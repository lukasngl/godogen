package testsuite

import (
	"context"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var opts = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "pretty",
	Paths:  []string{"../features"},
	Strict: true,
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	tc := &TestContext{}

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		tc.Reset()
		return ctx, nil
	})

	ctx.Step(`^the input:$`, tc.SetInput)
	ctx.Step(`^I format$`, tc.Format)
	ctx.Step(`^the output is:$`, tc.CheckOutput)
	ctx.Step(`^the output ends with a single newline$`, tc.CheckEndsWithNewline)
}
