---
name: godogen
description: Get help with Godogen BDD development - step definitions, CLI commands, configuration, and troubleshooting
---

# Godogen Reference

Godogen is a Go code generator for [godog](https://github.com/cucumber/godog) BDD step definitions. It allows you to colocate step patterns with their implementations using directive comments.

## Quick Install

```bash
go install github.com/lukasngl/godogen@latest
go install github.com/lukasngl/godogen/godogen-language-server@latest
```

For more options (Go 1.24+ tool directive, Nix flake, editor setup), see [Installation](docs/installation.md).

## Writing Step Definitions

Step definitions are Go functions with `//godogen:` directive comments:

```go
//godogen:given ^I am logged in as "([^"]*)"$
func (s *Suite) iAmLoggedInAs(username string) error {
    return nil
}

//godogen:when ^I click the "([^"]*)" button$
func (s *Suite) iClickButton(name string) error {
    return nil
}

//godogen:then ^I should see "([^"]*)"$
func (s *Suite) iShouldSee(text string) error {
    return nil
}
```

### Directive Types

| Directive                   | Generated Code                 | Use Case                |
| --------------------------- | ------------------------------ | ----------------------- |
| `//godogen:step <pattern>`  | `ctx.Step(pattern, fn)`        | Matches Given/When/Then |
| `//godogen:given <pattern>` | `ctx.Given(pattern, fn)`       | Preconditions           |
| `//godogen:when <pattern>`  | `ctx.When(pattern, fn)`        | Actions                 |
| `//godogen:then <pattern>`  | `ctx.Then(pattern, fn)`        | Assertions              |
| `//godogen:before`          | `ctx.Before(fn)`               | Before scenario hook    |
| `//godogen:after`           | `ctx.After(fn)`                | After scenario hook     |
| `//godogen:before_step`     | `ctx.StepContext().Before(fn)` | Before step hook        |
| `//godogen:after_step`      | `ctx.StepContext().After(fn)`  | After step hook         |

### Pattern Requirements

- Patterns must start with `^` and end with `$` (anchored regex)
- Use `([^"]*)` for quoted string captures
- Use `(\d+)` for integer captures
- Escape special characters: `\.`, `\(`, `\)`, etc.

### Generic Receivers

Generic context structs are supported. Type parameters are propagated to the generated initializer and renamed to `T1`, `T2`, ... to avoid collisions:

```go
type Suite[T any] struct{ state T }

//godogen:given ^I have (\d+) items$
func (s *Suite[T]) iHaveItems(count int) error {
    return nil
}
```

generates `func InitializeSteps[T1 any](sc *godog.ScenarioContext, r1 *Suite[T1])`.

The type declaration must be in the same file as the methods. For types declared elsewhere, use a type alias.

### Valid Return Types

Step functions can return:

- `error`
- `context.Context`
- `(context.Context, error)`
- `godog.Steps`
- nothing (void)

## Running the Generator

```bash
# In directory with go:generate directive
go generate ./...

# Or run godogen directly
godogen

# With Go 1.24+ tool directive
go tool godogen
```

The generator creates `*_initializer.go` files with step registration functions.

## CLI Commands

The `godogen-language-server` provides CLI commands for analysis:

### diagnose

Find issues in your BDD test suite:

```bash
godogen-language-server diagnose                    # All issues
godogen-language-server diagnose --severity error   # Only errors
godogen-language-server diagnose --format json      # JSON output
```

Reports:

- **Undefined steps** (error): Feature steps without definitions
- **Ambiguous steps** (warning): Steps matching multiple definitions
- **Duplicate definitions** (error): Same pattern defined twice
- **Unused definitions** (hint): Steps not used in features
- **Invalid patterns** (error): Missing anchors or invalid regex

### list-steps

List step definitions:

```bash
godogen-language-server list-steps                  # All steps
godogen-language-server list-steps --kind When      # Only When steps
godogen-language-server list-steps --format json    # JSON output
```

### find-definition

Navigate from feature step to Go definition:

```bash
godogen-language-server find-definition features/login.feature:10
```

### find-references

Find feature steps using a definition:

```bash
godogen-language-server find-references steps/auth.go:15:1
```

### hover

Get info about a position:

```bash
godogen-language-server hover features/login.feature:10:5
```

### symbols

List symbols in a file:

```bash
godogen-language-server symbols features/login.feature
godogen-language-server symbols steps/auth.go
```

### Global Flags

All commands support:

- `--root <dir>` - Workspace root (default: `.`)
- `--config <file>` - Config file path
- `--format, -f <text|json>` - Output format

## Configuration

Create `.godogen-language-server.json` in your project root:

```json
{
  "stepPatterns": ["**"]
}
```

The config file is **optional**. By default, the language server watches all files (`**`).

**Common configurations:**

```json
{
  "stepPatterns": [
    "**/*_steps.go",
    "**/*.feature"
  ]
}
```

For more patterns (monorepo, external features), see [Project Organization](docs/project-organization.md).

## CI/CD Integration

```bash
# Fail on any errors
godogen-language-server diagnose --severity error
if [ $? -ne 0 ]; then
  echo "BDD validation failed"
  exit 1
fi
```

Exit codes:

- `0`: No issues at the requested severity level
- `1`: Issues found or error running command

## Detailed Guides

- [Installation](docs/installation.md) - Go install, Go tool directive, Nix flake, editor setup
- [Project Organization](docs/project-organization.md) - Structure, TestContext pattern, composable steps
- [Scenario Isolation](docs/scenario-isolation.md) - Three-tier testing, transaction rollback, message queue isolation
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions
