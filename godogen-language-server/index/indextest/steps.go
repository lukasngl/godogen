//go:generate go run github.com/lukasngl/godogen
package indextest

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
)

//godogen:given ^(.+) is added to the workspace:$
func (tc *TestContext) AddFileToWorkspace(path string, content *godog.DocString) error {
	return tc.addFileToWorkspace(path, content.Content)
}

//godogen:given ^(.+) is added to the filesystem:$
func (tc *TestContext) AddFileToFilesystem(path string, content *godog.DocString) error {
	return tc.addFileToDisk(path, content.Content)
}

func (tc *TestContext) addFileToWorkspace(path string, content string) error {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		return tc.AddWorkspaceGoFile(path, []byte(content))
	case ".feature":
		return tc.AddWorkspaceFeatureFile(path, []byte(content))
	default:
		return fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func (tc *TestContext) addFileToDisk(path string, content string) error {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		return tc.AddDiskGoFile(path, []byte(content))
	case ".feature":
		return tc.AddDiskFeatureFile(path, []byte(content))
	default:
		return fmt.Errorf("unsupported file extension: %s", ext)
	}
}

//godogen:when ^I request step definitions for (.+) line (\d+)$
func (tc *TestContext) RequestStepDefinitions(path string, lineStr string) error {
	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return fmt.Errorf("invalid line number: %s", lineStr)
	}
	tc.FindStepDefinitions(path, line)
	return nil
}

//godogen:when ^I request step references for (.+) line (\d+)()$
//godogen:when ^I request step references for (.+) line (\d+) column (\d+)$
func (tc *TestContext) RequestStepReferences(path string, lineStr string, columnStr string) error {
	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return fmt.Errorf("invalid line number: %s", lineStr)
	}

	column := 1 // Default to column 1 (beginning of line) if not specified
	if columnStr != "" {
		column, err = strconv.Atoi(columnStr)
		if err != nil {
			return fmt.Errorf("invalid column number: %s", columnStr)
		}
	}

	tc.FindStepReferences(path, line, column)
	return nil
}

//godogen:step ^(.+) is updated on the filesystem:$
func (tc *TestContext) UpdateFileOnFilesystem(path string, content *godog.DocString) error {
	return tc.addFileToDisk(path, content.Content)
}

//godogen:step ^(.+) workspace version is removed$
func (tc *TestContext) RemoveWorkspaceVersion(path string) error {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		tc.Index.RemoveWorkspaceGoFile(path)
	case ".feature":
		tc.Index.RemoveWorkspaceFeatureFile(path)
	default:
		return fmt.Errorf("unsupported file extension: %s", ext)
	}
	return nil
}

//godogen:then ^I get (\d+) results?$
func (tc *TestContext) CheckResultCount(countStr string) error {
	expectedCount, err := strconv.Atoi(countStr)
	if err != nil {
		return fmt.Errorf("invalid count: %s", countStr)
	}

	actualCount := tc.ResultCount()
	if actualCount != expectedCount {
		return fmt.Errorf("expected %d results, got %d", expectedCount, actualCount)
	}

	return nil
}

//godogen:then ^the results are:$
func (tc *TestContext) CheckResults(table *godog.Table) error {
	results := tc.GetResults()

	if len(table.Rows) < 2 {
		return fmt.Errorf("table must have at least a header row and one data row")
	}

	// Parse header to find column indices
	header := table.Rows[0].Cells
	pathIdx, lineIdx, columnIdx := -1, -1, -1
	for i, cell := range header {
		switch strings.ToLower(cell.Value) {
		case "path":
			pathIdx = i
		case "line":
			lineIdx = i
		case "column":
			columnIdx = i
		}
	}

	if pathIdx == -1 || lineIdx == -1 || columnIdx == -1 {
		return fmt.Errorf("table must have 'path', 'line', and 'column' columns")
	}

	// Check that we have the right number of results
	expectedRows := len(table.Rows) - 1
	if len(results) != expectedRows {
		return fmt.Errorf("expected %d results, got %d", expectedRows, len(results))
	}

	// Check each result
	for i, row := range table.Rows[1:] {
		expectedPath := row.Cells[pathIdx].Value
		expectedLine, err := strconv.Atoi(row.Cells[lineIdx].Value)
		if err != nil {
			return fmt.Errorf("invalid line number in row %d: %s", i+1, row.Cells[lineIdx].Value)
		}
		expectedColumn, err := strconv.Atoi(row.Cells[columnIdx].Value)
		if err != nil {
			return fmt.Errorf(
				"invalid column number in row %d: %s",
				i+1,
				row.Cells[columnIdx].Value,
			)
		}

		result := results[i]
		if result.Path != expectedPath {
			return fmt.Errorf("row %d: expected path %s, got %s", i+1, expectedPath, result.Path)
		}
		if result.Line != expectedLine {
			return fmt.Errorf("row %d: expected line %d, got %d", i+1, expectedLine, result.Line)
		}
		if result.Column != expectedColumn {
			return fmt.Errorf(
				"row %d: expected column %d, got %d",
				i+1,
				expectedColumn,
				result.Column,
			)
		}
	}

	return nil
}

