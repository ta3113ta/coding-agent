package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"coding-agent/compaction"
	"coding-agent/permission"
	"coding-agent/plan"
	"coding-agent/session"
	"coding-agent/tools"
	"coding-agent/types"

	"github.com/stretchr/testify/require"
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
	ag, err := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", perm, compactor, plan.NewSessionState(), false, true)
	require.NoError(t, err)
	require.NoError(t, ag.InitNewSession(context.Background()))
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
	require.NoError(t, err)
	require.Equal(t, "hello", answer)
	require.Equal(t, []string{"hel", "lo"}, got)
}

func TestRun_NilStreamCallback(t *testing.T) {
	provider := &fakeStreamProvider{
		deltas: []string{"skip"},
		text:   "done",
	}
	ag, _ := newTestAgent(t, provider)

	answer, err := ag.Run(context.Background(), "hi", nil)
	require.NoError(t, err)
	require.Equal(t, "done", answer)
}

func TestRun_AutoSave(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	ag, store := newTestAgent(t, provider)

	_, err := ag.Run(context.Background(), "hi", nil)
	require.NoError(t, err)
	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	require.NoError(t, err)
	require.Len(t, s.Messages, 2)
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

	require.NoError(t, ag.ResumeSession(context.Background(), "other"))
	require.Equal(t, "other", ag.CurrentSessionID())
	require.Equal(t, "prev task", ag.CurrentSessionName())
	require.Len(t, ag.messages, 2)
}

func TestResetSession(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	ag, store := newTestAgent(t, provider)

	_, err := ag.Run(context.Background(), "hi", nil)
	require.NoError(t, err)
	oldID := ag.CurrentSessionID()
	require.NoError(t, ag.ResetSession(context.Background()))
	require.NotEqual(t, oldID, ag.CurrentSessionID())
	require.Empty(t, ag.messages)
	require.Contains(t, store.sessions, ag.CurrentSessionID())
}

func TestSetSessionName(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	ag, store := newTestAgent(t, provider)

	require.NoError(t, ag.SetSessionName(context.Background(), "my task"))
	require.Equal(t, "my task ("+ag.CurrentSessionID()+")", ag.SessionLabel())
	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	require.NoError(t, err)
	require.Equal(t, "my task", s.Name)
}

func TestInitNewSession(t *testing.T) {
	provider := &fakeStreamProvider{text: "ok"}
	store := newMemStore()
	ag, err := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", nil, nil, plan.NewSessionState(), false, true)
	require.NoError(t, err)
	require.Empty(t, ag.CurrentSessionID())
	require.NoError(t, ag.InitNewSession(context.Background()))
	require.NotEmpty(t, ag.CurrentSessionID())
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
	ag, err := New(provider, reg, "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", perm, nil, plan.NewSessionState(), false, true)
	require.NoError(t, err)
	require.NoError(t, ag.InitNewSession(context.Background()))

	answer, err := ag.Run(context.Background(), "run echo", nil)
	require.NoError(t, err)
	require.Equal(t, "done after deny", answer)
	require.False(t, executed)

	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	require.NoError(t, err)
	var toolMsg *types.Message
	for i := range s.Messages {
		if s.Messages[i].Role == "tool" {
			toolMsg = &s.Messages[i]
			break
		}
	}
	require.NotNil(t, toolMsg)
	require.True(t, toolMsg.IsError)
	require.Contains(t, toolMsg.Content, "blocked for test")
}

type stubTool struct {
	onExecute func()
}

func (s *stubTool) Name() string { return "run_bash" }

func (s *stubTool) Definition() types.ToolDefinition {
	return types.ToolDefinition{Name: "run_bash"}
}

func (s *stubTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
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

	_, err := ag.Run(context.Background(), "hi", nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, comp.calls, 1)
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
	require.NoError(t, ag.CompactSession(context.Background(), ""))
	require.Len(t, ag.messages, 1)
	require.Len(t, ag.archive, 3)
	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	require.NoError(t, err)
	require.Len(t, s.Messages, 3)
	require.Len(t, s.Compactions, 1)
}

type loopingProvider struct {
	calls int
}

func (p *loopingProvider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	_ = ctx
	p.calls++
	return &types.CompleteResponse{
		Text: "thinking",
		ToolCalls: []types.ToolCall{{
			ID:    "tc_1",
			Name:  "read_file",
			Input: json.RawMessage(`{"path":"x"}`),
		}},
	}, nil
}

func TestRunSubtask_MaxTurns(t *testing.T) {
	provider := &loopingProvider{}
	reg := tools.NewRegistry(&stubReadTool{})
	ag, _ := newTestAgentWithRegistry(t, provider, reg)

	text, err := ag.RunSubtask(context.Background(), "explore", 2)
	require.NoError(t, err)
	require.Contains(t, text, "max turns reached")
	require.Equal(t, 2, provider.calls)
}

