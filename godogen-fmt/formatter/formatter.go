package formatter

import (
	"sort"
	"strings"

	gherkin "github.com/cucumber/gherkin/go/v28"
	messages "github.com/cucumber/messages/go/v24"
)

// formatContext holds state during formatting
type formatContext struct {
	b           *strings.Builder
	comments    []*messages.Comment
	commentIdx  int
}

// Format formats a Gherkin document.
func Format(input string) (string, error) {
	reader := strings.NewReader(input)
	doc, err := gherkin.ParseGherkinDocument(reader, (&messages.Incrementing{}).NewId)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	ctx := &formatContext{
		b:        &b,
		comments: doc.Comments,
	}
	// Sort comments by line
	sort.Slice(ctx.comments, func(i, j int) bool {
		return ctx.comments[i].Location.Line < ctx.comments[j].Location.Line
	})

	ctx.formatDocument(doc)
	// Trim trailing whitespace and ensure single trailing newline
	result := strings.TrimRight(b.String(), "\n") + "\n"
	return result, nil
}

// writeCommentsUntil outputs all comments with line < targetLine
func (ctx *formatContext) writeCommentsUntil(targetLine int64, indent string) {
	for ctx.commentIdx < len(ctx.comments) {
		comment := ctx.comments[ctx.commentIdx]
		if comment.Location.Line >= targetLine {
			break
		}
		ctx.b.WriteString(indent)
		ctx.b.WriteString(comment.Text)
		ctx.b.WriteString("\n")
		ctx.commentIdx++
	}
}

func (ctx *formatContext) formatDocument(doc *messages.GherkinDocument) {
	if doc.Feature == nil {
		return
	}
	ctx.formatFeature(doc.Feature)
}

func (ctx *formatContext) formatFeature(f *messages.Feature) {
	// Comments before feature (before tags or feature line)
	var featureLine int64 = f.Location.Line
	if len(f.Tags) > 0 {
		featureLine = f.Tags[0].Location.Line
	}
	ctx.writeCommentsUntil(featureLine, "")

	// Tags
	ctx.formatTags(f.Tags, "")

	// Feature line
	ctx.b.WriteString("Feature: ")
	ctx.b.WriteString(f.Name)
	ctx.b.WriteString("\n")

	// Description
	if f.Description != "" {
		ctx.formatDescription(f.Description, "  ")
	}

	// Children (Background, Scenarios, Rules)
	for _, child := range f.Children {
		if child.Background != nil {
			ctx.formatBackground(child.Background, "  ")
		}
		if child.Scenario != nil {
			ctx.formatScenario(child.Scenario, "  ")
		}
		if child.Rule != nil {
			ctx.formatRule(child.Rule, "  ")
		}
	}
}

func (ctx *formatContext) formatRule(r *messages.Rule, indent string) {
	ctx.b.WriteString("\n")

	// Comments before rule
	var ruleLine int64 = r.Location.Line
	if len(r.Tags) > 0 {
		ruleLine = r.Tags[0].Location.Line
	}
	ctx.writeCommentsUntil(ruleLine, indent)

	ctx.formatTags(r.Tags, indent)
	ctx.b.WriteString(indent)
	ctx.b.WriteString("Rule: ")
	ctx.b.WriteString(r.Name)
	ctx.b.WriteString("\n")

	// Description
	if r.Description != "" {
		ctx.formatDescription(r.Description, indent+"  ")
	}

	// Children
	for _, child := range r.Children {
		if child.Background != nil {
			ctx.formatBackground(child.Background, indent+"  ")
		}
		if child.Scenario != nil {
			ctx.formatScenario(child.Scenario, indent+"  ")
		}
	}
}

func (ctx *formatContext) formatBackground(bg *messages.Background, indent string) {
	ctx.b.WriteString("\n")
	ctx.b.WriteString(indent)
	ctx.b.WriteString("Background:")
	if bg.Name != "" {
		ctx.b.WriteString(" ")
		ctx.b.WriteString(bg.Name)
	}
	ctx.b.WriteString("\n")

	// Description
	if bg.Description != "" {
		ctx.formatDescription(bg.Description, indent+"  ")
	}

	// Steps
	for _, step := range bg.Steps {
		ctx.formatStep(step, indent+"  ")
	}
}

func (ctx *formatContext) formatScenario(s *messages.Scenario, indent string) {
	ctx.b.WriteString("\n")

	// Comments before scenario
	var scenarioLine int64 = s.Location.Line
	if len(s.Tags) > 0 {
		scenarioLine = s.Tags[0].Location.Line
	}
	ctx.writeCommentsUntil(scenarioLine, indent)

	ctx.formatTags(s.Tags, indent)
	ctx.b.WriteString(indent)

	keyword := "Scenario"
	if len(s.Examples) > 0 {
		keyword = "Scenario Outline"
	}
	ctx.b.WriteString(keyword)
	ctx.b.WriteString(": ")
	ctx.b.WriteString(s.Name)
	ctx.b.WriteString("\n")

	// Description
	if s.Description != "" {
		ctx.formatDescription(s.Description, indent+"  ")
	}

	// Steps
	for _, step := range s.Steps {
		ctx.formatStep(step, indent+"  ")
	}

	// Examples
	for _, examples := range s.Examples {
		ctx.formatExamples(examples, indent+"  ")
	}
}

