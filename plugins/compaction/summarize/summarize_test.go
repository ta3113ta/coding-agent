package summarize

import (
	"context"
	"strings"
	"testing"

	"coding-agent/compaction"
	"coding-agent/config"
	"coding-agent/types"

	"github.com/stretchr/testify/require"
)

type fakeProvider struct {
	lastReq *types.CompleteRequest
	text    string
}

func (f *fakeProvider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	cp := req
	f.lastReq = &cp
	return &types.CompleteResponse{Text: f.text}, nil
}

func TestMaybeCompact_UnderBudgetNoOp(t *testing.T) {
	p := &fakeProvider{text: "summary"}
	c := &Compactor{
		provider: p,
		cfg: config.Config{
			CompactionReserveTokens:    16384,
			CompactionKeepRecentTokens: 20000,
			CompactionContextWindow:    200000,
		},
	}

	msgs := []types.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	res, err := c.MaybeCompact(context.Background(), compaction.Request{
		Archive: msgs,
		Model:   "test",
	})
	require.NoError(t, err)
	require.False(t, res.Compacted)
	require.Nil(t, p.lastReq)
}

func TestMaybeCompact_ForceSummarizes(t *testing.T) {
	p := &fakeProvider{text: "## Goal\nbuild fizzbuzz"}
	c := &Compactor{
		provider: p,
		cfg: config.Config{
			CompactionReserveTokens:    16384,
			CompactionKeepRecentTokens: 4,
			CompactionContextWindow:    200000,
		},
	}

	msgs := []types.Message{
		{Role: "user", Content: "build fizzbuzz"},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "recent"},
	}
	res, err := c.MaybeCompact(context.Background(), compaction.Request{
		Archive:      msgs,
		SystemPrompt: "system",
		Model:        "test",
		Force:        true,
	})
	require.NoError(t, err)
	require.True(t, res.Compacted)
	require.NotNil(t, p.lastReq)
	require.Contains(t, p.lastReq.Messages[0].Content, "build fizzbuzz")
	require.Contains(t, p.lastReq.Messages[0].Content, "[User]: build fizzbuzz")
	require.Len(t, res.Projected, 2)
	require.True(t, strings.HasPrefix(res.Projected[0].Content, compaction.CompactedPrefix))
	require.Equal(t, "recent", res.Projected[1].Content)
	require.Len(t, res.Archive, 3)
	require.Len(t, res.Compactions, 1)
}

func TestMaybeCompact_OverBudget(t *testing.T) {
	p := &fakeProvider{text: "## Goal\nsummary"}
	c := &Compactor{
		provider: p,
		cfg: config.Config{
			CompactionReserveTokens:    1,
			CompactionKeepRecentTokens: 4,
			CompactionContextWindow:    10,
		},
	}

	msgs := []types.Message{
		{Role: "user", Content: strings.Repeat("x", 100)},
		{Role: "assistant", Content: "ok"},
		{Role: "user", Content: "tail"},
	}
	res, err := c.MaybeCompact(context.Background(), compaction.Request{
		Archive: msgs,
		Model:   "test",
	})
	require.NoError(t, err)
	require.True(t, res.Compacted)
}

func TestMaybeCompact_CustomInstructions(t *testing.T) {
	p := &fakeProvider{text: "## Goal\napi focus"}
	c := &Compactor{
		provider: p,
		cfg: config.Config{
			CompactionKeepRecentTokens: 4,
			CompactionContextWindow:    200000,
		},
	}
	msgs := []types.Message{
		{Role: "user", Content: "old"},
		{Role: "user", Content: "recent"},
	}
	_, err := c.MaybeCompact(context.Background(), compaction.Request{
		Archive:            msgs,
		Model:              "test",
		Force:              true,
		CustomInstructions: "focus on API changes",
	})
	require.NoError(t, err)
	require.Contains(t, p.lastReq.Messages[0].Content, "focus on API changes")
}
