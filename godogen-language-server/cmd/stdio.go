package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	gherkin "github.com/cucumber/gherkin/go/v36"
	"github.com/lukasngl/godogen/godogen-language-server/fsys"
	"github.com/lukasngl/godogen/godogen-language-server/index"
	"github.com/spf13/cobra"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	protocol17 "github.com/tliron/glsp/protocol_3_17"
	"github.com/tliron/glsp/server"
)

const lsName = "godogen-language-server"

var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Start the language server using stdio",
	Long:  `Starts the Godogen language server using stdio transport.`,
	RunE:  runStdio,
}

func init() {
	rootCmd.AddCommand(stdioCmd)
}

func runStdio(cmd *cobra.Command, args []string) error {
	srv, err := NewServer()
	if err != nil {
		return err
	}
	defer srv.Close()

	return srv.server.RunStdio()
}

// Server implements the language server.
type Server struct {
	index   *index.Index
	watcher *fsys.Watcher
	server  *server.Server
	handler *protocol17.Handler
	cancel  context.CancelFunc
}

// NewServer creates a new language server instance.
func NewServer() (*Server, error) {
	srv := &Server{
		index: index.NewIndex(),
	}

	watcher, err := fsys.NewWatcher(srv.onDiskFileChanged, srv.onDiskFileDeleted)
	if err != nil {
		return nil, err
	}

	srv.watcher = watcher

	srv.handler = &protocol17.Handler{
		// diagnostics
		TextDocumentDiagnostic: srv.textDocumentDiagnostic,
		// lifecycle
		Initialize: srv.initialize,
		Handler: protocol.Handler{
			Shutdown: srv.shutdown,
			// document sync
			TextDocumentDidOpen:   srv.textDocumentDidOpen,
			TextDocumentDidChange: srv.textDocumentDidChange,
			TextDocumentDidClose:  srv.textDocumentDidClose,
			// completion
			TextDocumentCompletion: srv.textDocumentCompletion,
			// navigation
			TextDocumentDefinition:     srv.textDocumentDefinition,
			TextDocumentImplementation: srv.textDocumentImplementation,
			TextDocumentReferences:     srv.textDocumentReferences,
			// autofix
			TextDocumentCodeAction: srv.textDocumentCodeAction,
		},
	}

	srv.server = server.NewServer(srv.handler, lsName, true)

	return srv, nil
}

// Close closes the server and releases resources.
func (srv *Server) Close() error {
	return srv.watcher.Close()
}

// === Disk file change handlers

func (srv *Server) onDiskFileChanged(path string, content []byte) error {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		return srv.index.IndexDiskGoFile(path, content)
	case ".feature":
		return srv.index.IndexDiskFeatureFile(path, content)
	}
	return nil
}

func (srv *Server) onDiskFileDeleted(path string) {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		srv.index.RemoveDiskGoFile(path)
	case ".feature":
		srv.index.RemoveDiskFeatureFile(path)
	}
}

// === LSP Handlers

func (srv *Server) initialize(
	_ *glsp.Context,
	params *protocol17.InitializeParams,
) (any, error) {
	slog.Info("Initializing "+lsName, "version", version, "rootURI", *params.RootURI)

	path, isFile := strings.CutPrefix(*params.RootURI, "file://")
	if !isFile {
		return nil, fmt.Errorf("root URI is not a file path: %s", *params.RootURI)
	}

	// Start watching and discover files
	ctx := context.Background()
	if err := srv.watcher.DiscoverAndWatch(ctx, path); err != nil {
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

func (srv *Server) shutdown(_ *glsp.Context) error {
	return nil
}

func (srv *Server) textDocumentDidOpen(
	_ *glsp.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil
	}

	switch params.TextDocument.LanguageID {
	case "go":
		srv.index.IndexWorkspaceGoFile(path, []byte(params.TextDocument.Text))
	case "cucumber":
		srv.index.IndexWorkspaceFeatureFile(path, []byte(params.TextDocument.Text))
	}

	return nil
}

func (srv *Server) textDocumentDidChange(
	_ *glsp.Context,
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
		srv.index.IndexWorkspaceGoFile(path, []byte(change.Text))
	case ".feature":
		srv.index.IndexWorkspaceFeatureFile(path, []byte(change.Text))
	}

	return nil
}

