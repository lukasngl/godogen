package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lukasngl/godogen/godogen-language-server/cli"
	"github.com/spf13/cobra"
)

var kindFilter string

var listStepsCmd = &cobra.Command{
	Use:   "list-steps",
	Short: "List all step definitions in the workspace",
	Long: `List all step definitions found in Go files in the workspace.

Step definitions are Go functions annotated with //godogen:step directives.`,
	RunE: runListSteps,
}

func init() {
	rootCmd.AddCommand(listStepsCmd)
	listStepsCmd.Flags().StringVar(&kindFilter, "kind", "all", "Filter by kind: Given, When, Then, Step, or all")
}

type stepOutput struct {
	Kind    string `json:"kind"`
	Pattern string `json:"pattern"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Valid   bool   `json:"valid"`
}

type listStepsResult struct {
	Steps []stepOutput `json:"steps"`
	Total int          `json:"total"`
}

func runListSteps(_ *cobra.Command, _ []string) error {
	ws, err := cli.LoadWorkspace(rootDir, configFile)
	if err != nil {
		return err
	}

	var steps []stepOutput

	for _, path := range ws.AllGoFiles() {
		goFile := ws.Index.GetGoFile(path)
		if goFile == nil {
			continue
		}

		for _, step := range goFile.AllSteps() {
			if kindFilter != "all" && step.Kind != kindFilter {
				continue
			}

			pos := goFile.Position(step.Node.Pos())
			steps = append(steps, stepOutput{
				Kind:    step.Kind,
				Pattern: step.Pattern,
				File:    path,
				Line:    pos.Line,
				Valid:   step.Regexp != nil,
			})
		}
	}

	// Sort by kind, then file, then line
	slices.SortFunc(steps, func(a, b stepOutput) int {
		if a.Kind != b.Kind {
			return strings.Compare(a.Kind, b.Kind)
		}
		if a.File != b.File {
			return strings.Compare(a.File, b.File)
		}
		return a.Line - b.Line
	})

	result := listStepsResult{
		Steps: steps,
		Total: len(steps),
	}

	if outputFmt == "json" {
		return outputJSON(result)
	}

	return outputListStepsText(result)
}

func outputListStepsText(result listStepsResult) error {
	if result.Total == 0 {
		fmt.Println("No step definitions found.")
		return nil
	}

	// Find max pattern length for alignment
	maxPatternLen := 0
	for _, s := range result.Steps {
		if len(s.Pattern) > maxPatternLen {
			maxPatternLen = len(s.Pattern)
		}
	}

	// Cap at reasonable width
	if maxPatternLen > 60 {
		maxPatternLen = 60
	}

	for _, s := range result.Steps {
		pattern := s.Pattern
		if len(pattern) > maxPatternLen {
			pattern = pattern[:maxPatternLen-3] + "..."
		}

		status := ""
		if !s.Valid {
			status = " [invalid]"
		}

		fmt.Printf("%-5s: %-*s  (%s:%d)%s\n", s.Kind, maxPatternLen, pattern, s.File, s.Line, status)
	}

	fmt.Printf("\nTotal: %d step definitions\n", result.Total)
	return nil
}
