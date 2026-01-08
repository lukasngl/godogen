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

// Global flags for CLI commands.
var (
	rootDir    string
	configFile string
	outputFmt  string
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

func init() {
	rootCmd.PersistentFlags().StringVar(&rootDir, "root", ".", "Workspace root directory")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (default: .godogen-language-server.json in root)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "format", "f", "text", "Output format: text or json")
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
