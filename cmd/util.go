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
	"strings"
	"time"

	"github.com/charmbracelet/log"

	"gopkg.in/yaml.v2"
)

type PushGetResponse struct {
	LastCommitSha             string `json:"last_commit_sha"`
	RemoteChanges             bool   `json:"remote_changes"`
	SyncStatusUpdateTimestamp string `json:"sync_status_update_timestamp" yaml:"sync_status_update_timestamp"`
}

type CourseLectureYaml struct {
	CourseLectureID int    `yaml:"course_lecture_id,omitempty"`
	Directory       string `yaml:"directory,omitempty"`
	LectureID       int    `yaml:"lecture_id,omitempty"`
}

type CourseYaml struct {
	CourseType                string              `yaml:"course_type,omitempty"`
	EndDate                   string              `yaml:"end_date,omitempty"`
	ID                        int                 `yaml:"id,omitempty"`
	LastCommitSHA             string              `yaml:"last_commit_sha,omitempty"`
	Lectures                  []CourseLectureYaml `yaml:"lectures,omitempty"`
	Name                      string              `yaml:"name,omitempty"`
	Number                    string              `yaml:"number,omitempty"`
	Slug                      string              `yaml:"slug,omitempty"`
	StartDate                 string              `yaml:"start_date,omitempty"`
	SyncStatusUpdateTimestamp string              `yaml:"sync_status_update_timestamp,omitempty"`
}

type Question struct {
	// All
	ID           int      `yaml:"id,omitempty"`
	QuestionType string   `yaml:"question_type,omitempty"`
	QuestionText string   `yaml:"question_text,omitempty"`
	Topics       []string `yaml:"topics,omitempty"`
	Difficulty   string   `yaml:"difficulty,omitempty"`
	Points       int      `yaml:"points,omitempty"`
	Solution     string   `yaml:"solution,omitempty"`

	// SingleSelection or MultipleSelection
	Options []string `yaml:"options,omitempty"`

	// Code, Freeform, FillInBlank
	StartingCode string `yaml:"starting_code,omitempty"`

	// Code
	SetupCode string `yaml:"setup_code,omitempty"`
	TestCode  string `yaml:"test_code,omitempty"`
}

type Quiz struct {
	QuizID      int        `yaml:"quiz_id,omitempty"`
	MaxAttempts int        `yaml:"max_attempts,omitempty"`
	Topics      []string   `yaml:"topics,omitempty"`
	StartCode   string     `yaml:"start_code,omitempty"`
	Questions   []Question `yaml:"questions"`
}

type ContentBlockYaml struct {
	ContentBlockID   int      `yaml:"content_block_id,omitempty"`
	Description      string   `yaml:"description,omitempty"`
	Filename         string   `yaml:"filename,omitempty"`
	LectureContentID int      `yaml:"lecture_content_id,omitempty"`
	Title            string   `yaml:"title,omitempty"`
	Type             string   `yaml:"type,omitempty"`
	URL              string   `yaml:"url,omitempty"`
	VimeoVideoID     string   `yaml:"vimeo_video_id,omitempty"`
	YoutubeVideoID   string   `yaml:"youtube_video_id,omitempty"`
	NUploads         int      `yaml:"n_uploads,omitempty"`
	UploadExtensions []string `yaml:"upload_extensions,omitempty"`
	Quiz             Quiz     `yaml:"quiz,omitempty"`
}

type LectureYaml struct {
	ContentBlocks   []ContentBlockYaml `yaml:"content_blocks"`
	CourseLectureID int                `yaml:"course_lecture_id,omitempty"`
	Description     string             `yaml:"description,omitempty"`
	LectureID       int                `yaml:"lecture_id,omitempty"`
	Title           string             `yaml:"title,omitempty"`
}

