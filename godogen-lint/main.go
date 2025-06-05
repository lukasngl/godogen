package main

import (
	"github.com/lukasngl/godogen/godogen-lint/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analyzer.Analyzer)
}
