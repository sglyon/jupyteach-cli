package git

import (
	"log"
	"strings"

	lib "github.com/ldez/go-git-cmd-wrapper/v2/git"
	"github.com/ldez/go-git-cmd-wrapper/v2/revparse"
	"github.com/ldez/go-git-cmd-wrapper/v2/status"
	"github.com/ldez/go-git-cmd-wrapper/v2/types"
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

func GetLatestCommitSha(path string) (string, error) {
	x, err := lib.RevParse(revparse.Args("HEAD"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(x), nil
}

func IsShaInHistory(sha string) (bool, error) {
	_, err := lib.RevParse(revparse.Args(sha))
	if err != nil {
		log.Println("We got an error looking up sha in history", sha, err)
		return false, err
	}
	return true, nil
}

func ChangesSinceCommit(path, sha string) (map[string]string, error) {
	x, err := lib.Raw("diff", func(g *types.Cmd) {
		g.AddOptions("--name-status")
		g.AddOptions(sha)
		lib.Debug(g)
	})
	if err != nil {
		return nil, err
	}

	xLines := strings.Split(x, "\n")
	out := make(map[string]string, len(xLines))

	for _, line := range xLines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		out[parts[1]] = parts[0]
	}
	return out, nil
}
