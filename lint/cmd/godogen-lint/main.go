package main

import (
	lint "github.com/lukasngl/godogen/godogen-lint"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(lint.Analyzer)
}
