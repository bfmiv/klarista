package cmd

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/mholt/archiver/v3"
	"github.com/spf13/cast"
	"github.com/thanhpk/randstr"
	"github.com/thoas/go-funk"
)

// Version - klarista cli version
var Version = "latest"

func getAutoFlags(override bool) string {
	if override || os.Getenv("CI") != "" {
		return "-auto-approve"
	}
	return ""
}

func getVarFileFlags(inputIds []string) string {
	return strings.Join(
		cast.ToStringSlice(
			funk.Map(inputIds, func(id string) string {
				return fmt.Sprintf(`-var-file "inputs/%s"`, id)
			}),
		),
		" ",
	)
}

func getOutputJSONBytes() ([]byte, error) {
	command := exec.Command("terraform", "output", "-json")
	outputBytes, err := command.Output()
	if err != nil {
		return nil, err
	}

	var output map[string]interface{}
	err = json.Unmarshal(outputBytes, &output)
	if err != nil {
		return nil, err
	}

	for key, value := range output {
		output[key] = value.(map[string]interface{})["value"]
	}

	outputBytes, err = json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, err
	}

	return outputBytes, nil
}

func shell(command string, args ...string) {
	filteredArgs := funk.Compact(args).([]string)
	cmd := exec.Command(command, filteredArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	sigs := make(chan os.Signal)
	done := make(chan bool)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if <-done {
			return
		}

		sig := <-sigs
		Logger.Debug("GOT SIGNAL ", sig)

		if cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
			if err := cmd.Process.Kill(); err != nil {
				Logger.Fatal("Failed to kill process: ", err)
			}
		}
	}()

	Logger.Debugf("%s %s", command, strings.Join(filteredArgs, " "))

	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	done <- true
}

func useWorkDir(wd string, cb func()) {
	// Get the pwd
	originalWd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	wd, err = filepath.Abs(wd)
	if err != nil {
		panic(err)
	}

	if wd == originalWd {
		Logger.Debugf(`Already in WD %s`, wd)
		cb()
	} else {
		// Change to the target wd
		Logger.Debugf(`Using WD %s`, wd)
		if err = os.Chdir(wd); err != nil {
			panic(err)
		}

		defer func() {
			// Return to the original wd
			Logger.Debugf(`Returning to WD %s`, originalWd)
			if err = os.Chdir(originalWd); err != nil {
				panic(err)
			}
		}()

		// Do work
		cb()
	}
}

// UseTempDirCallback - useTempDir callback function
type UseTempDirCallback = func(string)

func useTempDir(args ...interface{}) {
	var autoremove bool = true
	var cb UseTempDirCallback
	var name string

	for _, v := range args {
		switch arg := v.(type) {
		case bool:
			autoremove = arg
		case string:
			name = arg
		default:
			cb = func(tmpdir string) {
				fn := reflect.ValueOf(arg)
				fnArgs := []reflect.Value{}
				if fn.Type().NumIn() == 1 {
					fnArgs = append(fnArgs, reflect.ValueOf(tmpdir))
				}
				reflect.ValueOf(arg).Call(fnArgs)
			}
		}
	}

	if name == "" {
		name = randstr.Hex(8)
	}

	tmpdir := path.Join(os.TempDir(), name)
	if err := os.MkdirAll(tmpdir, 0755); err != nil {
		panic(err)
	}

	if autoremove {
		defer os.RemoveAll(tmpdir)
	}

	useWorkDir(tmpdir, func() {
		cb(tmpdir)
	})
}

func useRemoteState(clusterName, bucket string, cb func()) {
	remoteStateKey := "klarista.state.tar"

	sess := session.Must(session.NewSession())
	uploader := s3manager.NewUploader(sess)
	downloader := s3manager.NewDownloader(sess)

	useTempDir(func(stateTmpDir string) {
		localStateFilePath := path.Join(stateTmpDir, remoteStateKey)

		stateFile, err := os.Create(localStateFilePath)
		if err != nil {
			panic(fmt.Errorf("Failed to create file %q, %v", localStateFilePath, err))
		}

		_, err = downloader.Download(stateFile, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(remoteStateKey),
		})

		useTempDir(clusterName, false, func() {
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case s3.ErrCodeNoSuchKey:
						Logger.Warn(aerr.Error())
						// noop
					default:
						panic(aerr.Error())
					}
				} else {
					panic(fmt.Errorf("Failed to download file, %v", err))
				}
			} else {
				Logger.Debugf("Reading state from s3://%s/%s", bucket, remoteStateKey)
				tar := &archiver.Tar{
					// ContinueOnError:        true,
					ImplicitTopLevelFolder: false,
					MkdirAll:               true,
					OverwriteExisting:      true,
					StripComponents:        0,
				}
				if err := tar.Unarchive(localStateFilePath, "."); err != nil {
					panic(err)
				}
			}

			defer func() {
				type ArchiveElement struct {
					Body []byte
					Path string
					Size int64
				}

				var archiveContent []*ArchiveElement

				filepath.Walk(".", func(fpath string, info os.FileInfo, err error) error {
					if err != nil {
						panic(err)
					}

					if info.IsDir() {
						if info.Name() == ".terraform" {
							return filepath.SkipDir
						}
					} else {
						if strings.HasSuffix(info.Name(), ".backup") {
							return filepath.SkipDir
						}

						body, err := ioutil.ReadFile(fpath)
						if err != nil {
							panic(err)
						}

						archiveContent = append(archiveContent, &ArchiveElement{
							Body: body,
							Path: fpath,
							Size: info.Size(),
						})
					}

					return nil
				})

				var data bytes.Buffer
				writer := tar.NewWriter(&data)

				for _, file := range archiveContent {
					hdr := &tar.Header{
						Name: file.Path,
						Mode: 0755,
						Size: file.Size,
					}
					if err := writer.WriteHeader(hdr); err != nil {
						panic(err)
					}
					if _, err := writer.Write(file.Body); err != nil {
						panic(err)
					}
				}
				if err := writer.Close(); err != nil {
					panic(err)
				}
				if err := stateFile.Truncate(0); err != nil {
					panic(err)
				}
				if _, err := stateFile.Write(data.Bytes()); err != nil {
					panic(err)
				}
				if _, err := stateFile.Seek(0, 0); err != nil {
					panic(err)
				}

				Logger.Infof("Writing state to s3://%s/%s", bucket, remoteStateKey)

				result, err := uploader.Upload(&s3manager.UploadInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(remoteStateKey),
					Body:   stateFile,
				})

				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						switch aerr.Code() {
						case s3.ErrCodeNoSuchBucket:
							Logger.Warn(aerr.Error())
							// noop
						default:
							panic(aerr.Error())
						}
					} else {
						panic(fmt.Errorf("Failed to upload file, %v", err))
					}
				} else {
					Logger.Infof("State written successfully to %s", result.Location)
				}
			}()

			cb()
		})
	})
}
