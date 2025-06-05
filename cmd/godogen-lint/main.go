package main

import (
	"github.com/lukasngl/godogen"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(godogen.Analyzer)
}
