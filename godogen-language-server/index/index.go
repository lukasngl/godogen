// Package index provides an in-memory index of Gherkin feature files and Go step definition files.
// It maintains separate versions for workspace (editor-open) and disk files, with workspace files
// taking precedence when both versions exist.
package index

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"iter"
	"log/slog"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	gherkin "github.com/cucumber/gherkin/go/v36"
	messages "github.com/cucumber/messages/go/v30"
	"github.com/google/uuid"
	godogen "github.com/lukasngl/godogen/pkg"
)

// Index maintains an in-memory index of feature files and Go step definition files.
// It tracks both workspace versions (from LSP clients) and disk versions of files,
// preferring workspace versions when both exist. All operations are thread-safe.
type Index struct {
	mx sync.RWMutex

	Features map[string]*FeatureFileVersions
	GoFiles  map[string]*GoFileVersions
}

// NewIndex creates a new empty index.
func NewIndex() *Index {
	return &Index{
		mx:       sync.RWMutex{},
		Features: make(map[string]*FeatureFileVersions),
		GoFiles:  make(map[string]*GoFileVersions),
	}
}

// GoFileVersions holds both workspace and disk versions of a Go file.
type GoFileVersions struct {
	workspace *GoFile
	disk      *GoFile
}

// FeatureFileVersions holds both workspace and disk versions of a feature file.
type FeatureFileVersions struct {
	workspace *FeatureFile
	disk      *FeatureFile
}

// GoFile represents a parsed Go file containing godogen step definitions.
type GoFile struct {
	*token.FileSet
	godogen.StepFuncs
}

// FeatureFile represents a parsed Gherkin feature file.
type FeatureFile struct {
	*messages.Feature
}

// getOrCreate returns the FileVersions for a path, creating if needed.
func (gfv *GoFileVersions) getOrCreate() *GoFileVersions {
	if gfv == nil {
		return &GoFileVersions{}
	}

	return gfv
}

func (ffv *FeatureFileVersions) getOrCreate() *FeatureFileVersions {
	if ffv == nil {
		return &FeatureFileVersions{}
	}

	return ffv
}

// get returns the preferred file version (workspace over disk).
func (gfv *GoFileVersions) get() *GoFile {
	if gfv.workspace != nil {
		return gfv.workspace
	}

	if gfv.disk != nil {
		return gfv.disk
	}

	return nil
}

func (ffv *FeatureFileVersions) get() *FeatureFile {
	if ffv.workspace != nil {
		return ffv.workspace
	}

	if ffv.disk != nil {
		return ffv.disk
	}

	return nil
}

// IndexWorkspaceGoFile indexes a Go file from the workspace (LSP client).
// The file type is explicitly known to be Go, so no extension checking is performed.
// If the file contains no godogen directives, any existing workspace version is removed.
func (index *Index) IndexWorkspaceGoFile(path string, content []byte) error {
	slog.Debug(
		"indexing file",
		"component",
		"index",
		"path",
		path,
		"isWorkspace",
		true,
		"type",
		"go",
	)

	if !bytes.Contains(content, []byte("\n//godogen:")) {
		slog.Debug("no directives found", "component", "index", "path", path)
		index.RemoveWorkspaceGoFile(path)
		return nil
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return err
	}

	index.mx.Lock()
	defer index.mx.Unlock()

	versions := index.GoFiles[path].getOrCreate()
	goFile := &GoFile{
		FileSet:   fset,
		StepFuncs: godogen.GetStepDefinitions(fset, file),
	}

	versions.workspace = goFile
	index.GoFiles[path] = versions

	return nil
}

// IndexDiskGoFile indexes a Go file from disk.
// If the file contains no godogen directives, any existing disk version is removed.
func (index *Index) IndexDiskGoFile(path string, content []byte) error {
	slog.Debug(
		"indexing file",
		"component",
		"index",
		"path",
		path,
		"isWorkspace",
		false,
		"type",
		"go",
	)

	if !bytes.Contains(content, []byte("\n//godogen:")) {
		slog.Debug("no directives found", "component", "index", "path", path)
		index.RemoveDiskGoFile(path)
		return nil
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return err
	}

	index.mx.Lock()
	defer index.mx.Unlock()

	versions := index.GoFiles[path].getOrCreate()
	goFile := &GoFile{
		FileSet:   fset,
		StepFuncs: godogen.GetStepDefinitions(fset, file),
	}

	versions.disk = goFile
	index.GoFiles[path] = versions

	return nil
}

// RemoveWorkspaceGoFile removes the workspace version of a Go file from the index.
// If both workspace and disk versions are nil after removal, the file entry is deleted entirely.
func (index *Index) RemoveWorkspaceGoFile(path string) {
	slog.Debug(
		"removing file",
		"component",
		"index",
		"path",
		path,
		"isWorkspace",
		true,
		"type",
		"go",
	)

	index.mx.Lock()
	defer index.mx.Unlock()

	versions := index.GoFiles[path]
	if versions == nil {
		return
	}

	versions.workspace = nil

	// If both versions are nil, remove the entry entirely
	if versions.workspace == nil && versions.disk == nil {
		delete(index.GoFiles, path)
	}
}

// RemoveDiskGoFile removes the disk version of a Go file from the index.
// If both workspace and disk versions are nil after removal, the file entry is deleted entirely.
func (index *Index) RemoveDiskGoFile(path string) {
	slog.Debug(
		"removing file",
		"component",
		"index",
		"path",
		path,
		"isWorkspace",
		false,
		"type",
		"go",
	)

	index.mx.Lock()
	defer index.mx.Unlock()

	versions := index.GoFiles[path]
	if versions == nil {
		return
	}

	versions.disk = nil

	// If both versions are nil, remove the entry entirely
	if versions.workspace == nil && versions.disk == nil {
		delete(index.GoFiles, path)
	}
}

// IndexWorkspaceFeatureFile indexes a Gherkin feature file from the workspace (LSP client).
// The file type is explicitly known to be a feature file, so no extension checking is performed.
func (index *Index) IndexWorkspaceFeatureFile(path string, content []byte) error {
	slog.Debug(
		"indexing file",
		"component",
		"index",
		"path",
		path,
		"isWorkspace",
		true,
		"type",
		"feature",
	)

	reader := bytes.NewReader(content)

	document, err := gherkin.ParseGherkinDocument(reader, uuid.NewString)
	if err != nil {
		slog.Debug("parse error", "component", "index", "path", path, "error", err)
		return err
	}

	index.mx.Lock()
	defer index.mx.Unlock()

	versions := index.Features[path].getOrCreate()
	featureFile := &FeatureFile{
		Feature: document.Feature,
	}

	versions.workspace = featureFile
	index.Features[path] = versions

	return nil
}

