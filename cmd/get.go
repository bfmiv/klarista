package cmd

import (
	"encoding/json"
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
		out := path.Join(os.TempDir(), name)
		pathOnly, _ := cmd.Flags().GetBool("path")

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
		})

		var result string
		var err error

		useRemoteState(name, stateBucketName, func() {
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
