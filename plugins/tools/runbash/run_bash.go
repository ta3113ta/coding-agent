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
		Description: "รันคำสั่ง bash ใน working directory ปัจจุบัน ใช้สำหรับ grep, find, git, go build/test ฯลฯ มี timeout 60 วินาที",
		Properties: map[string]any{
			"command": map[string]any{"type": "string", "description": "คำสั่ง bash ที่จะรัน"},
		},
		Required: []string{"command"},
	}
}

func (RunBash) Execute(input json.RawMessage) (string, error) {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()

	result := truncate(out.String(), 30_000)
	if ctx.Err() == context.DeadlineExceeded {
		return result + "\n(timeout หลัง 60 วินาที)", nil
	}
	if err != nil {
		return fmt.Sprintf("%s\n(exit error: %v)", result, err), nil
	}
	if result == "" {
		return "(ไม่มี output, exit 0)", nil
	}
	return result, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	head := max * 6 / 10
	tail := max - head
	return s[:head] + fmt.Sprintf("\n... (ตัด %d bytes ตรงกลาง) ...\n", len(s)-max) + s[len(s)-tail:]
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/runbash" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterTools(app, RunBash{})
	return nil
}

var _ tools.Tool = RunBash{}
