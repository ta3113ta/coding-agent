package memory

import (
	"context"
	"testing"

	"coding-agent/types"

	"github.com/stretchr/testify/require"
)

func TestStoreRoundTrip(t *testing.T) {
	store := New()
	ctx := context.Background()

	s, err := store.Create(ctx, "anthropic", "model")
	require.NoError(t, err)
	s.Messages = []types.Message{{Role: "user", Content: "hi"}}
	s.Name = "test"
	require.NoError(t, store.Save(ctx, s))

	got, err := store.Get(ctx, s.ID)
	require.NoError(t, err)
	require.Equal(t, "test", got.Name)
	require.Len(t, got.Messages, 1)

	metas, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, metas, 1)
	require.Equal(t, "test", metas[0].Name)
}

func TestStoreGetMissing(t *testing.T) {
	store := New()
	_, err := store.Get(context.Background(), "missing")
	require.Error(t, err)
}
