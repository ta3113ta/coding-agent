package openrouter

import (
	"context"
	"encoding/json"
	"fmt"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/models/operations"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"

	"coding-agent/config"
	"coding-agent/llm"
	"coding-agent/plugin"
	"coding-agent/types"
)

type provider struct {
	client *openrouter.OpenRouter
}

func newProvider(cfg config.Config) (llm.Provider, error) {
	return &provider{
		client: openrouter.New(openrouter.WithSecurity(cfg.OpenRouterAPIKey)),
	}, nil
}

func (p *provider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8096
	}

	messages := []components.ChatMessages{
		components.CreateChatMessagesSystem(components.ChatSystemMessage{
			Content: components.CreateChatSystemMessageContentStr(req.SystemPrompt),
			Role:    components.ChatSystemMessageRoleSystem,
		}),
	}
	messages = append(messages, toOpenRouterMessages(req.Messages)...)

	res, err := p.client.Chat.Send(ctx, components.ChatRequest{
		Model:     openrouter.Pointer(req.Model),
		MaxTokens: optionalnullable.From(openrouter.Pointer(int64(maxTokens))),
		Messages:  messages,
		Tools:     toOpenRouterTools(req.Tools),
		Stream:    openrouter.Pointer(false),
	})
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}

	if res.Type != operations.SendChatCompletionRequestResponseTypeChatResult || res.ChatResult == nil {
		return nil, fmt.Errorf("openrouter: unexpected response type")
	}

	choices := res.ChatResult.GetChoices()
	if len(choices) == 0 {
		return nil, fmt.Errorf("openrouter: no choices in response")
	}

	msg := choices[0].GetMessage()
	out := &types.CompleteResponse{}

	content, ok := msg.GetContent().GetOrZero()
	if ok && content.Type == components.ChatAssistantMessageContentTypeStr && content.Str != nil {
		out.Text = *content.Str
	}

	for _, tc := range msg.GetToolCalls() {
		fn := tc.GetFunction()
		out.ToolCalls = append(out.ToolCalls, types.ToolCall{
			ID:    tc.GetID(),
			Name:  fn.Name,
			Input: json.RawMessage(fn.Arguments),
		})
	}
	return out, nil
}

func toOpenRouterTools(tools []types.ToolDefinition) []components.ChatFunctionTool {
	out := make([]components.ChatFunctionTool, 0, len(tools))
	for _, t := range tools {
		params := map[string]any{
			"type":       "object",
			"properties": t.Properties,
		}
		if len(t.Required) > 0 {
			params["required"] = t.Required
		}
		out = append(out, components.CreateChatFunctionToolChatFunctionToolFunction(components.ChatFunctionToolFunction{
			Type: components.ChatFunctionToolTypeFunction,
			Function: components.ChatFunctionToolFunctionFunction{
				Name:        t.Name,
				Description: openrouter.String(t.Description),
				Parameters:  params,
			},
		}))
	}
	return out
}

func toOpenRouterMessages(msgs []types.Message) []components.ChatMessages {
	out := make([]components.ChatMessages, 0, len(msgs))
	for _, msg := range msgs {
		switch msg.Role {
		case "user":
			out = append(out, components.CreateChatMessagesUser(components.ChatUserMessage{
				Content: components.CreateChatUserMessageContentStr(msg.Content),
				Role:    components.ChatUserMessageRoleUser,
			}))
		case "assistant":
			am := components.ChatAssistantMessage{
				Role:      components.ChatAssistantMessageRoleAssistant,
				ToolCalls: toOpenRouterToolCalls(msg.ToolCalls),
			}
			if msg.Content != "" {
				content := components.CreateChatAssistantMessageContentStr(msg.Content)
				am.Content = optionalnullable.From(&content)
			}
			out = append(out, components.CreateChatMessagesAssistant(am))
		case "tool":
			out = append(out, components.CreateChatMessagesTool(components.ChatToolMessage{
				Content:    components.CreateChatToolMessageContentStr(msg.Content),
				Role:       components.ChatToolMessageRoleTool,
				ToolCallID: msg.ToolCallID,
			}))
		}
	}
	return out
}

func toOpenRouterToolCalls(calls []types.ToolCall) []components.ChatToolCall {
	out := make([]components.ChatToolCall, 0, len(calls))
	for _, tc := range calls {
		out = append(out, components.ChatToolCall{
			ID:   tc.ID,
			Type: components.ChatToolCallTypeFunction,
			Function: components.ChatToolCallFunction{
				Name:      tc.Name,
				Arguments: string(tc.Input),
			},
		})
	}
	return out
}

type Plugin struct{}

func (Plugin) Name() string { return "providers/openrouter" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterProvider(app, providerPlugin{})
	return nil
}

type providerPlugin struct{}

func (providerPlugin) ProviderName() config.ProviderName { return config.ProviderOpenRouter }

func (providerPlugin) NewProvider(cfg config.Config) (llm.Provider, error) {
	return newProvider(cfg)
}
