# godogen

Now with Methods! 🎉

godogen is golang codegenerator, that allows you to colocate [godog] step definitions with their pattern.

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
//go:generate go run github.com/lukasngl/godogen

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
- generic methods are not yet supported

[godog]: https://github.com/cucumber/godog
