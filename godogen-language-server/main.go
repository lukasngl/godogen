package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	gherkin "github.com/cucumber/gherkin/go/v36"
	messages "github.com/cucumber/messages/go/v30"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/lukasngl/godogen"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	protocol17 "github.com/tliron/glsp/protocol_3_17"
	"github.com/tliron/glsp/server"
)

const lsName string = "godogen-language-server"

var version string = "0.0.1"

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt, os.Kill,
	)
	defer cancel()

	server, err := NewServer()
	if err != nil {
		return err
	}

	return server.Run(ctx)
}

type Server struct {
	index   *Index
	watcher *fsnotify.Watcher
	server  *server.Server
	handler *protocol17.Handler
	cancel  context.CancelFunc
}

func NewServer() (*Server, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	server := &Server{
		index:   NewIndex(),
		watcher: watcher,
	}

	server.handler = &protocol17.Handler{
		// diagnostics
		TextDocumentDiagnostic: server.textDocumentDiagnostic,
		// lifecycle
		Initialize: server.initialize,
		Handler: protocol.Handler{
			Shutdown: server.shutdown,
			// document sync
			TextDocumentDidOpen:   server.textDocumentDidOpen,
			TextDocumentDidChange: server.textDocumentDidChange,
			// completion
			TextDocumentCompletion: server.textDocumentCompletion,
			// navigation
			TextDocumentDefinition:     server.textDocumentDefinition,
			TextDocumentImplementation: server.textDocumentImplementation,
			TextDocumentReferences:     server.textDocumentReferences,
			// autofix
			TextDocumentCodeAction: server.textDocumentCodeAction,
		},
	}

	return server, nil
}

// === LSP Handlers

func (srv *Server) initialize(
	context *glsp.Context,
	params *protocol17.InitializeParams,
) (any, error) {
	slog.Info("Initializing "+lsName, "version", version, "rootURI", *params.RootURI)

	path, isFile := strings.CutPrefix(*params.RootURI, "file://")
	if isFile {
		err := srv.watcher.Add(path)
		if err != nil {
			return nil, fmt.Errorf("failed to add root URI to watcher: %w", err)
		}

		slog.Info("Watching root URI", "path", path)
	}

	err := srv.discoverFiles(path)
	if err != nil {
		return nil, err
	}

	capabilities := srv.handler.CreateServerCapabilities()
	capabilities.CompletionProvider = &protocol.CompletionOptions{}
	capabilities.TextDocumentSync = protocol.TextDocumentSyncKindFull

	return protocol17.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    lsName,
			Version: &version,
		},
	}, nil
}

func (srv *Server) shutdown(context *glsp.Context) error {
	srv.cancel()
	return nil
}

func (srv *Server) textDocumentDidOpen(
	context *glsp.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil
	}

	switch params.TextDocument.LanguageID {
	case "go":
		srv.index.GoFileChanged(path, []byte(params.TextDocument.Text))
	case "cucumber":
		srv.index.FeatureChanged(path, []byte(params.TextDocument.Text))
	}

	return nil
}

func (srv *Server) textDocumentDidChange(
	context *glsp.Context,
	params *protocol.DidChangeTextDocumentParams,
) error {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil
	}

	change := params.ContentChanges[0].(protocol.TextDocumentContentChangeEventWhole)

	// TODO: remember filetype when opening the file
	switch filepath.Ext(path) {
	case ".go":
		srv.index.GoFileChanged(path, []byte(change.Text))
	case ".feature":
		srv.index.FeatureChanged(path, []byte(change.Text))
	}

	return nil
}

// returns DocumentDiagnosticReport = RelatedFullDocumentDiagnosticReport | RelatedUnchangedDocumentDiagnosticReport
func (srv *Server) textDocumentDiagnostic(
	context *glsp.Context,
	params *protocol17.DocumentDiagnosticParams,
) (any, error) {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil, nil
	}

	srv.index.mx.RLock()
	defer srv.index.mx.RUnlock()

	goFile, isGoFile := srv.index.GoFiles[path]
	if !isGoFile {
		return nil, nil
	}

	diagnostics := []protocol.Diagnostic{}

	stepFuncs := godogen.GetStepDefinitions(goFile.FileSet, goFile.File)

	for validationErr := range stepFuncs.ValidationErrors() {
		start := goFile.Position(validationErr.Pos())
		end := goFile.Position(validationErr.End())

		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      protocol.UInteger(start.Line - 1),
					Character: protocol.UInteger(start.Column - 1),
				},
				End: protocol.Position{
					Line:      protocol.UInteger(end.Line - 1),
					Character: protocol.UInteger(end.Column - 1),
				},
			},
			Severity: box(protocol.DiagnosticSeverityWarning),
			Source:   box("godogen"),
			Message:  validationErr.Message,
		})
	}

	return protocol17.RelatedFullDocumentDiagnosticReport{
		FullDocumentDiagnosticReport: protocol17.FullDocumentDiagnosticReport{
			Kind:  string(protocol17.DocumentDiagnosticReportKindFull),
			Items: diagnostics,
		},
	}, nil
}

