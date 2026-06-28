package tools

import (
	"context"
	"encoding/json"
	"testing"

	"coding-agent/types"
)

func TestDefinitionsFiltered(t *testing.T) {
	r := NewRegistry(
		stubTool{name: "read_file"},
		stubTool{name: "write_file"},
		stubTool{name: "grep"},
	)

	all := r.Definitions()
	if len(all) != 3 {
		t.Fatalf("expected 3 definitions, got %d", len(all))
	}

	filtered := r.DefinitionsFiltered(map[string]bool{
		"read_file": true,
		"grep":      true,
	})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered definitions, got %d", len(filtered))
	}
}

type stubTool struct {
	name string
}

func (s stubTool) Name() string { return s.name }
func (s stubTool) Definition() types.ToolDefinition {
	return types.ToolDefinition{Name: s.name}
}
func (s stubTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return "", nil
}
