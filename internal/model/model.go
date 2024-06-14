package model

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

type CourseLectureYaml struct {
	CourseLectureID int    `yaml:"course_lecture_id,omitempty"`
	Directory       string `yaml:"directory,omitempty"`
	LectureID       int    `yaml:"lecture_id,omitempty"`
	AvailableAt     string `yaml:"available_at,omitempty"`
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
	CLIDirectoryWordSeparator string              `yaml:"cli_directory_word_separator,omitempty"`
	CLICommitSHA              string              `yaml:"cli_commit_sha,omitempty"`
}

var CourseTypes = [...]string{"semester", "ongoing", "mooc"}

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

var (
	ContentBlockTypes = [...]string{"video", "notebook", "markdown", "link", "quiz"}
	VideoSources      = [...]string{"youtube", "vimeo", "url"}
)

type LectureYaml struct {
	ContentBlocks   []ContentBlockYaml `yaml:"content_blocks"`
	CourseLectureID int                `yaml:"course_lecture_id,omitempty"`
	Description     string             `yaml:"description,omitempty"`
	LectureID       int                `yaml:"lecture_id,omitempty"`
	Title           string             `yaml:"title,omitempty"`
}

func ParseCourseYaml(dirname string) (*CourseYaml, error) {
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

func (c CourseYaml) Sep() string {
	if c.CLIDirectoryWordSeparator != "" {
		return c.CLIDirectoryWordSeparator
	}
	return "-"
}

func ParseLectureYaml(lectureYmlPath string) (*LectureYaml, error) {
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

type SpecForZip struct {
	Name, Path string
}

func (c *CourseYaml) WriteYaml(dirname string) error {
	// finally marshal the course yaml and write it to the zip file
	courseYamlBytes, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	path := filepath.Join(dirname, "_course.yml")
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := f.Write(courseYamlBytes); err != nil {
		return err
	}
	return nil
}

func (c *CourseYaml) CreateZip(path string) ([]byte, []SpecForZip, error) {
	// first create a zip file in memory and load up the file at `path/_course.yml` and `path/syllabus.md`
	// then add all the files in the lectures directories
	// then return the zip file as a byte slice

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)
	// Create a new zip archive.
	w := zip.NewWriter(buf)

	// Add files to the archive.
	files := []SpecForZip{
		{"syllabus.md", filepath.Join(path, "syllabus.md")},
	}

	log.Printf("These are the lectures %+v\n\n", c.Lectures)

	// loop over c.lectures
	for _, l := range c.Lectures {
		// now read `path/directory/_lecture.yml`
		lectureYamlPath := filepath.Join(path, l.Directory, "_lecture.yml")
		lecture, err := ParseLectureYaml(lectureYamlPath)
		if err != nil {
			return nil, nil, err
		}

		files = append(
			files,
			SpecForZip{
				Name: filepath.Join(l.Directory, "_lecture.yml"),
				Path: lectureYamlPath,
			},
		)

		// loop over lecture.ContentBlocks
		for _, cb := range lecture.ContentBlocks {
			if cb.Filename != "" {
				files = append(
					files,
					SpecForZip{
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
			return nil, nil, err
		}

		fileContent, err := os.ReadFile(file.Path)
		if err != nil {
			return nil, nil, err
		}

		_, err = f.Write(fileContent)
		if err != nil {
			return nil, nil, err
		}
	}

	// finally marshal the course yaml and write it to the zip file
	courseYamlBytes, err := yaml.Marshal(c)
	if err != nil {
		return nil, nil, err
	}

	f, err := w.Create("_course.yml")
	if err != nil {
		return nil, nil, err
	}

	_, err = f.Write(courseYamlBytes)
	if err != nil {
		return nil, nil, err
	}

	// Make sure to check the error on Close.

	if err := w.Close(); err != nil {
		return nil, nil, err
	}

	return buf.Bytes(), files, nil
}

func (c *CourseYaml) CheckLectureDirectories() error {
	// We need to make sure that for each `cl CourseLectureYaml` in `c.Lectures` that the
	// name is the slugified version of the title in `cl.Directory/_lecture.yml` file

	for _, cl := range c.Lectures {
		lectureYamlPath := filepath.Join(".", cl.Directory, "_lecture.yml")
		lecture, err := ParseLectureYaml(lectureYamlPath)
		if err != nil {
			return err
		}
		// Check slugified version
		slug := Slugify(lecture.Title, c.Sep())
		if cl.Directory != slug {
			return fmt.Errorf("Directory name %s does not match lecture title %s", cl.Directory, slug)
		}
	}
	return nil
}