func parseCourseYaml(dirname string) (*CourseYaml, error) {
	yamlPath := filepath.Join(dirname, "_course.yml")
	// Check if _course.yml exists
	_, errFile := os.Stat(yamlPath)
	if os.IsNotExist(errFile) {
		// Handle the case where the file does not exist
		log.Fatal("_course.yml does not exist. You must run `jupyteach pull {course_slug}`  to get course data")
	} else if errFile != nil {
		// Handle other errors, if any
		return nil, errFile
	}

	// Open the _course.yml file
	file, err := os.Open(yamlPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read the file content into a byte slice
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	// Parse the YAML data
	var course CourseYaml
	err = yaml.Unmarshal(byteValue, &course)
	if err != nil {
		return nil, err
	}

	return &course, nil
}

func (c CourseYaml) LastUpdateTimestamp() time.Time {
	t, err := time.Parse(time.RFC3339, c.SyncStatusUpdateTimestamp)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func parseLectureYaml(lectureYmlPath string) (*LectureYaml, error) {
	// Open the _course.yml file
	file, err := os.Open(lectureYmlPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file content into a byte slice
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	// Parse the YAML data
	var lecture LectureYaml
	err = yaml.Unmarshal(byteValue, &lecture)
	if err != nil {
		return nil, err
	}

	return &lecture, nil
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

type specForZip struct {
	Name, Path string
}

func (c CourseYaml) createZip(path string) ([]byte, error) {
	// first create a zip file in memory and load up the file at `path/_course.yml` and `path/syllabus.md`
	// then add all the files in the lectures directories
	// then return the zip file as a byte slice

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)
	// Create a new zip archive.
	w := zip.NewWriter(buf)

	// Add files to the archive.
	files := []specForZip{
		{"syllabus.md", filepath.Join(path, "syllabus.md")},
	}

	// loop over c.lectures
	for _, l := range c.Lectures {
		// now read `path/directory/_lecture.yml`
		lectureYamlPath := filepath.Join(path, l.Directory, "_lecture.yml")
		lecture, err := parseLectureYaml(lectureYamlPath)
		if err != nil {
			return nil, err
		}

		files = append(
			files,
			specForZip{
				Name: filepath.Join(l.Directory, "_lecture.yml"),
				Path: lectureYamlPath,
			},
		)

		// loop over lecture.ContentBlocks
		for _, cb := range lecture.ContentBlocks {
			if cb.Filename != "" {
				files = append(
					files,
					specForZip{
						Name: filepath.Join(l.Directory, cb.Filename),
						Path: filepath.Join(path, l.Directory, cb.Filename),
					},
				)
			}
		}
	}

	for _, file := range files {
		f, err := w.Create(file.Name)
		if err != nil {
			return nil, err
		}

		fileContent, err := os.ReadFile(file.Path)
		if err != nil {
			return nil, err
		}

		_, err = f.Write(fileContent)
		if err != nil {
			return nil, err
		}
	}

	// finally marshal the course yaml and write it to the zip file
	courseYamlBytes, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	f, err := w.Create("_course.yml")
	if err != nil {
		return nil, err
	}

	_, err = f.Write(courseYamlBytes)
	if err != nil {
		return nil, err
	}

	// Make sure to check the error on Close.

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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

func slugify(x string) string {
	// slugify a string. This will convert to lowercase and replace all spaces with hyphens

	// convert to lowercase
	x = strings.ToLower(x)
	// replace colon with nothing
	x = strings.ReplaceAll(x, ":", "")
	x = strings.ReplaceAll(x, "(", "")
	x = strings.ReplaceAll(x, ")", "")
	// replace spaces, parens  with hyphens
	x = strings.ReplaceAll(x, " ", "-")

	return x
}

func (c CourseYaml) checkLectureDirectories() error {
	// We need to make sure that for each `cl CourseLectureYaml` in `c.Lectures` that the
	// name is the slugified version of the title in `cl.Directory/_lecture.yml` file

	for _, cl := range c.Lectures {
		lectureYamlPath := filepath.Join(cl.Directory, "_lecture.yml")
		lecture, err := parseLectureYaml(lectureYamlPath)
		if err != nil {
			return err
		}
		// Check slugified version
		slug := slugify(lecture.Title)
		if cl.Directory != slug {
			return fmt.Errorf("Directory name %s does not match lecture title %s", cl.Directory, slug)
		}
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

	return nil
}
func getCourseSlug(args []string) string {
	// first check if we have a _course.yml file
	course, err := parseCourseYaml(".")
	if err == nil {
		return course.Slug
	}

	if len(args) == 0 {
		log.Fatal("No course slug provided")
	}

	return args[0]
}
