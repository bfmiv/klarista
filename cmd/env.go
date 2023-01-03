package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env <name>",
	Short: "Print the cluster environment",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		stateBucketName := strings.ReplaceAll(name, ".", "-") + "-state"
		localStateDir := path.Join(os.TempDir(), name)
		localEnvFile := path.Join(localStateDir, ".env")

		readEnv := func() string {
			b, err := ioutil.ReadFile(localEnvFile)
			if err != nil {
				panic(err)
			}
			return string(b)
		}

		var result string

		_, err := os.Stat(localEnvFile)
		if err == nil {
			result = readEnv()
		}

		if result == "" {
			pwd, err := os.Getwd()
			if err != nil {
				panic(err)
			}

			if !rootCmd.PersistentFlags().Changed("input") {
				inputs = getInitialInputs(localStateDir)
			}

			assetWriter := NewAssetWriter(pwd, localStateDir, assets)
			inputProcessor := NewInputProcessor(assetWriter)

			assetWriter.Digest("{tf_vars,tf_state}/*")

			inputIds := inputProcessor.Digest(inputs)

			setAwsEnv(localStateDir, inputIds)

			useRemoteState(name, stateBucketName, true, false, func() {
				_, err := os.Stat(localEnvFile)
				if err == nil {
					result = readEnv()
				}
			})
		}

		if result == "" {
			result = string(generateDefaultEnvironmentFile(name))
		}

		fmt.Print(result)
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
