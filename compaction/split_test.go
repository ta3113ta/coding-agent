package compaction

import (
	"encoding/json"
	"strings"
	"testing"

	"coding-agent/types"
)

func TestSplitMessages_KeepAllWhenShort(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	prefix, suffix := SplitMessages(msgs, 12)
	if prefix != nil {
		t.Fatalf("prefix = %v, want nil", prefix)
	}
	if len(suffix) != 2 {
		t.Fatalf("suffix len = %d, want 2", len(suffix))
	}
}

func TestSplitMessages_SplitsPrefixSuffix(t *testing.T) {
	msgs := make([]types.Message, 20)
	for i := range msgs {
		msgs[i] = types.Message{Role: "user", Content: "msg"}
	}
	prefix, suffix := SplitMessages(msgs, 5)
	if len(prefix) != 15 {
		t.Fatalf("prefix len = %d, want 15", len(prefix))
	}
	if len(suffix) != 5 {
		t.Fatalf("suffix len = %d, want 5", len(suffix))
	}
}

func TestSplitMessages_DoesNotSplitToolGroup(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "old reply"},
		{Role: "user", Content: "run tool"},
		{
			Role:    "assistant",
			Content: "calling",
			ToolCalls: []types.ToolCall{{
				ID:    "tc_1",
				Name:  "run_bash",
				Input: json.RawMessage(`{"command":"echo"}`),
			}},
		},
		{Role: "tool", Content: "echo", ToolCallID: "tc_1"},
		{Role: "assistant", Content: "done"},
		{Role: "user", Content: "recent"},
	}

	// keepRecent=3 would naively split before tool result; align should pull back
	prefix, suffix := SplitMessages(msgs, 3)
	for i, m := range suffix {
		if m.Role == "tool" {
			if i == 0 || suffix[i-1].Role != "assistant" || len(suffix[i-1].ToolCalls) == 0 {
				t.Fatalf("tool result at suffix[%d] not preceded by assistant tool_calls", i)
			}
		}
	}
	if len(prefix) == 0 {
		t.Fatal("expected non-empty prefix")
	}
	if len(suffix) < 3 {
		t.Fatalf("suffix len = %d, want at least 3", len(suffix))
	}
}

func TestEstimateTokens(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: "abcd"},
		{Role: "assistant", Content: "efgh"},
	}
	got := EstimateTokens(msgs)
	if got != 2 {
		t.Fatalf("EstimateTokens = %d, want 2", got)
	}
}

func TestSplitMessagesByTokens_KeepAllWhenShort(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	prefix, suffix := SplitMessagesByTokens(msgs, 20000)
	if prefix != nil {
		t.Fatalf("prefix = %v, want nil", prefix)
	}
	if len(suffix) != 2 {
		t.Fatalf("suffix len = %d, want 2", len(suffix))
	}
}

func TestSplitMessagesByTokens_SplitsByBudget(t *testing.T) {
	msgs := make([]types.Message, 10)
	for i := range msgs {
		msgs[i] = types.Message{Role: "user", Content: strings.Repeat("x", 400)} // ~100 tokens each
	}
	prefix, suffix := SplitMessagesByTokens(msgs, 200) // keep ~200 tokens = ~2 msgs
	if len(prefix) == 0 {
		t.Fatal("expected prefix")
	}
	if len(suffix) == 0 {
		t.Fatal("expected suffix")
	}
	if len(prefix)+len(suffix) != len(msgs) {
		t.Fatalf("split mismatch: %d + %d != %d", len(prefix), len(suffix), len(msgs))
	}
}

func TestSplitMessagesByTokens_DoesNotSplitToolGroup(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: strings.Repeat("a", 4000)},
		{Role: "user", Content: "run tool"},
		{
			Role:    "assistant",
			Content: "calling",
			ToolCalls: []types.ToolCall{{
				ID:    "tc_1",
				Name:  "run_bash",
				Input: json.RawMessage(`{"command":"echo"}`),
			}},
		},
		{Role: "tool", Content: "echo", ToolCallID: "tc_1"},
		{Role: "assistant", Content: "done"},
		{Role: "user", Content: "recent"},
	}
	prefix, suffix := SplitMessagesByTokens(msgs, 10)
	for i, m := range suffix {
		if m.Role == "tool" {
			if i == 0 || suffix[i-1].Role != "assistant" || len(suffix[i-1].ToolCalls) == 0 {
				t.Fatalf("tool result at suffix[%d] not preceded by assistant tool_calls", i)
			}
		}
	}
	if len(prefix) == 0 {
		t.Fatal("expected non-empty prefix")
	}
}
