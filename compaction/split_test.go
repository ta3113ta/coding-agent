package compaction

import (
	"encoding/json"
	"strings"
	"testing"

	"coding-agent/types"

	"github.com/stretchr/testify/require"
)

func TestSplitMessages_KeepAllWhenShort(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	prefix, suffix := SplitMessages(msgs, 12)
	require.Nil(t, prefix)
	require.Len(t, suffix, 2)
}

func TestSplitMessages_SplitsPrefixSuffix(t *testing.T) {
	msgs := make([]types.Message, 20)
	for i := range msgs {
		msgs[i] = types.Message{Role: "user", Content: "msg"}
	}
	prefix, suffix := SplitMessages(msgs, 5)
	require.Len(t, prefix, 15)
	require.Len(t, suffix, 5)
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
			require.False(t, i == 0 || suffix[i-1].Role != "assistant" || len(suffix[i-1].ToolCalls) == 0,
				"tool result at suffix[%d] not preceded by assistant tool_calls", i)
		}
	}
	require.NotEmpty(t, prefix)
	require.GreaterOrEqual(t, len(suffix), 3)
}

func TestEstimateTokens(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: "abcd"},
		{Role: "assistant", Content: "efgh"},
	}
	got := EstimateTokens(msgs)
	require.Equal(t, 2, got)
}

func TestSplitMessagesByTokens_KeepAllWhenShort(t *testing.T) {
	msgs := []types.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	prefix, suffix := SplitMessagesByTokens(msgs, 20000)
	require.Nil(t, prefix)
	require.Len(t, suffix, 2)
}

func TestSplitMessagesByTokens_SplitsByBudget(t *testing.T) {
	msgs := make([]types.Message, 10)
	for i := range msgs {
		msgs[i] = types.Message{Role: "user", Content: strings.Repeat("x", 400)} // ~100 tokens each
	}
	prefix, suffix := SplitMessagesByTokens(msgs, 200) // keep ~200 tokens = ~2 msgs
	require.NotEmpty(t, prefix)
	require.NotEmpty(t, suffix)
	require.Equal(t, len(msgs), len(prefix)+len(suffix))
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
			require.False(t, i == 0 || suffix[i-1].Role != "assistant" || len(suffix[i-1].ToolCalls) == 0,
				"tool result at suffix[%d] not preceded by assistant tool_calls", i)
		}
	}
	require.NotEmpty(t, prefix)
}
