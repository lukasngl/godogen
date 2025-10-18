// Package index provides an in-memory index of Gherkin feature files and Go step definition files.
// It maintains separate versions for workspace (editor-open) and disk files, with workspace files
// taking precedence when both versions exist.
package index

import (
	"bytes"
	"go/parser"
	"go/token"
	"iter"
	"log/slog"
	"sync"

	gherkin "github.com/cucumber/gherkin/go/v36"
	messages "github.com/cucumber/messages/go/v30"
	"github.com/google/uuid"
	"github.com/lukasngl/godogen"
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
	slog.Info("Indexing Go file", "path", path, "isWorkspace", true)

	if !bytes.Contains(content, []byte("\n//godogen:")) {
		slog.Info("file does not contain directives", "path", path)
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
	slog.Info("Indexing Go file", "path", path, "isWorkspace", false)

	if !bytes.Contains(content, []byte("\n//godogen:")) {
		slog.Info("file does not contain directives", "path", path)
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
	slog.Info("Removing Go file from index", "path", path, "isWorkspace", true)

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
	slog.Info("Removing Go file from index", "path", path, "isWorkspace", false)

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
	slog.Info("Indexing feature file", "path", path, "isWorkspace", true)

	reader := bytes.NewReader(content)

	document, err := gherkin.ParseGherkinDocument(reader, uuid.NewString)
	if err != nil {
		slog.Error("failed to parse feature file", "path", path, "error", err)
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
	slog.Info("Indexing feature file", "path", path, "isWorkspace", false)

	reader := bytes.NewReader(content)

	document, err := gherkin.ParseGherkinDocument(reader, uuid.NewString)
	if err != nil {
		slog.Error("failed to parse feature file", "path", path, "error", err)
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
	slog.Info("Removing feature file from index", "path", path, "isWorkspace", true)

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
	slog.Info("Removing feature file from index", "path", path, "isWorkspace", false)

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

// Diagnostic represents a diagnostic message with position information.
type Diagnostic struct {
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Message     string
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
		})
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
		slog.Info("not a feature file", "path", featurePath)
		return nil
	}

	featureFile := versions.get()
	if featureFile == nil {
		slog.Info("not a feature file", "path", featurePath)
		return nil
	}

	var locs []Location

	slog.Info("iterating steps", "file", featureFile)

	for kind, step := range featureFile.Steps() {
		slog.Info("checking step", "kind", kind, "text", step.Text)
		if step.Location.Line-1 != int64(line) {
			slog.Info("not on the same line",
				"expected", line,
				"got", step.Location.Line,
			)
			continue
		}

		slog.Info("matching against go files",
			"path", featurePath,
			"files", index.GoFiles,
		)

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

// FindStepReferences finds feature steps that reference the Go step pattern at the given line.
func (index *Index) FindStepReferences(goPath string, line int) []Location {
	index.mx.RLock()
	defer index.mx.RUnlock()

	versions := index.GoFiles[goPath]
	if versions == nil {
		slog.Info("not a go file", "path", goPath)
		return nil
	}

	goFile := versions.get()
	if goFile == nil {
		slog.Info("not a go file", "path", goPath)
		return nil
	}

	var locs []Location

	slog.Info("iterating stepDefs", "file", goFile)

	for _, stepDef := range goFile.AllSteps() {
		if goFile.Position(stepDef.Node.Pos()).Line-1 != line {
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
