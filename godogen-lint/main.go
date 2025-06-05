package main

import (
	"github.com/lukasngl/godogen/godogen"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

type Analyzer struct{}

// New satifies the plugin ABI for [golangci-lint]
//
// [golangci-lint]: https://golangci-lint.run/plugins/go-plugins
func New(conf any) ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{NewLinter()}, nil
}

// NewLinter creates a new instance of the godogen linter.
func NewLinter() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "godogenlint",
		Doc:  "godogenlint checks that godog step definitions are valid and have the correct number of parameters.",
		Run: func(pass *analysis.Pass) (any, error) {
			for _, file := range pass.Files {
				steps := godogen.GetStepDefinitions(pass.Fset, file)

				for _, step := range steps {
					for _, err := range step.ValidationErrors {
						pass.Report(analysis.Diagnostic{
							Message:  err.Message,
							Category: "godogen",
							Pos:      err.Pos,
							End:      err.End,
						})
					}
				}
			}

			return nil, nil
		},
	}
}

func main() {
	singlechecker.Main(NewLinter())
}