func (srv *Server) textDocumentDidClose(
	_ *glsp.Context,
	params *protocol.DidCloseTextDocumentParams,
) error {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil
	}

	// Remove only the workspace version
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		srv.index.RemoveWorkspaceGoFile(path)
	case ".feature":
		srv.index.RemoveWorkspaceFeatureFile(path)
	}

	// Disk version remains and will be used automatically
	return nil
}

// returns DocumentDiagnosticReport = RelatedFullDocumentDiagnosticReport | RelatedUnchangedDocumentDiagnosticReport
func (srv *Server) textDocumentDiagnostic(
	_ *glsp.Context,
	params *protocol17.DocumentDiagnosticParams,
) (any, error) {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil, nil
	}

	indexDiagnostics := srv.index.GetDiagnostics(path)
	if indexDiagnostics == nil {
		return nil, nil
	}

	var diagnostics []protocol.Diagnostic
	for _, diag := range indexDiagnostics {
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      protocol.UInteger(diag.StartLine - 1),
					Character: protocol.UInteger(diag.StartColumn - 1),
				},
				End: protocol.Position{
					Line:      protocol.UInteger(diag.EndLine - 1),
					Character: protocol.UInteger(diag.EndColumn - 1),
				},
			},
			Severity: box(protocol.DiagnosticSeverityWarning),
			Source:   box("godogen"),
			Message:  diag.Message,
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
	_ *glsp.Context,
	params *protocol.CodeActionParams,
) (any, error) {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil, nil
	}

	goFile := srv.index.GetGoFile(path)
	if goFile == nil {
		return nil, nil
	}

	var actions []protocol.CodeAction

	for validationErr := range goFile.ValidationErrors() {
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
	_ *glsp.Context,
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

	// TODO: handle partially files by using tree-sitter
	featureFile := srv.index.GetFeature(path)
	if featureFile == nil {
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
	if featureFile.Language != "" {
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
	_ *glsp.Context,
	params *protocol.ImplementationParams,
) (any, error) {
	return srv.getDefinitions(false, params.Position, params.TextDocument)
}

// Returns: Location | []Location | []LocationLink | nil
func (srv *Server) textDocumentDefinition(
	_ *glsp.Context,
	params *protocol.DefinitionParams,
) (any, error) {
	return srv.getDefinitions(true, params.Position, params.TextDocument)
}

func (srv *Server) getDefinitions(
	patternLoc bool,
	position protocol.Position,
	doc protocol.TextDocumentIdentifier,
) ([]protocol.Location, error) {
	slog.Info("textDocumentDefinition called",
		"uri", doc.URI,
		"line", position.Line,
	)

	path, isFile := strings.CutPrefix(doc.URI, "file://")
	if !isFile {
		slog.Info("not a file path")
		return nil, nil
	}

	indexLocs := srv.index.FindStepDefinitions(path, int(position.Line), patternLoc)

	var locs []protocol.Location
	for _, loc := range indexLocs {
		locs = append(locs, protocol.Location{
			URI: "file://" + loc.Path,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      protocol.UInteger(loc.Line - 1),
					Character: protocol.UInteger(loc.Column - 1),
				},
				End: protocol.Position{
					Line:      protocol.UInteger(loc.Line - 1),
					Character: protocol.UInteger(loc.Column - 1),
				},
			},
		})
	}

	return locs, nil
}

func (srv *Server) textDocumentReferences(
	_ *glsp.Context,
	params *protocol.ReferenceParams,
) ([]protocol.Location, error) {
	slog.Info("textDocumentReferences called",
		"uri", params.TextDocument.URI,
		"line", params.Position.Line,
	)

	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		slog.Info("not a file path")
		return nil, nil
	}

	indexLocs := srv.index.FindStepReferences(path, int(params.Position.Line))

	var locs []protocol.Location
	for _, loc := range indexLocs {
		locs = append(locs, protocol.Location{
			URI: "file://" + loc.Path,
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      protocol.UInteger(loc.Line - 1),
					Character: protocol.UInteger(loc.Column - 1),
				},
				End: protocol.Position{
					Line:      protocol.UInteger(loc.Line - 1),
					Character: protocol.UInteger(loc.Column - 1),
				},
			},
		})
	}

	return locs, nil
}

func box[T any](v T) *T {
	return &v
}
