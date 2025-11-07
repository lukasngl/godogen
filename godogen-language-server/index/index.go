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
	slog.Debug("indexing file", "component", "index", "path", path, "isWorkspace", true, "type", "go")

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
	slog.Debug("indexing file", "component", "index", "path", path, "isWorkspace", false, "type", "go")

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
	slog.Debug("removing file", "component", "index", "path", path, "isWorkspace", true, "type", "go")

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
	slog.Debug("removing file", "component", "index", "path", path, "isWorkspace", false, "type", "go")

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
	slog.Debug("indexing file", "component", "index", "path", path, "isWorkspace", true, "type", "feature")

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
	slog.Debug("indexing file", "component", "index", "path", path, "isWorkspace", false, "type", "feature")

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
	slog.Debug("removing file", "component", "index", "path", path, "isWorkspace", true, "type", "feature")

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
	slog.Debug("removing file", "component", "index", "path", path, "isWorkspace", false, "type", "feature")

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

// Diagnostic represents a diagnostic message with position information.
type Diagnostic struct {
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Message     string
	Severity    DiagnosticSeverity
}

// GetDiagnostics returns validation errors for a Go file at the given path.
func (index *Index) GetDiagnostics(path string) []Diagnostic {
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

	// Check for unused step definitions
	for _, stepFunc := range goFile.StepFuncs {
		for _, stepDef := range stepFunc.Steps {
			// Skip invalid patterns - they already have validation errors
			if stepDef.Regexp == nil {
				continue
			}

			// Check if this step is used anywhere
			if index.isStepUsed(stepDef) {
				continue
			}

			// Report as unused with Hint severity
			start := goFile.Position(stepDef.Node.Pos())
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

	// Check for duplicate step definitions
	duplicateDiags := index.findDuplicateSteps(path)
	diagnostics = append(diagnostics, duplicateDiags...)

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

			// For "Step" kind, we need to track it separately for each specific kind it conflicts with
			if stepDef.Kind == "Step" {
				// Generic step conflicts with all specific kinds
				for _, kind := range []string{"Given", "When", "Then", "Step"} {
					key := stepKey{kind: kind, pattern: stepDef.Pattern}
					pos := goFile.Position(stepDef.Node.Pos())
					allSteps[key] = append(allSteps[key], Location{
						Path:   filePath,
						Line:   pos.Line,
						Column: pos.Column,
					})
				}
			} else {
				// Specific kind: track both the specific kind and the generic "Step" kind
				for _, kind := range []string{stepDef.Kind, "Step"} {
					key := stepKey{kind: kind, pattern: stepDef.Pattern}
					pos := goFile.Position(stepDef.Node.Pos())
					allSteps[key] = append(allSteps[key], Location{
						Path:   filePath,
						Line:   pos.Line,
						Column: pos.Column,
					})
				}
			}
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

		// For this step, collect all locations that conflict with it
		if stepDef.Kind == "Step" {
			// Generic step: check for conflicts with any kind
			for _, kind := range []string{"Given", "When", "Then", "Step"} {
				key := stepKey{kind: kind, pattern: stepDef.Pattern}
				if locs, exists := allSteps[key]; exists {
					for _, loc := range locs {
						// Skip the current step itself
						pos := goFile.Position(stepDef.Node.Pos())
						if loc.Path == path && loc.Line == pos.Line {
							continue
						}
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
						// Skip the current step itself
						pos := goFile.Position(stepDef.Node.Pos())
						if loc.Path == path && loc.Line == pos.Line {
							continue
						}
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

		// If we found duplicates, create a diagnostic
		if len(duplicates) > 0 {
			pos := goFile.Position(stepDef.Node.Pos())
			end := goFile.Position(stepDef.Node.End())

			// Add current step to the list to find the actual first occurrence
			allOccurrences := append([]Location{{
				Path:   path,
				Line:   pos.Line,
				Column: pos.Column,
			}}, duplicates...)

			// Find the first occurrence (earliest line in same file, then alphabetically first file)
			firstOcc := allOccurrences[0]
			for _, occ := range allOccurrences[1:] {
				// If both in same file, use earlier line
				if occ.Path == firstOcc.Path {
					if occ.Line < firstOcc.Line {
						firstOcc = occ
					}
				} else {
					// Different files, use alphabetically first
					if occ.Path < firstOcc.Path {
						firstOcc = occ
					}
				}
			}

			// All duplicates (including first) report the first occurrence
			message := fmt.Sprintf("Duplicate step definition: pattern already defined at %s:%d", firstOcc.Path, firstOcc.Line)

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

		// Check if any step definition matches this step
		hasMatch := false

		for _, goFileVersions := range index.GoFiles {
			goFile := goFileVersions.get()
			if goFile == nil {
				continue
			}

			for _, stepDef := range goFile.AllSteps() {
				if stepMatchesDefinition(kind, step, stepDef) {
					hasMatch = true
					break
				}
			}

			if hasMatch {
				break
			}
		}

		if !hasMatch {
			// Report diagnostic for undefined step
			diagnostics = append(diagnostics, Diagnostic{
				StartLine:   int(step.Location.Line),
				StartColumn: int(step.Location.Column),
				EndLine:     int(step.Location.Line),
				EndColumn:   int(step.Location.Column) + len(step.Keyword) + len(step.Text),
				Message:     fmt.Sprintf("No step definition found for: %s %s", kind, step.Text),
				Severity:    DiagnosticSeverityError,
			})
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

// Steps returns an iterator over all steps in the feature file.
// It yields pairs of step kind (Given/When/Then) and the step itself,
// handling And/But conjunction steps by inheriting the previous step's kind.
func (f FeatureFile) Steps() iter.Seq2[string, *messages.Step] {
	return func(yield func(string, *messages.Step) bool) {
		for _, child := range f.Children {
			if child.Background != nil {
				if !yieldSteps(child.Background.Steps, yield) {
					return
				}
			}

			if child.Rule != nil {
				for _, ruleChild := range child.Rule.Children {
					if ruleChild.Background != nil {
						if !yieldSteps(ruleChild.Background.Steps, yield) {
							return
						}
					}
					if ruleChild.Scenario != nil {
						if !yieldSteps(ruleChild.Scenario.Steps, yield) {
							return
						}
					}
				}
			}

			if child.Scenario != nil {
				if !yieldSteps(child.Scenario.Steps, yield) {
					return
				}
			}
		}
	}
}

func yieldSteps(steps []*messages.Step, yield func(string, *messages.Step) bool) bool {
	lastKind := messages.StepKeywordType_UNKNOWN
	for _, step := range steps {
		lastKind = conjugateKind(lastKind, step)
		if !yield(mapStepKeywordType(lastKind), step) {
			return false
		}
	}

	return true
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

	slog.Debug("finding references", "component", "index", "path", goPath, "line", line, "column", column)

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

func stepMatchesDefinition(kind string, step *messages.Step, stepDef godogen.Step) bool {
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
func formatUndefinedStep(step *messages.Step) string {
	return fmt.Sprintf("**No step definition found**\n\nNo matching step definition for:\n```gherkin\n%s%s\n```",
		step.Keyword, step.Text)
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
		content.WriteString(fmt.Sprintf("**Pattern:** `%s`", match.stepDef.Pattern))
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

	// Write function keyword and name
	buf.WriteString("func ")
	buf.WriteString(funcDecl.Name.Name)

	// Write function type (parameters and return values)
	if err := format.Node(&buf, fset, funcDecl.Type); err != nil {
		// Fallback to just the name if formatting fails
		return "func " + funcDecl.Name.Name
	}

	return buf.String()
}
