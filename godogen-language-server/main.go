package main

import (
	_ "embed"
	"strings"

	"github.com/lukasngl/godogen/godogen-language-server/cmd"
)

//go:embed version.txt
var versionTxt string

var (
	version = strings.TrimSpace(versionTxt)
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, date)
	cmd.Execute()
}
