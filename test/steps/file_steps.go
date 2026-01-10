package steps

import (
	"os"

	"github.com/cucumber/godog"
)

//godogen:given ^a Go file "([^"]*)" with content:$
func (tc *TestContext) aGoFileWithContent(filename string, content *godog.DocString) error {
	return os.WriteFile(tc.FilePath(filename), []byte(content.Content), 0o644)
}
