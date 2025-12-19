package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/lukasngl/godogen/godogen-language-server/cli"
	"github.com/lukasngl/godogen/godogen-language-server/index"
	"github.com/spf13/cobra"
)

var severityFilter string

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Run diagnostics on the workspace",
	Long: `Analyze all Go and feature files in the workspace and report diagnostics.

Reports:
  - Undefined steps (feature steps without matching definitions)
  - Ambiguous steps (steps matching multiple definitions)
  - Duplicate step definitions
  - Unused step definitions
  - Invalid step patterns`,
	RunE: runDiagnose,
}

func init() {
	rootCmd.AddCommand(diagnoseCmd)
	diagnoseCmd.Flags().StringVar(&severityFilter, "severity", "all", "Filter by severity: error, warning, hint, or all")
}

type diagnosticOutput struct {
	File        string              `json:"file"`
	Line        int                 `json:"line"`
	Column      int                 `json:"column"`
	Severity    string              `json:"severity"`
	Message     string              `json:"message"`
	RelatedInfo []relatedInfoOutput `json:"relatedInfo,omitempty"`
}

type relatedInfoOutput struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

type diagnoseResult struct {
	Diagnostics []diagnosticOutput `json:"diagnostics"`
	Summary     diagnoseSummary    `json:"summary"`
}

type diagnoseSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Hints    int `json:"hints"`
	Total    int `json:"total"`
}

func runDiagnose(_ *cobra.Command, _ []string) error {
	ws, err := cli.LoadWorkspace(rootDir, configFile)
	if err != nil {
		return err
	}

	var diagnostics []diagnosticOutput

	// Collect diagnostics from Go files
	for _, path := range ws.AllGoFiles() {
		diags := ws.Index.GetDiagnostics(path)
		for _, diag := range diags {
			d := diagnosticOutput{
				File:        path,
				Line:        diag.StartLine,
				Column:      diag.StartColumn,
				Severity:    severityString(diag.Severity),
				Message:     diag.Message,
				RelatedInfo: toRelatedInfoOutput(diag.RelatedInformation),
			}
			if matchesSeverityFilter(d.Severity) {
				diagnostics = append(diagnostics, d)
			}
		}
	}

	// Collect diagnostics from feature files
	for _, path := range ws.AllFeatureFiles() {
		diags := ws.Index.GetFeatureDiagnostics(path)
		for _, diag := range diags {
			d := diagnosticOutput{
				File:        path,
				Line:        diag.StartLine,
				Column:      diag.StartColumn,
				Severity:    severityString(diag.Severity),
				Message:     diag.Message,
				RelatedInfo: toRelatedInfoOutput(diag.RelatedInformation),
			}
			if matchesSeverityFilter(d.Severity) {
				diagnostics = append(diagnostics, d)
			}
		}
	}

	// Sort by file, then line
	slices.SortFunc(diagnostics, func(a, b diagnosticOutput) int {
		if a.File != b.File {
			return strings.Compare(a.File, b.File)
		}
		return a.Line - b.Line
	})

	// Calculate summary
	summary := diagnoseSummary{}
	for _, d := range diagnostics {
		switch d.Severity {
		case "error":
			summary.Errors++
		case "warning":
			summary.Warnings++
		case "hint":
			summary.Hints++
		}
		summary.Total++
	}

	result := diagnoseResult{
		Diagnostics: diagnostics,
		Summary:     summary,
	}

	if outputFmt == "json" {
		return outputJSON(result)
	}

	return outputDiagnoseText(result)
}

func severityString(s index.DiagnosticSeverity) string {
	switch s {
	case index.DiagnosticSeverityError:
		return "error"
	case index.DiagnosticSeverityWarning:
		return "warning"
	case index.DiagnosticSeverityHint:
		return "hint"
	default:
		return "info"
	}
}

func matchesSeverityFilter(severity string) bool {
	if severityFilter == "all" {
		return true
	}
	return severity == severityFilter
}

func toRelatedInfoOutput(info []index.DiagnosticRelatedInformation) []relatedInfoOutput {
	if len(info) == 0 {
		return nil
	}
	result := make([]relatedInfoOutput, len(info))
	for i, ri := range info {
		result[i] = relatedInfoOutput{
			File:    ri.Path,
			Line:    ri.Line,
			Column:  ri.Column,
			Message: ri.Message,
		}
	}
	return result
}

func outputJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func outputDiagnoseText(result diagnoseResult) error {
	for _, d := range result.Diagnostics {
		fmt.Printf("%s:%d:%d: %s: %s\n", d.File, d.Line, d.Column, d.Severity, d.Message)
		for _, rel := range d.RelatedInfo {
			fmt.Printf("  -> %s:%d:%d: %s\n", rel.File, rel.Line, rel.Column, rel.Message)
		}
	}

	if result.Summary.Total > 0 {
		fmt.Printf("\nSummary: %d errors, %d warnings, %d hints\n",
			result.Summary.Errors, result.Summary.Warnings, result.Summary.Hints)
	} else {
		fmt.Println("No diagnostics found.")
	}

	return nil
}