// Returns: Command | []CodeAction | nil
func (srv *Server) textDocumentCodeAction(
	context *glsp.Context,
	params *protocol.CodeActionParams,
) (any, error) {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil, nil
	}

	srv.index.mx.RLock()
	defer srv.index.mx.RUnlock()

	goFile, isGoFile := srv.index.GoFiles[path]
	if !isGoFile {
		return nil, nil
	}

	var actions []protocol.CodeAction

	stepFuncs := godogen.GetStepDefinitions(goFile.FileSet, goFile.File)

	for validationErr := range stepFuncs.ValidationErrors() {
		for _, fix := range validationErr.SuggestedFixes {

			start := goFile.Position(validationErr.Pos())
			end := goFile.Position(validationErr.End())

			var edits []protocol.TextEdit
			for _, textEdit := range fix.TextEdits {
				editStart := goFile.Position(textEdit.Pos)
				editEnd := goFile.Position(textEdit.End)

				edits = append(edits, protocol.TextEdit{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      protocol.UInteger(editStart.Line - 1),
							Character: protocol.UInteger(editStart.Column - 1),
						},
						End: protocol.Position{
							Line:      protocol.UInteger(editEnd.Line - 1),
							Character: protocol.UInteger(editEnd.Column - 1),
						},
					},
					NewText: string(textEdit.NewText),
				})
			}

			actions = append(actions, protocol.CodeAction{
				Title: fix.Message,
				Edit: &protocol.WorkspaceEdit{
					Changes: map[protocol.DocumentUri][]protocol.TextEdit{
						params.TextDocument.URI: edits,
					},
				},
				Diagnostics: []protocol.Diagnostic{{
					Range: protocol.Range{
						Start: protocol.Position{
							Line:      protocol.UInteger(start.Line - 1),
							Character: protocol.UInteger(start.Column - 1),
						},
						End: protocol.Position{
							Line:      protocol.UInteger(end.Line - 1),
							Character: protocol.UInteger(end.Column - 1),
						},
					},
					Severity: box(protocol.DiagnosticSeverityWarning),
					Source:   box("godogen"),
					Message:  validationErr.Message,
				}},
			})
		}
	}

	return actions, nil
}

// Returns: []CompletionItem | CompletionList | nil
func (srv *Server) textDocumentCompletion(
	context *glsp.Context,
	params *protocol.CompletionParams,
) (any, error) {
	slog.Info("textDocumentCompletion called",
		"uri", params.TextDocument.URI,
		"position", params.Position,
	)
	candidates := []protocol.CompletionItem{}

	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		slog.Info("not a file path")
		return nil, nil
	}

	// TODO: handle unparseable files
	featureFile, isFeature := srv.index.Features[path]
	if !isFeature {
		slog.Info("not a featureFile")
		return nil, nil
	}

	dialectMap, ok := any(gherkin.DialectsBuiltin()).(map[string]*gherkin.Dialect)
	if ok {
		slog.Info("providing language candidates")
		for lang := range dialectMap {
			candidates = append(candidates, protocol.CompletionItem{
				Label:  lang,
				Detail: &[]string{"Language"}[0],
			})
		}
	}

	language := "en"
	if featureFile != nil && featureFile.Language != "" {
		language = featureFile.Language
	}

	dialect := gherkin.DialectsBuiltin().GetDialect(language)
	if dialect == nil {
		// fallback to English
		dialect = gherkin.DialectsBuiltin().GetDialect("en")
	}

	if dialect == nil {
		slog.Info("no dialect found", "language", featureFile.Language)
		return nil, nil // well, nothing we can do
	}

	slog.Info("providing keyword candidates", "language", dialect.Language)
	for kind, keywords := range dialect.Keywords {
		for _, keyword := range keywords {
			candidates = append(candidates, protocol.CompletionItem{
				Label:  keyword,
				Detail: &kind,
			})
		}
	}

	return candidates, nil
}

