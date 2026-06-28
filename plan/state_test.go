package plan

import "testing"

func TestSessionStateTodoReplace(t *testing.T) {
	s := NewSessionState()
	s.SetTodos([]TodoItem{
		{ID: "a", Content: "first", Status: TodoPending},
		{ID: "b", Content: "second", Status: TodoPending},
	})
	todos := s.Todos()
	if len(todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(todos))
	}
}

func TestSessionStateTodoMerge(t *testing.T) {
	s := NewSessionState()
	s.SetTodos([]TodoItem{
		{ID: "a", Content: "first", Status: TodoPending},
		{ID: "b", Content: "second", Status: TodoInProgress},
	})
	s.MergeTodos([]TodoItem{
		{ID: "b", Content: "second updated", Status: TodoCompleted},
		{ID: "c", Content: "third", Status: TodoPending},
	})
	todos := s.Todos()
	if len(todos) != 3 {
		t.Fatalf("expected 3 todos after merge, got %d", len(todos))
	}
	byID := make(map[string]TodoItem)
	for _, todo := range todos {
		byID[todo.ID] = todo
	}
	if byID["b"].Status != TodoCompleted {
		t.Fatalf("expected b to be completed, got %s", byID["b"].Status)
	}
	if byID["b"].Content != "second updated" {
		t.Fatalf("expected b content updated")
	}
}

func TestCanSwitchToAgent(t *testing.T) {
	s := NewSessionState()

	ok, msg := s.CanSwitchToAgent()
	if !ok || msg != "" {
		t.Fatalf("agent mode should always allow switch: ok=%v msg=%q", ok, msg)
	}

	s.SetMode(ModePlan)
	ok, msg = s.CanSwitchToAgent()
	if !ok || msg != "" {
		t.Fatalf("plan mode without plan should allow switch: ok=%v msg=%q", ok, msg)
	}

	s.CreateDraftPlan("t", "o", "body")
	ok, msg = s.CanSwitchToAgent()
	if ok || msg == "" {
		t.Fatalf("draft plan should block switch")
	}

	if err := s.ApprovePlan(); err != nil {
		t.Fatalf("approve: %v", err)
	}
	ok, msg = s.CanSwitchToAgent()
	if !ok || msg != "" {
		t.Fatalf("approved plan should allow switch: ok=%v msg=%q", ok, msg)
	}
}

func TestLoadSnapshotRoundTrip(t *testing.T) {
	s := NewSessionState()
	s.SetMode(ModePlan)
	s.SetTodos([]TodoItem{{ID: "x", Content: "task", Status: TodoPending}})
	s.CreateDraftPlan("title", "overview", "body")

	s2 := NewSessionState()
	s2.LoadSnapshot(s.Snapshot())

	if s2.Mode() != ModePlan {
		t.Fatalf("expected plan mode, got %s", s2.Mode())
	}
	if len(s2.Todos()) != 1 {
		t.Fatalf("expected 1 todo")
	}
	if p := s2.Plan(); p == nil || p.Title != "title" {
		t.Fatalf("expected plan title title, got %v", p)
	}
}
