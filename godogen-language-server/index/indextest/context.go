package indextest

import (
	"path/filepath"
	"strings"

	"github.com/lukasngl/godogen/godogen-language-server/index"
)

// TestContext holds the state for a test scenario.
type TestContext struct {
	index           *index.Index
	Index           *index.Index // Exported for direct access in step definitions
	results         []index.Location
	diagnostics     []index.Diagnostic
	diagnosticsPath string
	hoverInfo       *index.HoverInfo
	documentSymbols []index.DocumentSymbol
}

// NewTestContext creates a new test context for a scenario.
func NewTestContext() *TestContext {
	idx := index.NewIndex()
	return &TestContext{
		index: idx,
		Index: idx,
	}
}

// AddWorkspaceGoFile adds a Go file to the workspace.
func (tc *TestContext) AddWorkspaceGoFile(path string, content []byte) error {
	return tc.index.IndexWorkspaceGoFile(path, content)
}

// AddWorkspaceFeatureFile adds a feature file to the workspace.
func (tc *TestContext) AddWorkspaceFeatureFile(path string, content []byte) error {
	return tc.index.IndexWorkspaceFeatureFile(path, content)
}

// AddDiskGoFile adds a Go file from disk.
func (tc *TestContext) AddDiskGoFile(path string, content []byte) error {
	return tc.index.IndexDiskGoFile(path, content)
}

// AddDiskFeatureFile adds a feature file from disk.
func (tc *TestContext) AddDiskFeatureFile(path string, content []byte) error {
	return tc.index.IndexDiskFeatureFile(path, content)
}

// FindStepDefinitions queries for step definitions at the given location.
func (tc *TestContext) FindStepDefinitions(path string, line int) {
	tc.results = tc.index.FindStepDefinitions(path, line, false)
}

// FindStepReferences queries for step references at the given location.
func (tc *TestContext) FindStepReferences(path string, line int, column int) {
	tc.results = tc.index.FindStepReferences(path, line, column)
}

// GetResults returns the last query results.
func (tc *TestContext) GetResults() []index.Location {
	return tc.results
}

// ResultCount returns the number of results from the last query.
func (tc *TestContext) ResultCount() int {
	return len(tc.results)
}

// GetDiagnostics queries for diagnostics at the given path.
func (tc *TestContext) GetDiagnostics(path string) {
	tc.diagnosticsPath = path
	// Check file extension to determine which diagnostics method to call
	if strings.HasSuffix(path, ".feature") {
		tc.diagnostics = tc.index.GetFeatureDiagnostics(path)
	} else {
		tc.diagnostics = tc.index.GetDiagnostics(path)
	}
}

// GetDiagnosticsPath returns the path for which diagnostics were requested.
func (tc *TestContext) GetDiagnosticsPath() string {
	return tc.diagnosticsPath
}

// GetDiagnosticsResult returns the last diagnostics query results.
func (tc *TestContext) GetDiagnosticsResult() []index.Diagnostic {
	return tc.diagnostics
}

// DiagnosticCount returns the number of diagnostics from the last query.
func (tc *TestContext) DiagnosticCount() int {
	return len(tc.diagnostics)
}

// GetHoverInfo queries for hover information at the given position.
func (tc *TestContext) GetHoverInfo(path string, line int, column int) {
	if filepath.Ext(path) == ".feature" {
		tc.hoverInfo = tc.index.GetHoverInfoForFeature(path, line, column)
	} else if filepath.Ext(path) == ".go" {
		tc.hoverInfo = tc.index.GetHoverInfoForGo(path, line, column)
	}
}

// GetHoverInfoResult returns the last hover query result.
func (tc *TestContext) GetHoverInfoResult() *index.HoverInfo {
	return tc.hoverInfo
}

// GetDocumentSymbols queries for document symbols at the given path.
func (tc *TestContext) GetDocumentSymbols(path string, withHierarchy bool) {
	if filepath.Ext(path) == ".go" {
		tc.documentSymbols = tc.index.GetGoDocumentSymbols(path)
	} else {
		tc.documentSymbols = tc.index.GetFeatureDocumentSymbols(path, withHierarchy)
	}
}

// GetDocumentSymbolsResult returns the last document symbols query results.
func (tc *TestContext) GetDocumentSymbolsResult() []index.DocumentSymbol {
	return tc.documentSymbols
}

// DocumentSymbolsCount returns the number of document symbols from the last query.
func (tc *TestContext) DocumentSymbolsCount() int {
	return len(tc.documentSymbols)
}
