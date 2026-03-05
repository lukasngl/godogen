package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

var (
	debugFlag   bool
	logFileFlag string
)

var stdioCmd = &cobra.Command{
	Use:   "stdio",
	Short: "Start the language server using stdio",
	Long:  `Starts the Godogen language server using stdio transport.`,
	RunE:  runStdio,
}

func init() {
	rootCmd.AddCommand(stdioCmd)
	stdioCmd.Flags().BoolVarP(&debugFlag, "debug", "d", false, "Enable debug logging")
	stdioCmd.Flags().StringVar(&logFileFlag, "log-file", "", "Write logs to file instead of stderr")
}

func runStdio(cmd *cobra.Command, args []string) error {
	if err := setupLogging(debugFlag, logFileFlag); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	srv, err := NewServer()
	if err != nil {
		return err
	}
	defer func() {
		if err := srv.Close(); err != nil {
			slog.Error("failed to close server", "error", err)
		}
	}()

	return srv.server.RunStdio()
}

// setupLogging configures structured logging with JSON output.
func setupLogging(debug bool, logFile string) error {
	// Determine log level
	level := slog.LevelError // Default: silent except errors
	if debug {
		level = slog.LevelDebug
	}

	// Determine output destination
	writer := os.Stderr
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		writer = f
	}

	// Create JSON handler for structured, filterable logs
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: level,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return nil
}

// Server implements the language server.
type Server struct {
	index   *index.Index
	watcher *fsys.Watcher
	server  *server.Server
	handler *protocol17.Handler
	cancel  context.CancelFunc

	// diagnosticsCache stores the last published diagnostics per file for code actions
	diagMu           sync.RWMutex
	diagnosticsCache map[string][]index.Diagnostic
}

// cacheDiagnostics stores diagnostics for a file path.
func (srv *Server) cacheDiagnostics(path string, diags []index.Diagnostic) {
	srv.diagMu.Lock()
	defer srv.diagMu.Unlock()
	srv.diagnosticsCache[path] = diags
}

// getCachedDiagnostics retrieves cached diagnostics for a file path.
func (srv *Server) getCachedDiagnostics(path string) []index.Diagnostic {
	srv.diagMu.RLock()
	defer srv.diagMu.RUnlock()
	return srv.diagnosticsCache[path]
}

// Config contains godogen language server configuration.
// This can be loaded from a .godogen-language-server.json file in the workspace root,
// or provided via LSP initialization options.
type Config struct {
	// StepPatterns is a list of glob patterns for discovering step definitions.
	// Patterns can be absolute or relative to the workspace root.
	// Examples: "**/*_steps.go", "../shared-steps/**/*.go", "**/*.feature"
	// Default: ["**"] (watch everything in workspace)
	StepPatterns []string `json:"stepPatterns"`
}

