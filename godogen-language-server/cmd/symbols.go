package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lukasngl/godogen/godogen-language-server/cli"
	"github.com/lukasngl/godogen/godogen-language-server/index"
	"github.com/spf13/cobra"
)

var symbolsCmd = &cobra.Command{
	Use:   "symbols <file>",
	Short: "Get document symbols for a file",
	Long: `Get document symbols for a Go or feature file.

For Go files: Lists step definitions and hooks.
For feature files: Lists scenarios, backgrounds, and rules.

Arguments:
  file  The file path

Example:
  godogen-language-server symbols features/login.feature
  godogen-language-server symbols steps/auth_steps.go`,
	Args: cobra.ExactArgs(1),
	RunE: runSymbols,
}

func init() {
	rootCmd.AddCommand(symbolsCmd)
}

type symbolOutput struct {
	Name     string         `json:"name"`
	Kind     string         `json:"kind"`
	Line     int            `json:"line"`
	Children []symbolOutput `json:"children,omitempty"`
}

type symbolsResult struct {
	Symbols []symbolOutput `json:"symbols"`
	Total   int            `json:"total"`
}

func runSymbols(_ *cobra.Command, args []string) error {
	file := args[0]

	ws, err := cli.LoadWorkspace(rootDir, configFile)
	if err != nil {
		return err
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}

	var indexSymbols []index.DocumentSymbol

	ext := filepath.Ext(absFile)
	switch ext {
	case ".feature":
		indexSymbols = ws.Index.GetFeatureDocumentSymbols(absFile, true)
	case ".go":
		indexSymbols = ws.Index.GetGoDocumentSymbols(absFile)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	symbols := convertSymbols(indexSymbols)
	total := countSymbols(symbols)

	result := symbolsResult{
		Symbols: symbols,
		Total:   total,
	}

	if outputFmt == "json" {
		return outputJSON(result)
	}

	return outputSymbolsText(result.Symbols, 0)
}

func convertSymbols(indexSymbols []index.DocumentSymbol) []symbolOutput {
	var symbols []symbolOutput
	for _, s := range indexSymbols {
		symbols = append(symbols, symbolOutput{
			Name:     s.Name,
			Kind:     s.Kind,
			Line:     s.Line,
			Children: convertSymbols(s.Children),
		})
	}
	return symbols
}

func countSymbols(symbols []symbolOutput) int {
	count := len(symbols)
	for _, s := range symbols {
		count += countSymbols(s.Children)
	}
	return count
}

func outputSymbolsText(symbols []symbolOutput, indent int) error {
	prefix := strings.Repeat("  ", indent)
	for _, s := range symbols {
		fmt.Printf("%s%s  (%s:%d)\n", prefix, s.Name, s.Kind, s.Line)
		if len(s.Children) > 0 {
			if err := outputSymbolsText(s.Children, indent+1); err != nil {
				return err
			}
		}
	}
	return nil
}
