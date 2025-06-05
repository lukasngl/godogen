package analyzer

import (
	"github.com/lukasngl/godogen/pkg/godogen"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "godogen-lint",
	Doc:  "godogen-lint checks that godog step definitions are valid and have the correct number of parameters.",
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