// NewServer creates a new language server instance.
func NewServer() (*Server, error) {
	srv := &Server{
		index:            index.NewIndex(),
		diagnosticsCache: make(map[string][]index.Diagnostic),
	}

	watcher, err := fsys.NewWatcher(srv.onDiskFileChanged, srv.onDiskFileDeleted)
	if err != nil {
		return nil, err
	}

	srv.watcher = watcher

	srv.handler = &protocol17.Handler{
		// diagnostics
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
			// hover
			TextDocumentHover: srv.textDocumentHover,
			// document symbols
			TextDocumentDocumentSymbol: srv.textDocumentDocumentSymbol,
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
	slog.Debug("initialize", "component", "lsp", "version", version, "rootURI", *params.RootURI)

	path, isFile := strings.CutPrefix(*params.RootURI, "file://")
	if !isFile {
		return nil, fmt.Errorf("root URI is not a file path: %s", *params.RootURI)
	}

	srv.index.WorkspaceRoot = path

	// Load configuration with precedence: LSP options > config file > defaults
	config := loadConfig(path, params.InitializationOptions)

	slog.Debug("config loaded", "component", "lsp", "stepPatterns", config.StepPatterns)

	// Start watching and discover files
	ctx := context.Background()
	if err := srv.watcher.DiscoverAndWatch(ctx, path, config.StepPatterns); err != nil {
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
	ctx *glsp.Context,
	params *protocol.DidOpenTextDocumentParams,
) error {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil
	}

	switch params.TextDocument.LanguageID {
	case "go":
		_ = srv.index.IndexWorkspaceGoFile(path, []byte(params.TextDocument.Text))
	case "cucumber":
		_ = srv.index.IndexWorkspaceFeatureFile(path, []byte(params.TextDocument.Text))
	}

	// Republish diagnostics for all files to handle cross-file dependencies
	srv.publishAllDiagnostics(ctx)

	return nil
}

func (srv *Server) textDocumentDidChange(
	ctx *glsp.Context,
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
		_ = srv.index.IndexWorkspaceGoFile(path, []byte(change.Text))
	case ".feature":
		_ = srv.index.IndexWorkspaceFeatureFile(path, []byte(change.Text))
	}

	// Republish diagnostics for all files to handle cross-file dependencies
	srv.publishAllDiagnostics(ctx)

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

// publishAllDiagnostics republishes diagnostics for all tracked files.
// This ensures cross-file dependencies are updated (e.g., feature files
// see changes to step definitions in Go files).
func (srv *Server) publishAllDiagnostics(ctx *glsp.Context) {
	goPaths, featurePaths := srv.index.AllFilePaths()
	for _, path := range goPaths {
		srv.publishDiagnostics(ctx, path)
	}
	for _, path := range featurePaths {
		srv.publishDiagnostics(ctx, path)
	}
}

// publishDiagnostics pushes diagnostics to the client via PublishDiagnostics notification.
func (srv *Server) publishDiagnostics(ctx *glsp.Context, path string) {
	uri := "file://" + path

	// Get diagnostics based on file type
	var indexDiagnostics []index.Diagnostic
	if filepath.Ext(path) == ".feature" {
		indexDiagnostics = srv.index.GetFeatureDiagnostics(path)
	} else {
		indexDiagnostics = srv.index.GetDiagnostics(path)
	}

	// Cache the diagnostics for code actions
	srv.cacheDiagnostics(path, indexDiagnostics)

	// Initialize with empty slice (not nil) to ensure JSON marshals to [] not null
	diagnostics := []protocol.Diagnostic{}

	for _, diag := range indexDiagnostics {
		// Map index severity to LSP severity
		var severity protocol.DiagnosticSeverity
		switch diag.Severity {
		case 1:
			severity = protocol.DiagnosticSeverityError
		case 2:
			severity = protocol.DiagnosticSeverityWarning
		case 3:
			severity = protocol.DiagnosticSeverityInformation
		case 4:
			severity = protocol.DiagnosticSeverityHint
		default:
			severity = protocol.DiagnosticSeverityWarning
		}

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
			Severity:           box(severity),
			Source:             box("godogen"),
			Message:            diag.Message,
			RelatedInformation: convertRelatedInfo(diag.RelatedInformation),
		})
	}

	// Send PublishDiagnostics notification to client
	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

// Returns: Command | []CodeAction | nil.
func (srv *Server) textDocumentCodeAction(
	_ *glsp.Context,
	params *protocol.CodeActionParams,
) (any, error) {
	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		return nil, nil
	}

	var actions []protocol.CodeAction

	// Handle feature file code actions
	if strings.HasSuffix(path, ".feature") {
		// Use cached diagnostics
		diags := srv.getCachedDiagnostics(path)

		for _, diag := range diags {
			if len(diag.Fixes) == 0 {
				continue
			}

			// Check if diagnostic intersects with requested range
			diagLine := protocol.UInteger(diag.StartLine - 1)
			if diagLine < params.Range.Start.Line || diagLine > params.Range.End.Line {
				continue
			}

			// Create a code action for each fix
			for _, fix := range diag.Fixes {
				actions = append(actions, protocol.CodeAction{
					Title: fix.Title,
					Kind:  box(protocol.CodeActionKindQuickFix),
					Edit: &protocol.WorkspaceEdit{
						Changes: map[protocol.DocumentUri][]protocol.TextEdit{
							params.TextDocument.URI: {{
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
								NewText: fix.NewText,
							}},
						},
					},
					Diagnostics: []protocol.Diagnostic{{
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
						Severity: box(convertSeverity(diag.Severity)),
						Source:   box("godogen"),
						Message:  diag.Message,
					}},
				})
			}
		}
		return actions, nil
	}

	// Handle Go file code actions
	goFile := srv.index.GetGoFile(path)
	if goFile == nil {
		return nil, nil
	}

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

// Returns: []CompletionItem | CompletionList | nil.
func (srv *Server) textDocumentCompletion(
	_ *glsp.Context,
	params *protocol.CompletionParams,
) (any, error) {
	slog.Debug("completion request", "component", "lsp", "uri", params.TextDocument.URI, "position", params.Position)

	candidates := []protocol.CompletionItem{}

	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		slog.Debug("not a file URI", "component", "lsp", "uri", params.TextDocument.URI)
		return nil, nil
	}

	// TODO: handle partially files by using tree-sitter
	featureFile := srv.index.GetFeature(path)
	if featureFile == nil {
		slog.Debug("not a feature file", "component", "lsp", "path", path)
		return nil, nil
	}

	dialectMap, ok := any(gherkin.DialectsBuiltin()).(map[string]*gherkin.Dialect)
	if ok {
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
		slog.Debug("no dialect found", "component", "lsp", "language", featureFile.Language)
		return nil, nil // well, nothing we can do
	}

	slog.Debug("providing completions", "component", "lsp", "language", dialect.Language, "count", len(candidates))

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

