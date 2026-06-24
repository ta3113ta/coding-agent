package runner

import (
	"context"
	"encoding/json"
	"testing"

	"coding-agent/spawn"
	"coding-agent/tools"
	"coding-agent/types"

	"github.com/stretchr/testify/require"
)

type countingProvider struct {
	calls int
	text  string
}

func (p *countingProvider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	_ = ctx
	p.calls++
	return &types.CompleteResponse{Text: p.text}, nil
}

func TestRunner_RunSubAgent(t *testing.T) {
	provider := &countingProvider{text: "child done"}
	reg := tools.NewRegistry(stubTool{name: "read_file"}, stubTool{name: "list_dir"})
	r := &Runner{
		provider:     provider,
		basePrompt:   "base prompt",
		model:        "test-model",
		providerName: "anthropic",
		tools:        reg,
		maxTurns:     5,
	}

	res, err := r.Run(context.Background(), spawn.Request{
		Type:        spawn.TypeExplore,
		Description: "scan files",
		Prompt:      "list the repo",
	})
	require.NoError(t, err)
	require.Equal(t, "child done", res.Text)
	require.Equal(t, 1, provider.calls)
}

func TestRunner_RejectsEmptyPrompt(t *testing.T) {
	r := &Runner{tools: tools.NewRegistry(), maxTurns: 5}
	_, err := r.Run(context.Background(), spawn.Request{Type: spawn.TypeShell, Prompt: "  "})
	require.ErrorContains(t, err, "prompt")
}

func TestRunner_RejectsInvalidType(t *testing.T) {
	reg := tools.NewRegistry(stubTool{name: "read_file"})
	r := &Runner{tools: reg, maxTurns: 5}
	_, err := r.Run(context.Background(), spawn.Request{Type: "bogus", Prompt: "do work"})
	require.Error(t, err)
}

type stubTool struct {
	name string
}

func (s stubTool) Name() string { return s.name }

func (s stubTool) Definition() types.ToolDefinition {
	return types.ToolDefinition{Name: s.name}
}

func (s stubTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	_ = input
	return "ok", nil
}
