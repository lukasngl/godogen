//go:generate go tool godogen
package godogs

import (
	"context"
	"fmt"

	"github.com/cucumber/godog"
)

type (
	ScState  struct{}
	ScState2 struct{}
)

//godogen:when ^I want to use methods$
func (*ScState) NowWithMethods() {}

//godogen:then ^I can$
func (*ScState) ThenICan() {}

//godogen:given ^I need more than two state objects$
func (*ScState2) ItWorks(int) {}

func InitializeScenario(ctx *godog.ScenarioContext) {
	InitializeGodogSteps(ctx, &ScState{}, &ScState2{})
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

//godogen:before_step
func printStep(ctx context.Context, step *godog.Step) (context.Context, error) {
	fmt.Println(step.Text)
	return ctx, godog.ErrPending
}
