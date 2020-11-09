package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get <name> <path>",
	Short: "Get a file from the cluster state",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		requestedPath := args[1]
		localStateDir := path.Join(os.TempDir(), name)
		pathOnly, _ := cmd.Flags().GetBool("path")
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

		var result string

		useRemoteState(name, stateBucketName, func() {
			var err error
			if pathOnly {
				result, err = filepath.Abs(requestedPath)
				if err != nil {
					panic(err)
				}
			} else {
				var content []byte
				content, err = ioutil.ReadFile(requestedPath)
				if err != nil {
					panic(err)
				}
				result = string(content)
			}
		})

		fmt.Print(result)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().Bool("path", false, "Return file path rather than content")
}
