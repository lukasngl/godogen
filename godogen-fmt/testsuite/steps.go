package testsuite

import (
	"errors"
	"strings"

	"github.com/cucumber/godog"
	"github.com/lukasngl/godogen/godogen-fmt/formatter"
)

// SetInput stores the input Gherkin content.
func (tc *TestContext) SetInput(content *godog.DocString) error {
	tc.input = content.Content
	return nil
}

// Format runs the formatter on the input.
func (tc *TestContext) Format() error {
	output, err := formatter.Format(tc.input)
	if err != nil {
		return err
	}
	tc.output = output
	return nil
}

// CheckOutput verifies the formatted output matches expected.
func (tc *TestContext) CheckOutput(expected *godog.DocString) error {
	// Doc strings don't include trailing newlines, but formatter adds one
	// Trim both for comparison
	got := strings.TrimRight(tc.output, "\n")
	want := strings.TrimRight(expected.Content, "\n")
	if got != want {
		return errors.New("output does not match expected:\n" +
			"=== GOT ===\n" + got + "\n" +
			"=== EXPECTED ===\n" + want)
	}
	return nil
}

// CheckEndsWithNewline verifies output ends with exactly one newline.
func (tc *TestContext) CheckEndsWithNewline() error {
	if !strings.HasSuffix(tc.output, "\n") {
		return errors.New("output does not end with newline")
	}
	if strings.HasSuffix(tc.output, "\n\n") {
		return errors.New("output ends with multiple newlines")
	}
	return nil
}
