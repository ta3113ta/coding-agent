package session

import (
	"context"
	"sort"
	"time"

	"coding-agent/types"
)

type Session struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Provider  string
	Model     string
	Name      string
	Messages  []types.Message
}

type Meta struct {
	ID           string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Provider     string
	Model        string
	Name         string
	MessageCount int
}

type Store interface {
	Create(ctx context.Context, provider, model string) (*Session, error)
	Get(ctx context.Context, id string) (*Session, error)
	Save(ctx context.Context, s *Session) error
	List(ctx context.Context) ([]Meta, error)
}

func Latest(metas []Meta) *Meta {
	if len(metas) == 0 {
		return nil
	}
	sorted := append([]Meta(nil), metas...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt)
	})
	return &sorted[0]
}

func SortByUpdatedDesc(metas []Meta) []Meta {
	sorted := append([]Meta(nil), metas...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt)
	})
	return sorted
}
