package openrouter

import (
	"testing"

	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/stretchr/testify/require"

	"coding-agent/types"
)

func TestApplyPromptCache_Disabled(t *testing.T) {
	chatReq := &components.ChatRequest{}
	applyPromptCache(chatReq, types.PromptCacheConfig{Enabled: false})
	require.Nil(t, chatReq.CacheControl)
}

func TestApplyPromptCache_EnabledDefaultTTL(t *testing.T) {
	chatReq := &components.ChatRequest{}
	applyPromptCache(chatReq, types.PromptCacheConfig{Enabled: true, TTL: "5m"})
	require.NotNil(t, chatReq.CacheControl)
	require.Equal(t, components.AnthropicCacheControlDirectiveTypeEphemeral, chatReq.CacheControl.Type)
	require.Nil(t, chatReq.CacheControl.TTL)
}

func TestApplyPromptCache_EnabledOneHourTTL(t *testing.T) {
	chatReq := &components.ChatRequest{}
	applyPromptCache(chatReq, types.PromptCacheConfig{Enabled: true, TTL: "1h"})
	require.NotNil(t, chatReq.CacheControl.TTL)
	require.Equal(t, components.AnthropicCacheControlTTLOneh, *chatReq.CacheControl.TTL)
}
