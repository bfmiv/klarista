package cmd

import (
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state <command>",
	Short: "Manage klarista state",
	Args:  cobra.MinimumNArgs(1),
}

// statePushCmd represents the state push command
var statePushCmd = &cobra.Command{
	Use:   "push <name>",
	Short: "Push local klarista state to remote",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		localStateDir := path.Join(os.TempDir(), name)
		stateBucketName := strings.ReplaceAll(name, ".", "-") + "-state"

		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		if !rootCmd.PersistentFlags().Changed("input") {
			inputs = getInitialInputs(localStateDir)
		}

		writeAssets := createAssetWriter(pwd, localStateDir, assets)
		processInputs := createInputProcessor(pwd, localStateDir, assets, writeAssets)

		writeAssets("tf_vars/*")

		inputIds := processInputs(inputs)

		setAwsEnv(localStateDir, inputIds)

		useRemoteState(name, stateBucketName, false, true, func() {
			// noop
		})
	},
}

func init() {
	stateCmd.AddCommand(statePushCmd)
	rootCmd.AddCommand(stateCmd)
}
