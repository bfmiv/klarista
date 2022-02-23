package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy <name>",
	Short: "Destroy an existing cluster",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		localStateDir := path.Join(os.TempDir(), name)
		stateBucketName := strings.ReplaceAll(name, ".", "-") + "-state"

		yes, _ := cmd.Flags().GetBool("yes")
		autoFlags := getAutoFlags(yes)

		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		if !rootCmd.PersistentFlags().Changed("input") {
			inputs = getInitialInputs(localStateDir)
		}

		writeAssets := createAssetWriter(pwd, localStateDir, assets)
		processInputs := createInputProcessor(pwd, localStateDir, assets, writeAssets)

		writeAssets()

		inputIds := processInputs(inputs)

		setAwsEnv(localStateDir, inputIds)

		if err = os.Setenv("KOPS_STATE_STORE", "s3://"+stateBucketName+"/kops"); err != nil {
			panic(err)
		}

		Logger.Infof(`Destroying cluster "%s"`, name)

		useRemoteState(name, stateBucketName, true, true, func() {
			if err = os.Setenv("CLUSTER", name); err != nil {
				panic(err)
			}

			if err = os.Setenv("KUBECONFIG", path.Join(localStateDir, "kubeconfig.yaml")); err != nil {
				panic(err)
			}

			useWorkDir("tf", func() {
				defer func() {
					if r := recover(); r != nil {
						Logger.Infof("Recovered: %s", r)
					}
				}()

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

				shell(
					"bash",
					"-c",
					`
						if kops get cluster $CLUSTER > /dev/null; then
							kops delete cluster $CLUSTER --yes
						fi
					`,
				)
			})

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

			shell("bash", "-c", "ls -a1 | tail -n +3 | xargs rm -rf")
		})
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().Bool("yes", false, "Skip confirmation")
}