// Returns: Location | []Location | []LocationLink | nil
func (srv *Server) textDocumentImplementation(
	context *glsp.Context,
	params *protocol.ImplementationParams,
) (any, error) {
	return srv.getDefinitions(context, false, params.Position, params.TextDocument)
}

// Returns: Location | []Location | []LocationLink | nil
func (srv *Server) textDocumentDefinition(
	context *glsp.Context,
	params *protocol.DefinitionParams,
) (any, error) {
	return srv.getDefinitions(context, true, params.Position, params.TextDocument)
}

func (srv *Server) getDefinitions(
	context *glsp.Context,
	patternLoc bool,
	position protocol.Position,
	doc protocol.TextDocumentIdentifier,
) ([]protocol.Location, error) {
	// TODO: optimize with indexes

	slog.Info("textDocumentDefinition called",
		"uri", doc.URI,
		"line", position.Line,
	)

	srv.index.mx.RLock()
	defer srv.index.mx.RUnlock()

	path, isFile := strings.CutPrefix(doc.URI, "file://")
	if !isFile {
		slog.Info("not a file path")
		return nil, nil
	}

	featureFile, found := srv.index.Features[path]
	if !found {
		slog.Info("not a feature file")
		return nil, nil
	}

	var locs []protocol.Location

	slog.Info("iterating steps", "file", featureFile)

	for kind, step := range Steps(featureFile) {
		slog.Info("checking step", "kind", kind, "text", step.Text)
		if step.Location.Line-1 != int64(position.Line) {
			slog.Info("not on the same line",
				"expected", position.Line,
				"got", step.Location.Line,
			)
			continue
		}

		slog.Info("matching against go filess",
			"path", path,
			"files", srv.index.GoFiles,
		)

		for path, goFile := range srv.index.GoFiles {
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

					locs = append(locs, protocol.Location{
						URI: "file://" + path,
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      protocol.UInteger(pos.Line - 1),
								Character: protocol.UInteger(pos.Column - 1),
							},
							End: protocol.Position{
								Line:      protocol.UInteger(pos.Line - 1),
								Character: protocol.UInteger(pos.Column - 1),
							},
						},
					})
				}
			}
		}
	}

	return locs, nil
}

func (srv *Server) textDocumentReferences(
	context *glsp.Context,
	params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	// TODO: optimize with indexes

	slog.Info("textDocumentDefinition called",
		"uri", params.TextDocument.URI,
		"line", params.Position.Line,
	)

	srv.index.mx.RLock()
	defer srv.index.mx.RUnlock()

	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		slog.Info("not a file path")
		return nil, nil
	}

	goFile, found := srv.index.GoFiles[path]
	if !found {
		slog.Info("not a go file")
		return nil, nil
	}

	var locs []protocol.Location

	slog.Info("iterating stepDefs", "file", goFile)

	stepFuncs := godogen.GetStepDefinitions(goFile.FileSet, goFile.File)

	for _, stepFunc := range stepFuncs {
		for _, stepDef := range stepFunc.Steps {
			if goFile.Position(stepDef.Node.Pos()).Line-1 != int(params.Position.Line) {
				continue
			}

			for path, featureFile := range srv.index.Features {
				for kind, step := range Steps(featureFile) {
					slog.Info("matching against def", "kind", stepDef.Kind, "text", stepDef.Pattern)
					// TODO: cache compiled regexes
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

					pos := step.Location

					locs = append(locs, protocol.Location{
						URI: "file://" + path,
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      protocol.UInteger(pos.Line - 1),
								Character: protocol.UInteger(pos.Column - 1),
							},
							End: protocol.Position{
								Line:      protocol.UInteger(pos.Line - 1),
								Character: protocol.UInteger(pos.Column - 1),
							},
						},
					})
				}
			}
		}
	}

	return locs, nil
}

