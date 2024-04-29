package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/charmbracelet/log"
	selfupdate "github.com/creativeprojects/go-selfupdate"
)

func SelfUpdate() error {
	latest, found, err := selfupdate.DetectLatest(context.Background(), selfupdate.ParseSlug(repo))
	if err != nil {
		return fmt.Errorf("error occurred while detecting version: %w", err)
	}
	if !found {
		return fmt.Errorf("latest version for %s/%s could not be found from github repository", runtime.GOOS, runtime.GOARCH)
	}

	if latest.LessOrEqual(version) {
		log.Infof("Current version (%s) is the latest", version)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return errors.New("could not locate executable path")
	}
	if err := selfupdate.UpdateTo(context.Background(), latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}
	log.Infof("Successfully updated to version %s", latest.Version())
	return nil
}
