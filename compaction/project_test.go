package compaction

import (
	"strings"
	"testing"
	"time"

	"coding-agent/session"
	"coding-agent/types"

	"github.com/stretchr/testify/require"
)

func TestProjectMessages_NoCompactions(t *testing.T) {
	archive := []types.Message{{Role: "user", Content: "hi"}}
	got := ProjectMessages(archive, nil)
	require.Len(t, got, 1)
	require.Equal(t, "hi", got[0].Content)
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
	require.Len(t, got, 2)
	require.True(t, strings.HasPrefix(got[0].Content, CompactedPrefix))
	require.Equal(t, "recent", got[1].Content)
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
	require.Len(t, got, 2)
	require.Contains(t, got[0].Content, "second")
}
