package tools

import (
	"context"
	"encoding/json"
	"testing"

	"coding-agent/types"

	"github.com/stretchr/testify/require"
)

type stub struct {
	name string
}

func (s stub) Name() string { return s.name }

func (s stub) Definition() types.ToolDefinition {
	return types.ToolDefinition{Name: s.name}
}

func (s stub) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	_ = input
	return "ok", nil
}

func TestFilter(t *testing.T) {
	src := NewRegistry(stub{name: "a"}, stub{name: "b"}, stub{name: "c"})
	filtered := Filter(src, map[string]bool{"a": true, "c": true})
	defs := filtered.Definitions()
	require.Len(t, defs, 2)
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}
	require.True(t, names["a"])
	require.True(t, names["c"])
	require.False(t, names["b"])
}
