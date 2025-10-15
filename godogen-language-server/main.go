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
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	gherkin "github.com/cucumber/gherkin/go/v36"
	messages "github.com/cucumber/messages/go/v30"
	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
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
	handler *protocol.Handler
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

	server.handler = &protocol.Handler{
		// lifecycle
		Initialize: server.initialize,
		Shutdown:   server.shutdown,
		// document sync
		TextDocumentDidOpen:   server.textDocumentDidOpen,
		TextDocumentDidChange: server.textDocumentDidChange,
		// completion
		TextDocumentCompletion: server.textDocumentCompletion,
	}

	return server, nil
}

// === LSP Handlers

func (srv *Server) initialize(
	context *glsp.Context,
	params *protocol.InitializeParams,
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

	return protocol.InitializeResult{
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

// Returns: []CompletionItem | CompletionList | nil
func (srv *Server) textDocumentCompletion(
	context *glsp.Context,
	params *protocol.CompletionParams,
) (any, error) {
	candidates := []protocol.CompletionItem{}

	for uri := range srv.index.Features {
		candidates = append(candidates, protocol.CompletionItem{
			Label:  uri,
			Detail: opt("Feature file"),
		})
	}

	for uri := range srv.index.GoFiles {
		slog.Info("adding go file to completion", "uri", uri, "all", srv.index.GoFiles)
		candidates = append(candidates, protocol.CompletionItem{
			Label:  uri,
			Detail: opt("Go file"),
		})
	}

	return candidates, nil
}

func opt[T any](p T) *T {
	return &p
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
		srv.server.GetStdio().Close(),
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

	fset := token.NewFileSet()

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

	file.File, err = parser.ParseFile(fset, uri, content, parser.ParseComments)
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
