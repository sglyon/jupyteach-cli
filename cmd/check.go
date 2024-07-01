/*
Copyright © 2024 Spencer Lyon spencerlyon2@gmail.com
*/
package cmd

import (
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate the structure of the directory and content of _course.yml and _lecture.yml files.",
	Long:  `TODO: long description`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("check called")
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			logger.Fatalf("Must provide a path")
		}
		createRepoZip(path)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// checkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// checkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