//godogen:step ^I request diagnostics for (.+)$
func (tc *TestContext) RequestDiagnostics(path string) error {
	tc.GetDiagnostics(path)
	return nil
}

//godogen:then ^I get (\d+) diagnostics?$
func (tc *TestContext) CheckDiagnosticCount(countStr string) error {
	expectedCount, err := strconv.Atoi(countStr)
	if err != nil {
		return fmt.Errorf("invalid count: %s", countStr)
	}

	actualCount := tc.DiagnosticCount()
	if actualCount != expectedCount {
		return fmt.Errorf("expected %d diagnostics, got %d", expectedCount, actualCount)
	}

	return nil
}

//godogen:then ^diagnostic (\d+) message is "([^"]+)"$
func (tc *TestContext) CheckDiagnosticMessage(indexStr string, expectedMessage string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	diagnostics := tc.GetDiagnosticsResult()
	if index >= len(diagnostics) {
		return fmt.Errorf(
			"diagnostic index %d out of range (have %d diagnostics)",
			index,
			len(diagnostics),
		)
	}

	actualMessage := diagnostics[index].Message
	if actualMessage != expectedMessage {
		return fmt.Errorf("expected message %q, got %q", expectedMessage, actualMessage)
	}

	return nil
}

//godogen:then ^diagnostic (\d+) message contains "([^"]+)"$
func (tc *TestContext) CheckDiagnosticMessageContains(indexStr string, substring string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	diagnostics := tc.GetDiagnosticsResult()
	if index >= len(diagnostics) {
		return fmt.Errorf(
			"diagnostic index %d out of range (have %d diagnostics)",
			index,
			len(diagnostics),
		)
	}

	actualMessage := diagnostics[index].Message
	if !strings.Contains(actualMessage, substring) {
		return fmt.Errorf("expected message to contain %q, got %q", substring, actualMessage)
	}

	return nil
}

//godogen:when ^I hover over line (\d+) column (\d+) in (.+)$
func (tc *TestContext) HoverOverPosition(lineStr string, columnStr string, path string) error {
	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return fmt.Errorf("invalid line number: %s", lineStr)
	}

	column, err := strconv.Atoi(columnStr)
	if err != nil {
		return fmt.Errorf("invalid column number: %s", columnStr)
	}

	// LSP uses 0-indexed lines, but 0-indexed columns
	// The feature uses 1-indexed for both, so convert
	tc.GetHoverInfo(path, line-1, column-1)
	return nil
}

//godogen:then ^I get hover content:$
func (tc *TestContext) CheckHoverContent(expected *godog.DocString) error {
	hoverInfo := tc.GetHoverInfoResult()
	if hoverInfo == nil {
		return fmt.Errorf("expected hover content, got nil")
	}

	expectedContent := strings.TrimSpace(expected.Content)
	actualContent := strings.TrimSpace(hoverInfo.Content)

	if actualContent != expectedContent {
		return fmt.Errorf(
			"hover content mismatch:\nExpected:\n%s\n\nActual:\n%s",
			expectedContent,
			actualContent,
		)
	}

	return nil
}

//godogen:then ^I get no hover content$
func (tc *TestContext) CheckNoHoverContent() error {
	hoverInfo := tc.GetHoverInfoResult()
	if hoverInfo != nil {
		return fmt.Errorf("expected no hover content, got: %s", hoverInfo.Content)
	}
	return nil
}

//godogen:then ^diagnostic (\d+) message does not contain "([^"]+)"$
func (tc *TestContext) CheckDiagnosticMessageDoesNotContainVerbose(
	indexStr string,
	substring string,
) error {
	return tc.CheckDiagnosticMessageDoesNotContain(indexStr, substring)
}