func (ctx *formatContext) formatStep(step *messages.Step, indent string) {
	ctx.b.WriteString(indent)
	ctx.b.WriteString(step.Keyword)
	ctx.b.WriteString(step.Text)
	ctx.b.WriteString("\n")

	// Data table
	if step.DataTable != nil {
		ctx.formatDataTable(step.DataTable, indent+"  ")
	}

	// Doc string
	if step.DocString != nil {
		ctx.formatDocString(step.DocString, indent+"  ")
	}
}

func (ctx *formatContext) formatDataTable(dt *messages.DataTable, indent string) {
	if len(dt.Rows) == 0 {
		return
	}

	// Calculate column widths (with escaped pipes)
	widths := make([]int, len(dt.Rows[0].Cells))
	for _, row := range dt.Rows {
		for i, cell := range row.Cells {
			escaped := escapeCell(cell.Value)
			if len(escaped) > widths[i] {
				widths[i] = len(escaped)
			}
		}
	}

	// Format rows
	for _, row := range dt.Rows {
		ctx.b.WriteString(indent)
		ctx.b.WriteString("|")
		for i, cell := range row.Cells {
			escaped := escapeCell(cell.Value)
			ctx.b.WriteString(" ")
			ctx.b.WriteString(escaped)
			// Pad to column width
			for j := len(escaped); j < widths[i]; j++ {
				ctx.b.WriteString(" ")
			}
			ctx.b.WriteString(" |")
		}
		ctx.b.WriteString("\n")
	}
}

func escapeCell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}

func (ctx *formatContext) formatDocString(ds *messages.DocString, indent string) {
	ctx.b.WriteString(indent)
	ctx.b.WriteString("\"\"\"")
	if ds.MediaType != "" {
		ctx.b.WriteString(ds.MediaType)
	}
	ctx.b.WriteString("\n")

	// Content lines
	lines := strings.Split(ds.Content, "\n")
	for _, line := range lines {
		ctx.b.WriteString(indent)
		ctx.b.WriteString(line)
		ctx.b.WriteString("\n")
	}

	ctx.b.WriteString(indent)
	ctx.b.WriteString("\"\"\"\n")
}

func (ctx *formatContext) formatExamples(ex *messages.Examples, indent string) {
	ctx.b.WriteString("\n")
	ctx.formatTags(ex.Tags, indent)
	ctx.b.WriteString(indent)
	ctx.b.WriteString("Examples:")
	if ex.Name != "" {
		ctx.b.WriteString(" ")
		ctx.b.WriteString(ex.Name)
	}
	ctx.b.WriteString("\n")

	// Description
	if ex.Description != "" {
		ctx.formatDescription(ex.Description, indent+"  ")
	}

	// Table
	if ex.TableHeader != nil {
		// Calculate column widths
		widths := make([]int, len(ex.TableHeader.Cells))
		for i, cell := range ex.TableHeader.Cells {
			if len(cell.Value) > widths[i] {
				widths[i] = len(cell.Value)
			}
		}
		for _, row := range ex.TableBody {
			for i, cell := range row.Cells {
				if len(cell.Value) > widths[i] {
					widths[i] = len(cell.Value)
				}
			}
		}

		// Format header
		ctx.b.WriteString(indent + "  ")
		ctx.b.WriteString("|")
		for i, cell := range ex.TableHeader.Cells {
			ctx.b.WriteString(" ")
			ctx.b.WriteString(cell.Value)
			for j := len(cell.Value); j < widths[i]; j++ {
				ctx.b.WriteString(" ")
			}
			ctx.b.WriteString(" |")
		}
		ctx.b.WriteString("\n")

		// Format body
		for _, row := range ex.TableBody {
			ctx.b.WriteString(indent + "  ")
			ctx.b.WriteString("|")
			for i, cell := range row.Cells {
				ctx.b.WriteString(" ")
				ctx.b.WriteString(cell.Value)
				for j := len(cell.Value); j < widths[i]; j++ {
					ctx.b.WriteString(" ")
				}
				ctx.b.WriteString(" |")
			}
			ctx.b.WriteString("\n")
		}
	}
}

func (ctx *formatContext) formatTags(tags []*messages.Tag, indent string) {
	if len(tags) == 0 {
		return
	}

	// Group tags by line
	var currentLine int64 = -1
	for _, tag := range tags {
		if tag.Location.Line != currentLine {
			if currentLine != -1 {
				ctx.b.WriteString("\n")
			}
			ctx.b.WriteString(indent)
			currentLine = tag.Location.Line
		} else {
			ctx.b.WriteString(" ")
		}
		ctx.b.WriteString(tag.Name)
	}
	ctx.b.WriteString("\n")
}

func (ctx *formatContext) formatDescription(desc string, indent string) {
	lines := strings.Split(strings.TrimSpace(desc), "\n")
	for _, line := range lines {
		if line == "" {
			ctx.b.WriteString("\n")
		} else {
			ctx.b.WriteString(indent)
			ctx.b.WriteString(line)
			ctx.b.WriteString("\n")
		}
	}
}
