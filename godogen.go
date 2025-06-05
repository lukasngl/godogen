package godogen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"regexp"
	"strings"

	"github.com/iancoleman/strcase"
)

type (
	// Step represents a step definition in a Go file.
	// Each step coressponds directive comment on a function.
	// A function may have multiple directives, e.g. for different patterns.
	Step struct {
		// One of "Step", "Given", "When", "Then", "Before", or "After"
		Kind string
		// Pattern of steps, execpt before and after
		Pattern string
		// Name of the function to pass as the step func.
		Function string
		// Errors that occurred while validating the step.
		ValidationErrors []Error
	}
	// Error represents a validation error for a step definition.
	Error struct {
		// Node is the AST node where the error occurred.
		ast.Node
		// Message is the description of the error.
		Message string
	}
)

// GetStepDefinitions returns all step definitions found in the given file.
func GetStepDefinitions(fset *token.FileSet, file *ast.File) []Step {
	visitor := &fileVisitor{fset: fset}

	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case nil:
			return false
		case *ast.FuncDecl:
			visitor.steps = append(visitor.steps, visitor.VisitFuncDecl(node)...)
		}

		return true
	})

	return visitor.steps
}

type fileVisitor struct {
	// Accumulated steps
	steps []Step
	// For resolving token locations
	fset *token.FileSet
}

func (file *fileVisitor) VisitFuncDecl(funcdecl *ast.FuncDecl) []Step {
	var steps []Step

	if funcdecl.Doc == nil {
		return nil
	}

	for _, comment := range funcdecl.Doc.List {
		steps = append(steps, file.VisitComment(funcdecl, comment)...)
	}

	return steps
}

func (visitor *fileVisitor) VisitComment(funcdecl *ast.FuncDecl, comment *ast.Comment) []Step {
	position := visitor.fset.Position(comment.Pos())
	funname := funcdecl.Name.Name

	directive, isDirective := strings.CutPrefix(comment.Text, "//godogen:")
	if !isDirective {
		return nil
	}

	parts := strings.SplitN(directive, " ", 2)
	directive = parts[0]

	pattern := ""
	if len(parts) > 1 {
		pattern = parts[1]
	}

	switch directive {
	case "before", "after", "step", "given", "when", "then":
		step := Step{
			Kind:     strcase.ToCamel(directive),
			Pattern:  pattern,
			Function: funname,
		}

		if directive != "before" && directive != "after" {
			step.ValidationErrors = visitor.validateStep(funcdecl, comment, step)
		}

		visitor.steps = append(visitor.steps, step)

	default:
		log.Printf("%s: WARN decl %q has unknown directive %q",
			position, funname, directive,
		)
	}

	return nil
}

func (visitor *fileVisitor) validateStep(
	funcdecl *ast.FuncDecl,
	directive *ast.Comment,
	step Step,
) []Error {
	if step.Pattern == "" {
		return []Error{{
			Message: "pattern is empty",
			Node:    directive,
		}}
	}

	reg, err := regexp.Compile(step.Pattern)
	if err != nil {
		return []Error{{
			Message: fmt.Sprintf("regex pattern does not compile: %v", err),
			Node:    directive,
		}}
	}

	// Note: this does not include (?:non capturing groups)
	expectedParams := reg.NumSubexp()
	actualParams := 0

	var errs []Error

	n := len(funcdecl.Type.Params.List)
	for i, param := range funcdecl.Type.Params.List {
		m := len(param.Names)
		for j, name := range param.Names {
			isFirst := i == 0
			isLast := i == n-1 && j == m-1
			paramType := exprToString(visitor.fset, param.Type)

			switch paramType {
			case "context.Context":
				if !isFirst {
					errs = append(errs, Error{
						Message: fmt.Sprintf(
							"parameter %q of type %q must be first parameter",
							name,
							paramType,
						),
						Node: name,
					})
				}

			case "*godog.Table", "*godog.DocString":
				if !isLast {
					errs = append(errs, Error{
						Message: fmt.Sprintf(
							"parameter %q of type %q must be last parameter",
							name,
							paramType,
						),
						Node: name,
					})
				}

			case "int", "int8", "int16", "int32", "int64",
				"float32", "float64",
				"string":
				actualParams++

			default:
				errs = append(errs, Error{
					Message: fmt.Sprintf(
						"parameter %q has unexpected type %q",
						name,
						paramType,
					),
					Node: name,
				})

			}
		}
	}

	if actualParams != expectedParams {
		errs = append(errs, Error{
			Message: fmt.Sprintf(
				"pattern has %d groups, but function has %d regular parameters",
				expectedParams, actualParams,
			),
			Node: directive,
		})
	}

	return errs
}

func exprToString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer

	err := format.Node(&buf, fset, expr)
	if err != nil {
		panic("unreachable: format.Node should never fail for expression and a buffer")
	}

	return buf.String()
}
