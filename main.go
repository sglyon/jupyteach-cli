/*
Copyright © 2024 Spencer Lyon spencerlyon2@gmail.com
*/
package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/sglyon/jupyteach/cmd"
)

// These fields are populated by the goreleaser build
var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

const repo = "sglyon/jupyteach-cli"

func main() {
	vi := cmd.VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
		BuiltBy: builtBy,
	}
	updated, err := SelfUpdate()
	if err != nil {
		log.Fatal(err)
	}

	if updated {
		log.Info("Updated successfully. Please re-run the command.")
		os.Exit(0)
	}
	cmd.Execute(vi)
}
