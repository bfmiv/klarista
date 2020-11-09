package cmd

import (
	"github.com/spf13/cobra"
)

var inputs []string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "klarista",
	Long:    `klarista is a command line tool that generates terraform modules for kops clusters`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		Logger.Error(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringArrayVarP(&inputs, "input", "i", []string{}, "Path(s) to the cluster input file(s)")
}
