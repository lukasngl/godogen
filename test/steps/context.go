//go:generate go run github.com/lukasngl/godogen

package steps

import (
	"os"
	"path/filepath"
	"strings"
)

// TestContext holds state for a single scenario.
type TestContext struct {
	tempDir      string
	lastExitCode int
	lastOutput   string
}

// NewTestContext creates a new test context with a temp directory.
func NewTestContext() (*TestContext, error) {
	dir, err := os.MkdirTemp("", "godogen-test-*")
	if err != nil {
		return nil, err
	}
	return &TestContext{tempDir: dir}, nil
}

// Cleanup removes the temp directory.
func (tc *TestContext) Cleanup() error {
	if tc.tempDir != "" {
		return os.RemoveAll(tc.tempDir)
	}
	return nil
}

// FilePath returns the full path for a filename in the temp directory.
func (tc *TestContext) FilePath(filename string) string {
	return filepath.Join(tc.tempDir, filename)
}

// NormalizeOutput replaces the temp dir path with /test for golden comparison.
func (tc *TestContext) NormalizeOutput(output string) string {
	return strings.ReplaceAll(output, tc.tempDir, "/test")
}

// GetNormalizedOutput returns the last command output with paths normalized.
func (tc *TestContext) GetNormalizedOutput() string {
	return tc.NormalizeOutput(tc.lastOutput)
}

// ReadFile reads a file from the temp directory.
func (tc *TestContext) ReadFile(filename string) (string, error) {
	content, err := os.ReadFile(tc.FilePath(filename))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// FileExists checks if a file exists in the temp directory.
func (tc *TestContext) FileExists(filename string) bool {
	_, err := os.Stat(tc.FilePath(filename))
	return err == nil
}
