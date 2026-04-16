package executor

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"time"
)

// Result holds the output of a gh command execution.
type Result struct {
	Stdout   []byte        `json:"stdout"`
	Stderr   []byte        `json:"stderr"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration_ms"`
}

// Execute runs a gh command with the given arguments and returns its output.
// If workDir is non-empty and absolute, the command runs in that directory.
func Execute(ctx context.Context, ghPath string, args []string, workDir string) *Result {
	start := time.Now()

	cmd := exec.CommandContext(ctx, ghPath, args...)
	if workDir != "" && filepath.IsAbs(workDir) {
		cmd.Dir = workDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: 0,
		Duration: time.Since(start),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
			result.Stderr = append(result.Stderr, []byte("\nghx: "+err.Error())...)
		}
	}

	return result
}

// ExecutePassthrough runs a gh command directly, inheriting stdio.
// Used for interactive/mutating commands that can't be cached.
func ExecutePassthrough(ghPath string, args []string) int {
	cmd := exec.Command(ghPath, args...)
	cmd.Stdin = nil // stdin is not forwarded through daemon
	cmd.Stdout = nil
	cmd.Stderr = nil

	// For passthrough, we connect to the calling process's stdio
	// This is only used when ghx falls back to direct execution
	cmd.Stdin = nil

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}
	return 0
}
