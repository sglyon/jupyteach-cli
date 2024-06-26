package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/sglyon/jupyteach/internal/git"
	"github.com/sglyon/jupyteach/internal/model"
	"github.com/spf13/viper"

	"gopkg.in/yaml.v2"
)

type PushGetResponse struct {
	LastCommitSha             string `json:"last_commit_sha"`
	RemoteChanges             bool   `json:"remote_changes"`
	SyncStatusUpdateTimestamp string `json:"sync_status_update_timestamp" yaml:"sync_status_update_timestamp"`
}

func requestGetPush(apiKey, baseURL, courseSlug string) (*PushGetResponse, error) {
	// First do a get request to see if there are any changes
	url := fmt.Sprintf("%s/api/v1/course/%s/push", baseURL, courseSlug)
	client := &http.Client{}
	req1, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req1.Header.Add("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req1)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// we need to Unmarshal into a struct
	var pushGetResponse PushGetResponse
	err = json.Unmarshal(bodyBytes, &pushGetResponse)
	if err != nil {
		return nil, err
	}

	return &pushGetResponse, nil
}

func requestPostRecordSha(apiKey, baseURL, courseSlug, sha string) (*PushGetResponse, error) {
	// First do a get request to see if there are any changes
	url := fmt.Sprintf("%s/api/v1/course/%s/response_commit_sha", baseURL, courseSlug)
	client := &http.Client{}

	body := map[string]string{"response_sha": sha}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req1, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	req1.Header.Add("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}

	req1.Header.Add("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req1)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// we need to Unmarshal into a struct
	var pushGetResponse PushGetResponse
	err = json.Unmarshal(bodyBytes, &pushGetResponse)
	if err != nil {
		return nil, err
	}

	return &pushGetResponse, nil
}

func ListFilesInDirectory(path string, extensions []string) ([]string, error) {
	var files []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// check that path ends in one of the given extensions
		if len(extensions) > 0 {
			for _, ext := range extensions {
				if filepath.Ext(path) == ext {
					files = append(files, path)
				}
			}
		} else {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func writeYaml(path string, data interface{}) error {
	// write the data to a file at path
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, yamlBytes, 0o644); err != nil {
		return err
	}

	return nil
}

func unpackZip(zipReader *zip.Reader) error {
	// Iterate through the files in the archive,
	// creating them in the current directory
	for _, file := range zipReader.File {
		outputFilePath := filepath.Join(".", file.Name)

		if file.FileInfo().IsDir() {
			// Create directory
			if err := os.MkdirAll(outputFilePath, file.Mode()); err != nil {
				return err
			}
			continue
		}

		// Open the file inside the zip archive
		zippedFile, err := file.Open()
		if err != nil {
			log.Fatal(err)
		}
		defer zippedFile.Close()

		// Create all necessary directories in the path
		outputDir := filepath.Dir(outputFilePath)
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			log.Fatal(err)
		}

		outputFile, err := os.OpenFile(outputFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			log.Fatal(err)
		}
		defer outputFile.Close()

		// Copy the contents of the file to the current directory
		_, err = io.Copy(outputFile, zippedFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func unpackZipResponse(resp *http.Response) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyReader := bytes.NewReader(bodyBytes)
	zipReader, err := zip.NewReader(bodyReader, int64(bodyReader.Len()))
	if err != nil {
		return err
	}

	if err := unpackZip(zipReader); err != nil {
		return err
	}

	// Unmarshal and remarshall the course yaml to make sure it is formatted correctly
	course, err := model.ParseCourseYaml(".")
	if err != nil {
		return err
	}

	courseYamlBytes, err := yaml.Marshal(course)
	if err != nil {
		return err
	}

	if err := os.WriteFile("_course.yml", courseYamlBytes, 0o644); err != nil {
		return err
	}

	// now do the same for all `_lecture.yml` files
	for _, cl := range course.Lectures {
		lectureYamlPath := filepath.Join(".", cl.Directory, "_lecture.yml")
		lecture, err := model.ParseLectureYaml(lectureYamlPath)
		if err != nil {
			return err
		}

		lectureYamlBytes, err := yaml.Marshal(lecture)
		if err != nil {
			return err
		}

		if err := os.WriteFile(lectureYamlPath, lectureYamlBytes, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func getCourseSlug(args []string) string {
	// first check if we have a _course.yml file
	course, err := model.ParseCourseYaml(".")
	if err == nil {
		return course.Slug
	}

	if len(args) == 0 {
		log.Fatal("No course slug provided")
	}

	return args[0]
}

func FilterChanged(changed map[string]string, files []model.SpecForZip) map[string]string {
	// filter the changed map to only include files that appear in `files`
	filtered := make(map[string]string)
	for _, file := range files {
		if _, ok := changed[file.Path]; ok {
			filtered[file.Path] = changed[file.Path]
		}
	}
	return filtered
}

func updateServerWithCommitSHA(path, courseSlug string) error {
	// get sha of latest commit
	apiKey := viper.GetString("API_KEY")
	baseURL := viper.GetString("BASE_URL")
	if apiKey == "" {
		log.Fatal("API Key not set. Please run `jupyteach login`")
	}

	sha, err := git.GetLatestCommitSha(path)
	if err != nil {
		return fmt.Errorf("Error getting latest commit sha %e", err)
	}
	resp, errFinal := requestPostRecordSha(apiKey, baseURL, courseSlug, sha)
	if errFinal != nil {
		return fmt.Errorf("Error upating server with most recent sha %e", errFinal)
	}
	log.Infof("Server updated with this info: %+v\n", resp)
	return nil
}

func commitAllAndUpdateServer(path, courseSlug string) error {
	committed, err := git.CommitAll(path, "jupyteach cli pull response")

	if err != nil {
		return err
	}

	if committed {
		// TODO: need to DRY this out. Also repeated in push.go
		log.Info("Successfully committed changes to local git repository")
		// get sha of latest commit
		if err := updateServerWithCommitSHA(path, courseSlug); err != nil {
			return err
		}
	} else {
		log.Info("Update successful. No changes to commit")
	}

	return nil
}