// IndexDiskFeatureFile indexes a Gherkin feature file from disk.
func (index *Index) IndexDiskFeatureFile(path string, content []byte) error {
	slog.Debug(
		"indexing file",
		"component",
		"index",
		"path",
		path,
		"isWorkspace",
		false,
		"type",
		"feature",
	)

	reader := bytes.NewReader(content)

	document, err := gherkin.ParseGherkinDocument(reader, uuid.NewString)
	if err != nil {
		slog.Debug("parse error", "component", "index", "path", path, "error", err)
		return err
	}

	index.mx.Lock()
	defer index.mx.Unlock()

	versions := index.Features[path].getOrCreate()
	featureFile := &FeatureFile{
		Feature: document.Feature,
	}

	versions.disk = featureFile
	index.Features[path] = versions

	return nil
}

// RemoveWorkspaceFeatureFile removes the workspace version of a feature file from the index.
// If both workspace and disk versions are nil after removal, the file entry is deleted entirely.
func (index *Index) RemoveWorkspaceFeatureFile(path string) {
	slog.Debug(
		"removing file",
		"component",
		"index",
		"path",
		path,
		"isWorkspace",
		true,
		"type",
		"feature",
	)

	index.mx.Lock()
	defer index.mx.Unlock()

	versions := index.Features[path]
	if versions == nil {
		return
	}

	versions.workspace = nil

	// If both versions are nil, remove the entry entirely
	if versions.workspace == nil && versions.disk == nil {
		delete(index.Features, path)
	}
}

// RemoveDiskFeatureFile removes the disk version of a feature file from the index.
// If both workspace and disk versions are nil after removal, the file entry is deleted entirely.
func (index *Index) RemoveDiskFeatureFile(path string) {
	slog.Debug(
		"removing file",
		"component",
		"index",
		"path",
		path,
		"isWorkspace",
		false,
		"type",
		"feature",
	)

	index.mx.Lock()
	defer index.mx.Unlock()

	versions := index.Features[path]
	if versions == nil {
		return
	}

	versions.disk = nil

	// If both versions are nil, remove the entry entirely
	if versions.workspace == nil && versions.disk == nil {
		delete(index.Features, path)
	}
}

// DiagnosticSeverity represents the severity level of a diagnostic.
type DiagnosticSeverity int

const (
	// DiagnosticSeverityError indicates an error.
	DiagnosticSeverityError DiagnosticSeverity = 1
	// DiagnosticSeverityWarning indicates a warning.
	DiagnosticSeverityWarning DiagnosticSeverity = 2
	// DiagnosticSeverityInformation indicates an informational message.
	DiagnosticSeverityInformation DiagnosticSeverity = 3
	// DiagnosticSeverityHint indicates a hint.
	DiagnosticSeverityHint DiagnosticSeverity = 4
)

// SeverityToString converts a DiagnosticSeverity to its string representation.
func (index *Index) SeverityToString(severity DiagnosticSeverity) string {
	switch severity {
	case DiagnosticSeverityError:
		return "Error"
	case DiagnosticSeverityWarning:
		return "Warning"
	case DiagnosticSeverityInformation:
		return "Information"
	case DiagnosticSeverityHint:
		return "Hint"
	default:
		return "Unknown"
	}
}

// DiagnosticRelatedInformation represents additional location information related to a diagnostic.
type DiagnosticRelatedInformation struct {
	Path    string
	Line    int
	Column  int
	Message string
}

// DiagnosticFix represents a suggested fix for a diagnostic.
type DiagnosticFix struct {
	Title   string // e.g., "Change 'And' to 'When'"
	NewText string // The replacement text
}

// Diagnostic represents a diagnostic message with position information.
type Diagnostic struct {
	StartLine          int
	StartColumn        int
	EndLine            int
	EndColumn          int
	Message            string
	Severity           DiagnosticSeverity
	RelatedInformation []DiagnosticRelatedInformation
	Fix                *DiagnosticFix // Optional fix for the diagnostic
}

// GetDiagnostics returns validation errors for a Go file at the given path.
// If the path is a feature file, returns feature-specific diagnostics.
func (index *Index) GetDiagnostics(path string) []Diagnostic {
	index.mx.RLock()
	defer index.mx.RUnlock()

	// Check if it's a feature file
	if filepath.Ext(path) == ".feature" {
		return index.GetFeatureDiagnostics(path)
	}

	// Otherwise, check if it's a Go file
	versions := index.GoFiles[path]
	if versions == nil {
		return nil
	}

	goFile := versions.get()
	if goFile == nil {
		return nil
	}

	var diagnostics []Diagnostic
	for validationErr := range goFile.ValidationErrors() {
		start := goFile.Position(validationErr.Pos())
		end := goFile.Position(validationErr.End())

		diagnostics = append(diagnostics, Diagnostic{
			StartLine:   start.Line,
			StartColumn: start.Column,
			EndLine:     end.Line,
			EndColumn:   end.Column,
			Message:     validationErr.Message,
			Severity:    DiagnosticSeverityError,
		})
	}

	// Check for duplicate step definitions first
	duplicateDiags := index.findDuplicateSteps(path)
	diagnostics = append(diagnostics, duplicateDiags...)

	// Build set of lines with duplicate errors to skip in unused check
	duplicateLines := make(map[int]bool)
	for _, diag := range duplicateDiags {
		duplicateLines[diag.StartLine] = true
	}

	// Check for unused step definitions (only if there are feature files to use them)
	// Skip this check if there are no feature files at all
	hasFeatureFiles := len(index.Features) > 0
	if hasFeatureFiles {
		for _, stepFunc := range goFile.StepFuncs {
			for _, stepDef := range stepFunc.Steps {
				// Skip invalid patterns - they already have validation errors
				if stepDef.Regexp == nil {
					continue
				}

				// Skip if this step has a duplicate error
				start := goFile.Position(stepDef.Node.Pos())
				if duplicateLines[start.Line] {
					continue
				}

				// Check if this step is used anywhere
				if index.isStepUsed(stepDef) {
					continue
				}

				// Report as unused with Hint severity
				end := goFile.Position(stepDef.Node.End())

				diagnostics = append(diagnostics, Diagnostic{
					StartLine:   start.Line,
					StartColumn: start.Column,
					EndLine:     end.Line,
					EndColumn:   end.Column,
					Message:     "Step definition is not used in any feature file",
					Severity:    DiagnosticSeverityHint,
				})
			}
		}
	}

	return diagnostics
}

