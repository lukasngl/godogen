package godogen-lint

import (
	"github.com/golangci/plugin-module-register/register"
	"github.com/lukasngl/godogen/pkg/godogen"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("godogen-lint", New)
}

// New satifies the plugin ABI for [golangci-lint]
//
// [golangci-lint]: https://golangci-lint.run/plugins/go-plugins
func New(conf any) ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

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
