package permission

import "sync"

type SessionRules struct {
	mu           sync.Mutex
	allowedTools map[string]bool
	allowAll     bool
}

func NewSessionRules() *SessionRules {
	return &SessionRules{
		allowedTools: make(map[string]bool),
	}
}

func (s *SessionRules) Allows(tool string) bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.allowAll {
		return true
	}
	return s.allowedTools[tool]
}

func (s *SessionRules) AllowTool(tool string) {
	if s == nil || tool == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.allowedTools == nil {
		s.allowedTools = make(map[string]bool)
	}
	s.allowedTools[tool] = true
}

func (s *SessionRules) AllowAll() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowAll = true
}

func (s *SessionRules) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowAll = false
	s.allowedTools = make(map[string]bool)
}
