package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"coding-agent/compaction"
	"coding-agent/permission"
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
	if len(s.Compactions) > 0 {
		cp.Compactions = make([]session.CompactionRecord, len(s.Compactions))
		copy(cp.Compactions, s.Compactions)
	}
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
	if len(s.Compactions) > 0 {
		cp.Compactions = make([]session.CompactionRecord, len(s.Compactions))
		copy(cp.Compactions, s.Compactions)
	}
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
	return newTestAgentWithPermission(t, provider, nil)
}

func newTestAgentWithPermission(t *testing.T, provider *fakeStreamProvider, perm *permission.Chain) (*Agent, *memStore) {
	return newTestAgentWithOptions(t, provider, perm, nil)
}

func newTestAgentWithOptions(t *testing.T, provider *fakeStreamProvider, perm *permission.Chain, compactor compaction.Compactor) (*Agent, *memStore) {
	t.Helper()
	store := newMemStore()
	ag, err := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", perm, compactor)
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
	ag, err := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", nil, nil)
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

type fakeToolProvider struct {
	calls int
}

func (f *fakeToolProvider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	f.calls++
	if f.calls == 1 {
		return &types.CompleteResponse{
			Text: "running tool",
			ToolCalls: []types.ToolCall{{
				ID:    "tc_1",
				Name:  "run_bash",
				Input: json.RawMessage(`{"command":"echo hi"}`),
			}},
		}, nil
	}
	return &types.CompleteResponse{Text: "done after deny"}, nil
}

type denyHook struct{}

func (denyHook) BeforeToolUse(ctx context.Context, req permission.ToolUseRequest) (permission.Result, error) {
	_ = ctx
	return permission.Result{Decision: permission.Deny, Message: "blocked for test"}, nil
}

func TestRun_PermissionDenied(t *testing.T) {
	provider := &fakeToolProvider{}
	perm := permission.NewChain(denyHook{})
	reg := tools.NewRegistry()
	executed := false
	reg.Register(&stubTool{onExecute: func() { executed = true }})

	store := newMemStore()
	ag, err := New(provider, reg, "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", perm, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := ag.InitNewSession(context.Background()); err != nil {
		t.Fatalf("InitNewSession: %v", err)
	}

	answer, err := ag.Run(context.Background(), "run echo", nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if answer != "done after deny" {
		t.Fatalf("answer = %q, want done after deny", answer)
	}
	if executed {
		t.Fatal("tool should not execute when permission denied")
	}

	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	var toolMsg *types.Message
	for i := range s.Messages {
		if s.Messages[i].Role == "tool" {
			toolMsg = &s.Messages[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatal("expected tool message in history")
	}
	if !toolMsg.IsError || !strings.Contains(toolMsg.Content, "blocked for test") {
		t.Fatalf("tool message = %+v, want permission error", toolMsg)
	}
}

type stubTool struct {
	onExecute func()
}

func (s *stubTool) Name() string { return "run_bash" }

func (s *stubTool) Definition() types.ToolDefinition {
	return types.ToolDefinition{Name: "run_bash"}
}

func (s *stubTool) Execute(input json.RawMessage) (string, error) {
	if s.onExecute != nil {
		s.onExecute()
	}
	return "executed", nil
}

type recordingCompactor struct {
	calls int
}

func (r *recordingCompactor) MaybeCompact(ctx context.Context, req compaction.Request) (compaction.Result, error) {
	_ = ctx
	r.calls++
	projected := compaction.ProjectMessages(req.Archive, req.Compactions)
	return compaction.Result{Archive: req.Archive, Compactions: req.Compactions, Projected: projected}, nil
}

func TestRun_InvokesCompactorBeforeComplete(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	comp := &recordingCompactor{}
	ag, _ := newTestAgentWithOptions(t, provider, nil, comp)

	if _, err := ag.Run(context.Background(), "hi", nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if comp.calls < 1 {
		t.Fatalf("compactor calls = %d, want at least 1", comp.calls)
	}
}

type forceCompactor struct{}

func (forceCompactor) MaybeCompact(ctx context.Context, req compaction.Request) (compaction.Result, error) {
	_ = ctx
	projected := compaction.ProjectMessages(req.Archive, req.Compactions)
	if len(req.Archive) <= 1 {
		return compaction.Result{Archive: req.Archive, Compactions: req.Compactions, Projected: projected}, nil
	}
	record := session.CompactionRecord{
		Summary:        "compact",
		FirstKeptIndex: len(req.Archive) - 1,
	}
	comps := append(append([]session.CompactionRecord(nil), req.Compactions...), record)
	return compaction.Result{
		Archive:     req.Archive,
		Compactions: comps,
		Projected:   []types.Message{{Role: "user", Content: "[compacted]"}},
		Compacted:   true,
	}, nil
}

func TestCompactSession(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	ag, store := newTestAgentWithOptions(t, provider, nil, forceCompactor{})

	ag.appendArchive(
		types.Message{Role: "user", Content: "old"},
		types.Message{Role: "assistant", Content: "reply"},
		types.Message{Role: "user", Content: "recent"},
	)
	if err := ag.CompactSession(context.Background(), ""); err != nil {
		t.Fatalf("CompactSession: %v", err)
	}
	if len(ag.messages) != 1 {
		t.Fatalf("projected len = %d, want 1 after compact", len(ag.messages))
	}
	if len(ag.archive) != 3 {
		t.Fatalf("archive len = %d, want 3 unchanged", len(ag.archive))
	}
	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(s.Messages) != 3 {
		t.Fatalf("persisted archive = %d, want 3", len(s.Messages))
	}
	if len(s.Compactions) != 1 {
		t.Fatalf("persisted compactions = %d, want 1", len(s.Compactions))
	}
}
