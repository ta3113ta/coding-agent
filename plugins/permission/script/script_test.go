package script

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"coding-agent/permission"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHookOutput(t *testing.T) {
	tests := []struct {
		in       hookOutput
		decision permission.Decision
		message  string
	}{
		{hookOutput{Permission: "allow"}, permission.Allow, ""},
		{hookOutput{Permission: "deny", AgentMessage: "blocked"}, permission.Deny, "blocked"},
		{hookOutput{Permission: "ask", UserMessage: "confirm?"}, permission.Ask, "confirm?"},
	}
	for _, tc := range tests {
		res, err := parseHookOutput(tc.in)
		require.NoError(t, err)
		assert.Equal(t, tc.decision, res.Decision)
		if tc.message != "" {
			assert.Equal(t, tc.message, res.Message)
		}
	}
}

func TestRunHook_Allow(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "allow.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho '{\"permission\":\"allow\"}'\n"), 0o755))

	res, err := runHook(context.Background(), Def{Command: "allow.sh"}, permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	}, dir)
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}

func TestRunHook_DenyExitCode(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "deny.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho blocked 1>&2\nexit 2\n"), 0o755))

	res, err := runHook(context.Background(), Def{Command: "deny.sh", FailClosed: true}, permission.ToolUseRequest{
		ToolName: "run_bash",
	}, dir)
	require.NoError(t, err)
	require.Equal(t, permission.Deny, res.Decision)
}

func TestHook_Matcher(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "deny.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\necho '{\"permission\":\"deny\",\"agent_message\":\"no bash\"}'\n"), 0o755))

	cfg := &Config{
		Version: 1,
		Hooks: map[string][]Def{
			"preToolUse": {{
				Command: "deny.sh",
				Matcher: "run_bash",
			}},
		},
	}
	hook, err := NewHook(cfg, dir)
	require.NoError(t, err)

	res, err := hook.BeforeToolUse(context.Background(), permission.ToolUseRequest{ToolName: "run_bash"})
	require.NoError(t, err)
	require.Equal(t, permission.Deny, res.Decision)

	res, err = hook.BeforeToolUse(context.Background(), permission.ToolUseRequest{ToolName: "read_file"})
	require.NoError(t, err)
	require.Equal(t, permission.Allow, res.Decision)
}

func TestLoadConfig_MissingFile(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	require.NoError(t, err)
	require.Nil(t, cfg)
}
