/*
Copyright © 2024 Spencer Lyon spencerlyon2@gmail.com
*/
package cmd

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"

	"github.com/sglyon/jupyteach/internal/git"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull {course_slug}",
	Short: "Pull changes from the Jupyteach application to local directory",
	Long:  `TODO: long description`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		courseSlug := args[0]
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			log.Fatal("Must provide a path")
		}
		git.CheckCleanFatal(path)

		apiKey := viper.GetString("API_KEY")
		baseURL := viper.GetString("BASE_URL")
		if apiKey == "" {
			log.Fatal("API Key not set. Please run `jupyteach login`")
		}

		url := fmt.Sprintf("%s/api/v1/course/%s/pull", baseURL, courseSlug)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Add("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		log.Info("Response received", "statusCode", resp.StatusCode)

		if err := unpackZipResponse(resp); err != nil {
			log.Fatal(err)
		}

		log.Info("Successfully pulled course contents. Please use `git` commands to save changes.")
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pullCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pullCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
