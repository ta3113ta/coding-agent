package agent

import (
	"context"
	"fmt"

	"coding-agent/llm"
	"coding-agent/tools"
	"coding-agent/types"
)

type Agent struct {
	provider     llm.Provider
	registry     *tools.Registry
	messages     []types.Message
	model        string
	systemPrompt string
	verbose      bool
}

func New(provider llm.Provider, registry *tools.Registry, model, systemPrompt string, verbose bool) *Agent {
	return &Agent{
		provider:     provider,
		registry:     registry,
		model:        model,
		systemPrompt: systemPrompt,
		verbose:      verbose,
	}
}

// Run รับ input จากผู้ใช้ แล้ววน loop จน LLM หยุดเรียก tool
// คืน text สุดท้ายที่ LLM ตอบ
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	a.messages = append(a.messages, types.Message{
		Role:    "user",
		Content: userInput,
	})

	for {
		resp, err := a.provider.Complete(ctx, types.CompleteRequest{
			SystemPrompt: a.systemPrompt,
			Messages:     a.messages,
			Tools:        a.registry.Definitions(),
			Model:        a.model,
			MaxTokens:    8096,
		})
		if err != nil {
			return "", fmt.Errorf("llm call: %w", err)
		}

		a.messages = append(a.messages, types.Message{
			Role:      "assistant",
			Content:   resp.Text,
			ToolCalls: resp.ToolCalls,
		})

		if len(resp.ToolCalls) == 0 {
			return resp.Text, nil
		}

		if a.verbose && resp.Text != "" {
			fmt.Printf("\n💭 %s\n", resp.Text)
		}

		for _, tc := range resp.ToolCalls {
			if a.verbose {
				fmt.Printf("🔧 %s(%s)\n", tc.Name, string(tc.Input))
			}
			result, err := a.registry.Dispatch(tc.Name, tc.Input)
			isError := false
			if err != nil {
				result = fmt.Sprintf("error: %v", err)
				isError = true
			}
			a.messages = append(a.messages, types.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
				IsError:    isError,
			})
		}
	}
}
