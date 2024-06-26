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
	"net/http"
	"os"

	"github.com/sglyon/jupyteach/internal/git"
	"github.com/spf13/cobra"
)

func doClone(path, courseSlug string) (int, error) {
	return doPullOrClone(path, courseSlug, "clone")
}

// cloneCmd represents the clone command
var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone an existing course to a new directory",
	Long: `Clone a full Jupyteach course (for which you are an admin)
	to a new directory. The name of the directory will match the course slug.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		courseSlug := args[0]
		// path, err := cmd.Flags().GetString("path")
		// if err != nil {
		// 	logger.Fatal("Must provide a path")
		// }
		path = courseSlug

		// We need the path to not exist
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			logger.Fatal("Path already exists. Please provide a new path")
		}

		// now let's mkdir the path
		if err := os.MkdirAll(path, 0o755); err != nil {
			logger.Fatal(err)
		}

		// Now we are ready to pull
		retCode, err := doClone(path, courseSlug)
		if err != nil {
			cleanupFailure(path)
			logger.Fatal(err)
		}
		logger.Info("Successfully cloned course contents. ", "directory", path)

		if retCode == http.StatusCreated {
			// we only need to init the git repo if it was the first clone
			// subsequent clones will have the repo already initialized
			if err := git.Init(path); err != nil {
				cleanupFailure(path)
				logger.Fatal(err)
			}

			if _, _, err := commitAllAndUpdateServer(path, courseSlug, "jupyteach cli clone response"); err != nil {
				cleanupFailure(path)
				logger.Fatal(err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cloneCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cloneCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