//godogen:then ^diagnostic (\d+) does not contain "([^"]+)"$
func (tc *TestContext) CheckDiagnosticMessageDoesNotContain(
	indexStr string,
	substring string,
) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	diagnostics := tc.GetDiagnosticsResult()
	if index >= len(diagnostics) {
		return fmt.Errorf(
			"diagnostic index %d out of range (have %d diagnostics)",
			index,
			len(diagnostics),
		)
	}

	actualMessage := diagnostics[index].Message
	if strings.Contains(actualMessage, substring) {
		return fmt.Errorf("expected message not to contain %q, got %q", substring, actualMessage)
	}

	return nil
}

//godogen:then ^diagnostic (\d+) severity is "([^"]+)"$
func (tc *TestContext) CheckDiagnosticSeverity(indexStr string, severity string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	diagnostics := tc.GetDiagnosticsResult()
	if index >= len(diagnostics) {
		return fmt.Errorf(
			"diagnostic index %d out of range (have %d diagnostics)",
			index,
			len(diagnostics),
		)
	}

	actualSeverity := tc.Index.SeverityToString(diagnostics[index].Severity)
	if actualSeverity != severity {
		return fmt.Errorf("expected severity %q, got %q", severity, actualSeverity)
	}

	return nil
}

//godogen:then ^diagnostic (\d+) is on line (\d+)$
func (tc *TestContext) CheckDiagnosticLine(indexStr string, lineStr string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	diagnostics := tc.GetDiagnosticsResult()
	if index >= len(diagnostics) {
		return fmt.Errorf(
			"diagnostic index %d out of range (have %d diagnostics)",
			index,
			len(diagnostics),
		)
	}

	expectedLine, err := strconv.Atoi(lineStr)
	if err != nil {
		return fmt.Errorf("invalid line number: %s", lineStr)
	}

	actualLine := diagnostics[index].StartLine
	if actualLine != expectedLine {
		return fmt.Errorf("expected line %d, got %d", expectedLine, actualLine)
	}

	return nil
}

//godogen:then ^diagnostic (\d+) has (\d+) related info$
func (tc *TestContext) CheckDiagnosticRelatedInfoCount(indexStr string, countStr string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	diagnostics := tc.GetDiagnosticsResult()
	if index >= len(diagnostics) {
		return fmt.Errorf(
			"diagnostic index %d out of range (have %d diagnostics)",
			index,
			len(diagnostics),
		)
	}

	expectedCount, err := strconv.Atoi(countStr)
	if err != nil {
		return fmt.Errorf("invalid count: %s", countStr)
	}

	actualCount := len(diagnostics[index].RelatedInformation)
	if actualCount != expectedCount {
		return fmt.Errorf("expected %d related info, got %d", expectedCount, actualCount)
	}

	return nil
}

//godogen:then ^diagnostic (\d+) related info (\d+) is on line (\d+)$
func (tc *TestContext) CheckDiagnosticRelatedInfoLine(diagIndexStr string, relIndexStr string, lineStr string) error {
	diagIndex, err := strconv.Atoi(diagIndexStr)
	if err != nil {
		return fmt.Errorf("invalid diagnostic index: %s", diagIndexStr)
	}

	diagnostics := tc.GetDiagnosticsResult()
	if diagIndex >= len(diagnostics) {
		return fmt.Errorf(
			"diagnostic index %d out of range (have %d diagnostics)",
			diagIndex,
			len(diagnostics),
		)
	}

	relIndex, err := strconv.Atoi(relIndexStr)
	if err != nil {
		return fmt.Errorf("invalid related info index: %s", relIndexStr)
	}

	relatedInfo := diagnostics[diagIndex].RelatedInformation
	if relIndex >= len(relatedInfo) {
		return fmt.Errorf(
			"related info index %d out of range (have %d related info)",
			relIndex,
			len(relatedInfo),
		)
	}

	expectedLine, err := strconv.Atoi(lineStr)
	if err != nil {
		return fmt.Errorf("invalid line number: %s", lineStr)
	}

	actualLine := relatedInfo[relIndex].Line
	if actualLine != expectedLine {
		return fmt.Errorf("expected related info on line %d, got %d", expectedLine, actualLine)
	}

	return nil
}

//godogen:when ^I request document symbols for (.+) with hierarchy$
func (tc *TestContext) RequestDocumentSymbolsWithHierarchy(path string) error {
	tc.GetDocumentSymbols(path, true)
	return nil
}

//godogen:when ^I request document symbols for ([^ ]+)$
func (tc *TestContext) RequestDocumentSymbols(path string) error {
	tc.GetDocumentSymbols(path, false)
	return nil
}