// Returns: Location | []Location | []LocationLink | nil.
func (srv *Server) textDocumentImplementation(
	_ *glsp.Context,
	params *protocol.ImplementationParams,
) (any, error) {
	return srv.getDefinitions(false, params.Position, params.TextDocument)
}

// Returns: Location | []Location | []LocationLink | nil.
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
	slog.Debug("definition request", "component", "lsp", "uri", doc.URI, "line", position.Line, "patternLoc", patternLoc)

	path, isFile := strings.CutPrefix(doc.URI, "file://")
	if !isFile {
		slog.Debug("not a file URI", "component", "lsp", "uri", doc.URI)
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
	slog.Debug("references request", "component", "lsp", "uri", params.TextDocument.URI, "line", params.Position.Line, "character", params.Position.Character)

	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		slog.Debug("not a file URI", "component", "lsp", "uri", params.TextDocument.URI)
		return nil, nil
	}

	// LSP positions are 0-indexed, but our index uses 1-indexed columns
	indexLocs := srv.index.FindStepReferences(path, int(params.Position.Line), int(params.Position.Character)+1)

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

// Returns: Hover | nil.
func (srv *Server) textDocumentHover(
	_ *glsp.Context,
	params *protocol.HoverParams,
) (*protocol.Hover, error) {
	slog.Debug("hover request", "component", "lsp", "uri", params.TextDocument.URI, "line", params.Position.Line, "character", params.Position.Character)

	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		slog.Debug("not a file URI", "component", "lsp", "uri", params.TextDocument.URI)
		return nil, nil
	}

	ext := filepath.Ext(path)

	switch ext {
	case ".feature":
		return srv.hoverOnFeatureFile(path, params.Position)
	case ".go":
		return srv.hoverOnGoFile(path, params.Position)
	default:
		return nil, nil
	}
}

func (srv *Server) hoverOnFeatureFile(path string, position protocol.Position) (*protocol.Hover, error) {
	hoverInfo := srv.index.GetHoverInfoForFeature(path, int(position.Line), int(position.Character)+1)
	if hoverInfo == nil {
		return nil, nil
	}

	content := protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: hoverInfo.Content,
	}

	return &protocol.Hover{
		Contents: content,
	}, nil
}

func (srv *Server) hoverOnGoFile(path string, position protocol.Position) (*protocol.Hover, error) {
	hoverInfo := srv.index.GetHoverInfoForGo(path, int(position.Line), int(position.Character)+1)
	if hoverInfo == nil {
		return nil, nil
	}

	content := protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: hoverInfo.Content,
	}

	return &protocol.Hover{
		Contents: content,
	}, nil
}

// Returns: []DocumentSymbol | []SymbolInformation | nil.
func (srv *Server) textDocumentDocumentSymbol(
	_ *glsp.Context,
	params *protocol.DocumentSymbolParams,
) (any, error) {
	slog.Debug("document symbols request", "component", "lsp", "uri", params.TextDocument.URI)

	path, isFile := strings.CutPrefix(params.TextDocument.URI, "file://")
	if !isFile {
		slog.Debug("not a file URI", "component", "lsp", "uri", params.TextDocument.URI)
		return nil, nil
	}

	ext := filepath.Ext(path)
	var indexSymbols []index.DocumentSymbol

	switch ext {
	case ".go":
		indexSymbols = srv.index.GetGoDocumentSymbols(path)
	case ".feature":
		// For feature files, we check if client supports hierarchical symbols
		// by always providing hierarchical structure (Feature as root)
		indexSymbols = srv.index.GetFeatureDocumentSymbols(path, false)
	default:
		return nil, nil
	}

	if indexSymbols == nil {
		return nil, nil
	}

	// Convert index symbols to protocol symbols
	var protocolSymbols []protocol.DocumentSymbol
	for _, sym := range indexSymbols {
		protocolSymbols = append(protocolSymbols, convertDocumentSymbol(sym))
	}

	return protocolSymbols, nil
}

