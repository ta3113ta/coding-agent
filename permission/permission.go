package permission

import (
	"context"
	"encoding/json"
	"fmt"
)

type Decision int

const (
	Allow Decision = iota
	Deny
	Ask
)

type ToolUseRequest struct {
	ToolName   string
	Input      json.RawMessage
	ToolCallID string
	AskHint    string // set when a prior hook returned Ask with a message
}

type Result struct {
	Decision     Decision
	Message      string
	UpdatedInput json.RawMessage
}

type Hook interface {
	BeforeToolUse(ctx context.Context, req ToolUseRequest) (Result, error)
}

type Chain struct {
	hooks []Hook
}

func NewChain(hooks ...Hook) *Chain {
	return &Chain{hooks: hooks}
}

func (c *Chain) Register(h Hook) {
	if h == nil {
		return
	}
	c.hooks = append(c.hooks, h)
}

func (c *Chain) Empty() bool {
	return c == nil || len(c.hooks) == 0
}

func (c *Chain) Evaluate(ctx context.Context, req ToolUseRequest) (Result, error) {
	if c == nil || len(c.hooks) == 0 {
		return Result{Decision: Allow}, nil
	}

	var updatedInput json.RawMessage
	var askHint string

	for _, hook := range c.hooks {
		req.AskHint = askHint

		res, err := hook.BeforeToolUse(ctx, req)
		if err != nil {
			return Result{
				Decision: Deny,
				Message:  fmt.Sprintf("permission hook error: %v", err),
			}, nil
		}

		if res.UpdatedInput != nil {
			updatedInput = res.UpdatedInput
			req.Input = res.UpdatedInput
		}

		switch res.Decision {
		case Deny:
			if res.Message == "" {
				res.Message = "permission denied"
			}
			return res, nil
		case Ask:
			if res.Message != "" {
				askHint = res.Message
			}
			continue
		case Allow:
			continue
		default:
			return Result{
				Decision: Deny,
				Message:  fmt.Sprintf("permission hook returned unknown decision: %d", res.Decision),
			}, nil
		}
	}

	out := Result{Decision: Allow}
	if updatedInput != nil {
		out.UpdatedInput = updatedInput
	}

	return out, nil
}
