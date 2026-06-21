package interactive

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"coding-agent/permission"
)

func TestHook_AutoAllowReadOnly(t *testing.T) {
	h := NewHook(strings.NewReader(""), &strings.Builder{})
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{ToolName: "read_file"})
	if err != nil {
		t.Fatalf("BeforeToolUse: %v", err)
	}
	if res.Decision != permission.Allow {
		t.Fatalf("decision = %v, want Allow", res.Decision)
	}
}

func TestHook_PromptApprove(t *testing.T) {
	h := NewHook(strings.NewReader("y\n"), &strings.Builder{})
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"echo hi"}`),
	})
	if err != nil {
		t.Fatalf("BeforeToolUse: %v", err)
	}
	if res.Decision != permission.Allow {
		t.Fatalf("decision = %v, want Allow", res.Decision)
	}
}

func TestHook_PromptDeny(t *testing.T) {
	h := NewHook(strings.NewReader("n\n"), &strings.Builder{})
	res, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "write_file",
		Input:    json.RawMessage(`{"path":"x.txt","content":"hi"}`),
	})
	if err != nil {
		t.Fatalf("BeforeToolUse: %v", err)
	}
	if res.Decision != permission.Deny {
		t.Fatalf("decision = %v, want Deny", res.Decision)
	}
}

func TestHook_AskHintShown(t *testing.T) {
	var out strings.Builder
	h := NewHook(strings.NewReader("y\n"), &out)
	_, err := h.BeforeToolUse(context.Background(), permission.ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"curl example.com"}`),
		AskHint:  "Network command detected",
	})
	if err != nil {
		t.Fatalf("BeforeToolUse: %v", err)
	}
	if !strings.Contains(out.String(), "Network command detected") {
		t.Fatalf("output = %q, want ask hint", out.String())
	}
}
