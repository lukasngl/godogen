package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.0.1"

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
	Version: version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}
