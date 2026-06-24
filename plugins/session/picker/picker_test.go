package picker

import (
	"context"
	"strings"
	"testing"
	"time"

	"coding-agent/session"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.Empty(t, id)
	require.Contains(t, out.String(), "no previous sessions")
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
	require.NoError(t, err)
	require.Equal(t, "session-a", id)
}

func TestSelectInvalid(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		metas: []session.Meta{{ID: "only", UpdatedAt: now}},
	}
	_, err := Select(context.Background(), store, strings.NewReader("9\n"), &strings.Builder{})
	require.Error(t, err)
}

func TestSelectCancel(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{
		metas: []session.Meta{{ID: "only", UpdatedAt: now}},
	}
	_, err := Select(context.Background(), store, strings.NewReader("q\n"), &strings.Builder{})
	require.Error(t, err)
}