// stepKey creates a unique key for a step based on its kind and pattern.
// Generic "Step" kind matches all specific kinds (Given/When/Then).
type stepKey struct {
	kind    string
	pattern string
}

// findDuplicateSteps finds duplicate step definitions within and across files.
// It reports diagnostics for steps that have the same kind+pattern combination.
// Generic "Step" kind is treated as a wildcard that conflicts with any specific kind.
func (index *Index) findDuplicateSteps(path string) []Diagnostic {
	// Build a map of all step definitions across all files
	// Map structure: kind+pattern -> []location
	allSteps := make(map[stepKey][]Location)

	for filePath, goFileVersions := range index.GoFiles {
		goFile := goFileVersions.get()
		if goFile == nil {
			continue
		}

		for _, stepDef := range goFile.AllSteps() {
			// Skip steps with invalid regex patterns (they already have validation errors)
			if stepDef.Regexp == nil {
				continue
			}

			// Track step under its own kind only
			key := stepKey{kind: stepDef.Kind, pattern: stepDef.Pattern}
			pos := goFile.Position(stepDef.Node.Pos())
			allSteps[key] = append(allSteps[key], Location{
				Path:   filePath,
				Line:   pos.Line,
				Column: pos.Column,
			})
		}
	}

	// Now find duplicates for the requested path
	var diagnostics []Diagnostic

	versions := index.GoFiles[path]
	if versions == nil {
		return diagnostics
	}

	goFile := versions.get()
	if goFile == nil {
		return diagnostics
	}

	for _, stepDef := range goFile.AllSteps() {
		// Skip steps with invalid regex patterns
		if stepDef.Regexp == nil {
			continue
		}

		// Check for duplicates with this step's kind+pattern
		var duplicates []Location

		// For this step, collect all locations that conflict with it (including itself)
		if stepDef.Kind == "Step" {
			// Generic step: check for conflicts with any kind
			for _, kind := range []string{"Given", "When", "Then", "Step"} {
				key := stepKey{kind: kind, pattern: stepDef.Pattern}
				if locs, exists := allSteps[key]; exists {
					for _, loc := range locs {
						// Check if already in duplicates
						found := false
						for _, dup := range duplicates {
							if dup.Path == loc.Path && dup.Line == loc.Line {
								found = true
								break
							}
						}
						if !found {
							duplicates = append(duplicates, loc)
						}
					}
				}
			}
		} else {
			// Specific kind: check for conflicts with same kind and generic "Step"
			for _, kind := range []string{stepDef.Kind, "Step"} {
				key := stepKey{kind: kind, pattern: stepDef.Pattern}
				if locs, exists := allSteps[key]; exists {
					for _, loc := range locs {
						// Check if already in duplicates
						found := false
						for _, dup := range duplicates {
							if dup.Path == loc.Path && dup.Line == loc.Line {
								found = true
								break
							}
						}
						if !found {
							duplicates = append(duplicates, loc)
						}
					}
				}
			}
		}

		// If we found more than one location (duplicates), create a diagnostic
		// Since we include the current step, len > 1 means there are duplicates
		if len(duplicates) > 1 {
			pos := goFile.Position(stepDef.Node.Pos())
			end := goFile.Position(stepDef.Node.End())

			// Sort duplicates for consistent ordering
			slices.SortFunc(duplicates, func(a, b Location) int {
				if a.Path != b.Path {
					return strings.Compare(a.Path, b.Path)
				}
				return a.Line - b.Line
			})

			// Determine which location to point to:
			// - For same-file duplicates: point to the first occurrence
			// - For cross-file duplicates: point to a different file if current is first
			currentLoc := Location{Path: path, Line: pos.Line, Column: pos.Column}
			pointTo := duplicates[0]

			// If current step is the first occurrence and there are cross-file duplicates,
			// point to the first one from a different file
			if pointTo.Path == currentLoc.Path && pointTo.Line == currentLoc.Line {
				for _, loc := range duplicates[1:] {
					if loc.Path != path {
						pointTo = loc
						break
					}
				}
				// If all duplicates are in the same file, just point to the first (self)
			}

			message := fmt.Sprintf(
				"Duplicate step definition: pattern already defined at %s:%d",
				pointTo.Path,
				pointTo.Line,
			)

			diagnostics = append(diagnostics, Diagnostic{
				StartLine:   pos.Line,
				StartColumn: pos.Column,
				EndLine:     end.Line,
				EndColumn:   end.Column,
				Message:     message,
				Severity:    DiagnosticSeverityError,
			})
		}
	}

	return diagnostics
}

// isStepUsed checks if a step definition is used in any feature file.
// This method assumes the index read lock is already held by the caller.
func (index *Index) isStepUsed(stepDef godogen.Step) bool {
	for _, featureFileVersions := range index.Features {
		featureFile := featureFileVersions.get()
		if featureFile == nil {
			continue
		}

		for kind, step := range featureFile.Steps() {
			if stepMatchesDefinition(kind, step, stepDef) {
				return true
			}
		}
	}

	return false
}

