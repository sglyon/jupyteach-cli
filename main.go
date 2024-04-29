/*
Copyright © 2024 Spencer Lyon spencerlyon2@gmail.com
*/
package main

import (
	"github.com/charmbracelet/log"
	"github.com/sglyon/jupyteach/cmd"
)

// These fields are populated by the goreleaser build
var (
	version = "0.1.0-rc1"
	commit  = ""
	date    = ""
	builtBy = ""
)

func main() {
	vi := cmd.VersionInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
		BuiltBy: builtBy,
	}
	cmd.Execute(vi)
}
