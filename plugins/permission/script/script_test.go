package script

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"coding-agent/permission"
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
		if err != nil {
			t.Fatalf("parseHookOutput(%+v): %v", tc.in, err)
		}
		if res.Decision != tc.decision {
			t.Fatalf("decision = %v, want %v", res.Decision, tc.decision)
		}
		if tc.message != "" && res.Message != tc.message {
			t.Fatalf("message = %q, want %q", res.Message, tc.message)
		}
	}
}

func TestRunHook_Allow(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "allow.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho '{\"permission\":\"allow\"}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := runHook(context.Background(), Def{Command: "allow.sh"}, permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	}, dir)
	if err != nil {
		t.Fatalf("runHook: %v", err)
	}
	if res.Decision != permission.Allow {
		t.Fatalf("decision = %v, want Allow", res.Decision)
	}
}

func TestRunHook_DenyExitCode(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "deny.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho blocked 1>&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := runHook(context.Background(), Def{Command: "deny.sh", FailClosed: true}, permission.ToolUseRequest{
		ToolName: "run_bash",
	}, dir)
	if err != nil {
		t.Fatalf("runHook: %v", err)
	}
	if res.Decision != permission.Deny {
		t.Fatalf("decision = %v, want Deny", res.Decision)
	}
}

func TestHook_Matcher(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "deny.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho '{\"permission\":\"deny\",\"agent_message\":\"no bash\"}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("NewHook: %v", err)
	}

	res, err := hook.BeforeToolUse(context.Background(), permission.ToolUseRequest{ToolName: "run_bash"})
	if err != nil {
		t.Fatalf("BeforeToolUse: %v", err)
	}
	if res.Decision != permission.Deny {
		t.Fatalf("decision = %v, want Deny for matched tool", res.Decision)
	}

	res, err = hook.BeforeToolUse(context.Background(), permission.ToolUseRequest{ToolName: "read_file"})
	if err != nil {
		t.Fatalf("BeforeToolUse: %v", err)
	}
	if res.Decision != permission.Allow {
		t.Fatalf("decision = %v, want Allow for unmatched tool", res.Decision)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config for missing file")
	}
}
