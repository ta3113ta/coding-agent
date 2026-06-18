package agent

import (
	"context"
	"testing"

	"coding-agent/tools"
	"coding-agent/types"
)

type fakeStreamProvider struct {
	deltas []string
	text   string
}

func (f *fakeStreamProvider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	for _, d := range f.deltas {
		if req.OnStream != nil {
			req.OnStream(types.StreamEvent{TextDelta: d})
		}
	}
	return &types.CompleteResponse{Text: f.text}, nil
}

func TestRun_StreamCallback(t *testing.T) {
	provider := &fakeStreamProvider{
		deltas: []string{"hel", "lo"},
		text:   "hello",
	}
	ag := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false)

	var got []string
	answer, err := ag.Run(context.Background(), "hi", func(ev types.StreamEvent) {
		got = append(got, ev.TextDelta)
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if answer != "hello" {
		t.Fatalf("answer = %q, want hello", answer)
	}
	if len(got) != 2 || got[0] != "hel" || got[1] != "lo" {
		t.Fatalf("stream deltas = %v, want [hel lo]", got)
	}
}

func TestRun_NilStreamCallback(t *testing.T) {
	provider := &fakeStreamProvider{
		deltas: []string{"skip"},
		text:   "done",
	}
	ag := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false)

	answer, err := ag.Run(context.Background(), "hi", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if answer != "done" {
		t.Fatalf("answer = %q, want done", answer)
	}
}
