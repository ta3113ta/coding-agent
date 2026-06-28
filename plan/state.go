package plan

import (
	"errors"
	"sync"
	"time"
)

type SessionState struct {
	mu        sync.RWMutex
	mode      Mode
	todos     []TodoItem
	plan      *Plan
	sessionID string
}

func NewSessionState() *SessionState {
	return &SessionState{mode: ModeAgent}
}

func (s *SessionState) SetSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = id
}

func (s *SessionState) SessionID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionID
}

func (s *SessionState) Mode() Mode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

func (s *SessionState) SetMode(m Mode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = m
}

func (s *SessionState) Todos() []TodoItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]TodoItem(nil), s.todos...)
}

func (s *SessionState) SetTodos(todos []TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos = append([]TodoItem(nil), todos...)
}

func (s *SessionState) MergeTodos(incoming []TodoItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	byID := make(map[string]TodoItem, len(s.todos))
	for _, t := range s.todos {
		byID[t.ID] = t
	}
	for _, t := range incoming {
		byID[t.ID] = t
	}
	s.todos = make([]TodoItem, 0, len(byID))
	for _, t := range byID {
		s.todos = append(s.todos, t)
	}
}

func (s *SessionState) Plan() *Plan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.plan == nil {
		return nil
	}
	p := *s.plan
	return &p
}

func (s *SessionState) SetPlan(p *Plan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p == nil {
		s.plan = nil
		return
	}
	cp := *p
	s.plan = &cp
}

func (s *SessionState) ApprovePlan() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.plan == nil {
		return errors.New("no plan to approve")
	}
	if s.plan.Status != PlanStatusDraft {
		return errors.New("plan is not in draft status")
	}
	s.plan.Status = PlanStatusApproved
	return nil
}

// CanSwitchToAgent reports whether switching from plan to agent mode is allowed.
func (s *SessionState) CanSwitchToAgent() (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.mode != ModePlan {
		return true, ""
	}
	if s.plan == nil || s.plan.Status == PlanStatusApproved {
		return true, ""
	}
	return false, "draft plan exists; run /approve or /approve <instructions> to implement"
}

type Snapshot struct {
	Mode  string
	Todos []TodoItem
	Plan  *Plan
}

func (s *SessionState) LoadSnapshot(snap Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if snap.Mode != "" {
		s.mode = Mode(snap.Mode)
	} else {
		s.mode = ModeAgent
	}
	s.todos = append([]TodoItem(nil), snap.Todos...)
	if snap.Plan != nil {
		cp := *snap.Plan
		s.plan = &cp
	} else {
		s.plan = nil
	}
}

func (s *SessionState) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap := Snapshot{
		Mode:  string(s.mode),
		Todos: append([]TodoItem(nil), s.todos...),
	}
	if s.plan != nil {
		cp := *s.plan
		snap.Plan = &cp
	}
	return snap
}

func (s *SessionState) CreateDraftPlan(title, overview, body string) *Plan {
	p := &Plan{
		Title:     title,
		Overview:  overview,
		Body:      body,
		Status:    PlanStatusDraft,
		CreatedAt: time.Now().UTC(),
	}
	s.SetPlan(p)
	return p
}
