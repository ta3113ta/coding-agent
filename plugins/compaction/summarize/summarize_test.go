package summarize

import (
	"context"
	"strings"
	"testing"

	"coding-agent/compaction"
	"coding-agent/config"
	"coding-agent/types"
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
	if err != nil {
		t.Fatalf("MaybeCompact: %v", err)
	}
	if res.Compacted {
		t.Fatal("expected no compaction under budget")
	}
	if p.lastReq != nil {
		t.Fatal("provider should not be called under budget")
	}
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
	if err != nil {
		t.Fatalf("MaybeCompact: %v", err)
	}
	if !res.Compacted {
		t.Fatal("expected compaction when forced")
	}
	if p.lastReq == nil {
		t.Fatal("provider should be called")
	}
	if !strings.Contains(p.lastReq.Messages[0].Content, "build fizzbuzz") {
		t.Fatalf("prefix not in summarize request: %q", p.lastReq.Messages[0].Content)
	}
	if !strings.Contains(p.lastReq.Messages[0].Content, "[User]: build fizzbuzz") {
		t.Fatalf("expected serialized format: %q", p.lastReq.Messages[0].Content)
	}
	if len(res.Projected) != 2 {
		t.Fatalf("projected len = %d, want 2 (compacted + recent)", len(res.Projected))
	}
	if !strings.HasPrefix(res.Projected[0].Content, compaction.CompactedPrefix) {
		t.Fatalf("first message = %q, want compacted prefix", res.Projected[0].Content)
	}
	if res.Projected[1].Content != "recent" {
		t.Fatalf("kept message = %q, want recent", res.Projected[1].Content)
	}
	if len(res.Archive) != 3 {
		t.Fatalf("archive len = %d, want 3 unchanged", len(res.Archive))
	}
	if len(res.Compactions) != 1 {
		t.Fatalf("compactions = %d, want 1", len(res.Compactions))
	}
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
	if err != nil {
		t.Fatalf("MaybeCompact: %v", err)
	}
	if !res.Compacted {
		t.Fatal("expected compaction over budget")
	}
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
	if err != nil {
		t.Fatalf("MaybeCompact: %v", err)
	}
	if !strings.Contains(p.lastReq.Messages[0].Content, "focus on API changes") {
		t.Fatalf("custom instructions missing: %q", p.lastReq.Messages[0].Content)
	}
}