// GetFeatureDiagnostics returns diagnostics for a feature file at the given path.
// It checks all steps in the feature file and reports errors for steps without matching definitions.
func (index *Index) GetFeatureDiagnostics(path string) []Diagnostic {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.Features[path]
	if versions == nil {
		return nil
	}

	featureFile := versions.get()
	if featureFile == nil {
		return nil
	}

	var diagnostics []Diagnostic

	for kind, step := range featureFile.Steps() {
		// Skip scenario outline steps with placeholders (e.g., "I have <count> cukes")
		// These will be instantiated with example values at runtime
		if strings.Contains(step.Text, "<") && strings.Contains(step.Text, ">") {
			continue
		}

		// Find all matching step definitions
		type matchInfo struct {
			Path    string
			Line    int
			Column  int
			Pattern string
		}
		var matches []matchInfo

		for goPath, goFileVersions := range index.GoFiles {
			goFile := goFileVersions.get()
			if goFile == nil {
				continue
			}

			for _, stepDef := range goFile.AllSteps() {
				if stepMatchesDefinition(kind, step, stepDef) {
					pos := goFile.Position(stepDef.Node.Pos())
					matches = append(matches, matchInfo{
						Path:    goPath,
						Line:    pos.Line,
						Column:  pos.Column,
						Pattern: stepDef.Pattern,
					})
				}
			}
		}

		// Check for undefined steps
		if len(matches) == 0 {
			// Use the original keyword from the file (e.g., "But") not the inherited kind (e.g., "Given")
			keyword := strings.TrimSpace(step.Step.Keyword)
			diag := Diagnostic{
				StartLine:   int(step.Location.Line),
				StartColumn: int(step.Location.Column),
				EndLine:     int(step.Location.Line),
				EndColumn:   int(step.Location.Column) + len(step.Step.Keyword) + len(step.Step.Text),
				Message:     fmt.Sprintf("No step definition found for: %s %s", keyword, step.Step.Text),
				Severity:    DiagnosticSeverityError,
			}

			// Look for similar step definitions that might be what the user meant
			if suggestion := index.findSimilarStepDefinition(kind, step); suggestion != nil {
				// Add related info pointing to the suggested definition
				diag.RelatedInformation = append(diag.RelatedInformation, DiagnosticRelatedInformation{
					Path:    suggestion.path,
					Line:    suggestion.line,
					Column:  suggestion.column,
					Message: fmt.Sprintf("similarly named %s step defined here", suggestion.kind),
				})

				// Create a hint diagnostic for the suggestion (rust-analyzer style)
				hintDiag := Diagnostic{
					StartLine:   int(step.Location.Line),
					StartColumn: int(step.Location.Column),
					EndLine:     int(step.Location.Line),
					EndColumn:   int(step.Location.Column) + len(keyword),
					Severity:    DiagnosticSeverityHint,
				}

				if suggestion.exactMatch {
					// Pattern matches but wrong kind - offer a fix to change keyword
					if isConjunction(keyword) {
						hintDiag.Message = fmt.Sprintf(
							"A matching '%s' step exists: %s",
							suggestion.kind, suggestion.pattern,
						)
					} else {
						hintDiag.Message = fmt.Sprintf(
							"A matching pattern exists but is defined as '%s': %s",
							suggestion.kind, suggestion.pattern,
						)
					}
					// Add fix to change the keyword
					hintDiag.Fix = &DiagnosticFix{
						Title:   fmt.Sprintf("Change '%s' to '%s'", keyword, suggestion.kind),
						NewText: suggestion.kind + " ",
					}
				} else {
					// Fuzzy match - similar pattern
					hintDiag.Message = fmt.Sprintf(
						"A step with a similar name exists: %s %s",
						suggestion.kind, suggestion.pattern,
					)
					// Only offer fix if the pattern is entirely literal (no capture groups)
					if suggestion.regexp != nil {
						if literal, complete := suggestion.regexp.LiteralPrefix(); complete {
							hintDiag.Fix = &DiagnosticFix{
								Title:   fmt.Sprintf("Change to '%s %s'", keyword, literal),
								NewText: keyword + " " + literal,
							}
						}
					}
				}

				// Add related info pointing back to original diagnostic
				hintDiag.RelatedInformation = append(hintDiag.RelatedInformation, DiagnosticRelatedInformation{
					Path:    path,
					Line:    int(step.Location.Line),
					Column:  int(step.Location.Column),
					Message: "original diagnostic",
				})

				diagnostics = append(diagnostics, hintDiag)
			}

			// Add related info for Scenario Outline steps
			if step.ExampleRow != nil {
				diag.RelatedInformation = append(diag.RelatedInformation, DiagnosticRelatedInformation{
					Path:    path,
					Line:    int(step.ExampleRow.Location.Line),
					Column:  int(step.ExampleRow.Location.Column),
					Message: "for the values of this example",
				})
			}
			diagnostics = append(diagnostics, diag)
			continue
		}

		// Check for ambiguous steps (multiple matches)
		if len(matches) > 1 {
			// Sort by path then line for consistent output
			slices.SortFunc(matches, func(a, b matchInfo) int {
				if a.Path != b.Path {
					return strings.Compare(a.Path, b.Path)
				}
				return a.Line - b.Line
			})

			diag := Diagnostic{
				StartLine:   int(step.Location.Line),
				StartColumn: int(step.Location.Column),
				EndLine:     int(step.Location.Line),
				EndColumn:   int(step.Location.Column) + len(step.Step.Keyword) + len(step.Step.Text),
				Message:     fmt.Sprintf("Ambiguous step: matches %d definitions", len(matches)),
				Severity:    DiagnosticSeverityWarning,
			}

			// Add related info for each matching definition
			for _, m := range matches {
				diag.RelatedInformation = append(diag.RelatedInformation, DiagnosticRelatedInformation{
					Path:    m.Path,
					Line:    m.Line,
					Column:  m.Column,
					Message: fmt.Sprintf("defined here: %s", m.Pattern),
				})
			}

			// Add related info for Scenario Outline example row
			if step.ExampleRow != nil {
				diag.RelatedInformation = append(diag.RelatedInformation, DiagnosticRelatedInformation{
					Path:    path,
					Line:    int(step.ExampleRow.Location.Line),
					Column:  int(step.ExampleRow.Location.Column),
					Message: "for the values of this example",
				})
			}
			diagnostics = append(diagnostics, diag)
		}
	}

	return diagnostics
}

// GetGoFile returns the Go file at the given path, or nil if not found.
func (index *Index) GetGoFile(path string) *GoFile {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.GoFiles[path]
	if versions == nil {
		return nil
	}

	return versions.get()
}

// GetFeature returns the feature at the given path, or nil if not found.
func (index *Index) GetFeature(path string) *messages.Feature {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.Features[path]
	if versions == nil {
		return nil
	}

	featureFile := versions.get()
	if featureFile == nil {
		return nil
	}

	return featureFile.Feature
}

