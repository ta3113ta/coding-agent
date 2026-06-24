package task

import (
	"context"
	"encoding/json"
	"testing"

	"coding-agent/spawn"

	"github.com/stretchr/testify/require"
)

type mockSpawner struct {
	last spawn.Request
	text string
	err  error
}

func (m *mockSpawner) Run(ctx context.Context, req spawn.Request) (spawn.Result, error) {
	_ = ctx
	m.last = req
	if m.err != nil {
		return spawn.Result{}, m.err
	}
	return spawn.Result{Text: m.text}, nil
}

func TestTool_Execute(t *testing.T) {
	spawner := &mockSpawner{text: "sub-agent result"}
	tool := Tool{spawner: spawner}

	input, _ := json.Marshal(map[string]any{
		"description":   "scan repo",
		"prompt":        "find main.go",
		"subagent_type": "explore",
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "sub-agent result", out)
	require.Equal(t, spawn.TypeExplore, spawner.last.Type)
	require.Equal(t, "find main.go", spawner.last.Prompt)
}

func TestTool_RequiresSpawner(t *testing.T) {
	tool := Tool{}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"x","subagent_type":"shell"}`))
	require.Error(t, err)
}
