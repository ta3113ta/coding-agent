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
	h := NewHook(strings.NewReader(""), &strings.Builder{})
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{ToolName: "read_file"})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}

func TestHook_PromptApprove(t *testing.T) {
	h := NewHook(strings.NewReader("y\n"), &strings.Builder{})
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}

func TestHook_PromptDeny(t *testing.T) {
	h := NewHook(strings.NewReader("n\n"), &strings.Builder{})
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "write_file",
		Input:    json.RawMessage(`{"path":"x.txt","content":"hi"}`),
	})
	require.NoError(t, err)
	require.Equal(t, permission.Deny, res.Decision)
}

func TestHook_AskHintShown(t *testing.T) {
	var out strings.Builder
	h := NewHook(strings.NewReader("y\n"), &out)
	_, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"curl example.com"}`),
		AskHint:  "Network command detected",
	})
	require.NoError(t, err)
	require.Contains(t, out.String(), "Network command detected")
}