type Step struct {
	*messages.Step

	// The Keyword after resolving conjunctions.
	Keyword messages.StepKeywordType `json:"keyword"`

	// If the step is from a scenario outline, the example row used for interpolation
	Example    *messages.Examples `json:"example,omitempty"`
	ExampleRow *messages.TableRow `json:"exampleRow,omitempty"`
	// Interpolated text with example values applied, otherwise the original step text.
	Text string `json:"text"`
}

// Steps returns an iterator over all steps in the feature file.
// It yields pairs of step kind (Given/When/Then) and the step itself,
// handling And/But conjunction steps by inheriting the previous step's kind.
func (f FeatureFile) Steps() iter.Seq2[string, *Step] {
	return func(yield func(string, *Step) bool) {
		for _, child := range f.Children {
			if child.Background != nil {
				if !yieldSteps(yield, child.Background.Steps) {
					return
				}
			}

			if child.Rule != nil {
				for _, ruleChild := range child.Rule.Children {
					if bg := ruleChild.Background; bg != nil {
						if !yieldSteps(yield, bg.Steps) {
							return
						}
					}

					if sc := ruleChild.Scenario; sc != nil {
						if !interpolateWithExamples(yield, sc.Examples, sc.Steps) {
							return
						}
					}
				}
			}

			if sc := child.Scenario; sc != nil {
				if !interpolateWithExamples(yield, sc.Examples, sc.Steps) {
					return
				}
			}
		}
	}
}

func yieldSteps(
	yield func(string, *Step) bool,
	steps []*messages.Step,
	transforms ...func(*Step),
) bool {
	lastKind := messages.StepKeywordType_UNKNOWN
	for _, step := range steps {
		lastKind = conjugateKind(lastKind, step)
		s := Step{
			Step:    step,
			Keyword: lastKind,
			Text:    step.Text,
		}
		// Apply transforms (e.g., placeholder interpolation)
		for _, transform := range transforms {
			transform(&s)
		}
		if !yield(mapStepKeywordType(lastKind), &s) {
			return false
		}
	}

	return true
}

func interpolateWithExamples(
	yield func(string, *Step) bool,
	examples []*messages.Examples,
	steps []*messages.Step,
) bool {
	if len(examples) == 0 {
		return yieldSteps(yield, steps)
	}

	for _, example := range examples {
		for _, row := range example.TableBody {
			replacer := buildReplacer(example, row)
			if !yieldSteps(yield, steps, replacer) {
				return false
			}
		}
	}

	return true
}

func buildReplacer(
	example *messages.Examples,
	row *messages.TableRow,
) func(*Step) {
	oldnew := make([]string, 0, 2*len(example.TableHeader.Cells))
	for i, cell := range example.TableHeader.Cells {
		old := fmt.Sprintf("<%s>", cell.Value)
		new := row.Cells[i].Value
		oldnew = append(oldnew, old, new)
	}
	replacer := strings.NewReplacer(oldnew...)

	return func(step *Step) {
		step.Text = replacer.Replace(step.Text)
		step.Example = example
		step.ExampleRow = row
	}
}

func conjugateKind(
	lastKind messages.StepKeywordType,
	step *messages.Step,
) messages.StepKeywordType {
	if step.KeywordType == messages.StepKeywordType_CONJUNCTION {
		return lastKind
	}

	return step.KeywordType
}

func mapStepKeywordType(t messages.StepKeywordType) string {
	switch t {
	case messages.StepKeywordType_ACTION:
		return "When"
	case messages.StepKeywordType_CONTEXT:
		return "Given"
	case messages.StepKeywordType_OUTCOME:
		return "Then"
	default:
		return ""
	}
}

// Location represents a position in a file.
type Location struct {
	Path   string
	Line   int
	Column int
}

// FindStepDefinitions finds Go step definitions that match a feature step at the given line.
// If patternLoc is true, returns the location of the step pattern comment;
// otherwise returns the function declaration location.
func (index *Index) FindStepDefinitions(featurePath string, line int, patternLoc bool) []Location {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.Features[featurePath]
	if versions == nil {
		slog.Debug("file not found", "component", "index", "path", featurePath, "type", "feature")
		return nil
	}

	featureFile := versions.get()
	if featureFile == nil {
		slog.Debug("file not indexed", "component", "index", "path", featurePath, "type", "feature")
		return nil
	}

	var locs []Location

	slog.Debug("finding definitions", "component", "index", "path", featurePath, "line", line)

	for kind, step := range featureFile.Steps() {
		if step.Location.Line-1 != int64(line) {
			continue
		}

		slog.Debug("matching step", "component", "index", "kind", kind, "text", step.Text)

		for path, goFileVersions := range index.GoFiles {
			goFile := goFileVersions.get()
			if goFile == nil {
				continue
			}

			for stepFunc, stepDef := range goFile.AllSteps() {
				if !stepMatchesDefinition(kind, step, stepDef) {
					continue
				}

				var pos token.Position
				if patternLoc {
					pos = goFile.Position(stepDef.Node.Pos())
				} else {
					pos = goFile.Position(stepFunc.Node.Pos())
				}

				locs = append(locs, Location{
					Path:   path,
					Line:   pos.Line,
					Column: pos.Column,
				})
			}
		}
	}

	return locs
}

// FindStepReferences finds feature steps that reference the Go step pattern at the given line and column.
// Column is 1-indexed (1 = first character of line).
// Returns references only if the cursor is on a pattern comment (//godogen:...).
func (index *Index) FindStepReferences(goPath string, line int, column int) []Location {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.GoFiles[goPath]
	if versions == nil {
		slog.Debug("file not found", "component", "index", "path", goPath, "type", "go")
		return nil
	}

	goFile := versions.get()
	if goFile == nil {
		slog.Debug("file not indexed", "component", "index", "path", goPath, "type", "go")
		return nil
	}

	var locs []Location

	slog.Debug(
		"finding references",
		"component",
		"index",
		"path",
		goPath,
		"line",
		line,
		"column",
		column,
	)

	for stepFunc, stepDef := range goFile.AllSteps() {
		// Only proceed if cursor is on pattern comment or function name
		if !cursorInStepPattern(goFile, stepDef, line, column) &&
			!cursorInFunctionName(goFile, stepFunc, line, column) {
			continue
		}

		for path, featureFileVersions := range index.Features {
			featureFile := featureFileVersions.get()
			if featureFile == nil {
				continue
			}

			for kind, step := range featureFile.Steps() {
				if !stepMatchesDefinition(kind, step, stepDef) {
					continue
				}

				pos := step.Location

				locs = append(locs, Location{
					Path:   path,
					Line:   int(pos.Line),
					Column: int(pos.Column),
				})
			}
		}
	}

	return locs
}

