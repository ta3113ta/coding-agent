package compaction

import (
	"strings"
	"testing"

	"coding-agent/types"
)

func TestSerializeConversation_Labels(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: "hello"},
		{
			Role:    "assistant",
			Content: "reading",
			ToolCalls: []types.ToolCall{{
				Name:  "read_file",
				Input: []byte(`{"path":"foo.go"}`),
			}},
		},
		{Role: "tool", Content: "file contents", ToolCallID: "tc1"},
	}
	got := SerializeConversation(msgs, SerializeOptions{})
	if !strings.Contains(got, "[User]: hello") {
		t.Fatalf("missing user label: %q", got)
	}
	if !strings.Contains(got, "[Assistant]: reading") {
		t.Fatalf("missing assistant label: %q", got)
	}
	if !strings.Contains(got, "[Assistant tool calls]: read_file") {
		t.Fatalf("missing tool calls label: %q", got)
	}
	if !strings.Contains(got, "[Tool result]: file contents") {
		t.Fatalf("missing tool result label: %q", got)
	}
}

func TestSerializeConversation_TruncatesToolResult(t *testing.T) {
	long := strings.Repeat("x", 3000)
	msgs := []types.Message{{Role: "tool", Content: long}}
	got := SerializeConversation(msgs, SerializeOptions{ToolResultMaxLen: 2000})
	if len(got) > 2100 {
		t.Fatalf("expected truncation, got len %d", len(got))
	}
	if !strings.Contains(got, "[truncated") {
		t.Fatalf("missing truncation marker: %q", got)
	}
}
