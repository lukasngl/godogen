# godogen

godogen is quick prototype, that allows you to colocate [godog] step definitions with their pattern.

Example:

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

[godog]: https://github.com/cucumber/godog