// cursorInStepPattern checks if the cursor is positioned on a step pattern comment.
func cursorInStepPattern(goFile *GoFile, stepDef godogen.Step, line int, column int) bool {
	stepDefPos := goFile.Position(stepDef.Node.Pos())
	stepDefEnd := goFile.Position(stepDef.Node.End())
	stepDefLine := stepDefPos.Line - 1

	return stepDefLine == line && column >= stepDefPos.Column && column <= stepDefEnd.Column
}

// cursorInFunctionName checks if the cursor is positioned on the function name.
// Returns false if cursor is on func keyword, parameters, return type, or body.
func cursorInFunctionName(goFile *GoFile, stepFunc godogen.StepFunc, line int, column int) bool {
	funcNamePos := goFile.Position(stepFunc.Node.Name.Pos())
	funcNameEnd := goFile.Position(stepFunc.Node.Name.End())
	funcNameLine := funcNamePos.Line - 1

	return funcNameLine == line && column >= funcNamePos.Column && column < funcNameEnd.Column
}

func stepMatchesDefinition(kind string, step *Step, stepDef godogen.Step) bool {
	if stepDef.Regexp == nil {
		return false
	}

	if stepDef.Kind != "Step" && stepDef.Kind != kind {
		return false
	}

	if !stepDef.Regexp.MatchString(step.Text) {
		return false
	}

	return true
}

// isConjunction returns true if the keyword is And or But (or their localized equivalents).
func isConjunction(keyword string) bool {
	k := strings.ToLower(keyword)
	return k == "and" || k == "but" || k == "*"
}

// stepSuggestion holds info about a step definition that might match what the user meant.
type stepSuggestion struct {
	kind       string
	path       string
	line       int
	column     int
	pattern    string
	regexp     *regexp.Regexp // compiled regex for checking literal prefix
	exactMatch bool           // true if the regex matches exactly, false if fuzzy match
	score      float64        // similarity score for fuzzy matches
}

// findSimilarStepDefinition looks for step definitions that might be what the user meant.
// It first looks for exact regex matches with different kinds, then falls back to fuzzy matching.
// This method assumes the index read lock is already held by the caller.
func (index *Index) findSimilarStepDefinition(inheritedKind string, step *Step) *stepSuggestion {
	var bestFuzzy *stepSuggestion

	for goPath, goFileVersions := range index.GoFiles {
		goFile := goFileVersions.get()
		if goFile == nil {
			continue
		}

		for _, stepDef := range goFile.AllSteps() {
			if stepDef.Regexp == nil {
				continue
			}

			// Skip generic "Step" definitions (they should have matched already)
			if stepDef.Kind == "Step" {
				continue
			}

			pos := goFile.Position(stepDef.Node.Pos())
			sameKind := stepDef.Kind == inheritedKind

			// Check for exact regex match (only useful if different kind)
			if !sameKind && stepDef.Regexp.MatchString(step.Text) {
				return &stepSuggestion{
					kind:       stepDef.Kind,
					path:       goPath,
					line:       pos.Line,
					column:     pos.Column,
					pattern:    stepDef.Pattern,
					regexp:     stepDef.Regexp,
					exactMatch: true,
					score:      1.0,
				}
			}

			// Compute fuzzy similarity (for both same and different kinds)
			score := patternSimilarity(stepDef.Pattern, step.Text)
			if score >= 0.75 && (bestFuzzy == nil || score > bestFuzzy.score) {
				bestFuzzy = &stepSuggestion{
					kind:       stepDef.Kind,
					path:       goPath,
					line:       pos.Line,
					column:     pos.Column,
					pattern:    stepDef.Pattern,
					regexp:     stepDef.Regexp,
					exactMatch: false,
					score:      score,
				}
			}
		}
	}

	return bestFuzzy
}

// patternSimilarity computes similarity between a regex pattern and step text.
// Returns a score between 0 and 1, where 1 is a perfect match.
func patternSimilarity(pattern, text string) float64 {
	normalized := normalizePattern(pattern)
	return stringSimilarity(normalized, text)
}

// normalizePattern converts a regex pattern to a normalized form for comparison.
// Replaces capture groups with representative placeholders.
func normalizePattern(pattern string) string {
	// Remove anchors
	s := strings.TrimPrefix(pattern, "^")
	s = strings.TrimSuffix(s, "$")

	// Replace common capture groups with placeholders
	s = strings.ReplaceAll(s, `(\d+)`, "123")
	s = strings.ReplaceAll(s, `([^"]*)`, "text")
	s = strings.ReplaceAll(s, `(.*)`, "text")
	s = strings.ReplaceAll(s, `(.+)`, "text")
	s = strings.ReplaceAll(s, `(\w+)`, "word")

	// Remove remaining regex escapes
	s = strings.ReplaceAll(s, `\.`, ".")
	s = strings.ReplaceAll(s, `\(`, "(")
	s = strings.ReplaceAll(s, `\)`, ")")
	s = strings.ReplaceAll(s, `\[`, "[")
	s = strings.ReplaceAll(s, `\]`, "]")

	return s
}

// stringSimilarity computes similarity using longest common subsequence ratio.
func stringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if a == b {
		return 1.0
	}

	lcs := longestCommonSubsequence(a, b)
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 0.0
	}

	return float64(lcs) / float64(maxLen)
}

// longestCommonSubsequence returns the length of the LCS.
func longestCommonSubsequence(a, b string) int {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return 0
	}

	prev := make([]int, n+1)
	curr := make([]int, n+1)

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1] + 1
			} else {
				curr[j] = max(prev[j], curr[j-1])
			}
		}
		prev, curr = curr, prev
	}

	return prev[n]
}

// HoverInfo represents hover information to display.
type HoverInfo struct {
	Content string
}

