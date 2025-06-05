package lint

import (
	"github.com/golangci/plugin-module-register/register"
	"github.com/lukasngl/godogen/pkg/godogen"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "godogen",
	Doc:  "godogen checks that godog step definitions are valid and have the correct number of parameters.",
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

func init() {
	register.Plugin("godogen", func(conf any) (register.LinterPlugin, error) {
		return &Plugin{}, nil
	})
}

type Plugin struct{}

// BuildAnalyzers implements register.LinterPlugin.
func (p *Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

// GetLoadMode implements register.LinterPlugin.
func (p *Plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

// New satifies the plugin ABI for [golangci-lint]
//
// [golangci-lint]: https://golangci-lint.run/plugins/go-plugins
func New(conf any) ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}
