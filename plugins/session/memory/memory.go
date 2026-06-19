package memory

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"coding-agent/plugin"
	"coding-agent/session"
	"coding-agent/types"
)

type Store struct {
	mu       sync.Mutex
	sessions map[string]*session.Session
}

func New() *Store {
	return &Store{sessions: make(map[string]*session.Session)}
}

func (m *Store) Create(ctx context.Context, provider, model string) (*session.Session, error) {
	_ = ctx
	id, err := newUUID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	s := &session.Session{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Provider:  provider,
		Model:     model,
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	return s, nil
}

func (m *Store) Get(ctx context.Context, id string) (*session.Session, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	return cloneSession(s), nil
}

func (m *Store) Save(ctx context.Context, s *session.Session) error {
	_ = ctx
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[s.ID]; !ok {
		return fmt.Errorf("session %q not found", s.ID)
	}
	cp := cloneSession(s)
	cp.UpdatedAt = time.Now().UTC()
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = cp.UpdatedAt
	}
	m.sessions[s.ID] = cp
	return nil
}

func (m *Store) List(ctx context.Context) ([]session.Meta, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	metas := make([]session.Meta, 0, len(m.sessions))
	for _, s := range m.sessions {
		metas = append(metas, session.Meta{
			ID:           s.ID,
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
			Provider:     s.Provider,
			Model:        s.Model,
			Name:         s.Name,
			MessageCount: len(s.Messages),
		})
	}
	return metas, nil
}

func cloneSession(s *session.Session) *session.Session {
	cp := *s
	cp.Messages = append([]types.Message(nil), s.Messages...)
	return &cp
}

func newUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

type Plugin struct{}

func (Plugin) Name() string { return "session/memory" }

func (Plugin) Register(app *plugin.App) error {
	_ = app
	return nil
}

var _ session.Store = (*Store)(nil)
