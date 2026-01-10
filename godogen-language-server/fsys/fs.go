// Package fsys implements discovery and file system watchers for feature and go files.
package fsys

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fsnotify/fsnotify"
)

// OnFileChangedFunc is called when a file is created or modified on disk.
type OnFileChangedFunc func(path string, content []byte) error

// OnFileDeletedFunc is called when a file is deleted from disk.
type OnFileDeletedFunc func(path string)

// Watcher watches directories for file changes and discovers existing files.
type Watcher struct {
	watcher       *fsnotify.Watcher
	onFileChanged OnFileChangedFunc
	onFileDeleted OnFileDeletedFunc
	// root is the workspace root directory used for resolving relative patterns
	root string
	// patterns are glob patterns for matching files
	patterns []string
	// watchedDirs tracks which directories are being watched
	watchedDirs map[string]bool
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

	return NewWatcherWithFsnotify(fsWatcher, onFileChanged, onFileDeleted), nil
}

// NewWatcherWithFsnotify creates a watcher with a provided fsnotify.Watcher.
// This is useful for testing where you want access to the underlying watcher.
func NewWatcherWithFsnotify(
	fsWatcher *fsnotify.Watcher,
	onFileChanged OnFileChangedFunc,
	onFileDeleted OnFileDeletedFunc,
) *Watcher {
	return &Watcher{
		watcher:       fsWatcher,
		onFileChanged: onFileChanged,
		onFileDeleted: onFileDeleted,
	}
}

// DiscoverAndWatch discovers all existing files matching the patterns and starts watching for changes.
// Patterns can be absolute or relative to the root directory.
// Examples: "**/*_steps.go", "../shared-steps/**/*.go", "/absolute/path/**/*.feature".
func (w *Watcher) DiscoverAndWatch(ctx context.Context, root string, patterns []string) error {
	w.root = root
	w.patterns = patterns
	w.watchedDirs = make(map[string]bool)

	slog.Debug("discovering files", "component", "fsys", "root", root, "patterns", patterns)

	// Discover and watch files for each pattern
	for _, pattern := range patterns {
		// Resolve pattern against root if relative
		resolvedPattern := pattern
		if !filepath.IsAbs(pattern) {
			resolvedPattern = filepath.Join(root, pattern)
		}

		slog.Debug("processing pattern", "component", "fsys", "pattern", pattern, "resolved", resolvedPattern)

		// Get the base directory to watch (the part before any wildcards)
		watchDir := getWatchDir(resolvedPattern)

		// Add directory to watcher if not already watching
		if !w.watchedDirs[watchDir] {
			if err := w.watcher.Add(watchDir); err != nil {
				slog.Debug("failed to watch directory", "component", "fsys", "path", watchDir, "error", err)
				continue
			}
			w.watchedDirs[watchDir] = true
			slog.Debug("watching directory", "component", "fsys", "path", watchDir)
		}

		// Discover existing files matching this pattern
		if err := w.discoverPattern(resolvedPattern); err != nil {
			slog.Error("failed to discover files", "component", "fsys", "pattern", resolvedPattern, "error", err)
		}
	}

	// Start watching for changes
	go w.watch(ctx)

	return nil
}

// getWatchDir returns the directory component before any glob wildcards.
// Example: "/path/to/**/*.go" -> "/path/to".
func getWatchDir(pattern string) string {
	cleaned := filepath.Clean(pattern)

	// Find the first component with a wildcard
	parts := strings.Split(cleaned, string(filepath.Separator))
	var baseParts []string

	for _, part := range parts {
		if strings.ContainsAny(part, "*?[]") {
			break
		}
		baseParts = append(baseParts, part)
	}

	if len(baseParts) == 0 {
		return "."
	}

	result := filepath.Join(baseParts...)

	// Preserve leading slash for absolute paths
	if filepath.IsAbs(cleaned) && !filepath.IsAbs(result) {
		result = string(filepath.Separator) + result
	}

	return result
}

