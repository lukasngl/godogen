//go:generate go run github.com/lukasngl/godogen
package fsystest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cucumber/godog"
)

//godogen:after
func (tc *TestContext) cleanupAfterSceanrio(
	ctx context.Context,
	_ *godog.Scenario,
	_ error,
) (context.Context, error) {
	return ctx, tc.Cleanup()
}

//godogen:given ^we watch pattern "([^"]+)"$
func (tc *TestContext) WatchPattern(pattern string) error {
	tc.AddPattern(pattern)
	return nil
}

//godogen:given ^file "([^"]+)" exists$
func (tc *TestContext) FileExists(path string) error {
	content := []byte("package test\n//godogen:given ^test step$\nfunc TestStep() {}")
	if filepath.Ext(path) == ".feature" {
		content = []byte("Feature: Test\n  Scenario: Test\n    Given test step")
	}
	return tc.CreateFile(path, content)
}

//godogen:given ^directory "([^"]+)" exists$
func (tc *TestContext) DirectoryExists(path string) error {
	return tc.CreateDirectory(path)
}

//godogen:given ^discovery has run$
func (tc *TestContext) DiscoveryHasRun() error {
	return tc.RunDiscovery()
}

//godogen:when ^discovery runs$
func (tc *TestContext) DiscoveryRuns() error {
	return tc.RunDiscovery()
}

//godogen:when ^file "([^"]+)" is created$
func (tc *TestContext) FileIsCreated(path string) error {
	content := []byte("package test\n//godogen:given ^test step$\nfunc TestStep() {}")
	if filepath.Ext(path) == ".feature" {
		content = []byte("Feature: Test\n  Scenario: Test\n    Given test step")
	}

	// Real filesystem operation - fsnotify will detect it
	return tc.CreateFile(path, content)
}

//godogen:when ^file "([^"]+)" is modified$
func (tc *TestContext) FileIsModified(path string) error {
	fullPath := filepath.Join(tc.tempDir, path)
	content := []byte("package test\n//godogen:given ^modified step$\nfunc ModifiedStep() {}")
	if filepath.Ext(path) == ".feature" {
		content = []byte("Feature: Modified\n  Scenario: Modified\n    Given modified step")
	}

	// Real filesystem operation - fsnotify will detect it
	return os.WriteFile(fullPath, content, 0o644)
}

//godogen:when ^file "([^"]+)" is deleted$
func (tc *TestContext) FileIsDeleted(path string) error {
	fullPath := filepath.Join(tc.tempDir, path)
	// Real filesystem operation - fsnotify will detect it
	return os.Remove(fullPath)
}

//godogen:when ^directory "([^"]+)" is created$
func (tc *TestContext) DirectoryIsCreated(path string) error {
	// Real filesystem operation - fsnotify will detect it
	return tc.CreateDirectory(path)
}

//godogen:when ^directory "([^"]+)" is created with file "([^"]+)"$
func (tc *TestContext) DirectoryIsCreatedWithFile(dirPath string, fileName string) error {
	if err := tc.CreateDirectory(dirPath); err != nil {
		return err
	}

	filePath := filepath.Join(dirPath, fileName)
	content := []byte("package test\n//godogen:given ^test step$\nfunc TestStep() {}")

	// Real filesystem operation - fsnotify will detect it
	return tc.CreateFile(filePath, content)
}

//godogen:then ^"([^"]+)" should be indexed$
func (tc *TestContext) FileShouldBeIndexed(path string) error {
	if !tc.IsFileIndexed(path) {
		return fmt.Errorf("file %q was not indexed", path)
	}
	return nil
}

//godogen:then ^"([^"]+)" should not be indexed$
func (tc *TestContext) FileShouldNotBeIndexed(path string) error {
	if tc.IsFileIndexed(path) {
		return fmt.Errorf("file %q was indexed but should not be", path)
	}
	return nil
}

//godogen:then ^"([^"]+)" should be reindexed$
func (tc *TestContext) FileShouldBeReindexed(path string) error {
	if !tc.IsFileReindexed(path) {
		return fmt.Errorf("file %q was not reindexed", path)
	}
	return nil
}

//godogen:then ^"([^"]+)" should be removed from index$
func (tc *TestContext) FileShouldBeRemovedFromIndex(path string) error {
	if !tc.IsFileDeleted(path) {
		return fmt.Errorf("file %q was not removed from index", path)
	}
	return nil
}

//godogen:then ^directory "([^"]+)" should be watched$
func (tc *TestContext) DirectoryShouldBeWatched(path string) error {
	if !tc.IsDirectoryWatched(path) {
		return fmt.Errorf("directory %q is not being watched", path)
	}
	return nil
}

//godogen:then ^directory "([^"]+)" should not be watched$
func (tc *TestContext) DirectoryShouldNotBeWatched(path string) error {
	if tc.IsDirectoryWatched(path) {
		return fmt.Errorf("directory %q is being watched but should not be", path)
	}
	return nil
}
