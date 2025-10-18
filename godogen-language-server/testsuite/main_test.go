package testsuite

import (
	"context"
	"os"
	"testing"

	"github.com/cucumber/godog"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(sc *godog.ScenarioContext) {
	// Index test context for index-related tests
	indexTC := NewTestContext()
	InitializeIndexSteps(sc, indexTC)

	// Fsys test context for filesystem watching tests
	fsysTC, err := NewFsysTestContext()
	if err != nil {
		panic(err)
	}

	// Cleanup after scenario
	sc.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if cleanupErr := fsysTC.Cleanup(); cleanupErr != nil {
			return ctx, cleanupErr
		}
		return ctx, err
	})

	InitializeFsysSteps(sc, fsysTC)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
