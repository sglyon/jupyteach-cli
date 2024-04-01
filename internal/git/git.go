package git

import (
	"log"

	lib "github.com/ldez/go-git-cmd-wrapper/v2/git"
	"github.com/ldez/go-git-cmd-wrapper/v2/status"
)

func IsClean(path string) (bool, error) {
	x, err := lib.Status(status.Porcelain(""))
	if err != nil {
		return false, err
	}
	return len(x) == 0, nil
}

func CheckCleanFatal(path string) {
	clean, err := IsClean(path)
	if err != nil {
		log.Printf("Error checking if repo is clean, verify that you are in a git repository and `git` is installed\n")
		log.Fatal(err)
	}

	if !clean {
		log.Fatal("Repository is not clean, please commit or stash changes and try again")
	}
}
