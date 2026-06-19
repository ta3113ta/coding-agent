package picker

import (
	"context"
	"strings"
	"testing"
	"time"

	"coding-agent/session"
)

type fakeStore struct {
	metas []session.Meta
}

func (f *fakeStore) Create(ctx context.Context, provider, model string) (*session.Session, error) {
	return nil, nil
}

func (f *fakeStore) Get(ctx context.Context, id string) (*session.Session, error) {
	return nil, nil
}

func (f *fakeStore) Save(ctx context.Context, s *session.Session) error {
	return nil
}

func (f *fakeStore) List(ctx context.Context) ([]session.Meta, error) {
	return f.metas, nil
}

func TestSelectEmpty(t *testing.T) {
	store := &fakeStore{}
	var out strings.Builder
	id, err := Select(context.Background(), store, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if id != "" {
		t.Fatalf("id = %q, want empty", id)
	}
	if !strings.Contains(out.String(), "no previous sessions") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestSelectValid(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		metas: []session.Meta{
			{ID: "session-b", Name: "beta", UpdatedAt: now},
			{ID: "session-a", Name: "alpha", UpdatedAt: now.Add(-time.Hour)},
		},
	}
	var out strings.Builder
	id, err := Select(context.Background(), store, strings.NewReader("2\n"), &out)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if id != "session-a" {
		t.Fatalf("id = %q, want session-a", id)
	}
}

func TestSelectInvalid(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		metas: []session.Meta{{ID: "only", UpdatedAt: now}},
	}
	_, err := Select(context.Background(), store, strings.NewReader("9\n"), &strings.Builder{})
	if err == nil {
		t.Fatal("expected error for invalid selection")
	}
}

func TestSelectCancel(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		metas: []session.Meta{{ID: "only", UpdatedAt: now}},
	}
	_, err := Select(context.Background(), store, strings.NewReader("q\n"), &strings.Builder{})
	if err == nil {
		t.Fatal("expected cancel error")
	}
}
