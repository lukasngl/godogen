package testsuite

import (
	"testing"

	"github.com/cucumber/godog"
	"github.com/lukasngl/godogen/godogen-language-server/fsys/fsystest"
	"github.com/lukasngl/godogen/godogen-language-server/index/indextest"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			indextest.InitializeSteps(sc, indextest.NewTestContext())
			fsystest.InitializeSteps(sc, fsystest.MustNewTestContext())
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../features"},
			TestingT: t,
			Strict:   true,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
