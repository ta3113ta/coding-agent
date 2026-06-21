package permission

import (
	"context"
	"encoding/json"
	"testing"
)

type stubHook struct {
	decision Decision
	message  string
	err      error
	called   int
}

func (s *stubHook) BeforeToolUse(ctx context.Context, req ToolUseRequest) (Result, error) {
	s.called++
	if s.err != nil {
		return Result{}, s.err
	}
	return Result{Decision: s.decision, Message: s.message}, nil
}

type inputRewriteHook struct {
	newInput json.RawMessage
}

func (h *inputRewriteHook) BeforeToolUse(ctx context.Context, req ToolUseRequest) (Result, error) {
	return Result{Decision: Allow, UpdatedInput: h.newInput}, nil
}

func TestChain_EmptyAllows(t *testing.T) {
	c := NewChain()
	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Decision != Allow {
		t.Fatalf("decision = %v, want Allow", res.Decision)
	}
}

func TestChain_NilAllows(t *testing.T) {
	var c *Chain
	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Decision != Allow {
		t.Fatalf("decision = %v, want Allow", res.Decision)
	}
}

func TestChain_AllAllow(t *testing.T) {
	h1 := &stubHook{decision: Allow}
	h2 := &stubHook{decision: Allow}
	c := NewChain(h1, h2)

	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "write_file"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Decision != Allow {
		t.Fatalf("decision = %v, want Allow", res.Decision)
	}
	if h1.called != 1 || h2.called != 1 {
		t.Fatalf("calls = %d, %d, want 1, 1", h1.called, h2.called)
	}
}

func TestChain_DenyShortCircuit(t *testing.T) {
	h1 := &stubHook{decision: Deny, message: "blocked"}
	h2 := &stubHook{decision: Allow}
	c := NewChain(h1, h2)

	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Decision != Deny || res.Message != "blocked" {
		t.Fatalf("result = %+v, want Deny blocked", res)
	}
	if h2.called != 0 {
		t.Fatal("second hook should not run after deny")
	}
}

func TestChain_AskContinuesChain(t *testing.T) {
	h1 := &stubHook{decision: Ask, message: "confirm?"}
	h2 := &stubHook{decision: Allow}
	c := NewChain(h1, h2)

	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Decision != Allow {
		t.Fatalf("result = %+v, want Allow after Ask passes through", res)
	}
	if h1.called != 1 || h2.called != 1 {
		t.Fatalf("calls = %d, %d, want both hooks to run", h1.called, h2.called)
	}
}

func TestChain_HookErrorFailsClosed(t *testing.T) {
	h := &stubHook{err: context.Canceled}
	c := NewChain(h)

	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Decision != Deny {
		t.Fatalf("decision = %v, want Deny on hook error", res.Decision)
	}
}

func TestChain_UpdatedInput(t *testing.T) {
	newInput := json.RawMessage(`{"command":"echo safe"}`)
	c := NewChain(&inputRewriteHook{newInput: newInput})

	res, err := c.Evaluate(context.Background(), ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"rm -rf /"}`),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if res.Decision != Allow {
		t.Fatalf("decision = %v, want Allow", res.Decision)
	}
	if string(res.UpdatedInput) != string(newInput) {
		t.Fatalf("UpdatedInput = %s, want %s", res.UpdatedInput, newInput)
	}
}