//godogen:then ^I get (\d+) symbols?$
func (tc *TestContext) CheckSymbolCount(countStr string) error {
	expectedCount, err := strconv.Atoi(countStr)
	if err != nil {
		return fmt.Errorf("invalid count: %s", countStr)
	}

	actualCount := tc.DocumentSymbolsCount()
	if actualCount != expectedCount {
		return fmt.Errorf("expected %d symbols, got %d", expectedCount, actualCount)
	}

	return nil
}

//godogen:then ^symbol (\d+) name is "([^"]+)"$
func (tc *TestContext) CheckSymbolName(indexStr string, expectedName string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	symbols := tc.GetDocumentSymbolsResult()
	if index >= len(symbols) {
		return fmt.Errorf("symbol index %d out of range (have %d symbols)", index, len(symbols))
	}

	actualName := symbols[index].Name
	if actualName != expectedName {
		return fmt.Errorf("expected name %q, got %q", expectedName, actualName)
	}

	return nil
}

//godogen:then ^symbol (\d+) kind is "([^"]+)"$
func (tc *TestContext) CheckSymbolKind(indexStr string, expectedKind string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	symbols := tc.GetDocumentSymbolsResult()
	if index >= len(symbols) {
		return fmt.Errorf("symbol index %d out of range (have %d symbols)", index, len(symbols))
	}

	actualKind := symbols[index].Kind
	if actualKind != expectedKind {
		return fmt.Errorf("expected kind %q, got %q", expectedKind, actualKind)
	}

	return nil
}

//godogen:then ^symbol (\d+) is on line (\d+)$
func (tc *TestContext) CheckSymbolLine(indexStr string, lineStr string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return fmt.Errorf("invalid line: %s", lineStr)
	}

	symbols := tc.GetDocumentSymbolsResult()
	if index >= len(symbols) {
		return fmt.Errorf("symbol index %d out of range (have %d symbols)", index, len(symbols))
	}

	actualLine := symbols[index].Line
	if actualLine != line {
		return fmt.Errorf("expected line %d, got %d", line, actualLine)
	}

	return nil
}

//godogen:then ^symbol (\d+) has (\d+) child(?:ren)?$
func (tc *TestContext) CheckSymbolChildCount(indexStr string, countStr string) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	expectedCount, err := strconv.Atoi(countStr)
	if err != nil {
		return fmt.Errorf("invalid count: %s", countStr)
	}

	symbols := tc.GetDocumentSymbolsResult()
	if index >= len(symbols) {
		return fmt.Errorf("symbol index %d out of range (have %d symbols)", index, len(symbols))
	}

	actualCount := len(symbols[index].Children)
	if actualCount != expectedCount {
		return fmt.Errorf("expected %d children, got %d", expectedCount, actualCount)
	}

	return nil
}

//godogen:then ^symbol (\d+) child (\d+) name is "([^"]+)"$
func (tc *TestContext) CheckSymbolChildName(
	indexStr string,
	childIndexStr string,
	expectedName string,
) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	childIndex, err := strconv.Atoi(childIndexStr)
	if err != nil {
		return fmt.Errorf("invalid child index: %s", childIndexStr)
	}

	symbols := tc.GetDocumentSymbolsResult()
	if index >= len(symbols) {
		return fmt.Errorf("symbol index %d out of range (have %d symbols)", index, len(symbols))
	}

	children := symbols[index].Children
	if childIndex >= len(children) {
		return fmt.Errorf(
			"child index %d out of range (have %d children)",
			childIndex,
			len(children),
		)
	}

	actualName := children[childIndex].Name
	if actualName != expectedName {
		return fmt.Errorf("expected name %q, got %q", expectedName, actualName)
	}

	return nil
}

//godogen:then ^symbol (\d+) child (\d+) kind is "([^"]+)"$
func (tc *TestContext) CheckSymbolChildKind(
	indexStr string,
	childIndexStr string,
	expectedKind string,
) error {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid index: %s", indexStr)
	}

	childIndex, err := strconv.Atoi(childIndexStr)
	if err != nil {
		return fmt.Errorf("invalid child index: %s", childIndexStr)
	}

	symbols := tc.GetDocumentSymbolsResult()
	if index >= len(symbols) {
		return fmt.Errorf("symbol index %d out of range (have %d symbols)", index, len(symbols))
	}

	children := symbols[index].Children
	if childIndex >= len(children) {
		return fmt.Errorf(
			"child index %d out of range (have %d children)",
			childIndex,
			len(children),
		)
	}

	actualKind := children[childIndex].Kind
	if actualKind != expectedKind {
		return fmt.Errorf("expected kind %q, got %q", expectedKind, actualKind)
	}

	return nil
}