// GetHoverInfoForFeature returns hover information for a position in a feature file.
// Returns nil if no hover information is available.
func (index *Index) GetHoverInfoForFeature(featurePath string, line int, column int) *HoverInfo {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.Features[featurePath]
	if versions == nil {
		return nil
	}

	featureFile := versions.get()
	if featureFile == nil {
		return nil
	}

	// Find the step at the given position
	for kind, step := range featureFile.Steps() {
		// Check if position is on this step (line is 0-indexed in LSP)
		if step.Location.Line-1 != int64(line) {
			continue
		}

		// Check if cursor is within the step text
		startCol := int(step.Location.Column)
		endCol := startCol + len(step.Keyword) + len(step.Text)
		if column < startCol || column > endCol {
			continue
		}

		// Find matching step definitions
		var matches []stepDefMatch
		for path, goFileVersions := range index.GoFiles {
			goFile := goFileVersions.get()
			if goFile == nil {
				continue
			}

			for stepFunc, stepDef := range goFile.AllSteps() {
				if !stepMatchesDefinition(kind, step, stepDef) {
					continue
				}

				// Use function line for hover display (users expect to see where the function is)
				funcPos := goFile.Position(stepFunc.Node.Pos())
				matches = append(matches, stepDefMatch{
					path:     path,
					stepFunc: stepFunc,
					stepDef:  stepDef,
					goFile:   goFile,
					line:     funcPos.Line,
				})
			}
		}

		if len(matches) == 0 {
			// No matching step definition found
			return &HoverInfo{
				Content: formatUndefinedStep(step),
			}
		}

		if len(matches) == 1 {
			// Single match
			return &HoverInfo{
				Content: formatSingleStepDef(matches[0]),
			}
		}

		// Multiple matches (ambiguous)
		return &HoverInfo{
			Content: formatAmbiguousStepDefs(matches),
		}
	}

	// No step at this position
	return nil
}

// GetHoverInfoForGo returns hover information for a position in a Go file.
// Returns nil if no hover information is available.
func (index *Index) GetHoverInfoForGo(goPath string, line int, column int) *HoverInfo {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.GoFiles[goPath]
	if versions == nil {
		return nil
	}

	goFile := versions.get()
	if goFile == nil {
		return nil
	}

	// Find step definition at cursor position (only on pattern comments)
	for _, stepDef := range goFile.AllSteps() {
		// Check if cursor is on pattern comment
		if !cursorInStepPattern(goFile, stepDef, line, column) {
			continue
		}

		// Find all references to this step definition
		var refs []stepRefLocation
		for path, featureFileVersions := range index.Features {
			featureFile := featureFileVersions.get()
			if featureFile == nil {
				continue
			}

			for kind, step := range featureFile.Steps() {
				if !stepMatchesDefinition(kind, step, stepDef) {
					continue
				}

				refs = append(refs, stepRefLocation{
					path: path,
					line: int(step.Location.Line),
				})
			}
		}

		return &HoverInfo{
			Content: formatStepUsage(stepDef, refs),
		}
	}

	return nil
}

// stepDefMatch holds information about a matching step definition.
type stepDefMatch struct {
	path     string
	stepFunc godogen.StepFunc
	stepDef  godogen.Step
	goFile   *GoFile
	line     int
}

// stepRefLocation holds a reference location.
type stepRefLocation struct {
	path string
	line int
}

// formatUndefinedStep formats hover content for an undefined step.
func formatUndefinedStep(step *Step) string {
	return fmt.Sprintf(
		"**No step definition found**\n\nNo matching step definition for:\n```gherkin\n%s%s\n```",
		step.Step.Keyword,
		step.Step.Text,
	)
}

// formatSingleStepDef formats hover content for a single step definition.
func formatSingleStepDef(match stepDefMatch) string {
	var content strings.Builder
	content.WriteString("**Step Definition**\n\n")

	// Add godoc if available
	if match.stepFunc.Node.Doc != nil {
		docText := match.stepFunc.Node.Doc.Text()
		if docText != "" {
			content.WriteString(strings.TrimSpace(docText))
			content.WriteString("\n\n")
		}
	}

	// Add function signature
	sig := formatFunctionSignature(match.goFile.FileSet, match.stepFunc.Node)
	content.WriteString("```go\n")
	content.WriteString(sig)
	content.WriteString("\n```\n\n")

	// Add file location and pattern
	filename := filepath.Base(match.path)
	content.WriteString(fmt.Sprintf("**File:** %s:%d\n", filename, match.line))
	content.WriteString(fmt.Sprintf("**Pattern:** `%s`", match.stepDef.Pattern))

	return content.String()
}

// formatAmbiguousStepDefs formats hover content for multiple matching step definitions.
func formatAmbiguousStepDefs(matches []stepDefMatch) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("**Step Definitions (%d matches)**\n\n", len(matches)))

	for i, match := range matches {
		if i > 0 {
			content.WriteString("\n---\n\n")
		}

		sig := formatFunctionSignature(match.goFile.FileSet, match.stepFunc.Node)
		content.WriteString("```go\n")
		content.WriteString(sig)
		content.WriteString("\n```\n")

		filename := filepath.Base(match.path)
		content.WriteString(fmt.Sprintf("**File:** %s:%d\n", filename, match.line))
		content.WriteString(fmt.Sprintf("**Pattern:** `%s`\n", match.stepDef.Pattern))
	}

	return content.String()
}

// formatStepUsage formats hover content for step usage information.
func formatStepUsage(stepDef godogen.Step, refs []stepRefLocation) string {
	var content strings.Builder
	content.WriteString("**Step Definition**\n\n")
	content.WriteString(fmt.Sprintf("**Pattern:** `%s`\n", stepDef.Pattern))
	content.WriteString(fmt.Sprintf("**Kind:** %s\n", stepDef.Kind))

	count := len(refs)
	if count == 0 {
		content.WriteString("**Used in:** 0 places (unused)")
	} else {
		placeWord := "place"
		if count > 1 {
			placeWord = "places"
		}
		content.WriteString(fmt.Sprintf("**Used in:** %d %s\n\n", count, placeWord))

		for _, ref := range refs {
			filename := filepath.Base(ref.path)
			content.WriteString(fmt.Sprintf("- %s:%d\n", filename, ref.line))
		}
	}

	return content.String()
}

