package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"coding-agent/plugin"
	"coding-agent/spawn"
	"coding-agent/tools"
	"coding-agent/types"
)

type Tool struct {
	spawner spawn.Runner
}

func (Tool) Name() string { return "task" }

func (Tool) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name: "task",
		Description: "Launch a sub-agent to handle a focused sub-task in isolation. " +
			"The sub-agent runs with a fresh context and returns a final summary. " +
			"Use explore for read-only codebase search, shell for bash commands, generalPurpose for broader tasks.",
		Properties: map[string]any{
			"description": map[string]any{
				"type":        "string",
				"description": "Short 3-5 word title for the sub-task",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "Detailed task instructions for the sub-agent (no parent conversation history)",
			},
			"subagent_type": map[string]any{
				"type":        "string",
				"enum":        []string{string(spawn.TypeGeneralPurpose), string(spawn.TypeExplore), string(spawn.TypeShell)},
				"description": "Sub-agent profile: generalPurpose, explore (read-only), or shell (bash only)",
			},
		},
		Required: []string{"description", "prompt", "subagent_type"},
	}
}

func (t Tool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	if t.spawner == nil {
		return "", fmt.Errorf("sub-agent spawning is disabled")
	}

	var args struct {
		Description  string `json:"description"`
		Prompt       string `json:"prompt"`
		SubagentType string `json:"subagent_type"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Prompt) == "" {
		return "", fmt.Errorf("prompt is required")
	}
	if strings.TrimSpace(args.SubagentType) == "" {
		return "", fmt.Errorf("subagent_type is required")
	}

	res, err := t.spawner.Run(ctx, spawn.Request{
		Type:        spawn.Type(args.SubagentType),
		Description: strings.TrimSpace(args.Description),
		Prompt:      args.Prompt,
	})
	if err != nil {
		return "", err
	}
	return res.Text, nil
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/task" }

func (Plugin) Register(app *plugin.App) error {
	if app.Spawner == nil {
		return nil
	}
	plugin.RegisterTools(app, Tool{spawner: app.Spawner})
	return nil
}

var _ tools.Tool = Tool{}
