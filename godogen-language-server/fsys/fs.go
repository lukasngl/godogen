// Package fsys implements discovery and file system watchers for feature and go files.
package fsys

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// OnFileChangedFunc is called when a file is created or modified on disk.
type OnFileChangedFunc func(path string, content []byte) error

// OnFileDeletedFunc is called when a file is deleted from disk.
type OnFileDeletedFunc func(path string)

// Watcher watches a directory for file changes and discovers existing files.
type Watcher struct {
	watcher       *fsnotify.Watcher
	onFileChanged OnFileChangedFunc
	onFileDeleted OnFileDeletedFunc
}

// NewWatcher creates a new file system watcher.
func NewWatcher(
	onFileChanged OnFileChangedFunc,
	onFileDeleted OnFileDeletedFunc,
) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Watcher{
		watcher:       fsWatcher,
		onFileChanged: onFileChanged,
		onFileDeleted: onFileDeleted,
	}, nil
}

// DiscoverAndWatch discovers all existing files in the root directory and starts watching for changes.
func (w *Watcher) DiscoverAndWatch(ctx context.Context, root string) error {
	// Add root to watcher
	if err := w.watcher.Add(root); err != nil {
		return fmt.Errorf("failed to add root to watcher: %w", err)
	}

	slog.Info("Watching directory", "path", root)

	// Discover existing files
	if err := w.discover(root); err != nil {
		return err
	}

	// Start watching for changes
	go w.watch(ctx)

	return nil
}

// discover walks the directory and indexes all existing files.
func (w *Watcher) discover(root string) error {
	fsys, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("failed to open workspace root: %w", err)
	}

	return fs.WalkDir(fsys.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk workspace root: %w", err)
		}

		if !d.Type().IsRegular() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".feature" {
			fullPath := filepath.Join(root, path)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				slog.Error("failed to read file during discovery", "path", fullPath, "error", err)
				return nil
			}
			if err := w.onFileChanged(fullPath, content); err != nil {
				slog.Error("failed to index file during discovery", "path", fullPath, "error", err)
			}
		}

		return nil
	})
}

// watch runs the file watching loop.
func (w *Watcher) watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-w.watcher.Events:
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

			ext := filepath.Ext(event.Name)
			if ext != ".go" && ext != ".feature" {
				continue
			}

			slog.Info("File event", "path", event.Name, "op", event.Op)

			if deleted {
				w.onFileDeleted(event.Name)
			} else {
				content, err := os.ReadFile(event.Name)
				if err != nil {
					slog.Error("failed to read file", "path", event.Name, "error", err)
					continue
				}
				if err := w.onFileChanged(event.Name, content); err != nil {
					slog.Error("failed to index file", "path", event.Name, "error", err)
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				slog.Info("file watcher errors channel closed")
				return
			}

			slog.Error("file watcher error", "error", err)
		}
	}
}

// Close closes the watcher.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}
