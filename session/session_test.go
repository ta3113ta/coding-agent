package session

import (
	"testing"
	"time"
)

func TestLatestEmpty(t *testing.T) {
	if Latest(nil) != nil {
		t.Fatal("expected nil for empty list")
	}
}

func TestLatestReturnsMostRecent(t *testing.T) {
	now := time.Now().UTC()
	metas := []Meta{
		{ID: "old", UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "new", UpdatedAt: now},
		{ID: "mid", UpdatedAt: now.Add(-1 * time.Hour)},
	}
	got := Latest(metas)
	if got == nil || got.ID != "new" {
		t.Fatalf("Latest() = %+v, want new", got)
	}
}

func TestSortByUpdatedDesc(t *testing.T) {
	now := time.Now().UTC()
	metas := []Meta{
		{ID: "a", UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "b", UpdatedAt: now},
	}
	sorted := SortByUpdatedDesc(metas)
	if sorted[0].ID != "b" || sorted[1].ID != "a" {
		t.Fatalf("sort order = %v, want [b a]", []string{sorted[0].ID, sorted[1].ID})
	}
}
