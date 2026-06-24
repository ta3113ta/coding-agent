package compaction

import (
	"encoding/json"
	"testing"

	"coding-agent/types"

	"github.com/stretchr/testify/require"
)

func TestExtractFileOps(t *testing.T) {
	msgs := []types.Message{
		{
			Role: "assistant",
			ToolCalls: []types.ToolCall{
				{Name: "read_file", Input: json.RawMessage(`{"path":"a.go"}`)},
				{Name: "write_file", Input: json.RawMessage(`{"path":"b.go","content":"x"}`)},
			},
		},
	}
	ops := ExtractFileOps(msgs, FileOps{})
	require.Len(t, ops.ReadFiles, 1)
	require.Equal(t, "a.go", ops.ReadFiles[0])
	require.Len(t, ops.ModifiedFiles, 1)
	require.Equal(t, "b.go", ops.ModifiedFiles[0])
}

func TestExtractFileOps_Cumulative(t *testing.T) {
	prior := FileOps{ReadFiles: []string{"old.go"}, ModifiedFiles: []string{"edited.go"}}
	msgs := []types.Message{
		{
			Role: "assistant",
			ToolCalls: []types.ToolCall{
				{Name: "str_replace", Input: json.RawMessage(`{"path":"new.go"}`)},
			},
		},
	}
	ops := ExtractFileOps(msgs, prior)
	require.Len(t, ops.ReadFiles, 1)
	require.Equal(t, "old.go", ops.ReadFiles[0])
	require.Len(t, ops.ModifiedFiles, 2)
}
