package steps

import (
	"bytes"
	"os/exec"
	"strings"
)

//godogen:when ^I run godogen$
func (tc *TestContext) iRunGodogen() error {
	// Get the path to the godogen tool binary
	toolPath, err := exec.Command("go", "tool", "-n", "godogen").Output()
	if err != nil {
		return err
	}

	// Run godogen from the temp dir (godogen writes output to cwd)
	cmd := exec.Command(strings.TrimSpace(string(toolPath)))
	cmd.Dir = tc.tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Combine stdout and stderr for output
	tc.lastOutput = stdout.String() + stderr.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			tc.lastExitCode = exitErr.ExitCode()
		} else {
			tc.lastExitCode = 1
		}
	} else {
		tc.lastExitCode = 0
	}

	return nil // Don't fail the step on non-zero exit code
}
