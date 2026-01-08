package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/lukasngl/godogen/godogen-language-server/cli"
	"github.com/spf13/cobra"
)

var hoverCmd = &cobra.Command{
	Use:   "hover <file:line:column>",
	Short: "Get hover information for a position",
	Long: `Get hover information for a position in a feature or Go file.

For feature files: Shows the matching step definition(s) and their documentation.
For Go files: Shows step pattern usage and references.

Arguments:
  file:line:column  The file path, line number, and column (1-indexed)

Example:
  godogen-language-server hover features/login.feature:10:5
  godogen-language-server hover steps/auth_steps.go:15:1`,
	Args: cobra.ExactArgs(1),
	RunE: runHover,
}

func init() {
	rootCmd.AddCommand(hoverCmd)
}

type hoverResult struct {
	Content string `json:"content"`
	Found   bool   `json:"found"`
}

func runHover(_ *cobra.Command, args []string) error {
	file, line, col, err := parseFileLineColumn(args[0])
	if err != nil {
		return err
	}

	ws, err := cli.LoadWorkspace(rootDir, configFile)
	if err != nil {
		return err
	}

	absFile, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}

	var result hoverResult

	ext := filepath.Ext(absFile)
	switch ext {
	case ".feature":
		info := ws.Index.GetHoverInfoForFeature(absFile, line-1, col)
		if info != nil {
			result.Content = info.Content
			result.Found = true
		}
	case ".go":
		info := ws.Index.GetHoverInfoForGo(absFile, line-1, col)
		if info != nil {
			result.Content = info.Content
			result.Found = true
		}
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	if outputFmt == "json" {
		return outputJSON(result)
	}

	if !result.Found {
		fmt.Println("No hover information available at this position.")
		return nil
	}

	fmt.Println(result.Content)
	return nil
}
