package openrouter

import (
	"strings"
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

func TestApplySessionID_Empty(t *testing.T) {
	chatReq := &components.ChatRequest{}
	applySessionID(chatReq, "")
	require.Nil(t, chatReq.SessionID)
}

func TestApplySessionID_Set(t *testing.T) {
	chatReq := &components.ChatRequest{}
	applySessionID(chatReq, "64d0a2fb-ed49-4d6e-96a5-3dd44a1d115c")
	require.NotNil(t, chatReq.SessionID)
	require.Equal(t, "64d0a2fb-ed49-4d6e-96a5-3dd44a1d115c", *chatReq.SessionID)
}

func TestApplySessionID_TruncatesLongID(t *testing.T) {
	chatReq := &components.ChatRequest{}
	longID := strings.Repeat("a", 300)
	applySessionID(chatReq, longID)
	require.NotNil(t, chatReq.SessionID)
	require.Len(t, *chatReq.SessionID, 256)
}
