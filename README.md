# godogen

godogen is golang codegenerator, that allows you to colocate [godog] step definitions with their pattern.

## Example

```go
// file: godog_steps.go
package godogs
//go:generate go run github.com/lukasngl/godogen

//godog:step ^I eat (\d+)$
func iEat(ctx context.Context, num int) (context.Context, error) {
        // … get the available value from context etc.
	available -= num
	return context.WithValue(ctx, godogsCtxKey{}, available), nil
}
```

will generate:

```go
// file: godog_steps_initializer.go
package godogs

import "github.com/cucumber/godog"

// InitializeGodogSteps registers steps defined in "godog_steps.go" with the [godog.ScenarioContext].
func InitializeGodogSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^I eat (\d+)$`, iEat)
}
```

See [./_example](./_example) for a larger demo.

## Features

- colocate step definition (i.e. the function declaration) with the pattern,
using the following directives on a function named `<FUNCTION>`:
  - `//godogen:step <PATTERN>` will generate `ctx.Step(<PATTERN>, <FUNCTION>)`
  - `//godogen:given <PATTERN>` will generate `ctx.Given(<PATTERN>, <FUNCTION>)`
  - `//godogen:when <PATTERN>` will generate `ctx.When(<PATTERN>, <FUNCTION>)`
  - `//godogen:then <PATTERN>` will generate `ctx.Then(<PATTERN>, <FUNCTION>)`
  - `//godogen:after` will generate `ctx.After(<FUNCTION>)`
  - `//godogen:before` will generate `ctx.Before(<FUNCTION>)`

[godog]: https://github.com/cucumber/godog
