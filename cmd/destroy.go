package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy <name>",
	Short: "Destroy an existing cluster",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		out := path.Join(os.TempDir(), name)

		yes, _ := cmd.Flags().GetBool("yes")
		autoFlags := getAutoFlags(yes)

		Logger.Infof(`Destroying cluster "%s"`, name)

		stateBucketName := strings.ReplaceAll(name, ".", "-") + "-state"

		useWorkDir(path.Join(out, "tf_state"), func() {
			outputBytes, err := getOutputJSONBytes()
			if err != nil {
				panic(err)
			}

			var output map[string]interface{}
			err = json.Unmarshal(outputBytes, &output)
			if err != nil {
				panic(err)
			}

			awsProfile := output["aws_profile"].(string)
			awsRegion := output["aws_region"].(string)

			if err = os.Setenv("AWS_PROFILE", awsProfile); err != nil {
				panic(err)
			}

			if err = os.Setenv("AWS_REGION", awsRegion); err != nil {
				panic(err)
			}

			if err = os.Setenv("KOPS_STATE_STORE", "s3://"+stateBucketName+"/kops"); err != nil {
				panic(err)
			}
		})

		useRemoteState(name, stateBucketName, func() {
			inputFileInfo, err := ioutil.ReadDir("tf/inputs")
			if err != nil {
				panic(err)
			}

			inputIds := cast.ToStringSlice(
				funk.Map(inputFileInfo, func(file os.FileInfo) string {
					return file.Name()
				}),
			)

			if err = os.Setenv("CLUSTER", name); err != nil {
				panic(err)
			}

			if err = os.Setenv("KUBECONFIG", path.Join(out, "kubeconfig.yaml")); err != nil {
				panic(err)
			}

			useWorkDir("tf", func() {
				shell("terraform", "init")

				shell(
					"bash",
					"-c",
					fmt.Sprintf(
						`
							terraform destroy \
								-compact-warnings \
								-var "cluster_name=%s" \
								-var "state_bucket_name=%s" \
								%s \
								%s
						`,
						name,
						stateBucketName,
						autoFlags,
						getVarFileFlags(inputIds),
					),
				)
			})

			shell(
				"bash",
				"-c",
				`
					if kops get cluster $CLUSTER > /dev/null; then
						kops delete cluster $CLUSTER --yes
					fi
				`,
			)

			useWorkDir("tf_state", func() {
				shell("terraform", "init")

				shell(
					"bash",
					"-c",
					fmt.Sprintf(
						`
							terraform destroy \
								-compact-warnings \
								-var "cluster_name=%s" \
								-var "state_bucket_name=%s" \
								%s \
								%s
						`,
						name,
						stateBucketName,
						autoFlags,
						getVarFileFlags(inputIds),
					),
				)
			})

			shell("bash", "-c", "rm -rf *")
		})
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().Bool("yes", false, "Skip confirmation")
}
