/*
Copyright © 2024 Spencer Lyon spencerlyon2@gmail.com
*/
package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/sglyon/jupyteach/internal/git"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func doPull(path, courseSlug string) error {
	_, err := doPullOrClone(path, courseSlug, "pull")
	return err
}

func doPullOrClone(path, courseSlug, operation string) (int, error) {
	if operation != "pull" && operation != "clone" {
		return 1, errors.New("operation must be either 'pull' or 'clone'")
	}
	if operation == "pull" {
		git.CheckCleanFatal(path)
	}
	// We will have a bare directory if are to clone

	apiKey := viper.GetString("API_KEY")
	baseURL := viper.GetString("BASE_URL")
	if apiKey == "" {
		return 1, errors.New("API Key not set. Please run `jupyteach login`")
	}

	url := fmt.Sprintf("%s/api/v1/course/%s/%s", baseURL, courseSlug, operation)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 1, err
	}

	req.Header.Add("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return resp.StatusCode, err
	}
	defer resp.Body.Close()

	logger.Info("Response received", "statusCode", resp.StatusCode)

	if err := checkRespError(resp); err != nil {
		return resp.StatusCode, err
	}
	if err := git.WithDirectory(path, func() error {
		return unpackZipResponse(resp)
	}); err != nil {
		return resp.StatusCode, err
	}

	return resp.StatusCode, nil
}

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull {course_slug}",
	Short: "Pull changes from the Jupyteach application to local directory",
	Long:  `TODO: long description`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		courseSlug := getCourseSlug(args)
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			logger.Fatal("Must provide a path")
		}

		if err := doPull(path, courseSlug); err != nil {
			logger.Fatal(err)
		}

		logger.Info("Successfully pulled course contents.")

		if err := commitAllAndUpdateServer(path, courseSlug, "jupyteach cli pull response"); err != nil {
			logger.Fatal(err)
		}
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
