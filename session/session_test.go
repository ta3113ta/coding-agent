package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLatestEmpty(t *testing.T) {
	require.Nil(t, Latest(nil))
}

func TestLatestReturnsMostRecent(t *testing.T) {
	now := time.Now().UTC()
	metas := []Meta{
		{ID: "old", UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "new", UpdatedAt: now},
		{ID: "mid", UpdatedAt: now.Add(-1 * time.Hour)},
	}
	got := Latest(metas)
	require.NotNil(t, got)
	require.Equal(t, "new", got.ID)
}

func TestSortByUpdatedDesc(t *testing.T) {
	now := time.Now().UTC()
	metas := []Meta{
		{ID: "a", UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "b", UpdatedAt: now},
	}
	sorted := SortByUpdatedDesc(metas)
	require.Equal(t, "b", sorted[0].ID)
	require.Equal(t, "a", sorted[1].ID)
}
