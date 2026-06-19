package memory

import (
	"context"
	"testing"

	"coding-agent/types"
)

func TestStoreRoundTrip(t *testing.T) {
	store := New()
	ctx := context.Background()

	s, err := store.Create(ctx, "anthropic", "model")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	s.Messages = []types.Message{{Role: "user", Content: "hi"}}
	s.Name = "test"
	if err := store.Save(ctx, s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test" || len(got.Messages) != 1 {
		t.Fatalf("got = %+v", got)
	}

	metas, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "test" {
		t.Fatalf("metas = %+v", metas)
	}
}

func TestStoreGetMissing(t *testing.T) {
	store := New()
	_, err := store.Get(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
}
