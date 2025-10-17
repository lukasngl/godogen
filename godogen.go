package godogen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"iter"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/go/analysis"
)

type (
	StepFuncs []StepFunc
	// StepFunc represents a step function in a Go file.
	StepFunc struct {
		Node *ast.FuncDecl
		// Name of the function to pass as the step func.
		Function string
		// Errors that occurred while validating the step.
		ValidationErrors []Error
		// Steps is a list of steps that coresspond to a directive comment.
		Steps []Step
		// Hooks is a list of hooks that coresspond to a directive comment.
		Hooks []Hook
		// Hooks is a list of hooks that coresspond to a directive comment.
		StepHooks []Hook
		// Receiver is the type of the method recevier, nil if not a method.
		Receiver *Receiver
	}
	// Step represents a step definition in a Go file.
	// Each step coressponds directive comment on a function.
	// A function may have multiple directives, e.g. for different patterns.
	Step struct {
		Node *ast.Comment
		// Name of the function to pass as the step func.
		Function string
		// One of "Step", "Given", "When", or "Then"
		Kind string
		// Pattern of steps, execpt before and after
		Pattern string
		// Compiled regular expression of the pattern, nil if pattern did not compile.
		Regexp *regexp.Regexp
		// Errors that occurred while validating the step.
		ValidationErrors []Error
	}
	Hook struct {
		Node *ast.Comment
		// Name of the function to pass as the step func.
		Function string
		// One of "Before", or "After"
		Kind string
	}
	// Error represents a validation error for a step definition.
	Error struct {
		// Node is the AST node where the error occurred.
		ast.Node
		// Message is the description of the error.
		Message string
		// SuggestedFix is an optional replacement text to fix the error.
		SuggestedFixes []analysis.SuggestedFix
	}
	Receiver struct {
		// Node is the AST node where the error occurred.
		ast.Node
		//
		Type     ast.Expr
		TypeName string
	}
)

// GetStepDefinitions returns all step definitions found in the given file.
func GetStepDefinitions(fset *token.FileSet, file *ast.File) StepFuncs {
	visitor := &fileVisitor{fset: fset}

	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case nil:
			return false
		case *ast.FuncDecl:
			visitor.stepsFuncs = append(visitor.stepsFuncs, visitor.visitFuncDecl(node)...)
		}

		return true
	})

	visitor.stepsFuncs.validate(fset)

	return visitor.stepsFuncs
}

func (stepFunc StepFunc) hasDirectives() bool {
	return len(stepFunc.Steps) > 0 || len(stepFunc.Hooks) > 0 || len(stepFunc.StepHooks) > 0
}

type fileVisitor struct {
	// Accumulated stepsFuncs
	stepsFuncs StepFuncs
	// For resolving token locations
	fset *token.FileSet
}

func (visitor *fileVisitor) visitFuncDecl(funcdecl *ast.FuncDecl) []StepFunc {
	if funcdecl.Doc == nil {
		return nil
	}

	stepFunc := StepFunc{
		Node:     funcdecl,
		Function: funcdecl.Name.Name,
		Receiver: visitor.maybeGetReceiver(funcdecl),
	}

	for _, comment := range funcdecl.Doc.List {
		visitor.visitComment(&stepFunc, comment)
	}

	if !stepFunc.hasDirectives() {
		return nil
	}

	return []StepFunc{stepFunc}
}

func (visitor *fileVisitor) maybeGetReceiver(funcdecl *ast.FuncDecl) *Receiver {
	if funcdecl.Recv == nil || funcdecl.Recv.List == nil {
		return nil
	}

	t := funcdecl.Recv.List[0].Type

	return &Receiver{
		Node:     funcdecl.Recv,
		Type:     t,
		TypeName: exprToString(visitor.fset, t),
	}
}

func (visitor *fileVisitor) visitComment(
	stepFunc *StepFunc,
	comment *ast.Comment,
) {
	position := visitor.fset.Position(comment.Pos())

	directive, isDirective := strings.CutPrefix(comment.Text, "//godogen:")
	if !isDirective {
		return
	}

	parts := strings.SplitN(directive, " ", 2)
	directive = parts[0]

	pattern := ""
	if len(parts) > 1 {
		pattern = parts[1]
	}

	switch directive {
	case "before_step", "after_step":
		stepFunc.StepHooks = append(stepFunc.StepHooks, Hook{
			Node:     comment,
			Function: stepFunc.Function,
			Kind:     strcase.ToCamel(strings.TrimSuffix(directive, "_step")),
		})

	case "before", "after":
		stepFunc.Hooks = append(stepFunc.Hooks, Hook{
			Node:     comment,
			Function: stepFunc.Function,
			Kind:     strcase.ToCamel(directive),
		})

	case "step", "given", "when", "then":
		stepFunc.Steps = append(stepFunc.Steps, Step{
			Node:     comment,
			Function: stepFunc.Function,
			Kind:     strcase.ToCamel(directive),
			Pattern:  pattern,
		})

	default:
		slog.Warn(
			"unknown directive",
			"position", position,
			"directive", directive,
			"function", stepFunc.Function,
		)
	}
}

// Receivers returns an iterator over all unique receivers found in the step functions.
func (stepFuncs StepFuncs) Receivers() iter.Seq[*Receiver] {
	return func(yield func(*Receiver) bool) {
		// for small number of receivers, a simple slice is more efficient than a map or set.
		var seen []string

		for _, stepFunc := range stepFuncs {
			if stepFunc.Receiver == nil {
				continue
			}

			if slices.Contains(seen, stepFunc.Receiver.TypeName) {
				continue
			}

			if !yield(stepFunc.Receiver) {
				return
			}

			seen = append(seen, stepFunc.Receiver.TypeName)
		}
	}
}

