package anthropic

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"coding-agent/config"
	"coding-agent/llm"
	"coding-agent/plugin"
	"coding-agent/types"
)

type provider struct {
	client anthropic.Client
}

func newProvider(cfg config.Config) (llm.Provider, error) {
	return &provider{
		client: anthropic.NewClient(option.WithAPIKey(cfg.AnthropicAPIKey)),
	}, nil
}

func (p *provider) Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8096
	}

	params := buildMessageParams(req, maxTokens)

	if req.OnStream == nil {
		resp, err := p.client.Messages.New(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("anthropic: %w", err)
		}
		return anthropicMessageToResponse(resp), nil
	}

	stream := p.client.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	var msg anthropic.Message
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return nil, fmt.Errorf("anthropic accumulate: %w", err)
		}
		if delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			if td, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok && td.Text != "" {
				req.OnStream(types.StreamEvent{TextDelta: td.Text})
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("anthropic: %w", err)
	}
	return anthropicMessageToResponse(&msg), nil
}

func buildMessageParams(req types.CompleteRequest, maxTokens int) anthropic.MessageNewParams {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: int64(maxTokens),
		System: []anthropic.TextBlockParam{
			{Text: req.SystemPrompt},
		},
		Messages: toAnthropicMessages(req.Messages),
		Tools:    toAnthropicTools(req.Tools),
	}
	applyPromptCache(&params, req.PromptCache)
	return params
}

func applyPromptCache(params *anthropic.MessageNewParams, cfg types.PromptCacheConfig) {
	if !cfg.Enabled {
		return
	}
	params.CacheControl = anthropic.NewCacheControlEphemeralParam()
	if cfg.TTL == "1h" {
		params.CacheControl.TTL = anthropic.CacheControlEphemeralTTLTTL1h
	}
}

func anthropicMessageToResponse(msg *anthropic.Message) *types.CompleteResponse {
	out := &types.CompleteResponse{}
	for _, block := range msg.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			out.Text += b.Text
		case anthropic.ToolUseBlock:
			out.ToolCalls = append(out.ToolCalls, types.ToolCall{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
			})
		}
	}
	return out
}

func toAnthropicTools(tools []types.ToolDefinition) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		def := anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: t.Properties,
				Required:   t.Required,
			},
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &def})
	}
	return out
}

func toAnthropicMessages(msgs []types.Message) []anthropic.MessageParam {
	var out []anthropic.MessageParam
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		switch msg.Role {
		case "user":
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		case "assistant":
			var blocks []anthropic.ContentBlockParamUnion
			if msg.Content != "" {
				blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, tc.Input, tc.Name))
			}
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		case "tool":
			var toolResults []anthropic.ContentBlockParamUnion
			for i < len(msgs) && msgs[i].Role == "tool" {
				tm := msgs[i]
				toolResults = append(toolResults,
					anthropic.NewToolResultBlock(tm.ToolCallID, tm.Content, tm.IsError))
				i++
			}
			out = append(out, anthropic.NewUserMessage(toolResults...))
			i--
		}
	}
	return out
}

type Plugin struct{}

func (Plugin) Name() string { return "providers/anthropic" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterProvider(app, providerPlugin{})
	return nil
}

type providerPlugin struct{}

func (providerPlugin) ProviderName() config.ProviderName { return config.ProviderAnthropic }

func (providerPlugin) NewProvider(cfg config.Config) (llm.Provider, error) {
	return newProvider(cfg)
}