// formatFunctionSignature formats a function signature for display.
func formatFunctionSignature(fset *token.FileSet, funcDecl *ast.FuncDecl) string {
	var buf bytes.Buffer

	// Create a copy of the function declaration without doc comments
	funcDeclCopy := *funcDecl
	funcDeclCopy.Doc = nil

	// Format the function declaration without doc
	if err := format.Node(&buf, fset, &funcDeclCopy); err != nil {
		// Fallback to just the name if formatting fails
		return "func " + funcDecl.Name.Name
	}

	// Extract just the signature (first line before the body)
	fullText := buf.String()

	// Find the opening brace or end of signature
	lines := strings.Split(fullText, "\n")
	if len(lines) > 0 {
		// The first line is the signature
		sig := strings.TrimSpace(lines[0])

		// Remove trailing body markers if present
		sig = strings.TrimSuffix(sig, " {}")
		sig = strings.TrimSuffix(sig, "{}")
		sig = strings.TrimSuffix(sig, " {")
		sig = strings.TrimSuffix(sig, "{")

		return strings.TrimSpace(sig)
	}

	return fullText
}

// DocumentSymbol represents a symbol in a document for the document symbols feature.
type DocumentSymbol struct {
	Name     string
	Kind     string
	Line     int
	Children []DocumentSymbol
}

// GetGoDocumentSymbols returns document symbols for a Go file.
// Returns step definitions and hooks as symbols.
func (index *Index) GetGoDocumentSymbols(path string) []DocumentSymbol {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.GoFiles[path]
	if versions == nil {
		return nil
	}

	goFile := versions.get()
	if goFile == nil {
		return nil
	}

	var symbols []DocumentSymbol

	// Add step definitions
	for _, stepDef := range goFile.AllSteps() {
		pos := goFile.Position(stepDef.Node.Pos())
		// Escape backslashes in pattern for display
		escapedPattern := strings.ReplaceAll(stepDef.Pattern, `\`, `\\`)
		name := fmt.Sprintf("%s: %s", stepDef.Kind, escapedPattern)

		symbols = append(symbols, DocumentSymbol{
			Name: name,
			Kind: "Function",
			Line: pos.Line,
		})
	}

	// Add hooks
	for _, stepFunc := range goFile.StepFuncs {
		for _, hook := range stepFunc.Hooks {
			pos := goFile.Position(hook.Node.Pos())
			name := fmt.Sprintf("%s Hook", hook.Kind)

			symbols = append(symbols, DocumentSymbol{
				Name: name,
				Kind: "Function",
				Line: pos.Line,
			})
		}

		for _, hook := range stepFunc.StepHooks {
			pos := goFile.Position(hook.Node.Pos())
			name := fmt.Sprintf("%s Step Hook", hook.Kind)

			symbols = append(symbols, DocumentSymbol{
				Name: name,
				Kind: "Function",
				Line: pos.Line,
			})
		}
	}

	return symbols
}

// GetFeatureDocumentSymbols returns document symbols for a feature file.
// If withFeature is true, returns the Feature as root container with children.
// Otherwise, returns scenarios/rules as top-level symbols.
func (index *Index) GetFeatureDocumentSymbols(path string, withFeature bool) []DocumentSymbol {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.Features[path]
	if versions == nil {
		return nil
	}

	featureFile := versions.get()
	if featureFile == nil {
		return nil
	}

	var symbols []DocumentSymbol

	for _, child := range featureFile.Children {
		if child.Background != nil {
			bg := child.Background
			bgSymbol := DocumentSymbol{
				Name: "Background",
				Kind: "Method",
				Line: int(bg.Location.Line),
			}

			for _, step := range bg.Steps {
				bgSymbol.Children = append(bgSymbol.Children, DocumentSymbol{
					Name: fmt.Sprintf("%s %s", strings.TrimSpace(step.Keyword), step.Text),
					Kind: "Property",
					Line: int(step.Location.Line),
				})
			}

			symbols = append(symbols, bgSymbol)
		}

		if child.Rule != nil {
			rule := child.Rule
			ruleSymbol := DocumentSymbol{
				Name: fmt.Sprintf("Rule: %s", rule.Name),
				Kind: "Class",
				Line: int(rule.Location.Line),
			}

			for _, ruleChild := range rule.Children {
				if ruleChild.Background != nil {
					bg := ruleChild.Background
					bgSymbol := DocumentSymbol{
						Name: "Background",
						Kind: "Method",
						Line: int(bg.Location.Line),
					}

					for _, step := range bg.Steps {
						bgSymbol.Children = append(bgSymbol.Children, DocumentSymbol{
							Name: fmt.Sprintf("%s %s", strings.TrimSpace(step.Keyword), step.Text),
							Kind: "Property",
							Line: int(step.Location.Line),
						})
					}

					ruleSymbol.Children = append(ruleSymbol.Children, bgSymbol)
				}

				if ruleChild.Scenario != nil {
					scenario := ruleChild.Scenario
					scenarioSymbol := buildScenarioSymbol(scenario)
					ruleSymbol.Children = append(ruleSymbol.Children, scenarioSymbol)
				}
			}

			symbols = append(symbols, ruleSymbol)
		}

		if child.Scenario != nil {
			scenario := child.Scenario
			scenarioSymbol := buildScenarioSymbol(scenario)
			symbols = append(symbols, scenarioSymbol)
		}
	}

	if withFeature {
		featureSymbol := DocumentSymbol{
			Name:     fmt.Sprintf("Feature: %s", featureFile.Name),
			Kind:     "Module",
			Line:     int(featureFile.Location.Line),
			Children: symbols,
		}
		return []DocumentSymbol{featureSymbol}
	}

	return symbols
}

func buildScenarioSymbol(scenario *messages.Scenario) DocumentSymbol {
	keyword := "Scenario"
	if len(scenario.Examples) > 0 {
		keyword = "Scenario Outline"
	}

	scenarioSymbol := DocumentSymbol{
		Name: fmt.Sprintf("%s: %s", keyword, scenario.Name),
		Kind: "Method",
		Line: int(scenario.Location.Line),
	}

	for _, step := range scenario.Steps {
		scenarioSymbol.Children = append(scenarioSymbol.Children, DocumentSymbol{
			Name: fmt.Sprintf("%s %s", strings.TrimSpace(step.Keyword), step.Text),
			Kind: "Property",
			Line: int(step.Location.Line),
		})
	}

	return scenarioSymbol
}
