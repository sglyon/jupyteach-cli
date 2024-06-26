/*
Copyright © 2024 Spencer Lyon <spencerlyon2@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/sglyon/jupyteach/internal/model"
	"github.com/spf13/cobra"
)

type CreateOptions struct {
	Type string
}
type CommonOptions struct {
	Title       string
	Description string
}

type LectureOptions struct {
	Directory   string
	AvailableAt string
	CommonOptions
}

type NotebookOptions struct {
	CommonOptions
}

func createLecture() error {
	var lectureOptions LectureOptions
	lectureOptions.AvailableAt = time.Now().Format(time.RFC3339)

	// Ensure `_course.yml` exists
	courseMetadata, err := model.ParseCourseYaml(".")
	if err != nil {
		return err
	}
	sep := courseMetadata.Sep()

	lectureForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Lecture Title").Value(&lectureOptions.Title),
			huh.NewInput().Title("Lecture Description").Value(&lectureOptions.Description),
			huh.NewInput().
				Title("Available At").
				Value(&lectureOptions.AvailableAt).
				Suggestions([]string{time.Now().Format(time.RFC3339)}).
				Validate(func(s string) error {
					_, err := time.Parse(time.RFC3339, s)
					if err != nil {
						return errors.New("Invalid date format. Must be in RFC3339 format (e.g. 2024-03-28T18:05:41Z)")
					}
					return nil
				}),
		),
	)

	if err := lectureForm.Run(); err != nil {
		return err
	}

	lectureOptions.Directory = model.Slugify(lectureOptions.Title, sep)

	// Make sure directory doesn't already exist
	if _, err := os.Stat(lectureOptions.Directory); !os.IsNotExist(err) {
		return fmt.Errorf("Directory %s already exists", lectureOptions.Directory)
	}

	// Create directory
	if err := os.Mkdir(lectureOptions.Directory, 0o755); err != nil {
		return err
	}

	newLecture := model.LectureYaml{
		Title:         lectureOptions.Title,
		Description:   lectureOptions.Description,
		ContentBlocks: []model.ContentBlockYaml{},
	}

	// Write newLecture to lectureOptions.Directory/_lecture.yml
	lecturePath := fmt.Sprintf("%s/_lecture.yml", lectureOptions.Directory)
	if err := writeYaml(lecturePath, newLecture); err != nil {
		return err
	}

	// Add this lecture to _course.yml
	newCourseLecture := model.CourseLectureYaml{
		Directory:   lectureOptions.Directory,
		AvailableAt: lectureOptions.AvailableAt, // current timestamp
	}

	courseMetadata.Lectures = append(courseMetadata.Lectures, newCourseLecture)
	if err := writeYaml("_course.yml", courseMetadata); err != nil {
		return err
	}

	return nil
}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new resource to your jupyteach course",
	Long: `Add a new resource to your jupyteach course.

	This command will prompt you to select the type of
	resource you want to add, guide you through defining all the necessary
	fields, and finally create the .yml entries for you in _course.yml and/or
	_lecture.yml`,
	Run: func(cmd *cobra.Command, args []string) {
		var options CreateOptions
		var contentBlock model.ContentBlockYaml
		var videoSource string
		var quizTopicsInput string

		lectureDirectory := "." // default to current directory

		createTypeSelect := huh.NewSelect[string]().
			Options(huh.NewOptions("lecture", "notebook", "markdown", "quiz", "video", "link")...).
			Title("Choose resource type").
			Value(&options.Type)

		if err := createTypeSelect.Run(); err != nil {
			log.Fatal(err)
		}

		if options.Type == "lecture" {
			if err := createLecture(); err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}

		contentBlock.Type = options.Type

		if options.Type == "quiz" {
			contentBlock.Quiz = model.Quiz{Questions: []model.Question{}} // initialize empty quiz
		}

		// Ensure `_lecture.yml` exists
		var lectureYaml *model.LectureYaml
		var errFile error
		lectureYaml, errFile = model.ParseLectureYaml("_lecture.yml")
		if errFile != nil {
			// try to let the user select an existing lecture
			courseYaml, err := model.ParseCourseYaml(".")
			if err != nil {
				log.Fatal(err)
			}

			options := make([]string, len(courseYaml.Lectures))
			for i, lecture := range courseYaml.Lectures {
				options[i] = lecture.Directory
			}
			lectureSelect := huh.NewSelect[string]().Options(huh.NewOptions(options...)...).Title("Select lecture").Value(&lectureDirectory)

			if err := lectureSelect.Run(); err != nil {
				log.Fatal(err)
			}

			// if we still can't find a lecture, bail
			lectureYaml, errFile = model.ParseLectureYaml(filepath.Join(lectureDirectory, "_lecture.yml"))
			if errFile != nil {
				log.Fatal(errFile)
			}
		}

		commonContentGroup := huh.NewGroup(
			huh.NewInput().Title(fmt.Sprintf("%s title (short)", options.Type)).Value(&contentBlock.Title),
			huh.NewInput().Title(fmt.Sprintf("%s Description (longer)", options.Type)).Value(&contentBlock.Description),
		)

		videoForm := huh.NewForm(
			commonContentGroup,
			huh.NewGroup(
				huh.NewSelect[string]().Title("Video source").Options(huh.NewOptions("url", "youtube", "vimeo")...).Value(&videoSource),
			).WithHideFunc(func() bool { return options.Type != "video" }),
			huh.NewGroup(
				huh.NewInput().Title("Video URL").Value(&contentBlock.URL),
			).WithHideFunc(func() bool { return videoSource != "url" }),
			huh.NewGroup(
				huh.NewInput().Title("YouTube Video ID").Value(&contentBlock.YoutubeVideoID),
			).WithHideFunc(func() bool { return videoSource != "youtube" }),
			huh.NewGroup(
				huh.NewInput().Title("Vimeo Video ID").Value(&contentBlock.VimeoVideoID),
			).WithHideFunc(func() bool { return videoSource != "vimeo" }),
		)

		linkForm := huh.NewForm(
			commonContentGroup,
			huh.NewGroup(
				huh.NewInput().Title("URL").Value(&contentBlock.URL).Validate(func(s string) error {
					if strings.HasPrefix(s, "http") {
						return nil
					} else {
						return fmt.Errorf("Must provide valid URL that begins with http(s)://")
					}
				}),
			),
		)

		quizForm := huh.NewForm(
			commonContentGroup,
			huh.NewGroup(
				huh.NewInput().Title("Start code").Description("Optional").Value(&contentBlock.Quiz.StartCode),
				huh.NewInput().Title("Topics").Description("Optional, Comma separated").Value(&quizTopicsInput),
				huh.NewSelect[int]().Title("Maximum attempts").Value(&contentBlock.Quiz.MaxAttempts).Options(huh.NewOptions(1, 2, 3, 4, 5, 1000)...),
			),
		)

		nbFiles, err := ListFilesInDirectory(lectureDirectory, []string{".ipynb"})
		if err != nil {
			log.Fatal(err)
		}
		if len(nbFiles) == 0 && options.Type == "notebook" {
			log.Fatal("No notebooks found in current directory. Notebooks must have .ipynb extension")
		}

		notebookForm := huh.NewForm(
			commonContentGroup,
			huh.NewGroup(
				huh.NewSelect[string]().
					Options(huh.NewOptions(nbFiles...)...).
					Title("Notebook file").
					Value(&contentBlock.Filename),
			),
		)

		mdFiles, err := ListFilesInDirectory(lectureDirectory, []string{".md"})
		if err != nil {
			log.Fatal(err)
		}

		if len(mdFiles) == 0 && options.Type == "markdown" {
			log.Fatal("No markdown files found in current directory. Markdown files must have .md extension")
		}

		mdForm := huh.NewForm(
			commonContentGroup,
			huh.NewGroup(
				huh.NewSelect[string]().
					Options(huh.NewOptions(mdFiles...)...).
					Title("Notebook file").
					Value(&contentBlock.Filename),
			),
		)

		var form *huh.Form
		switch options.Type {
		case "notebook":
			form = notebookForm
		case "markdown":
			form = mdForm
		case "video":
			form = videoForm
		case "link":
			form = linkForm
		case "quiz":
			form = quizForm
		}

		if err := form.Run(); err != nil {
			log.Fatal(err)
		}

		// Post processing
		switch options.Type {
		case "quiz":
			contentBlock.Quiz.Topics = make([]string, 0)
			for _, topic := range strings.Split(quizTopicsInput, ",") {
				contentBlock.Quiz.Topics = append(contentBlock.Quiz.Topics, strings.TrimSpace(topic))
			}
		case "notebook", "markdown":
			// strip lecture directory prefix from filename
			contentBlock.Filename = strings.TrimPrefix(contentBlock.Filename, lectureDirectory+string(filepath.Separator))
		}

		lectureYaml.ContentBlocks = append(lectureYaml.ContentBlocks, contentBlock)
		if err := writeYaml(filepath.Join(lectureDirectory, "_lecture.yml"), lectureYaml); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
