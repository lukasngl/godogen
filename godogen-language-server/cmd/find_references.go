package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/lukasngl/godogen/godogen-language-server/cli"
	"github.com/spf13/cobra"
)

var findRefsCmd = &cobra.Command{
	Use:   "find-references <file:line:column>",
	Short: "Find references to a step definition",
	Long: `Find all feature steps that reference a Go step definition at the given location.

Arguments:
  file:line:column  The Go file path, line number, and column (1-indexed)

The cursor should be on a //godogen:step comment or the function name.

Example:
  godogen-language-server find-references steps/auth_steps.go:15:1`,
	Args: cobra.ExactArgs(1),
	RunE: runFindReferences,
}

func init() {
	rootCmd.AddCommand(findRefsCmd)
}

type findRefsResult struct {
	Locations []locationOutput `json:"locations"`
	Total     int              `json:"total"`
}

func runFindReferences(_ *cobra.Command, args []string) error {
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

	// Note: FindStepReferences uses 0-indexed line numbers but 1-indexed columns
	locs := ws.Index.FindStepReferences(absFile, line-1, col)

	var locations []locationOutput
	for _, loc := range locs {
		locations = append(locations, locationOutput{
			File:   loc.Path,
			Line:   loc.Line,
			Column: loc.Column,
		})
	}

	result := findRefsResult{
		Locations: locations,
		Total:     len(locations),
	}

	if outputFmt == "json" {
		return outputJSON(result)
	}

	return outputLocationsText(result.Locations, "references")
}
