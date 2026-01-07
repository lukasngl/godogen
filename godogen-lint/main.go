package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	godogenlint "github.com/lukasngl/godogen/godogen-lint/pkg"
	"golang.org/x/tools/go/analysis/singlechecker"
)

//go:embed version.txt
var versionTxt string

var (
	version = strings.TrimSpace(versionTxt)
	commit  = "none"
	date    = "unknown"
)

func main() {
	godogenlint.Analyzer.Flags.Var(new(versionFlag), "V", "print version information and exit")
	singlechecker.Main(godogenlint.Analyzer)
}

// https://cs.opensource.google/go/x/tools/+/refs/tags/v0.35.0:go/analysis/internal/analysisflags/flags.go;l=188-237;drc=99337ebe7b90918701a41932abf121600b972e34
type versionFlag string

// Set implements flag.Value.
func (v versionFlag) Set(s string) error {
	extra := ""
	if s == "full" {
		extra = fmt.Sprintf(" %s buildID=%s\n", date, commit)
	}

	fmt.Printf("godogen-lint version %s%s\n", version, extra)

	os.Exit(0)
	return nil
}

// String implements flag.Value.
func (v versionFlag) IsBoolFlag() bool { return true }
func (v versionFlag) String() string   { return "" }
func (v versionFlag) Get() any         { return nil }
