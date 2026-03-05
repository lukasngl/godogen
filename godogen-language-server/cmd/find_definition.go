package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lukasngl/godogen/godogen-language-server/cli"
	"github.com/spf13/cobra"
)

var findDefCmd = &cobra.Command{
	Use:   "find-definition <file:line>",
	Short: "Find step definition for a feature step",
	Long: `Find the Go step definition that matches a feature step at the given location.

Arguments:
  file:line  The feature file path and line number (1-indexed)

Example:
  godogen-language-server find-definition features/login.feature:10`,
	Args: cobra.ExactArgs(1),
	RunE: runFindDefinition,
}

func init() {
	rootCmd.AddCommand(findDefCmd)
}

type locationOutput struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type findDefResult struct {
	Locations []locationOutput `json:"locations"`
	Total     int              `json:"total"`
}

func runFindDefinition(_ *cobra.Command, args []string) error {
	file, line, err := parseFileLine(args[0])
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

	// Note: FindStepDefinitions uses 0-indexed line numbers
	locs := ws.Index.FindStepDefinitions(absFile, line-1, true)

	var locations []locationOutput
	for _, loc := range locs {
		locations = append(locations, locationOutput{
			File:   loc.Path,
			Line:   loc.Line,
			Column: loc.Column,
		})
	}

	result := findDefResult{
		Locations: locations,
		Total:     len(locations),
	}

	if outputFmt == "json" {
		return outputJSON(result)
	}

	return outputLocationsText(result.Locations, "definition")
}

func parseFileLine(arg string) (string, int, error) {
	parts := strings.Split(arg, ":")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("expected format: file:line, got: %s", arg)
	}

	file := strings.Join(parts[:len(parts)-1], ":")
	lineStr := parts[len(parts)-1]

	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid line number: %s", lineStr)
	}

	return file, line, nil
}

func parseFileLineColumn(arg string) (string, int, int, error) {
	parts := strings.Split(arg, ":")
	if len(parts) < 3 {
		return "", 0, 0, fmt.Errorf("expected format: file:line:column, got: %s", arg)
	}

	file := strings.Join(parts[:len(parts)-2], ":")
	lineStr := parts[len(parts)-2]
	colStr := parts[len(parts)-1]

	line, err := strconv.Atoi(lineStr)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid line number: %s", lineStr)
	}

	col, err := strconv.Atoi(colStr)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid column number: %s", colStr)
	}

	return file, line, col, nil
}

func outputLocationsText(locations []locationOutput, kind string) error {
	if len(locations) == 0 {
		fmt.Printf("No %s found.\n", kind)
		return nil
	}

	for _, loc := range locations {
		fmt.Printf("%s:%d:%d\n", loc.File, loc.Line, loc.Column)
	}

	return nil
}
