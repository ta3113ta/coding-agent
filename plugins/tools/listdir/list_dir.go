package listdir

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"coding-agent/plugin"
	"coding-agent/tools"
	"coding-agent/types"
)

type ListDir struct{}

func (ListDir) Name() string { return "list_dir" }

func (ListDir) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name:        "list_dir",
		Description: "List files and directories at the given path. Omit path for the current directory.",
		Properties: map[string]any{
			"path": map[string]any{"type": "string", "description": "directory path; omit for '.'"},
		},
	}
}

func (ListDir) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	var args struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(input, &args)
	if args.Path == "" {
		args.Path = "."
	}
	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		b.WriteString(fmt.Sprintf("%-40s %8d\n", name, size))
	}
	if b.Len() == 0 {
		return "(empty directory)", nil
	}
	return b.String(), nil
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/listdir" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterTools(app, ListDir{})
	return nil
}

var _ tools.Tool = ListDir{}
