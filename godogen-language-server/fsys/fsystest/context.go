package fsystest

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lukasngl/godogen/godogen-language-server/fsys"
	"github.com/lukasngl/godogen/godogen-language-server/index"
)

// TestContext holds the state for filesystem watcher testing.
type TestContext struct {
	// Test filesystem
	tempDir string
	// fsnotify watcher for test assertions
	fsWatcher *fsnotify.Watcher
	// Watcher under test
	watcher *fsys.Watcher
	// Index to track indexed files
	index *index.Index
	// Patterns being watched
	patterns []string
	// Indexed files tracker (thread-safe)
	mu             sync.RWMutex
	indexedFiles   map[string]bool
	reindexedFiles map[string]bool
	deletedFiles   map[string]bool
	// Context for watcher
	ctx    context.Context
	cancel context.CancelFunc
}

// MustNewFsysTestContext creates a new filesystem test context or panics on error.
func MustNewTestContext() *TestContext {
	tc, err := NewTestContext()
	if err != nil {
		panic(err)
	}

	return tc
}

// NewFsysTestContext creates a new filesystem test context.
func NewTestContext() (*TestContext, error) {
	tempDir, err := os.MkdirTemp("", "godogen-fsys-test-*")
	if err != nil {
		return nil, err
	}

	idx := index.NewIndex()
	ctx, cancel := context.WithCancel(context.Background())

	tc := &TestContext{
		tempDir:        tempDir,
		index:          idx,
		patterns:       []string{},
		indexedFiles:   make(map[string]bool),
		reindexedFiles: make(map[string]bool),
		deletedFiles:   make(map[string]bool),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Create fsnotify watcher that we can inspect in tests
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		cancel()
		_ = os.RemoveAll(tempDir)
		return nil, err
	}
	tc.fsWatcher = fsWatcher

	// Create watcher with callbacks that track indexing
	tc.watcher = fsys.NewWatcherWithFsnotify(fsWatcher, tc.onFileChanged, tc.onFileDeleted)

	return tc, nil
}

// Cleanup removes temporary files and closes the watcher.
func (tc *TestContext) Cleanup() error {
	if tc.cancel != nil {
		tc.cancel()
	}
	if tc.watcher != nil {
		if err := tc.watcher.Close(); err != nil {
			return err
		}
	}
	if tc.tempDir != "" {
		return os.RemoveAll(tc.tempDir)
	}
	return nil
}

// onFileChanged is called when a file is indexed.
func (tc *TestContext) onFileChanged(path string, content []byte) error {
	relPath, _ := filepath.Rel(tc.tempDir, path)

	tc.mu.Lock()
	// Check if already indexed (for reindex detection)
	if tc.indexedFiles[relPath] {
		tc.reindexedFiles[relPath] = true
	}
	tc.indexedFiles[relPath] = true
	tc.mu.Unlock()

	// Index the file
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		return tc.index.IndexDiskGoFile(path, content)
	case ".feature":
		return tc.index.IndexDiskFeatureFile(path, content)
	}
	return nil
}

// onFileDeleted is called when a file is deleted.
func (tc *TestContext) onFileDeleted(path string) {
	relPath, _ := filepath.Rel(tc.tempDir, path)

	tc.mu.Lock()
	tc.deletedFiles[relPath] = true
	tc.mu.Unlock()

	// Remove from index
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		tc.index.RemoveDiskGoFile(path)
	case ".feature":
		tc.index.RemoveDiskFeatureFile(path)
	}
}

// AddPattern adds a watch pattern.
func (tc *TestContext) AddPattern(pattern string) {
	tc.patterns = append(tc.patterns, pattern)
}

// CreateFile creates a file in the test filesystem.
func (tc *TestContext) CreateFile(path string, content []byte) error {
	fullPath := filepath.Join(tc.tempDir, path)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, content, 0o644)
}

// CreateDirectory creates a directory in the test filesystem.
func (tc *TestContext) CreateDirectory(path string) error {
	fullPath := filepath.Join(tc.tempDir, path)
	return os.MkdirAll(fullPath, 0o755)
}

// RunDiscovery runs the discovery process.
func (tc *TestContext) RunDiscovery() error {
	return tc.watcher.DiscoverAndWatch(tc.ctx, tc.tempDir, tc.patterns)
}

// eventually waits for a condition to become true with tight polling interval.
// Uses 1ms interval and 100ms timeout for local filesystem operations.
func (tc *TestContext) eventually(check func() bool) bool {
	timeout := time.After(100 * time.Millisecond)
	tick := time.Tick(1 * time.Millisecond)

	for {
		select {
		case <-timeout:
			return false
		case <-tick:
			if check() {
				return true
			}
		}
	}
}

// IsFileIndexed checks if a file was indexed (with eventual consistency).
func (tc *TestContext) IsFileIndexed(path string) bool {
	return tc.eventually(func() bool {
		tc.mu.RLock()
		defer tc.mu.RUnlock()
		return tc.indexedFiles[path]
	})
}

// IsFileReindexed checks if a file was reindexed (with eventual consistency).
func (tc *TestContext) IsFileReindexed(path string) bool {
	return tc.eventually(func() bool {
		tc.mu.RLock()
		defer tc.mu.RUnlock()
		return tc.reindexedFiles[path]
	})
}

// IsFileDeleted checks if a file was deleted from index (with eventual consistency).
func (tc *TestContext) IsFileDeleted(path string) bool {
	return tc.eventually(func() bool {
		tc.mu.RLock()
		defer tc.mu.RUnlock()
		return tc.deletedFiles[path]
	})
}

// IsDirectoryWatched checks if a directory is being watched (with eventual consistency).
func (tc *TestContext) IsDirectoryWatched(path string) bool {
	fullPath := filepath.Join(tc.tempDir, path)
	return tc.eventually(func() bool {
		watchList := tc.fsWatcher.WatchList()
		for _, watched := range watchList {
			if watched == fullPath {
				return true
			}
		}
		return false
	})
}
