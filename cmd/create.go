package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

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

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new cluster",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		inputs, _ := cmd.Flags().GetStringArray("input")
		if !cmd.Flags().Changed("input") {
			inputs = []string{
				path.Join(name, "input.tfvars"),
			}
		}
		for i, input := range inputs {
			if !filepath.IsAbs(input) {
				inputs[i] = path.Join(pwd, input)
			}
		}

		out, _ := cmd.Flags().GetString("out")
		if !cmd.Flags().Changed("out") {
			out = name
		}

		fast, _ := cmd.Flags().GetBool("fast")

		yes, _ := cmd.Flags().GetBool("yes")
		autoFlags := getAutoFlags(yes)

		// These assets will only be written if they do not already exist
		politeAssets := map[string]bool{
			"kubeconfig.yaml": true,
		}

		writeAssets := func() {
			useWorkDir(pwd, func() {
				for _, file := range assets.List() {
					fp := path.Join(out, file)

					if politeAssets[file] {
						_, err := os.Stat(fp)
						if os.IsNotExist(err) {
							// noop
						} else {
							// The file exists; continue
							continue
						}
					}

					data, err := assets.Find(file)
					if err != nil {
						panic(err)
					}

					if err = os.MkdirAll(path.Dir(fp), 0755); err != nil {
						panic(err)
					}

					if err = ioutil.WriteFile(fp, data, 0644); err != nil {
						panic(err)
					}
				}
			})
		}

		Logger.Infof(`Applying changes to cluster "%s"`, name)
		Logger.Infof(
			"Reading input from [\n\t%s,\n]",
			strings.Join(inputs, ",\n\t"),
		)
		Logger.Infof(`Writing output to "%s"`, out)

		inputIds := make([]string, len(inputs))

		useWorkDir(out, func() {
			for i, input := range inputs {
				inputBytes, err := ioutil.ReadFile(input)
				if err != nil {
					panic(err)
				}
				inputIds[i] = fmt.Sprintf("%03d.tfvars", i)
				assets.AddBytes(
					path.Join("tf", "inputs", inputIds[i]),
					inputBytes,
				)
			}
			writeAssets()
		})

		useWorkDir(path.Join(out, "tf"), func() {
			// Clean up legacy files
			shell("bash", "-c", "rm -f locals.tf.json")

			shell("terraform", "init")

			shell(
				"bash",
				"-c",
				fmt.Sprintf(
					`terraform apply %s -var "cluster_name=%s" %s`,
					autoFlags,
					name,
					getVarFileFlags(inputIds),
				),
			)

			outputBytes, err := getOutputJSONBytes()
			if err != nil {
				panic(err)
			}

			var output map[string]interface{}
			err = json.Unmarshal(outputBytes, &output)
			if err != nil {
				panic(err)
			}

			assets.AddBytes(path.Join("tf", "output.json"), outputBytes)
			writeAssets()

			awsProfile := output["aws_profile"].(string)
			awsIamClusterAdminRoleArn := output["aws_iam_cluster_admin_role_arn"].(string)
			kopsStateBucket := "s3://" + output["kops_state_bucket"].(string)

			if err = os.Setenv("AWS_PROFILE", awsProfile); err != nil {
				panic(err)
			}

			if err = os.Setenv("CLUSTER", name); err != nil {
				panic(err)
			}

			if err = os.Setenv("KOPS_STATE_STORE", kopsStateBucket); err != nil {
				panic(err)
			}

			if err = os.Setenv("KOPS_FEATURE_FLAGS", "+TerraformJSON,-Terraform-0.12"); err != nil {
				panic(err)
			}

			if err = os.Setenv("KUBECONFIG", path.Join(pwd, out, "kubeconfig.yaml")); err != nil {
				panic(err)
			}

			var isNewCluster bool
			func() {
				defer func() {
					if r := recover(); r != nil {
						Logger.Debugf("Recovered: %s", r)
						isNewCluster = true
					}
				}()
				shell(
					"bash",
					"-c",
					"kops get cluster $CLUSTER &> /dev/null",
				)
			}()

			shell(
				"bash",
				"-c",
				fmt.Sprintf(
					`
						kops replace \
							%s \
							-f <(
								kops toolbox template \
									--name "$CLUSTER" \
									--set-string "cluster_name=$CLUSTER" \
									--values output.json \
									--template <(cat ../kops/*) \
									--format-yaml
							)
					`,
					func() string {
						if isNewCluster {
							// --force is required to replace a cluster that doesn't exist
							return "--force"
						}
						return ""
					}(),
				),
			)

			shell(
				"kops",
				"update",
				"cluster",
				name,
				func() string {
					if isNewCluster {
						return ""
					}
					return "--create-kube-config=false"
				}(),
				"--target", "terraform",
				"--out", ".",
				"--yes",
			)

			useWorkDir(pwd, func() {
				// Read the generated kops terraform
				kopsOutputFile := path.Join(out, "tf", "kubernetes.tf.json")
				kopsJSONBytes, err := ioutil.ReadFile(kopsOutputFile)
				if err != nil {
					panic(err)
				}

				var kopsJSON map[string]interface{}
				err = json.Unmarshal(kopsJSONBytes, &kopsJSON)
				if err != nil {
					panic(err)
				}

				// Remove providers from generated kops terraform
				// See https://discuss.hashicorp.com/t/terraform-v0-13-0-beta-program/9066/9
				delete(kopsJSON, "provider")
				// Remove duplicate output
				delete(kopsJSON["output"].(map[string]interface{}), "cluster_name")

				kopsJSONBytes, err = json.MarshalIndent(kopsJSON, "", "  ")
				if err != nil {
					panic(err)
				}

				err = ioutil.WriteFile(kopsOutputFile, kopsJSONBytes, 0644)
				if err != nil {
					panic(err)
				}
			})

			// Finish provisioning
			shell(
				"bash",
				"-c",
				fmt.Sprintf(
					`terraform apply -refresh=false %s -var "cluster_name=%s" %s`,
					autoFlags,
					name,
					getVarFileFlags(inputIds),
				),
			)

			// Write kops output
			outputBytes, err = getOutputJSONBytes()
			if err != nil {
				panic(err)
			}

			assets.AddBytes(path.Join("tf", "output.json"), outputBytes)
			writeAssets()

			if isNewCluster {
				Logger.Info("Waiting 3m for the cluster to come online")
				time.Sleep(3 * time.Minute)
			} else {
				shell(
					"bash",
					"-c",
					fmt.Sprintf(
						"kops rolling-update cluster %s %s --yes",
						name,
						func() string {
							if fast {
								return "--cloudonly"
							}
							return ""
						}(),
					),
				)
			}

			// Wait until the only validation failures are for aws-iam-authenticator
			for {
				validateCmd := exec.Command("kops", "validate", "cluster", name, "-o", "json")
				validateBytes, _ := validateCmd.Output()

				var validateJSON map[string]interface{}
				json.Unmarshal(validateBytes, &validateJSON)

				if validateJSON != nil {
					if validateJSON["failures"] == nil {
						break
					}

					failures := validateJSON["failures"].([]interface{})
					iamAuthenticatorFailureCount := 0

					for _, f := range failures {
						failure := f.(map[string]interface{})
						if strings.HasPrefix(failure["name"].(string), "kube-system/aws-iam-authenticator") {
							iamAuthenticatorFailureCount++
						}
					}

					if len(failures) == iamAuthenticatorFailureCount {
						break
					}
				}

				Logger.Info("Cluster validation failed, trying again in 30s")
				time.Sleep(30 * time.Second)
			}

			// Remove legacy kube-dns deployments (if migrating from kube-dns => core-dns)
			shell(
				"bash",
				"-c",
				`
					(
						if kubectl get deploy kube-dns -n kube-system; then
							kubectl delete deploy kube-dns-autoscaler kube-dns -n kube-system
						fi
					) &> /dev/null
				`,
			)

			// Create kubernetes resources
			shell(
				"bash",
				"-c",
				`
					kops toolbox template \
						--name "$CLUSTER" \
						--values output.json \
						--template <(cat ../k8s/*.yaml) \
						--format-yaml |
					kubectl apply -f -
				`,
			)

			// Create a new iam-authenticator user
			shell(
				"kubectl",
				"config",
				"set-credentials",
				name+".exec",
				"--exec-api-version", "client.authentication.k8s.io/v1alpha1",
				"--exec-command", "aws-iam-authenticator",
				"--exec-arg", "token",
				"--exec-arg", "-i",
				"--exec-arg", name,
				"--exec-arg", "-r",
				"--exec-arg", awsIamClusterAdminRoleArn,
			)

			// Update user in cluster context
			shell(
				"kubectl",
				"config",
				"set",
				fmt.Sprintf("contexts.%s.user", name),
				fmt.Sprintf("%s.exec", name),
			)
		})

		// Wait until the cluster is reachable with iam authenticator
		useWorkDir(pwd, func() {
			var isReady bool

			for {
				func() {
					defer func() {
						if r := recover(); r != nil {
							Logger.Debugf("Recovered: %s", r)
						} else {
							isReady = true
						}
					}()
					shell(
						"bash",
						"-c",
						"kubectl get pods -n kube-system -o name > /dev/null",
					)
				}()

				if isReady {
					break
				}

				Logger.Info("Cluster authentication failed, trying again in 30s")
				time.Sleep(30 * time.Second)
			}
		})

		// Cleanup template files
		useWorkDir(out, func() {
			shell("bash", "-c", "rm -rf k8s kops")
		})

		Logger.Info("☕️ Your cluster is ready!")
		Logger.Infof(`Output written to "%s"`, out)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().Bool("fast", false, "Apply updates as quickly as possible. This is not safe in production")
	createCmd.Flags().StringArrayP("input", "i", []string{"[name]/input.tfvars"}, "Path to the cluster input file(s)")
	createCmd.Flags().StringP("out", "o", "[name]", "Path to the klarista output directory")
	createCmd.Flags().Bool("yes", false, "Skip confirmation")
}
