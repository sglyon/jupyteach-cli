/*
Copyright © 2024 Spencer Lyon spencerlyon2@gmail.com
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/charmbracelet/log"

	"github.com/sglyon/jupyteach/internal/git"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push {course_slug}",
	Short: "Push local changes to the Jupyteach application",
	Long:  `TODO: long description`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Parse flags and config
		courseSlug := getCourseSlug(args)
		path, err := cmd.Flags().GetString("path")
		if err != nil {
			log.Fatalf("Must provide a path")
		}
		git.CheckCleanFatal(path)

		// Read the `sync_status_update_timestamp` field in `_course.yml`
		course, err := parseCourseYaml(path)
		if err != nil {
			log.Fatalf("Error parsing _course.yaml file %e", err)
		}

		if err := course.checkLectureDirectories(); err != nil {
			log.Fatalf("Error checking lecture directories %e", err)
		}

		// Now you can use course.SyncStatusUpdateTimestamp in your code...

		apiKey := viper.GetString("API_KEY")
		baseURL := viper.GetString("BASE_URL")
		if apiKey == "" {
			log.Fatalf("API Key not set. Please run `jupyteach login`")
		}

		pushGetResponse, err := requestGetPush(apiKey, baseURL, courseSlug)
		if err != nil {
			log.Fatalf("Error in GET `/.../push` %e", err)
		}

		log.Printf("Details: %+v", pushGetResponse)

		// // parse timestamp in form of "2024-03-28T18:05:41Z"
		// mostRecentUpdateTimestamp, err := time.Parse(time.RFC3339, pushGetResponse.SyncStatusUpdateTimestamp)
		// if err != nil {
		// 	log.Fatalf("Error parsing timestamp from GET `/.../push` response %e", err)
		// }

		// courseYamlUpdateTimestamp := course.LastUpdateTimestamp()
		// log.Printf("last timestamp: %+v", mostRecentUpdateTimestamp)
		// log.Printf("last timestamp: %+v", courseYamlUpdateTimestamp)

		if pushGetResponse.LastCommitSha != "" {
			// check if local commit is in history
			inHistory, _ := git.IsShaInHistory(path, pushGetResponse.LastCommitSha)
			if !inHistory {
				log.Fatalf("Latest commit known to server is not in local history. Use `git pull` pull to changes from remote first")
			}
		}

		// now check latest commit sha
		sha, err := git.GetLatestCommitSha(path)
		if err != nil {
			log.Fatalf("Error getting latest commit sha %e", err)
		}

		// now get list of all files that have changed
		changed, err := git.ChangesSinceCommit(path, course.LastCommitSHA)
		if err != nil {
			log.Fatalf("Error getting changes since last known commit sha %e", err)
		}

		course.LastCommitSHA = sha

		// Now create a zip file
		zipBytes, files, err := course.createZip(path)
		if err != nil {
			log.Fatalf("Error creating zip %e", err)
		}

		// filter changed to only include files that are in the zip
		filteredChanged := FilterChanged(changed, files)

		changedJsonBytes, err := json.Marshal(filteredChanged)
		if err != nil {
			log.Fatalf("Error encoding changes as json object %e", err)
		}

		// Finally, we need to POST the zip file and changesJSON to the server
		// using a multipart/form-data request
		// The server expects the zip file to be in a field called `zip` and the changesJSON
		// to be in a field called `changes`

		// Create a new buffer to write the zip file and changesJSON
		// to the request body
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)

		if err := writer.WriteField("latest_sha", sha); err != nil {
			log.Fatalf("Error writing latest_sha to form %e", err)
		}

		// Add the zip file to the request
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="course.zip"; filename="course.zip"`)
		h.Set("Content-Type", "application/zip")
		zipPart, err := writer.CreatePart(h)
		if err != nil {
			log.Fatalf("Error creating course.zip form item %e", err)
		}
		if _, err := zipPart.Write(zipBytes); err != nil {
			log.Fatalf("Error writing course.zip to form %e", err)
		}

		// Add the changed.json file to the request
		h = textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="changed.json"; filename="changed.json"`)
		h.Set("Content-Type", "application/json")
		jsonPart, err := writer.CreatePart(h)
		if err != nil {
			log.Fatalf("Error creating changed.json form item %e", err)
		}
		if _, err := jsonPart.Write(changedJsonBytes); err != nil {
			log.Fatalf("Error writing changed.json to form %e", err)
		}

		// Close the writer to finalize the multipart body
		if err := writer.Close(); err != nil {
			log.Fatalf("Error finalizing form %e", err)
		}

		url := fmt.Sprintf("%s/api/v1/course/%s/push", baseURL, courseSlug)
		req, err := http.NewRequest("POST", url, &buffer)
		if err != nil {
			log.Fatalf("Error creating request with body %e", err)
		}

		client := &http.Client{}
		req.Header.Add("Authorization", "Bearer "+apiKey)
		header := writer.FormDataContentType()
		log.Printf("Content-type Header: %s", header)
		req.Header.Set("Content-Type", header)
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error sending request %e", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			log.Fatalf("Error response from server: %s", resp.Status)
		}

		log.Infof("Response object: %+v", resp)

		// TODO 2024-05-02 15:53:11: check status code to make sure request was successful
		if err := unpackZipResponse(resp); err != nil {
			log.Fatal(err)
		}

		log.Printf("Pushed changes to server. Some metadata may have changed. ")

		committed, err := git.CommitAll(path, "jupyteach cli push response")
		if committed {
			log.Info("Successfully committed changes to local git repository")
		}
		if err != nil {
			log.Warn("Failed to create commit. Please use `git` manually to commit changes to repo in directory", "directory", path)
		}
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}
