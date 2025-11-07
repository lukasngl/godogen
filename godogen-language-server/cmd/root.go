package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version string
	commit  string
	date    string
)

var rootCmd = &cobra.Command{
	Use:   "godogen-language-server",
	Short: "Language server for Godogen BDD framework",
	Long: `A language server that provides IDE support for Godogen,
a BDD testing framework for Go using Gherkin syntax.

Provides features like:
  - Go-to-definition for step patterns
  - Find references for step implementations
  - Diagnostics for step validation errors
  - Auto-completion for Gherkin keywords
  - Code actions for suggested fixes`,
}

func SetVersion(v, c, d string) {
	version = v
	commit = c
	date = d
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(fmt.Sprintf("{{.Name}} %s (%s %s)\n", version, commit, date))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
