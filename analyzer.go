package godogen

import (
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "godogen",
	Doc:  "godogen checks that godog step definitions are valid and have the correct number of parameters.",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, file := range pass.Files {
			stepFuncs := GetStepDefinitions(pass.Fset, file)
			for err := range stepFuncs.ValidationErrors() {
				pass.Report(analysis.Diagnostic{
					Message:  err.Message,
					Category: "godogen",
					Pos:      err.Pos(),
					End:      err.End(),
				})
			}
		}

		return nil, nil
	},
}
