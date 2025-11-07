package godogenlint

import (
	godogen "github.com/lukasngl/godogen/pkg"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "godogen",
	Doc:  "godogen checks that godog step definitions are valid and have the correct number of parameters.",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, file := range pass.Files {
			stepFuncs := godogen.GetStepDefinitions(pass.Fset, file)
			for err := range stepFuncs.ValidationErrors() {
				// Convert godogen.SuggestedFix to analysis.SuggestedFix
				var analysisFixes []analysis.SuggestedFix
				for _, fix := range err.SuggestedFixes {
					var edits []analysis.TextEdit
					for _, edit := range fix.TextEdits {
						edits = append(edits, analysis.TextEdit{
							Pos:     edit.Pos,
							End:     edit.End,
							NewText: edit.NewText,
						})
					}
					analysisFixes = append(analysisFixes, analysis.SuggestedFix{
						Message:   fix.Message,
						TextEdits: edits,
					})
				}

				pass.Report(analysis.Diagnostic{
					Message:        err.Message,
					Category:       "godogen",
					SuggestedFixes: analysisFixes,
					Pos:            err.Pos(),
					End:            err.End(),
				})
			}
		}

		return nil, nil
	},
}
