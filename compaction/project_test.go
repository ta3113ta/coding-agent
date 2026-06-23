package compaction

import (
	"strings"
	"testing"
	"time"

	"coding-agent/session"
	"coding-agent/types"
)

func TestProjectMessages_NoCompactions(t *testing.T) {
	archive := []types.Message{{Role: "user", Content: "hi"}}
	got := ProjectMessages(archive, nil)
	if len(got) != 1 || got[0].Content != "hi" {
		t.Fatalf("got = %+v", got)
	}
}

func TestProjectMessages_OneCompaction(t *testing.T) {
	archive := []types.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "reply"},
		{Role: "user", Content: "recent"},
	}
	compactions := []session.CompactionRecord{{
		ID:             "c1",
		Timestamp:      time.Now(),
		Summary:        "user asked about foo",
		FirstKeptIndex: 2,
	}}
	got := ProjectMessages(archive, compactions)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if !strings.HasPrefix(got[0].Content, CompactedPrefix) {
		t.Fatalf("first = %q", got[0].Content)
	}
	if got[1].Content != "recent" {
		t.Fatalf("kept = %q", got[1].Content)
	}
}

func TestProjectMessages_LatestCompactionWins(t *testing.T) {
	archive := []types.Message{
		{Role: "user", Content: "1"},
		{Role: "user", Content: "2"},
		{Role: "user", Content: "3"},
	}
	compactions := []session.CompactionRecord{
		{Summary: "first", FirstKeptIndex: 1},
		{Summary: "second", FirstKeptIndex: 2},
	}
	got := ProjectMessages(archive, compactions)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if !strings.Contains(got[0].Content, "second") {
		t.Fatalf("summary = %q", got[0].Content)
	}
}