// ValidationErrors returns an iterator over all validation errors found in the step functions.
func (stepFuncs StepFuncs) ValidationErrors() iter.Seq[Error] {
	return func(yield func(Error) bool) {
		for _, stepFunc := range stepFuncs {
			for _, err := range stepFunc.ValidationErrors {
				if !yield(err) {
					return
				}
			}

			for _, step := range stepFunc.Steps {
				for _, err := range step.ValidationErrors {
					if !yield(err) {
						return
					}
				}
			}
		}
	}
}

func (stepFuncs StepFuncs) validate(fset *token.FileSet) {
	for i := range stepFuncs {
		stepFuncs[i].validate(fset)
	}
}

func (stepFunc *StepFunc) validate(fset *token.FileSet) {
	if len(stepFunc.Hooks) > 0 && len(stepFunc.Steps) > 0 {
		stepFunc.ValidationErrors = append(stepFunc.ValidationErrors, Error{
			Message: "step function cannot have both hook and step directives",
			Node:    stepFunc.Node,
		})
	} else if len(stepFunc.Steps) > 0 {
		actualParams, errs := validateStepFunc(fset, stepFunc.Node)
		stepFunc.ValidationErrors = errs

		for i := range stepFunc.Steps {
			stepFunc.Steps[i].validate(actualParams)
		}
	}
}

// validateStepFunc validates the parameter and returns types of the step function.
func validateStepFunc(
	fset *token.FileSet,
	funcdecl *ast.FuncDecl,
) (int, []Error) {
	actualParams := 0

	var errs []Error

	n := len(funcdecl.Type.Params.List)
	for i, param := range funcdecl.Type.Params.List {
		m := len(param.Names)
		for j, name := range param.Names {
			isFirst := i == 0
			isLast := i == n-1 && j == m-1
			paramType := exprToString(fset, param.Type)

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

	var resultTypes []*ast.Field

	results := funcdecl.Type.Results
	if results != nil {
		resultTypes = results.List
	}

	switch len(resultTypes) {
	case 0:
		// No return values.
	case 1:
		// One return value: should be error, Steps, or context.Context.
		resultType := exprToString(fset, resultTypes[0].Type)
		switch resultType {
		case "error", "godog.Steps", "context.Context":
			// ok
		default:
			errs = append(errs, Error{
				Message: "with one return value, function should return error, godog.Steps, or context.Context",
				Node:    results,
			})
		}
	case 2:
		// Two return values: should be (context.Context, error).
		resultTypeA := exprToString(fset, resultTypes[0].Type)
		resultTypeB := exprToString(fset, resultTypes[1].Type)
		if resultTypeA != "context.Context" || resultTypeB != "error" {
			errs = append(errs, Error{
				Message: "function should return (context.Context, error) with two return values",
				Node:    results,
			})
		}
	default:
		// More than two return values.
		errs = append(errs, Error{
			Message: fmt.Sprintf(
				"function should return at most two values, but got %d",
				len(resultTypes),
			),
			Node: results,
		})
	}

	return actualParams, errs
}

func (step *Step) validate(actualParams int) {
	if step.Pattern == "" {
		step.ValidationErrors = append(step.ValidationErrors, Error{
			Message: "pattern is empty",
			Node:    step.Node,
		})
		return
	}

	regexp, err := regexp.Compile(step.Pattern)
	if err != nil {
		step.ValidationErrors = append(step.ValidationErrors, Error{
			Message: fmt.Sprintf("regex pattern does not compile: %v", err),
			Node:    step.Node,
		})
		return
	}

	step.Regexp = regexp

	anchorErr, found := step.checkAnchors()
	if found {
		step.ValidationErrors = append(step.ValidationErrors, anchorErr)
	}

	paramErr, found := checkParmCount(step.Node, regexp, actualParams)
	if found {
		step.ValidationErrors = append(step.ValidationErrors, paramErr)
	}
}

func checkParmCount(node ast.Node, reg *regexp.Regexp, actualParams int) (Error, bool) {
	// Note: this does not include (?:non capturing groups)
	expectedParams := reg.NumSubexp()
	if actualParams == expectedParams {
		return Error{}, false
	}

	return Error{
		Message: fmt.Sprintf(
			"pattern has %d groups, but function has %d regular parameters",
			expectedParams, actualParams,
		),
		Node: node,
	}, true
}

func (step *Step) checkAnchors() (Error, bool) {
	hasStartAnchor := strings.HasPrefix(step.Pattern, "^")
	hasEndAnchor := strings.HasSuffix(step.Pattern, "$")
	if hasStartAnchor && hasEndAnchor {
		return Error{}, false
	}

	fixedPattern := step.Pattern
	if !hasStartAnchor {
		fixedPattern = "^" + fixedPattern
	}

	if !hasEndAnchor {
		fixedPattern = fixedPattern + "$"
	}

	fixedPattern = fmt.Sprintf("//godogen:%s %s", strings.ToLower(step.Kind), fixedPattern)

	return Error{
		Node:    step.Node,
		Message: "The pattern should start with '^' and end with '$'",
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: "Add missing anchors",
			TextEdits: []analysis.TextEdit{{
				Pos:     step.Node.Pos(),
				End:     step.Node.End(),
				NewText: []byte(fixedPattern),
			}},
		}},
	}, true
}

func exprToString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer

	err := format.Node(&buf, fset, expr)
	if err != nil {
		panic("unreachable: format.Node should never fail for expression and a buffer")
	}

	return buf.String()
}
