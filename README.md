# godogen

Now with Methods! 🎉

godogen is golang codegenerator, that allows you to colocate [godog] step definitions with their pattern.

## Project Structure

This repository is organized as a monorepo containing multiple tools:

- **Root (`main.go` + `pkg/`)** - Code generator for godog step definitions
- **`godogen-lint/`** - Standalone linter for validating godogen directives
- **`godogen-gcl/`** - golangci-lint plugin
- **`godogen-language-server/`** - LSP server for IDE integration

## Installation

```bash
go install github.com/lukasngl/godogen@latest
```

Or run directly:

```bash
go run github.com/lukasngl/godogen@latest
```

## Motivation

Similar to Java's Cucumber annotations, godogen allows you to colocate test step patterns with their implementations:

**Java (Cucumber):**

```java
@Given("there are {int} godogs")
public void thereAreGodogs(int count) {
    this.available = count;
}
```

**Go (godogen):**

```go
//godogen:given ^there are (\d+) godogs$
func (s *GodogsState) thereAreGodogs(available int) {
    s.available = available
}
```

This approach keeps your test definitions close to their patterns, making them easier to maintain and understand.

## Example

```go
// file: godog_steps.go
package godogs
//go:generate go run github.com/lukasngl/godogen@latest

import (
	"fmt"
	"github.com/cucumber/godog"
)

type GodogsState struct {
	available int
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &GodogsState{available: 0}
	InitializeGodogSteps(ctx, state)
}

//godogen:given ^there are (\d+) godogs$
func (s *GodogsState) thereAreGodogs(available int) {
	s.available = available
}

//godogen:when ^I eat (\d+)$
func (s *GodogsState) iEat(num int) {
	s.available -= num
}

//godogen:then ^there should be (\d+) remaining$
func (s *GodogsState) thereShouldBeRemaining(remaining int) error {
	if s.available != remaining {
		return fmt.Errorf("expected %d godogs, but there are %d", remaining, s.available)
	}
	return nil
}
```

will generate:

```go
// file: godog_steps_initializer.go
package godogs

import "github.com/cucumber/godog"

// InitializeGodogSteps registers steps defined in "godog_steps.go" with the [godog.ScenarioContext].
func InitializeGodogSteps(ctx *godog.ScenarioContext, r1 *GodogsState) {
	ctx.Given(`^there are (\d+) godogs$`, r1.thereAreGodogs)
	ctx.When(`^I eat (\d+)$`, r1.iEat)
	ctx.Then(`^there should be (\d+) remaining$`, r1.thereShouldBeRemaining)
}
```

### Generic Receivers

godogen supports generic context structs. Type parameters are propagated to the generated initializer function and renamed to `T1`, `T2`, ... to avoid collisions:

```go
type Suite[T any] struct{ state T }

//godogen:given ^I have (\d+) items$
func (s *Suite[T]) iHaveItems(count int) error {
    return nil
}
```

generates:

```go
func InitializeSteps[T1 any](sc *godog.ScenarioContext, r1 *Suite[T1]) {
    sc.Given(`^I have (\d+) items$`, r1.iHaveItems)
}
```

The type declaration must be in the same file as the methods. For types declared elsewhere, use a type alias:

```go
type MyConstraint = otherpackage.Constraint
type Suite[T MyConstraint] struct{ state T }
```

## Features

- colocate step definition (i.e. the function or method declaration) with the pattern,
  using the following directives on a function or method named `<FUNCTION>`:
  - `//godogen:step <PATTERN>` will generate `ctx.Step(<PATTERN>, <FUNCTION>)`
  - `//godogen:given <PATTERN>` will generate `ctx.Given(<PATTERN>, <FUNCTION>)`
  - `//godogen:when <PATTERN>` will generate `ctx.When(<PATTERN>, <FUNCTION>)`
  - `//godogen:then <PATTERN>` will generate `ctx.Then(<PATTERN>, <FUNCTION>)`
  - `//godogen:after` will generate `ctx.After(<FUNCTION>)`
  - `//godogen:before` will generate `ctx.Before(<FUNCTION>)`
  - `//godogen:after_step` will generate `ctx.StepContext().After(<FUNCTION>)`
  - `//godogen:before_step` will generate `ctx.StepContext().Before(<FUNCTION>)`
- methods on any type are supported as step definitions
- receiver instances are passed as parameters to the generated `InitializeGodogSteps` function
- generic receiver types are supported — type parameters are propagated to the generated initializer function

## Linting

godogen includes a linter that validates your godogen directives and step definitions:

- **Pattern validation**: ensures patterns are valid regex and properly anchored with `^` and `$` (auto-fixable)
- **Parameter validation**: checks parameter count matches regex groups and validates parameter types
- **Return type validation**: ensures return types are compatible with godog (error, godog.Steps, context.Context, or (context.Context, error))
- **Directive validation**: prevents mixing hook and step directives on the same function

**Installation:**

```bash
go install github.com/lukasngl/godogen/godogen-lint@latest
```

**Usage:**

```bash
godogen-lint [-fix] ./...
```

**golangci-lint plugin:**

```bash
# Build the plugin
cd godogen-gcl
go build -o godogen-gcl.so .

# Use with golangci-lint
golangci-lint run --load=./godogen-gcl.so
```

## Language Server

godogen includes a Language Server Protocol (LSP) implementation that provides IDE features for Gherkin feature files and Go step definitions.

**Features:**

- Go to Definition (feature step → pattern comment)
- Go to Implementation (feature step → function)
- Find References (pattern/function → feature steps)
- Diagnostics (undefined, ambiguous, unused, duplicate steps)
- Hover (step definition details and usage)
- Document Symbols (scenarios, step definitions)
- Code Actions (quick fixes)
- Completion (Gherkin keywords)

**Installation:**

```bash
go install github.com/lukasngl/godogen/godogen-language-server@latest
```

**CLI Commands:**

The language server also provides CLI commands for batch analysis and CI/CD:

```bash
# Run diagnostics on workspace
godogen-language-server diagnose

# List all step definitions
godogen-language-server list-steps

# Find step definition for a feature step
godogen-language-server find-definition features/login.feature:10

# Find references to a step definition
godogen-language-server find-references steps/auth.go:15:1

# JSON output for CI/CD
godogen-language-server diagnose --format json --severity error
```

See [godogen-language-server](./godogen-language-server) for full documentation and configuration options.

## Editor Integration

### Language Server (LSP)

See [godogen-language-server](./godogen-language-server/README.md) for detailed setup instructions.

**Quick Start - Neovim:**

```lua
require('lspconfig').godogen.setup({
  cmd = { 'godogen-language-server', 'stdio' },
  filetypes = { 'feature', 'gherkin' },
})
```

**VS Code:** Extension coming soon.

### Syntax Highlighting (Neovim/Tree-sitter)

To enable regex syntax highlighting for godogen directive patterns in Neovim, add the following injection query to `~/.config/nvim/after/queries/go/injections.scm`:

```scheme
;extends

; godogen directives
((comment) @injection.content
  (#match? @injection.content "^//godogen:")
  (#set! injection.language "regex"))

; godog step functions
(call_expression
  (selector_expression) @_function
  (#any-of? @_function "sc.When" "sc.Then" "sc.Given" "sc.Step")
  (argument_list
    .
    [
      (raw_string_literal
        (raw_string_literal_content) @injection.content)
      (interpreted_string_literal
        (interpreted_string_literal_content) @injection.content)
    ])
  (#set! injection.language "regex"))
```

[godog]: https://github.com/cucumber/godog
