package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	// No SDK per-request context timeout: Send() defers cancel() on return, which
	// breaks SSE reads. Use a long HTTP client timeout for queue + stream instead.
	return &provider{
		client: openrouter.New(
			openrouter.WithSecurity(cfg.OpenRouterAPIKey),
			openrouter.WithClient(&http.Client{Timeout: 3 * time.Minute}),
		),
	}, nil
}

func (p *provider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8096
	}

	messages := buildOpenRouterMessages(req)
	chatReq := components.ChatRequest{
		Model:     openrouter.Pointer(req.Model),
		MaxTokens: optionalnullable.From(openrouter.Pointer(int64(maxTokens))),
		Messages:  messages,
		Tools:     toOpenRouterTools(req.Tools),
	}
	applyPromptCache(&chatReq, req.PromptCache)
	applySessionID(&chatReq, req.SessionID)

	if req.OnStream == nil {
		chatReq.Stream = openrouter.Pointer(false)
		return p.completeNonStreaming(ctx, chatReq)
	}

	chatReq.Stream = openrouter.Pointer(true)
	return p.completeStreaming(ctx, req, chatReq)
}

func applyPromptCache(chatReq *components.ChatRequest, cfg types.PromptCacheConfig) {
	if !cfg.Enabled {
		return
	}
	directive := &components.AnthropicCacheControlDirective{
		Type: components.AnthropicCacheControlDirectiveTypeEphemeral,
	}
	if cfg.TTL == "1h" {
		directive.TTL = components.AnthropicCacheControlTTLOneh.ToPointer()
	}
	chatReq.CacheControl = directive
}

func applySessionID(chatReq *components.ChatRequest, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	if len(sessionID) > 256 {
		sessionID = sessionID[:256]
	}
	chatReq.SessionID = openrouter.Pointer(sessionID)
}

func buildOpenRouterMessages(req types.CompleteRequest) []components.ChatMessages {
	messages := []components.ChatMessages{
		components.CreateChatMessagesSystem(components.ChatSystemMessage{
			Content: components.CreateChatSystemMessageContentStr(req.SystemPrompt),
			Role:    components.ChatSystemMessageRoleSystem,
		}),
	}
	return append(messages, toOpenRouterMessages(req.Messages)...)
}

func (p *provider) completeNonStreaming(ctx context.Context, chatReq components.ChatRequest) (*types.CompleteResponse, error) {
	res, err := p.client.Chat.Send(ctx, chatReq)
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
	if strings.TrimSpace(out.Text) == "" && len(out.ToolCalls) == 0 {
		return nil, fmt.Errorf("openrouter: model returned empty response")
	}
	return out, nil
}

type openRouterToolCallAcc struct {
	id        string
	name      string
	arguments strings.Builder
}

func (p *provider) completeStreaming(ctx context.Context, req types.CompleteRequest, chatReq components.ChatRequest) (*types.CompleteResponse, error) {
	res, err := p.client.Chat.Send(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter: %w", err)
	}

	if res.Type != operations.SendChatCompletionRequestResponseTypeEventStream || res.EventStream == nil {
		return nil, fmt.Errorf("openrouter: expected event stream")
	}
	defer res.EventStream.Close()

	out := &types.CompleteResponse{}
	var toolCalls []*openRouterToolCallAcc

	for res.EventStream.Next() {
		chunk := res.EventStream.Value()
		if chunk == nil {
			continue
		}
		data := chunk.GetData()
		if streamErr := data.GetError(); streamErr != nil {
			return nil, fmt.Errorf("openrouter: %s (code %d)", streamErr.GetMessage(), streamErr.GetCode())
		}
		for _, choice := range (&data).GetChoices() {
			delta := choice.GetDelta()
			if content, ok := delta.GetContent().GetOrZero(); ok && content != "" {
				out.Text += content
				req.OnStream(types.StreamEvent{TextDelta: content})
			}
			if reasoning, ok := delta.GetReasoning().GetOrZero(); ok && reasoning != "" {
				// Stream reasoning for live feedback only; do not persist in out.Text.
				req.OnStream(types.StreamEvent{TextDelta: reasoning})
			}
			for _, tc := range delta.GetToolCalls() {
				idx := tc.GetIndex()
				for int64(len(toolCalls)) <= idx {
					toolCalls = append(toolCalls, &openRouterToolCallAcc{})
				}
				acc := toolCalls[idx]
				if id := tc.GetID(); id != nil {
					acc.id = *id
				}
				if fn := tc.GetFunction(); fn != nil {
					if name := fn.GetName(); name != nil {
						acc.name = *name
					}
					if args := fn.GetArguments(); args != nil {
						acc.arguments.WriteString(*args)
					}
				}
			}
		}
	}
	if err := res.EventStream.Err(); err != nil {
		return nil, fmt.Errorf("openrouter stream: %w", err)
	}

	for _, acc := range toolCalls {
		if acc == nil {
			continue
		}
		out.ToolCalls = append(out.ToolCalls, types.ToolCall{
			ID:    acc.id,
			Name:  acc.name,
			Input: json.RawMessage(acc.arguments.String()),
		})
	}
	if strings.TrimSpace(out.Text) == "" && len(out.ToolCalls) == 0 {
		return nil, fmt.Errorf("openrouter: model returned empty response")
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
