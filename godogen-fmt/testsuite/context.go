package testsuite

// TestContext holds the state for a test scenario.
type TestContext struct {
	input  string
	output string
}

// Reset clears the test context for a new scenario.
func (tc *TestContext) Reset() {
	tc.input = ""
	tc.output = ""
}