// discoverPattern discovers files matching a specific glob pattern.
// Supports ** for recursive directory matching.
func (w *Watcher) discoverPattern(pattern string) error {
	// Get the base directory to search from (the part before any wildcards)
	base := getWatchDir(pattern)

	// Walk all directories under base and add them to the watcher
	// This is needed because fsnotify doesn't watch recursively
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Add directory to watcher if not already watching
			if !w.watchedDirs[path] {
				if err := w.watcher.Add(path); err != nil {
					slog.Debug("failed to watch directory", "component", "fsys", "path", path, "error", err)
				} else {
					w.watchedDirs[path] = true
					slog.Debug("watching directory", "component", "fsys", "path", path)
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory tree: %w", err)
	}

	// Now discover files matching the pattern
	fsys := os.DirFS(base)

	// Make pattern relative to base for doublestar matching
	relPattern, err := filepath.Rel(base, pattern)
	if err != nil {
		relPattern = pattern
	}

	matches, err := doublestar.Glob(fsys, relPattern)
	if err != nil {
		return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	slog.Debug("pattern matched", "component", "fsys", "pattern", pattern, "count", len(matches))

	for _, relMatch := range matches {
		match := filepath.Join(base, relMatch)

		info, err := os.Stat(match)
		if err != nil {
			slog.Debug("failed to stat file", "component", "fsys", "path", match, "error", err)
			continue
		}

		if !info.IsDir() {
			// Regular file - check if it's a .go or .feature file
			ext := filepath.Ext(match)
			if ext == ".go" || ext == ".feature" {
				content, err := os.ReadFile(match)
				if err != nil {
					slog.Error("failed to read file", "component", "fsys", "path", match, "error", err)
					continue
				}
				if err := w.onFileChanged(match, content); err != nil {
					slog.Error("failed to index file", "component", "fsys", "path", match, "error", err)
				}
			}
		}
	}

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
				slog.Error("failed to read file", "component", "fsys", "path", fullPath, "error", err)
				return nil
			}
			if err := w.onFileChanged(fullPath, content); err != nil {
				slog.Error("failed to index file", "component", "fsys", "path", fullPath, "error", err)
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
				slog.Debug("watcher closed", "component", "fsys")
				return
			}

			fileinfo, err := os.Stat(event.Name)

			// Handle directory creation
			if err == nil && fileinfo.IsDir() && event.Has(fsnotify.Create) {
				slog.Debug("directory created", "component", "fsys", "path", event.Name)
				w.handleNewDirectory(event.Name)
				continue
			}

			// Skip other directory events and errors
			if (err != nil && !os.IsNotExist(err)) || (err == nil && fileinfo.IsDir()) {
				continue
			}

			deleted := os.IsNotExist(err) || event.Has(fsnotify.Remove) ||
				event.Has(fsnotify.Rename)

			ext := filepath.Ext(event.Name)
			if ext != ".go" && ext != ".feature" {
				continue
			}

			slog.Debug("file event", "component", "fsys", "path", event.Name, "op", event.Op.String(), "deleted", deleted)

			if deleted {
				w.onFileDeleted(event.Name)
			} else {
				// Check if file matches any of our patterns
				if !w.matchesAnyPattern(event.Name) {
					continue
				}

				content, err := os.ReadFile(event.Name)
				if err != nil {
					slog.Error("failed to read file", "component", "fsys", "path", event.Name, "error", err)
					continue
				}
				if err := w.onFileChanged(event.Name, content); err != nil {
					slog.Error("failed to index file", "component", "fsys", "path", event.Name, "error", err)
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				slog.Debug("watcher closed", "component", "fsys")
				return
			}

			slog.Error("watcher error", "component", "fsys", "error", err)
		}
	}
}

// handleNewDirectory handles a newly created directory by adding it and its subdirectories
// to the watcher, and discovering any files that match our patterns.
func (w *Watcher) handleNewDirectory(dirPath string) {
	// Walk the new directory tree and add all directories to the watcher
	_ = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Debug("failed to walk directory", "component", "fsys", "dir", dirPath, "path", path, "error", err)
			return nil // Continue walking despite errors
		}

		if d.IsDir() {
			// Add directory to watcher
			if !w.watchedDirs[path] {
				if err := w.watcher.Add(path); err != nil {
					slog.Debug("failed to watch directory", "component", "fsys", "path", path, "error", err)
				} else {
					w.watchedDirs[path] = true
					slog.Debug("watching directory", "component", "fsys", "path", path)
				}
			}
		} else if d.Type().IsRegular() {
			// Check if file matches any of our patterns and has correct extension
			ext := filepath.Ext(path)
			if ext == ".go" || ext == ".feature" {
				// Check if file matches any pattern
				if w.matchesAnyPattern(path) {
					content, err := os.ReadFile(path)
					if err != nil {
						slog.Error("failed to read file", "component", "fsys", "path", path, "error", err)
						return nil
					}
					if err := w.onFileChanged(path, content); err != nil {
						slog.Error("failed to index file", "component", "fsys", "path", path, "error", err)
					}
				}
			}
		}

		return nil
	})
}

// matchesAnyPattern checks if a file path matches any of the configured patterns.
func (w *Watcher) matchesAnyPattern(filePath string) bool {
	for _, pattern := range w.patterns {
		// Resolve pattern against root if relative
		resolvedPattern := pattern
		if !filepath.IsAbs(pattern) {
			resolvedPattern = filepath.Join(w.root, pattern)
		}

		// Get base directory for the pattern
		base := getWatchDir(resolvedPattern)

		// Make pattern relative to base
		relPattern, err := filepath.Rel(base, resolvedPattern)
		if err != nil {
			continue
		}

		// Make file path relative to base
		relPath, err := filepath.Rel(base, filePath)
		if err != nil {
			continue
		}

		// Check if file matches pattern
		matched, err := doublestar.Match(relPattern, relPath)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// Close closes the watcher.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}
