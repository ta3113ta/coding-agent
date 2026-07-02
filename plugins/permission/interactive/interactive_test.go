package interactive

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"coding-agent/permission"

	"github.com/stretchr/testify/require"
)

func TestHook_AutoAllowReadOnly(t *testing.T) {
	h := NewHook(strings.NewReader(""), &strings.Builder{}, nil)
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{ToolName: "read_file"})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}

func TestHook_AutoAllowEdits(t *testing.T) {
	h := NewHook(strings.NewReader(""), &strings.Builder{}, nil)
	for _, tool := range []string{"write_file", "str_replace"} {
		res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
			ToolName: tool,
			Input:    json.RawMessage(`{"path":"x.txt"}`),
		})
		require.NoError(t, err)
		require.Equal(t, permission.Allow, res.Decision, "tool %s", tool)
	}
}

func TestHook_PromptApprove(t *testing.T) {
	h := NewHook(strings.NewReader("y\n"), &strings.Builder{}, nil)
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}

func TestHook_PromptDeny(t *testing.T) {
	h := NewHook(strings.NewReader("n\n"), &strings.Builder{}, nil)
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Deny, res.Decision)
}

func TestHook_AskHintShown(t *testing.T) {
	var out strings.Builder
	h := NewHook(strings.NewReader("y\n"), &out, nil)
	_, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"curl example.com"}`),
		AskHint:  "Network command detected",
	})
	require.NoError(t, err)
	require.Contains(t, out.String(), "Network command detected")
}

func TestHook_RememberTool(t *testing.T) {
	rules := permission.NewSessionRules()
	h := NewHook(strings.NewReader("a\n"), &strings.Builder{}, rules)

	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)

	res, err = h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo bye"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}

func TestHook_RememberAll(t *testing.T) {
	rules := permission.NewSessionRules()
	h := NewHook(strings.NewReader("A\n"), &strings.Builder{}, rules)

	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)

	res, err = h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "task",
		Input:    json.RawMessage(`{"prompt":"do work"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}

func TestHook_SessionRulesSkipPrompt(t *testing.T) {
	rules := permission.NewSessionRules()
	rules.AllowTool("run_bash")
	h := NewHook(strings.NewReader(""), &strings.Builder{}, rules)

	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}
