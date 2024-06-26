package git

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"

	"github.com/ldez/go-git-cmd-wrapper/v2/add"
	"github.com/ldez/go-git-cmd-wrapper/v2/commit"
	lib "github.com/ldez/go-git-cmd-wrapper/v2/git"
	gitinit "github.com/ldez/go-git-cmd-wrapper/v2/init"
	"github.com/ldez/go-git-cmd-wrapper/v2/revparse"
	"github.com/ldez/go-git-cmd-wrapper/v2/status"
	"github.com/ldez/go-git-cmd-wrapper/v2/types"
)

func WithDirectory(path string, f func() error) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(cwd)

	if err := os.Chdir(path); err != nil {
		return err
	}
	return f()
}

func IsClean(path string) (bool, error) {
	var x string

	err := WithDirectory(path, func() error {
		var errOut error
		x, errOut = lib.Status(status.Porcelain(""))
		return errOut
	})
	if err != nil {
		return false, err
	}
	return len(x) == 0, nil
}

func Init(path string) error {
	msg, err := lib.Init(gitinit.Directory(path))
	if err != nil {
		return err
	}
	log.Infof(msg)
	return nil
}

func CheckCleanFatal(path string) {
	clean, err := IsClean(path)
	if err != nil {
		log.Warn("Error checking if repo is clean, verify that you are in a git repository and `git` is installed\n")
		log.Fatal(err)
	}

	if !clean {
		log.Fatal("Repository is not clean, please commit or stash changes and try again")
	}
}

func GetLatestCommitSha(path string) (string, error) {
	var x string

	err := WithDirectory(path, func() error {
		var errOut error
		x, errOut = lib.RevParse(revparse.Args("HEAD"))
		return errOut
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(x), nil
}

func MoreRecentSha(path, sha1, sha2 string) (string, error) {
	if sha1 == "" && sha2 == "" {
		return sha1, nil
	}
	// Check if both SHAs are valid commit hashes
	sha1Time, err1 := CommitTime(path, sha1)
	sha2Time, err2 := CommitTime(path, sha2)

	if err1 == nil && err2 == nil {
		if sha1Time > sha2Time {
			return sha1, nil
		}
		return sha2, nil
	}

	if err1 != nil && err2 == nil {
		return sha1, nil
	}

	if err1 == nil && err2 != nil {
		return sha2, nil
	}

	return "", errors.New("error getting time for both shas")
}

func CommitTime(path, sha string) (int64, error) {
	var x string

	err := WithDirectory(path, func() error {
		var errOut error
		// git log --pretty=format: --name-status --diff-filter=ACMRT
		x, errOut = lib.Raw("show", func(g *types.Cmd) {
			g.AddOptions("-s")
			g.AddOptions("--format=%ct")
			g.AddOptions(sha)
			// lib.Debug(g)
		})
		return errOut
	})

	// Parse the UNIX timestamp
	unixTime, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse commit time: %w", err)
	}
	return unixTime, nil
}

func IsShaInHistory(path, sha string) (bool, error) {
	err := WithDirectory(path, func() error {
		_, err := lib.RevParse(revparse.Args(sha))
		return err
	})
	if err != nil {
		log.Error("We got an error looking up sha in history", "sha", sha, "err", err)
		return false, err
	}
	return true, nil
}

func toJupyteachChangecode(s string) string {
	switch s {
	case "R":
		return "M"
	default:
		return s
	}
}

func MakeFileChangeMap(s string) map[string]string {
	xLines := strings.Split(s, "\n")
	out := make(map[string]string, len(xLines))

	for _, line := range xLines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		changecode := toJupyteachChangecode(string(parts[0][0]))
		if string(parts[0][0]) == "R" && len(parts) == 3 {
			out[parts[2]] = changecode
		} else {
			out[parts[1]] = changecode
		}
	}
	return out
}

func ListFiles(path string) (string, error) {
	var x string
	err := WithDirectory(path, func() error {
		var errOut error
		x, errOut = lib.Raw("ls-files", func(g *types.Cmd) {
			g.AddOptions("-t")
			lib.Debug(g)
		})
		return errOut
	})
	if err != nil {
		return "", err
	}
	return x, nil
}

func ChangesSinceCommit(path, sha string) (map[string]string, error) {
	var x string
	var err error

	if sha == "" {
		x, err = ListFiles(path)
	} else {
		err = WithDirectory(path, func() error {
			var errOut error
			x, errOut = lib.Raw("diff", func(g *types.Cmd) {
				g.AddOptions("--name-status")
				g.AddOptions(sha)
				// lib.Debug(g)
			})
			return errOut
		})
	}
	if err != nil {
		return nil, err
	}

	return MakeFileChangeMap(x), nil
}

func CommitAll(path, message string) (bool, error) {
	var committed bool
	err := WithDirectory(path, func() error {
		var errOut error
		var s string
		s, errOut = lib.Add(add.All)
		if errOut != nil {
			log.Error(s)
			return errOut
		}

		s, errOut = lib.Commit(commit.Message(message))
		if strings.Contains(s, "nothing to commit") {
			return nil
		}
		if errOut != nil {
			log.Error(s)
			return errOut
		}

		committed = true

		return nil
	})

	return committed, err
}