// convertDocumentSymbol converts an index.DocumentSymbol to protocol.DocumentSymbol.
func convertDocumentSymbol(sym index.DocumentSymbol) protocol.DocumentSymbol {
	// Map string kind to protocol.SymbolKind
	var kind protocol.SymbolKind
	switch sym.Kind {
	case "Function":
		kind = protocol.SymbolKindFunction
	case "Method":
		kind = protocol.SymbolKindMethod
	case "Property":
		kind = protocol.SymbolKindProperty
	case "Module":
		kind = protocol.SymbolKindModule
	case "Class":
		kind = protocol.SymbolKindClass
	default:
		kind = protocol.SymbolKindFunction
	}

	// LSP uses 0-indexed lines
	line := protocol.UInteger(sym.Line - 1)

	// Create range for the symbol (we use line-based ranges)
	symbolRange := protocol.Range{
		Start: protocol.Position{
			Line:      line,
			Character: 0,
		},
		End: protocol.Position{
			Line:      line,
			Character: 1000, // Use a large number to cover the whole line
		},
	}

	protocolSym := protocol.DocumentSymbol{
		Name:           sym.Name,
		Kind:           kind,
		Range:          symbolRange,
		SelectionRange: symbolRange,
	}

	// Convert children recursively
	if len(sym.Children) > 0 {
		for _, child := range sym.Children {
			protocolSym.Children = append(protocolSym.Children, convertDocumentSymbol(child))
		}
	}

	return protocolSym
}

func box[T any](v T) *T {
	return &v
}

func convertSeverity(s index.DiagnosticSeverity) protocol.DiagnosticSeverity {
	switch s {
	case index.DiagnosticSeverityError:
		return protocol.DiagnosticSeverityError
	case index.DiagnosticSeverityWarning:
		return protocol.DiagnosticSeverityWarning
	case index.DiagnosticSeverityHint:
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityInformation
	}
}

func convertRelatedInfo(rels []index.DiagnosticRelatedInformation) []protocol.DiagnosticRelatedInformation {
	if len(rels) == 0 {
		return nil
	}
	result := make([]protocol.DiagnosticRelatedInformation, 0, len(rels))
	for _, rel := range rels {
		result = append(result, protocol.DiagnosticRelatedInformation{
			Location: protocol.Location{
				URI: protocol.DocumentUri("file://" + rel.Path),
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      protocol.UInteger(rel.Line - 1),
						Character: protocol.UInteger(rel.Column - 1),
					},
					End: protocol.Position{
						Line:      protocol.UInteger(rel.Line - 1),
						Character: protocol.UInteger(rel.Column - 1),
					},
				},
			},
			Message: rel.Message,
		})
	}
	return result
}

// loadConfig loads configuration with the following precedence:
// 1. LSP initialization options (highest priority)
// 2. .godogen-language-server.json in workspace root
// 3. Default values (lowest priority).
func loadConfig(workspaceRoot string, lspOptions any) Config {
	// Start with defaults
	config := Config{
		StepPatterns: []string{"**"},
	}

	// Try to load from config file
	configPath := filepath.Join(workspaceRoot, ".godogen-language-server.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var fileConfig Config
		if err := json.Unmarshal(data, &fileConfig); err == nil {
			slog.Debug("config file loaded", "component", "lsp", "path", configPath)
			if len(fileConfig.StepPatterns) > 0 {
				config.StepPatterns = fileConfig.StepPatterns
			}
		} else {
			slog.Warn("failed to parse config file", "component", "lsp", "path", configPath, "error", err)
		}
	} else {
		slog.Debug("no config file", "component", "lsp", "path", configPath)
	}

	// Override with LSP initialization options if provided
	lspConfig := parseInitializationOptions(lspOptions)
	if len(lspConfig.StepPatterns) > 0 {
		config.StepPatterns = lspConfig.StepPatterns
		slog.Debug("using LSP init options", "component", "lsp", "stepPatterns", lspConfig.StepPatterns)
	}

	return config
}

// parseInitializationOptions parses the initialization options from the LSP client.
func parseInitializationOptions(raw any) Config {
	var options Config

	if raw == nil {
		return options
	}

	optsMap, ok := raw.(map[string]any)
	if !ok {
		slog.Debug("init options not a map", "component", "lsp")
		return options
	}

	// Parse stepPatterns
	if patternsRaw, ok := optsMap["stepPatterns"]; ok {
		if patternsList, ok := patternsRaw.([]any); ok {
			for _, p := range patternsList {
				if pattern, ok := p.(string); ok {
					options.StepPatterns = append(options.StepPatterns, pattern)
				} else {
					slog.Debug("invalid stepPattern value", "component", "lsp", "value", p)
				}
			}
		} else {
			slog.Debug("stepPatterns not an array", "component", "lsp", "type", fmt.Sprintf("%T", patternsRaw))
		}
	}

	return options
}
