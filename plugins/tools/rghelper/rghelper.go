package rghelper

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const DefaultMaxBytes = 30_000

var installHint = "ripgrep (rg) not found on PATH — Please install it first"

// Resolve returns the path to the rg binary or an error with install instructions.
func Resolve() (string, error) {
	path, err := exec.LookPath("rg")
	if err != nil {
		return "", fmt.Errorf("%s", installHint)
	}
	return path, nil
}

// Run executes rg with the given args, honoring ctx cancellation and a 30s timeout.
// Exit code 1 (no matches) is not treated as an error.
func Run(ctx context.Context, args []string) (string, error) {
	rg, err := Resolve()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, rg, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	runErr := cmd.Run()
	result := out.String()

	if ctx.Err() == context.DeadlineExceeded {
		return Truncate(result, DefaultMaxBytes) + "\n(timeout after 30 seconds)", nil
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		if result != "" {
			return "", fmt.Errorf("%s", result)
		}
		return "", runErr
	}
	return result, nil
}

// Truncate caps output size, keeping head and tail like run_bash.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	head := max * 6 / 10
	tail := max - head
	return s[:head] + fmt.Sprintf("\n... (truncated %d bytes in the middle) ...\n", len(s)-max) + s[len(s)-tail:]
}
