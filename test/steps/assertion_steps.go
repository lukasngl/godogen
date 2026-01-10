package steps

import (
	"fmt"
	"strings"

	"github.com/cucumber/godog"
)

//godogen:then ^the command should succeed$
func (tc *TestContext) theCommandShouldSucceed() error {
	if tc.lastExitCode != 0 {
		return fmt.Errorf("expected exit code 0, got %d\noutput:\n%s", tc.lastExitCode, tc.lastOutput)
	}
	return nil
}

//godogen:then ^the command should fail$
func (tc *TestContext) theCommandShouldFail() error {
	if tc.lastExitCode == 0 {
		return fmt.Errorf("expected non-zero exit code, got 0\noutput:\n%s", tc.lastOutput)
	}
	return nil
}

//godogen:then ^the command output should be:$
func (tc *TestContext) theCommandOutputShouldBe(expected *godog.DocString) error {
	normalized := tc.NormalizeOutput(tc.lastOutput)
	expectedContent := expected.Content

	if normalized != expectedContent {
		return fmt.Errorf("output mismatch\nexpected:\n%s\n\ngot:\n%s", expectedContent, normalized)
	}
	return nil
}

//godogen:then ^the command output should be empty$
func (tc *TestContext) theCommandOutputShouldBeEmpty() error {
	if tc.lastOutput != "" {
		return fmt.Errorf("expected empty output, got:\n%s", tc.lastOutput)
	}
	return nil
}

//godogen:then ^the command output should contain "([^"]*)"$
func (tc *TestContext) theCommandOutputShouldContain(substring string) error {
	if !strings.Contains(tc.lastOutput, substring) {
		return fmt.Errorf("output does not contain %q\noutput:\n%s", substring, tc.lastOutput)
	}
	return nil
}

//godogen:then ^the command output should contain:$
func (tc *TestContext) theCommandOutputShouldContainDocstring(expected *godog.DocString) error {
	normalized := tc.NormalizeOutput(tc.lastOutput)
	if !strings.Contains(normalized, expected.Content) {
		return fmt.Errorf("output does not contain expected content\nexpected:\n%s\n\ngot:\n%s", expected.Content, normalized)
	}
	return nil
}

//godogen:then ^the file "([^"]*)" should exist$
func (tc *TestContext) theFileShouldExist(filename string) error {
	if !tc.FileExists(filename) {
		return fmt.Errorf("file %q does not exist", filename)
	}
	return nil
}

//godogen:then ^the file "([^"]*)" should not exist$
func (tc *TestContext) theFileShouldNotExist(filename string) error {
	if tc.FileExists(filename) {
		return fmt.Errorf("file %q exists but should not", filename)
	}
	return nil
}

//godogen:then ^the file "([^"]*)" should have content:$
func (tc *TestContext) theFileShouldHaveContent(filename string, expected *godog.DocString) error {
	content, err := tc.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}

	// Normalize: trim trailing whitespace from both
	expectedContent := strings.TrimRight(expected.Content, " \t\n")
	actualContent := strings.TrimRight(content, " \t\n")

	if actualContent != expectedContent {
		return fmt.Errorf("file %q content mismatch\nexpected:\n%s\n\ngot:\n%s", filename, expectedContent, actualContent)
	}
	return nil
}

//godogen:then ^the file "([^"]*)" should contain:$
func (tc *TestContext) theFileShouldContain(filename string, expected *godog.DocString) error {
	content, err := tc.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}

	if !strings.Contains(content, expected.Content) {
		return fmt.Errorf("file %q does not contain expected content\nexpected:\n%s\n\ngot:\n%s", filename, expected.Content, content)
	}
	return nil
}
