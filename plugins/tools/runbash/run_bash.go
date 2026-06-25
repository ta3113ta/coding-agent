package runbash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"coding-agent/plugin"
	"coding-agent/tools"
	"coding-agent/types"
)

type RunBash struct{}

func (RunBash) Name() string { return "run_bash" }

func (RunBash) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name:        "run_bash",
		Description: "Run a bash command in the current working directory. Use for git, go build/test, etc. Not for search (use grep/glob instead). 60 second timeout.",
		Properties: map[string]any{
			"command": map[string]any{"type": "string", "description": "bash command to run"},
		},
		Required: []string{"command"},
	}
}

func (RunBash) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()

	result := truncate(out.String(), 30_000)
	if ctx.Err() == context.DeadlineExceeded {
		return result + "\n(timeout after 60 seconds)", nil
	}
	if err != nil {
		return fmt.Sprintf("%s\n(exit error: %v)", result, err), nil
	}
	if result == "" {
		return "(no output, exit 0)", nil
	}
	return result, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	head := max * 6 / 10
	tail := max - head
	return s[:head] + fmt.Sprintf("\n... (truncated %d bytes in the middle) ...\n", len(s)-max) + s[len(s)-tail:]
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/runbash" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterTools(app, RunBash{})
	return nil
}

var _ tools.Tool = RunBash{}
