package writefile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"coding-agent/plugin"
	"coding-agent/tools"
	"coding-agent/types"
)

type WriteFile struct{}

func (WriteFile) Name() string { return "write_file" }

func (WriteFile) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name:        "write_file",
		Description: "Write content to a file. Creates parent directories automatically if needed. Overwrites existing content entirely.",
		Properties: map[string]any{
			"path":    map[string]any{"type": "string", "description": "path of the file to write"},
			"content": map[string]any{"type": "string", "description": "full file content"},
		},
		Required: []string{"path", "content"},
	}
}

func (WriteFile) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	if dir := filepath.Dir(args.Path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(args.Path, []byte(args.Content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote file %s successfully (%d bytes)", args.Path, len(args.Content)), nil
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/writefile" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterTools(app, WriteFile{})
	return nil
}

var _ tools.Tool = WriteFile{}