func TestRunSubtask_IsolatedFromParentArchive(t *testing.T) {
	provider := &fakeStreamProvider{text: "sub done"}
	ag, store := newTestAgent(t, provider)

	ag.appendArchive(types.Message{Role: "user", Content: "parent only"})
	parentID := ag.CurrentSessionID()
	parentLen := len(ag.archive)

	childStore := newMemStore()
	child, err := New(provider, tools.NewRegistry(), "test-model", "system", types.PromptCacheConfig{}, false, childStore, "anthropic", nil, nil, plan.NewSessionState(), false, true)
	require.NoError(t, err)
	require.NoError(t, child.InitNewSession(context.Background()))
	_, err = child.RunSubtask(context.Background(), "child task", 0)
	require.NoError(t, err)

	require.Equal(t, parentLen, len(ag.archive))
	s, err := store.Get(context.Background(), parentID)
	require.NoError(t, err)
	require.Empty(t, s.Messages)
	childS, err := childStore.Get(context.Background(), child.CurrentSessionID())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(childS.Messages), 2)
}

type stubReadTool struct{}

func (stubReadTool) Name() string { return "read_file" }

func (stubReadTool) Definition() types.ToolDefinition {
	return types.ToolDefinition{Name: "read_file"}
}

func (stubReadTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	_ = input
	return "file contents", nil
}

func newTestAgentWithRegistry(t *testing.T, provider *loopingProvider, reg *tools.Registry) (*Agent, *memStore) {
	t.Helper()
	store := newMemStore()
	ag, err := New(provider, reg, "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", nil, nil, plan.NewSessionState(), false, true)
	require.NoError(t, err)
	require.NoError(t, ag.InitNewSession(context.Background()))
	return ag, store
}

type parallelDualToolProvider struct {
	calls int
}

func (p *parallelDualToolProvider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	_ = ctx
	_ = req
	p.calls++
	if p.calls == 1 {
		return &types.CompleteResponse{
			ToolCalls: []types.ToolCall{
				{ID: "tc_slow", Name: "tool_slow", Input: json.RawMessage(`{}`)},
				{ID: "tc_fast", Name: "tool_fast", Input: json.RawMessage(`{}`)},
			},
		}, nil
	}
	return &types.CompleteResponse{Text: "done"}, nil
}

type timedStubTool struct {
	name  string
	delay time.Duration
	order *[]string
	mu    *sync.Mutex
}

func (t timedStubTool) Name() string { return t.name }

func (t timedStubTool) Definition() types.ToolDefinition {
	return types.ToolDefinition{Name: t.name}
}

func (t timedStubTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	_ = input
	time.Sleep(t.delay)
	t.mu.Lock()
	*t.order = append(*t.order, t.name)
	t.mu.Unlock()
	return t.name + " ok", nil
}

func TestRun_ParallelToolCalls(t *testing.T) {
	provider := &parallelDualToolProvider{}
	var finishOrder []string
	var orderMu sync.Mutex

	reg := tools.NewRegistry(
		timedStubTool{name: "tool_slow", delay: 50 * time.Millisecond, order: &finishOrder, mu: &orderMu},
		timedStubTool{name: "tool_fast", delay: 10 * time.Millisecond, order: &finishOrder, mu: &orderMu},
	)

	store := newMemStore()
	ag, err := New(provider, reg, "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", nil, nil, plan.NewSessionState(), false, true)
	require.NoError(t, err)
	require.NoError(t, ag.InitNewSession(context.Background()))

	answer, err := ag.Run(context.Background(), "run both", nil)
	require.NoError(t, err)
	require.Equal(t, "done", answer)
	require.Equal(t, []string{"tool_fast", "tool_slow"}, finishOrder)

	s, err := store.Get(context.Background(), ag.CurrentSessionID())
	require.NoError(t, err)

	var toolMsgs []types.Message
	for _, m := range s.Messages {
		if m.Role == "tool" {
			toolMsgs = append(toolMsgs, m)
		}
	}
	require.Len(t, toolMsgs, 2)
	require.Equal(t, "tc_slow", toolMsgs[0].ToolCallID)
	require.Equal(t, "tool_slow ok", toolMsgs[0].Content)
	require.Equal(t, "tc_fast", toolMsgs[1].ToolCallID)
	require.Equal(t, "tool_fast ok", toolMsgs[1].Content)
}

func TestRun_ParallelToolCalls_Disabled(t *testing.T) {
	provider := &parallelDualToolProvider{}
	var finishOrder []string
	var orderMu sync.Mutex

	reg := tools.NewRegistry(
		timedStubTool{name: "tool_slow", delay: 50 * time.Millisecond, order: &finishOrder, mu: &orderMu},
		timedStubTool{name: "tool_fast", delay: 10 * time.Millisecond, order: &finishOrder, mu: &orderMu},
	)

	store := newMemStore()
	ag, err := New(provider, reg, "test-model", "system", types.PromptCacheConfig{}, false, store, "anthropic", nil, nil, plan.NewSessionState(), false, false)
	require.NoError(t, err)
	require.NoError(t, ag.InitNewSession(context.Background()))

	_, err = ag.Run(context.Background(), "run both", nil)
	require.NoError(t, err)
	require.Equal(t, []string{"tool_slow", "tool_fast"}, finishOrder)
}
