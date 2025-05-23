//go:generate go run github.com/lukasngl/godogen
package godogs

import (
	"context"

	"github.com/cucumber/godog"
)

func InitializeScenario(ctx *godog.ScenarioContext) {
	InitializeGodogSteps(ctx)
}

//godgen:step there are (\d+) godogs
func thereAreGodogs(ctx context.Context, available int) {
	return godog.ErrPending
}

//godgen:step ^I eat (\d+)$
func iEat(ctx context.Context, num int) error {
	return godog.ErrPending
}

//godgen:step ^there should be (\d+) remaining$
func thereShouldBeRemaining(ctx context.Context, remaining int) error {
	return godog.ErrPending
}

//godgen:step ^there should be none remaining$
func thereShouldBeNoneRemaining(ctx context.Context) error {
	return godog.ErrPending
}

//godgen:before
func resetGodogs(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return ctx, godog.ErrPending
}
