package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new cluster",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		localStateDir := path.Join(os.TempDir(), name)
		stateBucketName := strings.ReplaceAll(name, ".", "-") + "-state"

		fast, _ := cmd.Flags().GetBool("fast")
		yes, _ := cmd.Flags().GetBool("yes")
		autoFlags := getAutoFlags(yes)

		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		if err = os.MkdirAll(localStateDir, 0755); err != nil {
			panic(err)
		}

		if !rootCmd.PersistentFlags().Changed("input") {
			inputs = getInitialInputs(localStateDir)
		}

		writeAssets := createAssetWriter(pwd, localStateDir, assets)
		processInputs := createInputProcessor(pwd, localStateDir, assets, writeAssets)

		Logger.Infof(`Applying changes to cluster "%s"`, name)

		writeAssets("{tf_vars,tf_state}/*")

		inputIds := processInputs(inputs)

		setAwsEnv(localStateDir, inputIds)

		useWorkDir(path.Join(localStateDir, "tf_state"), func() {
			shell("terraform", "init")

			shell(
				"bash",
				"-c",
				fmt.Sprintf(
					`terraform apply -auto-approve -compact-warnings -var "cluster_name=%s" -var "state_bucket_name=%s" %s`,
					name,
					stateBucketName,
					getVarFileFlags(inputIds),
				),
			)
		})

		writeAssets()

		Logger.Infof(`Writing output to "s3://%s"`, stateBucketName)

		useRemoteState(name, stateBucketName, func() {
			useWorkDir("tf", func() {
				writeAssets()

				shell("terraform", "init", "-upgrade")

				shell(
					"bash",
					"-c",
					fmt.Sprintf(
						`terraform apply %s -compact-warnings -var "cluster_name=%s" -var "state_bucket_name=%s" %s`,
						autoFlags,
						name,
						stateBucketName,
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

				awsIamClusterAdminRoleArn := output["aws_iam_cluster_admin_role_arn"].(string)

				if err = os.Setenv("CLUSTER", name); err != nil {
					panic(err)
				}

				if err = os.Setenv("KOPS_STATE_STORE", "s3://"+stateBucketName+"/kops"); err != nil {
					panic(err)
				}

				if err = os.Setenv("KOPS_FEATURE_FLAGS", "+TerraformJSON,-TerraformManagedFiles"); err != nil {
					panic(err)
				}

				if err = os.Setenv("KUBECONFIG", path.Join(localStateDir, "kubeconfig.yaml")); err != nil {
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
						// --force is required to replace a cluster that doesn't exist
						// or to create a new node group in an existing cluster
						"--force",
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
					func() string {
						if isDebug() {
							return "-v7"
						}
						return ""
					}(),
				)

				useWorkDir(pwd, func() {
					// Read the generated kops terraform
					kopsOutputFile := path.Join(localStateDir, "tf", "kubernetes.tf.json")
					kopsJSONBytes, err := ioutil.ReadFile(kopsOutputFile)
					if err != nil {
						panic(err)
					}

					var kopsJSON map[string]interface{}
					err = json.Unmarshal(kopsJSONBytes, &kopsJSON)
					if err != nil {
						panic(err)
					}

					// Remove duplicate output
					delete(kopsJSON["output"].(map[string]interface{}), "cluster_name")

					// Remove providers from generated kops terraform
					// See https://discuss.hashicorp.com/t/terraform-v0-13-0-beta-program/9066/9
					delete(kopsJSON, "provider")

					// Remove duplicate terraform
					delete(kopsJSON, "terraform")

					// Get terraform json output
					outputJSON, err := getOutputJSON()
					if err != nil {
						panic(err)
					}

					kopsResources := kopsJSON["resource"].(map[string]interface{})

					// Enable root volume encryption
					// kops <= 1.19
					if kopsResources["aws_launch_configuration"] != nil {
						launchConfigs := kopsResources["aws_launch_configuration"].(map[string]interface{})
						for _, lc := range launchConfigs {
							rootVolume := lc.(map[string]interface{})["root_block_device"].(map[string]interface{})
							rootVolume["encrypted"] = true
							if outputJSON["encryption_key_arn"] != nil {
								rootVolume["kms_key_id"] = outputJSON["encryption_key_arn"]
							}
						}
					}

					// Enable root volume encryption
					// kops >= 1.20
					if kopsResources["aws_launch_template"] != nil {
						launchTemplates := kopsResources["aws_launch_template"].(map[string]interface{})
						for _, lt := range launchTemplates {
							blockDeviceMappings := lt.(map[string]interface{})["block_device_mappings"].([]interface{})
							for _, bd := range blockDeviceMappings {
								ebs := bd.(map[string]interface{})["ebs"].([]interface{})
								for _, vol := range ebs {
									volume := vol.(map[string]interface{})
									volume["encrypted"] = true
									if outputJSON["encryption_key_arn"] != nil {
										volume["kms_key_id"] = outputJSON["encryption_key_arn"]
									}
								}
							}
						}
					}

					// Remove extraneous type property
					// kops >= 1.22
					if kopsResources["aws_route53_record"] != nil {
						route53Records := kopsResources["aws_route53_record"].(map[string]interface{})
						for _, r := range route53Records {
							if r.(map[string]interface{})["alias"] != nil {
								alias := r.(map[string]interface{})["alias"].(map[string]interface{})
								delete(alias, "type")
							}
						}
					}

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
						`terraform apply -refresh=false %s -compact-warnings -var "cluster_name=%s" -var "state_bucket_name=%s" %s`,
						autoFlags,
						name,
						stateBucketName,
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
							"kops rolling-update cluster %s %s %s --yes",
							name,
							func() string {
								if fast {
									return "--cloudonly"
								}
								return ""
							}(),
							func() string {
								if isDebug() {
									return "-v7"
								}
								return ""
							}(),
						),
					)
				}

				// Wait until the only validation failures are for aws-iam-authenticator
				for {
					validateArgs := []string{
						"validate",
						"cluster",
						name,
						"-o",
						"json",
					}
					if isDebug() {
						validateArgs = append(validateArgs, "-v7")
					}
					validateCmd := exec.Command("kops", validateArgs...)
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

		useRemoteState(name, stateBucketName, func() {
			shell(
				"bash",
				"-c",
				`
					# Remove default users from kubeconfig
					# Original kubeconfig can be recovered with kops
					# See https://kops.sigs.k8s.io/cli/kops_export_kubecfg/
					kubectl config unset users.$CLUSTER
					kubectl config unset users.$CLUSTER-basic-auth
				`,
			)
		})

		Logger.Info("☕️ Your cluster is ready!")
		Logger.Infof(`Output written to "%s"`, localStateDir)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().Bool("fast", false, "Apply updates as quickly as possible. This is not safe in production")
	createCmd.Flags().Bool("yes", false, "Skip confirmation")
}
