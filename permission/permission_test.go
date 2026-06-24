package permission

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.Equal(t, Allow, res.Decision)
}

func TestChain_NilAllows(t *testing.T) {
	var c *Chain
	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	require.NoError(t, err)
	require.Equal(t, Allow, res.Decision)
}

func TestChain_AllAllow(t *testing.T) {
	h1 := &stubHook{decision: Allow}
	h2 := &stubHook{decision: Allow}
	c := NewChain(h1, h2)

	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "write_file"})
	require.NoError(t, err)
	require.Equal(t, Allow, res.Decision)
	require.Equal(t, 1, h1.called)
	require.Equal(t, 1, h2.called)
}

func TestChain_DenyShortCircuit(t *testing.T) {
	h1 := &stubHook{decision: Deny, message: "blocked"}
	h2 := &stubHook{decision: Allow}
	c := NewChain(h1, h2)

	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	require.NoError(t, err)
	require.Equal(t, Deny, res.Decision)
	require.Equal(t, "blocked", res.Message)
	require.Equal(t, 0, h2.called)
}

func TestChain_AskContinuesChain(t *testing.T) {
	h1 := &stubHook{decision: Ask, message: "confirm?"}
	h2 := &stubHook{decision: Allow}
	c := NewChain(h1, h2)

	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	require.NoError(t, err)
	require.Equal(t, Allow, res.Decision)
	require.Equal(t, 1, h1.called)
	require.Equal(t, 1, h2.called)
}

func TestChain_HookErrorFailsClosed(t *testing.T) {
	h := &stubHook{err: context.Canceled}
	c := NewChain(h)

	res, err := c.Evaluate(context.Background(), ToolUseRequest{ToolName: "run_bash"})
	require.NoError(t, err)
	require.Equal(t, Deny, res.Decision)
}

func TestChain_UpdatedInput(t *testing.T) {
	newInput := json.RawMessage(`{"command":"echo safe"}`)
	c := NewChain(&inputRewriteHook{newInput: newInput})

	res, err := c.Evaluate(context.Background(), ToolUseRequest{
		ToolName: "run_bash",
		Input:    json.RawMessage(`{"command":"rm -rf /"}`),
	})
	require.NoError(t, err)
	require.Equal(t, Allow, res.Decision)
	require.Equal(t, string(newInput), string(res.UpdatedInput))
}
