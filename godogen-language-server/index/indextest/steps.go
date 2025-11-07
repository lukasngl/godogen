//go:generate go run github.com/lukasngl/godogen
package indextest

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
)

//godogen:when ^(.+) is added to the workspace:$
func (tc *TestContext) AddFileToWorkspace(path string, content *godog.DocString) error {
	return tc.addFileToWorkspace(path, content.Content)
}

//godogen:when ^(.+) is added to the filesystem:$
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

//godogen:when ^(.+) is updated on the filesystem:$
func (tc *TestContext) UpdateFileOnFilesystem(path string, content *godog.DocString) error {
	return tc.addFileToDisk(path, content.Content)
}

//godogen:when ^(.+) workspace version is removed$
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

//godogen:when ^I request diagnostics for (.+)$
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
