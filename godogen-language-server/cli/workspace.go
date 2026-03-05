// Package cli provides CLI utilities for the godogen language server.
package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/lukasngl/godogen/godogen-language-server/index"
)

// Config contains godogen language server configuration.
type Config struct {
	StepPatterns []string `json:"stepPatterns"`
}

// Workspace holds the index and configuration for CLI commands.
type Workspace struct {
	Index  *index.Index
	Root   string
	Config Config
}

// LoadWorkspace loads a workspace from the given root directory.
// If configPath is empty, it looks for .godogen-language-server.json in the root.
func LoadWorkspace(root, configPath string) (*Workspace, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root path: %w", err)
	}

	// Load config
	config := loadConfig(absRoot, configPath)

	// Create index
	idx := index.NewIndex()
	idx.WorkspaceRoot = absRoot

	// Discover and index files
	if err := discoverFiles(absRoot, config.StepPatterns, idx); err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	return &Workspace{
		Index:  idx,
		Root:   absRoot,
		Config: config,
	}, nil
}

// loadConfig loads configuration from the config file.
func loadConfig(root, configPath string) Config {
	config := Config{
		StepPatterns: []string{"**"},
	}

	if configPath == "" {
		configPath = filepath.Join(root, ".godogen-language-server.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return config
	}

	var fileConfig Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return config
	}

	if len(fileConfig.StepPatterns) > 0 {
		config.StepPatterns = fileConfig.StepPatterns
	}

	return config
}

// discoverFiles discovers all .go and .feature files matching the patterns and indexes them.
func discoverFiles(root string, patterns []string, idx *index.Index) error {
	for _, pattern := range patterns {
		resolvedPattern := pattern
		if !filepath.IsAbs(pattern) {
			resolvedPattern = filepath.Join(root, pattern)
		}

		if err := discoverPattern(resolvedPattern, idx); err != nil {
			return err
		}
	}

	return nil
}

// discoverPattern discovers files matching a specific glob pattern.
func discoverPattern(pattern string, idx *index.Index) error {
	base := getWatchDir(pattern)

	fsys := os.DirFS(base)

	relPattern, err := filepath.Rel(base, pattern)
	if err != nil {
		relPattern = pattern
	}

	matches, err := doublestar.Glob(fsys, relPattern)
	if err != nil {
		return fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
	}

	for _, relMatch := range matches {
		match := filepath.Join(base, relMatch)

		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		ext := filepath.Ext(match)
		if ext != ".go" && ext != ".feature" {
			continue
		}

		content, err := os.ReadFile(match)
		if err != nil {
			continue
		}

		switch ext {
		case ".go":
			_ = idx.IndexDiskGoFile(match, content)
		case ".feature":
			_ = idx.IndexDiskFeatureFile(match, content)
		}
	}

	return nil
}

// getWatchDir returns the directory component before any glob wildcards.
func getWatchDir(pattern string) string {
	cleaned := filepath.Clean(pattern)

	parts := splitPath(cleaned)
	var baseParts []string

	for _, part := range parts {
		if containsGlobChars(part) {
			break
		}
		baseParts = append(baseParts, part)
	}

	if len(baseParts) == 0 {
		return "."
	}

	result := filepath.Join(baseParts...)

	if filepath.IsAbs(cleaned) && !filepath.IsAbs(result) {
		result = string(filepath.Separator) + result
	}

	return result
}

func splitPath(path string) []string {
	var parts []string
	for path != "" {
		dir, file := filepath.Split(path)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		if dir == "" || dir == string(filepath.Separator) {
			if dir == string(filepath.Separator) {
				parts = append([]string{string(filepath.Separator)}, parts...)
			}
			break
		}
		path = filepath.Clean(dir)
	}
	return parts
}

func containsGlobChars(s string) bool {
	for _, c := range s {
		if c == '*' || c == '?' || c == '[' || c == ']' {
			return true
		}
	}
	return false
}

// AllGoFiles returns an iterator over all Go file paths in the workspace.
func (w *Workspace) AllGoFiles() []string {
	var files []string
	for path := range w.Index.GoFiles {
		files = append(files, path)
	}
	return files
}

// AllFeatureFiles returns an iterator over all feature file paths in the workspace.
func (w *Workspace) AllFeatureFiles() []string {
	var files []string
	for path := range w.Index.Features {
		files = append(files, path)
	}
	return files
}

// WalkGoFiles walks all .go files in the workspace that match patterns.
func WalkGoFiles(root string, patterns []string, fn func(path string, content []byte) error) error {
	for _, pattern := range patterns {
		resolvedPattern := pattern
		if !filepath.IsAbs(pattern) {
			resolvedPattern = filepath.Join(root, pattern)
		}

		base := getWatchDir(resolvedPattern)

		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".go" {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			return fn(path, content)
		})
		if err != nil {
			return err
		}
	}

	return nil
}
