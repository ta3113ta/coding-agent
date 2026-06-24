package compaction

import (
	"strings"
	"testing"

	"coding-agent/types"

	"github.com/stretchr/testify/require"
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
	require.Contains(t, got, "[User]: hello")
	require.Contains(t, got, "[Assistant]: reading")
	require.Contains(t, got, "[Assistant tool calls]: read_file")
	require.Contains(t, got, "[Tool result]: file contents")
}

func TestSerializeConversation_TruncatesToolResult(t *testing.T) {
	long := strings.Repeat("x", 3000)
	msgs := []types.Message{{Role: "tool", Content: long}}
	got := SerializeConversation(msgs, SerializeOptions{ToolResultMaxLen: 2000})
	require.LessOrEqual(t, len(got), 2100)
	require.Contains(t, got, "[truncated")
}
