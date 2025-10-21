package main

import (
	godogen "github.com/lukasngl/godogen/godogen-lint/pkg"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(godogen.Analyzer)
}