func Steps(feature *messages.Feature) iter.Seq2[string, *messages.Step] {
	return func(yield func(string, *messages.Step) bool) {
		for _, child := range feature.Children {
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

func box[T any](v T) *T {
	return &v
}

// === Server Lifecycle

func (srv *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv.server = server.NewServer(srv.handler, lsName, true)
	srv.cancel = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-srv.watcher.Events:
				if !ok {
					slog.Info("file watcher events channel closed")
					return
				}

				fileinfo, err := os.Stat(event.Name)
				if (err != nil && !os.IsNotExist(err)) || (err == nil && fileinfo.IsDir()) {
					continue
				}

				deleted := os.IsNotExist(err) || event.Has(fsnotify.Remove) ||
					event.Has(fsnotify.Rename)

				switch filepath.Ext(event.Name) {
				case ".go":
					slog.Info("Go file event", "path", event.Name, "op", event.Op)
					if deleted {
						srv.index.GoFileDeleted(event.Name)
					} else {
						srv.index.GoFileChanged(event.Name, nil)
					}
				case ".feature":
					slog.Info("Feature file event", "path", event.Name, "op", event.Op)
					if deleted {
						srv.index.FeatureDeleted(event.Name)
					} else {
						srv.index.FeatureChanged(event.Name, nil)
					}
				}

			case err, ok := <-srv.watcher.Errors:
				if !ok {
					slog.Info("file watcher errors channel closed")
					return
				}

				slog.Error("file watcher error", "error", err)
			}
		}
	}()

	return srv.server.RunStdio()
}

func (srv *Server) Close() error {
	return errors.Join(
		srv.watcher.Close(),
	)
}

// === File Discovery

func (srv *Server) discoverFiles(root string) error {
	fsys, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("failed to open workspace root: %w", err)
	}

	return fs.WalkDir(fsys.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk workspace root: %w", err)
		}

		if !d.Type().IsRegular() {
			return nil // continue traversing
		}

		switch filepath.Ext(path) {
		case ".go":
			srv.index.GoFileChanged(filepath.Join(root, path), nil)
		case ".feature":
			srv.index.FeatureChanged(filepath.Join(root, path), nil)
		}

		return nil
	})
}

// === Index

type Index struct {
	mx sync.RWMutex

	// map from file URI to the parsed Gherkin document
	Features map[string]*messages.Feature
	GoFiles  map[string]GoFile

	// TODO: add more indexes for quick lookup
}

func NewIndex() *Index {
	return &Index{
		mx:       sync.RWMutex{},
		Features: make(map[string]*messages.Feature),
		GoFiles:  make(map[string]GoFile),
	}
}

type GoFile struct {
	*ast.File
	*token.FileSet
}

func (index *Index) GoFileChanged(uri string, content []byte) error {
	slog.Info("Go file changed", "uri", uri)

	var (
		file GoFile
		err  error
	)

	file.FileSet = token.NewFileSet()

	if content == nil {
		file, err := os.Open(uri)
		if err != nil {
			return err
		}

		content, err = io.ReadAll(file)
		if err != nil {
			return err
		}
	}

	if !bytes.Contains(content, []byte("\n//godogen:")) {
		slog.Info("file is does not contain directives", "uri", uri)
		index.GoFileDeleted(uri)
		return nil
	}

	file.File, err = parser.ParseFile(file.FileSet, uri, content, parser.ParseComments)
	if err != nil {
		return err
	}

	index.mx.Lock()
	defer index.mx.Unlock()
	index.GoFiles[uri] = file

	return nil
}

func (index *Index) GoFileDeleted(uri string) {
	slog.Info("Go file deleted", "uri", uri)

	index.mx.Lock()
	defer index.mx.Unlock()

	delete(index.GoFiles, uri)
}

func (index *Index) FeatureChanged(uri string, content []byte) error {
	slog.Info("Feature file changed", "uri", uri)

	var reader io.Reader

	if content == nil {
		file, err := os.Open(uri)
		if err != nil {
			return err
		}

		reader = file
	} else {
		reader = bytes.NewReader(content)
	}

	document, err := gherkin.ParseGherkinDocument(reader, uuid.NewString)
	if err != nil {
		slog.Error("failed to parse feature file", "uri", uri, "error", err)
		return err
	}

	index.mx.Lock()
	defer index.mx.Unlock()
	index.Features[uri] = document.Feature

	return nil
}

func (index *Index) FeatureDeleted(uri string) {
	slog.Info("Feature file deleted", "uri", uri)

	index.mx.Lock()
	defer index.mx.Unlock()

	delete(index.Features, uri)
}
