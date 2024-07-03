package cmd

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"

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
			logger.Fatal(err)
		}
		defer zippedFile.Close()

		// Create all necessary directories in the path
		outputDir := filepath.Dir(outputFilePath)
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			logger.Fatal(err)
		}

		outputFile, err := os.OpenFile(outputFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			logger.Fatal(err)
		}
		defer outputFile.Close()

		// Copy the contents of the file to the current directory
		_, err = io.Copy(outputFile, zippedFile)
		if err != nil {
			logger.Fatal(err)
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
		logger.Fatal("No course slug provided and no slug found in _course.yml file. Please check _course.yml")
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
		logger.Fatal("API Key not set. Please run `jupyteach login`")
	}

	sha, err := git.GetLatestCommitSha(path)
	if err != nil {
		return fmt.Errorf("Error getting latest commit sha %e", err)
	}
	resp, errFinal := requestPostRecordSha(apiKey, baseURL, courseSlug, sha)
	if errFinal != nil {
		return fmt.Errorf("Error upating server with most recent sha %e", errFinal)
	}
	logger.Infof("Server updated with this info: %+v\n", resp)
	return nil
}

func commitAllAndUpdateServer(path, courseSlug, msg string) (committed, postedZip bool, err error) {
	committed, err = git.CommitAll(path, msg)

	if err != nil {
		return committed, postedZip, err
	}

	if committed {
		// TODO: need to DRY this out. Also repeated in push.go
		logger.Info("Successfully committed changes to local git repository")
		// get sha of latest commit
		if err := updateServerWithCommitSHA(path, courseSlug); err != nil {
			return committed, postedZip, err
		}

		// we made a commit, so we need to push to the server
		err = postRepoAsZip(path, courseSlug)
		if err == nil {
			postedZip = true
		}
	} else {
		logger.Info("Update successful. No changes to commit")
	}

	return committed, postedZip, err
}

// zipDirectory zips the given directory and all its subdirectories, returning the zip contents as a byte slice.
func createRepoZip(directory string) ([]byte, error) {

	files, errList := getFilesToZipForRepoPush(path)
	if errList != nil {
		return nil, errList
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	err := filepath.Walk(directory, func(filePath string, info os.FileInfo, err error) error {

		// strip prefix `directory` from filePath
		relFilePath := strings.TrimPrefix(filePath, directory+string(filepath.Separator))
		if directory != "." {
			relFilePath = strings.TrimPrefix(relFilePath, directory)
		}
		isInGitDir := strings.HasPrefix(relFilePath, ".git")
		_, isFileInMap := files[relFilePath]

		if !isFileInMap && !isInGitDir {
			logger.Debugf("Skipping %s", relFilePath)
			return nil
		}
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil

		}

		logger.Debugf("zipping: %s", filePath)

		// Create a zip header based on the file info
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name, err = filepath.Rel(directory, filePath)
		if err != nil {
			return err
		}
		header.Method = zip.Deflate

		// Create a writer for the zip file
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Open the file to be zipped
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Copy the file contents to the zip writer
		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		zipWriter.Close()
		return nil, err
	}

	err = zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func getFilesToZipForRepoPush(path string) (map[string]string, error) {
	out, err := git.ListFiles(path)
	if err != nil {
		return nil, fmt.Errorf("Unable to git ls-files: %e", err)
	}
	myMap := git.MakeFileChangeMap(out)
	myMap[".git"] = "A"
	return myMap, nil
}

func postRepoAsZip(path, courseSlug string) error {
	apiKey := viper.GetString("API_KEY")
	baseURL := viper.GetString("BASE_URL")
	if apiKey == "" {
		logger.Fatal("API Key not set. Please run `jupyteach login`")
	}

	// Now create a zip file
	zipBytes, err := createRepoZip(path)
	if err != nil {
		return fmt.Errorf("Error creating zip %e", err)
	}

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	// Add the zip file to the request
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="repo.zip"; filename="repo.zip"`)
	h.Set("Content-Type", "application/zip")
	zipPart, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("Error creating course.zip form item %e", err)
	}
	if _, err := zipPart.Write(zipBytes); err != nil {
		return fmt.Errorf("Error writing course.zip to form %e", err)
	}

	// Close the writer to finalize the multipart body
	if err := writer.Close(); err != nil {
		return fmt.Errorf("Error finalizing form %e", err)
	}

	url := fmt.Sprintf("%s/api/v1/course/%s/upload_git_dir", baseURL, courseSlug)
	req, err := http.NewRequest("POST", url, &buffer)
	if err != nil {
		return fmt.Errorf("Error creating request with body %e", err)
	}

	client := &http.Client{}
	req.Header.Add("Authorization", "Bearer "+apiKey)
	header := writer.FormDataContentType()
	req.Header.Set("Content-Type", header)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending request %e", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Error response from server: %s", resp.Status)
	}

	logger.Info("Pushed git repo to server")

	return nil

}

func cleanupFailure(path string) error {
	// delete the directory at path
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	return nil
}

func checkRespError(resp *http.Response) error {
	// copy the response body to a buffer so we can read it twice
	if resp.StatusCode >= 400 {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, resp.Body); err != nil {
			return err
		}
		errVal := make(map[string]interface{})
		err := json.Unmarshal(buf.Bytes(), &errVal)
		if err != nil {
			return err
		}
		return fmt.Errorf("Error making request: %+v", errVal)
	}
	return nil
}
