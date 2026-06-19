package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"coding-agent/session"
	"coding-agent/tools"
	"coding-agent/types"
)

type memStore struct {
	mu       sync.Mutex
	sessions map[string]*session.Session
	nextID   int
}

func newMemStore() *memStore {
	return &memStore{sessions: make(map[string]*session.Session)}
}

func (m *memStore) Create(ctx context.Context, provider, model string) (*session.Session, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	id := fmt.Sprintf("session-%d", m.nextID)
	now := time.Now().UTC()
	s := &session.Session{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		Provider:  provider,
		Model:     model,
	}
	m.sessions[s.ID] = s
	return s, nil
}

func (m *memStore) Get(ctx context.Context, id string) (*session.Session, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %q not found", id)
	}
	cp := *s
	cp.Messages = append([]types.Message(nil), s.Messages...)
	return &cp, nil
}

func (m *memStore) Save(ctx context.Context, s *session.Session) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.sessions[s.ID]; ok && !existing.CreatedAt.IsZero() {
		s.CreatedAt = existing.CreatedAt
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	s.UpdatedAt = time.Now().UTC()
	cp := *s
	cp.Messages = append([]types.Message(nil), s.Messages...)
	m.sessions[s.ID] = &cp
	return nil
}

func (m *memStore) List(ctx context.Context) ([]session.Meta, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	var metas []session.Meta
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

func newTestAgent(t *testing.T, provider *fakeStreamProvider) (*Agent, *memStore) {
	t.Helper()
	store := newMemStore()
	ag, err := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := ag.InitNewSession(context.Background()); err != nil {
		t.Fatalf("InitNewSession: %v", err)
	}
	return ag, store
}

func TestRun_StreamCallback(t *testing.T) {
	provider := &fakeStreamProvider{
		deltas: []string{"hel", "lo"},
		text:   "hello",
	}
	ag, _ := newTestAgent(t, provider)

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
	ag, _ := newTestAgent(t, provider)

	answer, err := ag.Run(context.Background(), "hi", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if answer != "done" {
		t.Fatalf("answer = %q, want done", answer)
	}
}

func TestRun_AutoSave(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	ag, store := newTestAgent(t, provider)

	if _, err := ag.Run(context.Background(), "hi", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(s.Messages) != 2 {
		t.Fatalf("saved messages = %d, want 2", len(s.Messages))
	}
}

func TestResumeSession(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	ag, store := newTestAgent(t, provider)

	store.sessions["other"] = &session.Session{
		ID:   "other",
		Name: "prev task",
		Messages: []types.Message{
			{Role: "user", Content: "prev"},
			{Role: "assistant", Content: "answer"},
		},
	}

	if err := ag.ResumeSession(context.Background(), "other"); err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	if ag.CurrentSessionID() != "other" {
		t.Fatalf("session id = %q, want other", ag.CurrentSessionID())
	}
	if ag.CurrentSessionName() != "prev task" {
		t.Fatalf("session name = %q, want prev task", ag.CurrentSessionName())
	}
	if len(ag.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(ag.messages))
	}
}

func TestResetSession(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	ag, store := newTestAgent(t, provider)

	if _, err := ag.Run(context.Background(), "hi", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	oldID := ag.CurrentSessionID()
	if err := ag.ResetSession(context.Background()); err != nil {
		t.Fatalf("ResetSession: %v", err)
	}
	if ag.CurrentSessionID() == oldID {
		t.Fatal("expected new session id after reset")
	}
	if len(ag.messages) != 0 {
		t.Fatalf("messages = %d, want 0", len(ag.messages))
	}
	if _, ok := store.sessions[ag.CurrentSessionID()]; !ok {
		t.Fatal("new session not in store")
	}
}

func TestSetSessionName(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	ag, store := newTestAgent(t, provider)

	if err := ag.SetSessionName(context.Background(), "my task"); err != nil {
		t.Fatalf("SetSessionName: %v", err)
	}
	if ag.SessionLabel() != "my task ("+ag.CurrentSessionID()+")" {
		t.Fatalf("label = %q", ag.SessionLabel())
	}
	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.Name != "my task" {
		t.Fatalf("stored name = %q", s.Name)
	}
}

func TestInitNewSession(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	store := newMemStore()
	ag, err := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if ag.CurrentSessionID() != "" {
		t.Fatal("expected no session before InitNewSession")
	}
	if err := ag.InitNewSession(context.Background()); err != nil {
		t.Fatalf("InitNewSession: %v", err)
	}
	if ag.CurrentSessionID() == "" {
		t.Fatal("expected session id after InitNewSession")
	}
}
