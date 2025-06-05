//go:generate go run github.com/lukasngl/godogen/cmd/godogen
package godogs

import (
	"context"

	"github.com/cucumber/godog"
)

func InitializeScenario(ctx *godog.ScenarioContext) {
	InitializeGodogSteps(ctx)
}

//godogen:given there are (\d+) godogs
func thereAreGodogs(ctx context.Context, available int) error {
	return godog.ErrPending
}

//godogen:when ^I eat (\d+)$
//godogen:when ^I ingest (\d+)$
func iEat(ctx context.Context, dogs int) error {
	return godog.ErrPending
}

//godogen:then ^there should be (\d+) remaining$
func thereShouldBeRemaining(ctx context.Context, remaining int) error {
	return godog.ErrPending
}

//godogen:step ^there should be none remaining$
func thereShouldBeNoneRemaining(ctx context.Context) error {
	return godog.ErrPending
}

//godogen:before
func resetGodogs(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	return ctx, godog.ErrPending
}
