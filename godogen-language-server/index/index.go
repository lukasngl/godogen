package index

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"iter"
	"log/slog"
	"os"
	"regexp"
	"sync"

	gherkin "github.com/cucumber/gherkin/go/v36"
	messages "github.com/cucumber/messages/go/v30"
	"github.com/google/uuid"
	"github.com/lukasngl/godogen"
)

type Index struct {
	mx sync.RWMutex

	// TODO: prefer workspace filess over disk files
	Features map[string]FeatureFile
	GoFiles  map[string]GoFile

	// TODO: add more indexes for quick lookup
}

func NewIndex() *Index {
	return &Index{
		mx:       sync.RWMutex{},
		Features: make(map[string]FeatureFile),
		GoFiles:  make(map[string]GoFile),
	}
}

type GoFile struct {
	*ast.File
	*token.FileSet

	isWorkspaceFile bool
}

type FeatureFile struct {
	*messages.Feature

	isWorkspaceFile bool
}

func (index *Index) IndexGoFile(path string, content []byte, isWorkspace bool) error {
	slog.Info("Indexing Go file", "path", path, "isWorkspace", isWorkspace)

	if content == nil {
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		content, err = io.ReadAll(file)
		if err != nil {
			return err
		}
	}

	if !bytes.Contains(content, []byte("\n//godogen:")) {
		slog.Info("file does not contain directives", "path", path)
		index.RemoveGoFile(path)
		return nil
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return err
	}

	index.mx.Lock()
	defer index.mx.Unlock()

	// Only index disk files if no workspace version exists
	if !isWorkspace {
		if existing, exists := index.GoFiles[path]; exists && existing.isWorkspaceFile {
			slog.Info("workspace version exists, skipping disk file", "path", path)
			return nil
		}
	}

	index.GoFiles[path] = GoFile{
		File:            file,
		FileSet:         fset,
		isWorkspaceFile: isWorkspace,
	}

	return nil
}

func (index *Index) RemoveGoFile(path string) {
	slog.Info("Removing Go file from index", "path", path)

	index.mx.Lock()
	defer index.mx.Unlock()

	delete(index.GoFiles, path)
}

func (index *Index) IndexFeatureFile(path string, content []byte, isWorkspace bool) error {
	slog.Info("Indexing feature file", "path", path, "isWorkspace", isWorkspace)

	var reader io.Reader

	if content == nil {
		file, err := os.Open(path)
		if err != nil {
			return err
		}

		reader = file
	} else {
		reader = bytes.NewReader(content)
	}

	document, err := gherkin.ParseGherkinDocument(reader, uuid.NewString)
	if err != nil {
		slog.Error("failed to parse feature file", "path", path, "error", err)
		return err
	}

	index.mx.Lock()
	defer index.mx.Unlock()

	// Only index disk files if no workspace version exists
	if !isWorkspace {
		if existing, exists := index.Features[path]; exists && existing.isWorkspaceFile {
			slog.Info("workspace version exists, skipping disk file", "path", path)
			return nil
		}
	}

	index.Features[path] = FeatureFile{
		Feature:         document.Feature,
		isWorkspaceFile: isWorkspace,
	}

	return nil
}

func (index *Index) RemoveFeatureFile(path string) {
	slog.Info("Removing feature file from index", "path", path)

	index.mx.Lock()
	defer index.mx.Unlock()

	delete(index.Features, path)
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

	goFile, found := index.GoFiles[path]
	if !found {
		return nil
	}

	stepFuncs := godogen.GetStepDefinitions(goFile.FileSet, goFile.File)

	var diagnostics []Diagnostic
	for validationErr := range stepFuncs.ValidationErrors() {
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

	if file, found := index.GoFiles[path]; found {
		return &file
	}
	return nil
}

// GetFeature returns the feature at the given path, or nil if not found.
func (index *Index) GetFeature(path string) *messages.Feature {
	index.mx.RLock()
	defer index.mx.RUnlock()

	if feature, found := index.Features[path]; found {
		return feature.Feature
	}
	return nil
}

// Steps returns an iterator over all steps in this feature file.
// It yields the step kind (Given/When/Then) and the step itself.
func (f FeatureFile) Steps() iter.Seq2[string, *messages.Step] {
	return func(yield func(string, *messages.Step) bool) {
		for _, child := range f.Feature.Children {
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

// Location represents a position in a file
type Location struct {
	Path   string
	Line   int
	Column int
}

// FindStepDefinitions finds go step definitions that match a feature step at the given line.
// If patternLoc is true, returns the location of the step pattern comment, otherwise returns the function location.
func (index *Index) FindStepDefinitions(featurePath string, line int, patternLoc bool) []Location {
	index.mx.RLock()
	defer index.mx.RUnlock()

	featureFile, found := index.Features[featurePath]
	if !found {
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

		for path, goFile := range index.GoFiles {
			stepFuncs := godogen.GetStepDefinitions(goFile.FileSet, goFile.File)
			slog.Info("matching against steps",
				"path", path,
				"steps", len(stepFuncs),
			)

			for _, stepFunc := range stepFuncs {
				for _, stepDef := range stepFunc.Steps {
					slog.Info("matching against def", "kind", stepDef.Kind, "text", stepDef.Pattern)
					matcher, err := regexp.Compile(stepDef.Pattern)
					if err != nil {
						continue
					}

					if stepDef.Kind != "Step" && stepDef.Kind != kind {
						continue
					}

					if !matcher.MatchString(step.Text) {
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
	}

	return locs
}

// FindStepReferences finds feature steps that reference the go step pattern at the given line.
func (index *Index) FindStepReferences(goPath string, line int) []Location {
	index.mx.RLock()
	defer index.mx.RUnlock()

	goFile, found := index.GoFiles[goPath]
	if !found {
		slog.Info("not a go file", "path", goPath)
		return nil
	}

	var locs []Location

	slog.Info("iterating stepDefs", "file", goFile)

	stepFuncs := godogen.GetStepDefinitions(goFile.FileSet, goFile.File)

	for _, stepFunc := range stepFuncs {
		for _, stepDef := range stepFunc.Steps {
			if goFile.Position(stepDef.Node.Pos()).Line-1 != line {
				continue
			}

			for path, featureFile := range index.Features {
				for kind, step := range featureFile.Steps() {
					slog.Info("matching against def", "kind", stepDef.Kind, "text", stepDef.Pattern)
					if stepDef.Regexp == nil {
						continue
					}

					if stepDef.Kind != "Step" && stepDef.Kind != kind {
						continue
					}

					if !stepDef.Regexp.MatchString(step.Text) {
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
	}

	return locs
}
