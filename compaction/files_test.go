package compaction

import (
	"encoding/json"
	"testing"

	"coding-agent/types"
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
	if len(ops.ReadFiles) != 1 || ops.ReadFiles[0] != "a.go" {
		t.Fatalf("read files = %v", ops.ReadFiles)
	}
	if len(ops.ModifiedFiles) != 1 || ops.ModifiedFiles[0] != "b.go" {
		t.Fatalf("modified files = %v", ops.ModifiedFiles)
	}
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
	if len(ops.ReadFiles) != 1 || ops.ReadFiles[0] != "old.go" {
		t.Fatalf("read files = %v", ops.ReadFiles)
	}
	if len(ops.ModifiedFiles) != 2 {
		t.Fatalf("modified files = %v, want 2", ops.ModifiedFiles)
	}
}
