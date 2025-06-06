# godogen

godogen is a Go code generator that allows you to colocate [godog] step definitions with their patterns using directive comments.

[godog]: https://github.com/cucumber/godog

## Example

```go
// file: godog_steps.go

//go:generate go run github.com/lukasngl/godogen/cmd/godogen

package godogs

import (
	"context"
	"github.com/cucumber/godog"
)

//godogen:given there are (\d+) godogs
func thereAreGodogs(ctx context.Context, available int) error {
	// ... your step implementation
	return godog.ErrPending
}

//godogen:when ^I eat (\d+)$
func iEat(ctx context.Context, dogs int) error {
	// ... your step implementation
	return godog.ErrPending
}
```

will generate:

```go
// file: godog_steps_initializer.go
package godogs

import "github.com/cucumber/godog"

// InitializeGodogSteps registers steps defined in "godog_steps.go" with the [godog.ScenarioContext].
func InitializeGodogSteps(ctx *godog.ScenarioContext) {
	ctx.Given(`there are (\d+) godogs`, thereAreGodogs)
	ctx.When(`^I eat (\d+)$`, iEat)
}
```

See [./_example](./_example) for a complete working demo.

## Features

Colocate step definitions (function declarations) with their patterns using the following directives:

- `//godogen:step <PATTERN>` → generates `ctx.Step(<PATTERN>, <FUNCTION>)`
- `//godogen:given <PATTERN>` → generates `ctx.Given(<PATTERN>, <FUNCTION>)`
- `//godogen:when <PATTERN>` → generates `ctx.When(<PATTERN>, <FUNCTION>)`
- `//godogen:then <PATTERN>` → generates `ctx.Then(<PATTERN>, <FUNCTION>)`
- `//godogen:before` → generates `ctx.Before(<FUNCTION>)`
- `//godogen:after` → generates `ctx.After(<FUNCTION>)`

### Multiple patterns per function

You can define multiple patterns for the same function:

```go
//godogen:when ^I eat (\d+)$
//godogen:when ^I ingest (\d+)$
func iEat(ctx context.Context, dogs int) error {
	return godog.ErrPending
}

## Usage

1. Add the go:generate directive to your step definition file:
   ```go
   //go:generate go run github.com/lukasngl/godogen/cmd/godogen
   ```

2. Add godogen comments above your step functions

3. Run `go generate` to create the initializer file

4. Use the generated `InitializeGodogSteps` function in your test setup

## Linting

godogen will run validation check when generating and report errors. This functionality is additionally exposed as a standalone linter, that can be integrated into golangci-lint.

godogen will run the following checks:
- check if the pattern compiles
- check that the step function signature is valid
- check that the number of parameters matches the number of capture groups of each pattern.

### Standalone Linter

You can run the linter directly using:

```bash
go run github.com/lukasngl/godogen/cmd/godogen-lint ./...
```

Or install it globally:

```bash
go install github.com/lukasngl/godogen/@latest
godogen-lint ./...
```

### golangci-lint Plugin

godogen can be integrated with [golangci-lint] as a plugin:

1. Create a `.custom-gcl.yml` file in your project root:
   ```yaml
   version: v2.1.2
   plugins:
     - module: 'github.com/lukasngl/godogen/gcl-plugin'
       import: 'github.com/lukasngl/godogen/gcl-plugin'
       version: latest
   ```

2. Build a custom golangci-lint that includes godogen-lint:
   ```bash
   golangci-lint custom # will create the custom-gcl binary
   ```

3. Alias golangci-lint to the `custom-gcl` binary.
