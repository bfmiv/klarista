package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy [name]",
	Short: "Destroy an existing cluster",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		modulePath := path.Join(name, "tf")

		yes, _ := cmd.Flags().GetBool("yes")
		autoFlags := getAutoFlags(yes)

		Logger.Infof(`Destroying cluster "%s"`, name)

		useWorkDir(modulePath, func() {
			inputFileInfo, err := ioutil.ReadDir("inputs")
			if err != nil {
				panic(err)
			}

			inputIds := cast.ToStringSlice(
				funk.Map(inputFileInfo, func(file os.FileInfo) string {
					return file.Name()
				}),
			)

			outputBytes, err := ioutil.ReadFile("output.json")
			if err != nil {
				panic(err)
			}

			var output map[string]interface{}
			err = json.Unmarshal(outputBytes, &output)
			if err != nil {
				panic(err)
			}

			awsProfile := output["aws_profile"].(string)
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

			if err = os.Setenv("KUBECONFIG", path.Join("..", "kubeconfig.yaml")); err != nil {
				panic(err)
			}

			shell("terraform", "init")

			// Destroy everything except the kops state bucket
			shell(
				"bash",
				"-c",
				fmt.Sprintf(
					`
						terraform destroy %s \
							-var "cluster_name=%s" \
							%s \
							$(terraform state list | grep -v kops_state | awk '{ print "-target=" $0 }' | tr '\n' ' ')
					`,
					autoFlags,
					name,
					getVarFileFlags(inputIds),
				),
			)

			// Destroy kops cluster
			shell(
				"bash",
				"-c",
				`
					if kops get cluster $CLUSTER > /dev/null; then
						kops delete cluster $CLUSTER --yes
					fi
				`,
			)

			// Finish destroying
			shell(
				"bash",
				"-c",
				fmt.Sprintf(
					`
						terraform destroy \
							-auto-approve \
							-var "cluster_name=%s" \
							%s
					`,
					name,
					getVarFileFlags(inputIds),
				),
			)
		})

		// Cleanup
		shell(
			"bash",
			"-c",
			fmt.Sprintf(
				`rm -rf "%s" "%s" "%s"`,
				modulePath,
				path.Join(path.Dir(modulePath), "kubeconfig.yaml"),
				path.Join(path.Dir(modulePath), "*.tfstate*"),
			),
		)
	},
}

func init() {
	rootCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().Bool("yes", false, "Skip confirmation")
}
