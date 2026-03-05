//go:generate go run github.com/lukasngl/godogen
package indextest

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	"github.com/lukasngl/godogen/godogen-language-server/index"
)

//godogen:given ^the workspace root is (.+)$
func (tc *TestContext) SetWorkspaceRoot(root string) error {
	tc.index.WorkspaceRoot = root
	return nil
}

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

//godogen:when ^I request diagnostics for (.+)$
func (tc *TestContext) RequestDiagnostics(path string) error {
	tc.GetDiagnostics(path)
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

// annotationRegex matches inline diagnostic annotations like "    ^ ERROR: message".
var annotationRegex = regexp.MustCompile(`^\s*\^\s+(ERROR|WARNING|INFO|HINT):\s*(.+)$`)

// fixAnnotationRegex matches inline fix annotations like "    ^^ FIX: title -> expected result".
var fixAnnotationRegex = regexp.MustCompile(`^\s*\^\^\s+FIX:\s*(.+)\s*->\s*(.+)$`)

type expectedFix struct {
	Title        string
	ExpectedLine string // The expected line content after applying the fix
}

type expectedDiagnostic struct {
	Line     int
	Severity string
	Message  string
	Fixes    []expectedFix
}

func parseInlineAnnotations(content string) (string, []expectedDiagnostic) {
	lines := strings.Split(content, "\n")
	var cleanLines []string
	var diags []expectedDiagnostic

	for _, line := range lines {
		if match := fixAnnotationRegex.FindStringSubmatch(line); match != nil {
			// Fix annotation - attach to the most recent diagnostic
			if len(diags) > 0 {
				diags[len(diags)-1].Fixes = append(diags[len(diags)-1].Fixes, expectedFix{
					Title:        match[1],
					ExpectedLine: match[2],
				})
			}
		} else if match := annotationRegex.FindStringSubmatch(line); match != nil {
			// Diagnostic annotation line - extract diagnostic info
			diags = append(diags, expectedDiagnostic{
				Line:     len(cleanLines), // Points to the previous clean line (1-indexed)
				Severity: match[1],
				Message:  match[2],
			})
		} else {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n"), diags
}

func severityToInt(severity string) int {
	switch severity {
	case "ERROR":
		return 1
	case "WARNING":
		return 2
	case "INFO":
		return 3
	case "HINT":
		return 4
	default:
		return 0
	}
}

func severityFromInt(severity int) string {
	switch severity {
	case 1:
		return "ERROR"
	case 2:
		return "WARNING"
	case 3:
		return "INFO"
	case 4:
		return "HINT"
	default:
		return "UNKNOWN"
	}
}

//godogen:then ^(.+) has the following diagnostics:$
func (tc *TestContext) CheckInlineDiagnostics(path string, content *godog.DocString) error {
	// Parse annotations and clean code
	cleanCode, expectedDiags := parseInlineAnnotations(content.Content)

	// Add file to workspace
	if err := tc.addFileToWorkspace(path, cleanCode); err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	// Request diagnostics
	tc.GetDiagnostics(path)
	actualDiags := tc.GetDiagnosticsResult()

	// Check count matches
	if len(actualDiags) != len(expectedDiags) {
		var details []string
		for _, d := range actualDiags {
			details = append(details, fmt.Sprintf("  line %d: %s: %s",
				d.StartLine, severityFromInt(int(d.Severity)), d.Message))
		}
		return fmt.Errorf("expected %d diagnostics, got %d:\n%s",
			len(expectedDiags), len(actualDiags), strings.Join(details, "\n"))
	}

	// Group actual diagnostics by line
	actualByLine := make(map[int][]index.Diagnostic)
	for _, d := range actualDiags {
		actualByLine[d.StartLine] = append(actualByLine[d.StartLine], d)
	}

	// Get file content for fix verification
	fileLines := strings.Split(cleanCode, "\n")

	// For each expected diagnostic, find a matching actual on the same line
	usedActual := make(map[*index.Diagnostic]bool)
	for _, expected := range expectedDiags {
		actualsOnLine := actualByLine[expected.Line]
		if len(actualsOnLine) == 0 {
			return fmt.Errorf("expected diagnostic on line %d with message %q, but no diagnostics on that line",
				expected.Line, expected.Message)
		}

		var matchedActual *index.Diagnostic
		for i := range actualsOnLine {
			actual := &actualsOnLine[i]
			if usedActual[actual] {
				continue
			}

			expectedSev := severityToInt(expected.Severity)
			if int(actual.Severity) != expectedSev {
				continue
			}

			if !strings.Contains(actual.Message, expected.Message) {
				continue
			}

			matchedActual = actual
			usedActual[actual] = true
			break
		}

		if matchedActual == nil {
			var actualMsgs []string
			for _, a := range actualsOnLine {
				actualMsgs = append(actualMsgs, fmt.Sprintf("%s: %s",
					severityFromInt(int(a.Severity)), a.Message))
			}
			return fmt.Errorf("on line %d: expected %s containing %q, got: %s",
				expected.Line, expected.Severity, expected.Message, strings.Join(actualMsgs, "; "))
		}

		// Verify fixes if expected
		if len(expected.Fixes) > 0 {
			if err := verifyFixes(expected, matchedActual, fileLines); err != nil {
				return err
			}
		}
	}

	return nil
}

// verifyFixes checks that the actual diagnostic has the expected fixes.
func verifyFixes(expected expectedDiagnostic, actual *index.Diagnostic, fileLines []string) error {
	if len(actual.Fixes) < len(expected.Fixes) {
		return fmt.Errorf("on line %d: expected %d fixes, got %d",
			expected.Line, len(expected.Fixes), len(actual.Fixes))
	}

	lineIdx := actual.StartLine - 1
	if lineIdx < 0 || lineIdx >= len(fileLines) {
		return fmt.Errorf("diagnostic line %d out of range", actual.StartLine)
	}
	originalLine := fileLines[lineIdx]

	for _, expectedFix := range expected.Fixes {
		// Find a matching fix by title
		var matchedFix *index.DiagnosticFix
		for _, actualFix := range actual.Fixes {
			if actualFix.Title == expectedFix.Title {
				matchedFix = actualFix
				break
			}
		}

		if matchedFix == nil {
			var actualTitles []string
			for _, f := range actual.Fixes {
				actualTitles = append(actualTitles, f.Title)
			}
			return fmt.Errorf("on line %d: expected fix with title %q, got: %v",
				expected.Line, expectedFix.Title, actualTitles)
		}

		// Apply the fix and check the result
		startCol := actual.StartColumn - 1
		endCol := actual.EndColumn - 1
		if startCol < 0 || endCol > len(originalLine) || startCol > endCol {
			return fmt.Errorf("invalid column range: %d-%d for line of length %d",
				startCol, endCol, len(originalLine))
		}

		resultLine := originalLine[:startCol] + matchedFix.NewText + originalLine[endCol:]
		if resultLine != expectedFix.ExpectedLine {
			return fmt.Errorf("on line %d: fix %q produced %q, expected %q",
				expected.Line, expectedFix.Title, resultLine, expectedFix.ExpectedLine)
		}
	}

	return nil
}

func checkFixTitle(fix *index.DiagnosticFix, expectedTitle string) error {
	if fix.Title != expectedTitle {
		return fmt.Errorf("expected fix title %q, got %q", expectedTitle, fix.Title)
	}
	return nil
}

//godogen:then ^diagnostic (\d+) fix title is "(.+)"$
func (tc *TestContext) CheckDiagnosticFixTitle(indexStr, expectedTitle string) error {
	diagnostics := tc.GetDiagnosticsResult()

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid diagnostic index: %s", indexStr)
	}

	if index >= len(diagnostics) {
		return fmt.Errorf("diagnostic index %d out of range (have %d diagnostics)", index, len(diagnostics))
	}

	diag := diagnostics[index]
	if len(diag.Fixes) == 0 {
		return fmt.Errorf("diagnostic %d has no fixes", index)
	}

	var errs []error
	for _, fix := range diag.Fixes {
		if err := checkFixTitle(fix, expectedTitle); err != nil {
			errs = append(errs, err)
		} else {
			return nil
		}
	}
	return errors.Join(errs...)
}

func applyFixAndCheck(fix *index.DiagnosticFix, diag *index.Diagnostic, content, expectedContent string) error {
	lines := strings.Split(content, "\n")
	if diag.StartLine < 1 || diag.StartLine > len(lines) {
		return fmt.Errorf("diagnostic start line %d out of range", diag.StartLine)
	}

	lineIdx := diag.StartLine - 1
	line := lines[lineIdx]

	startCol := diag.StartColumn - 1
	endCol := diag.EndColumn - 1
	if startCol < 0 || endCol > len(line) || startCol > endCol {
		return fmt.Errorf("invalid column range: %d-%d for line of length %d", startCol, endCol, len(line))
	}

	newLine := line[:startCol] + fix.NewText + line[endCol:]
	lines[lineIdx] = newLine

	result := strings.Join(lines, "\n")
	if result != expectedContent {
		return fmt.Errorf("fix %q produced:\n%s\n\nexpected:\n%s", fix.Title, result, expectedContent)
	}
	return nil
}

//godogen:then ^diagnostic (\d+) fix applied to (.+) produces:$
func (tc *TestContext) CheckDiagnosticFixApplied(indexStr, path string, expected *godog.DocString) error {
	diagnostics := tc.GetDiagnosticsResult()

	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return fmt.Errorf("invalid diagnostic index: %s", indexStr)
	}

	if index >= len(diagnostics) {
		return fmt.Errorf("diagnostic index %d out of range (have %d diagnostics)", index, len(diagnostics))
	}

	diag := diagnostics[index]
	if len(diag.Fixes) == 0 {
		return fmt.Errorf("diagnostic %d has no fixes", index)
	}

	content, err := tc.GetFileContent(path)
	if err != nil {
		return fmt.Errorf("failed to get file content: %w", err)
	}

	var errs []error
	for _, fix := range diag.Fixes {
		if err := applyFixAndCheck(fix, &diag, content, expected.Content); err != nil {
			errs = append(errs, err)
		} else {
			return nil
		}
	}
	return errors.Join(errs...)
}
