package gclplugin

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	godogen "github.com/lukasngl/godogen/godogen-lint/pkg"
)

func init() {
	register.Plugin("godogen-lint", func(conf any) (register.LinterPlugin, error) {
		return &Plugin{}, nil
	})
}

type Plugin struct{}

// BuildAnalyzers implements register.LinterPlugin.
func (p *Plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{godogen.Analyzer}, nil
}

// GetLoadMode implements register.LinterPlugin.
func (p *Plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

// New satifies the plugin ABI for [golangci-lint]
//
// [golangci-lint]: https://golangci-lint.run/plugins/go-plugins
func New(conf any) ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{godogen.Analyzer}, nil
}
