package anthropic

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/require"

	"coding-agent/types"
)

func TestApplyPromptCache_Disabled(t *testing.T) {
	params := &anthropic.MessageNewParams{}
	applyPromptCache(params, types.PromptCacheConfig{Enabled: false})
	require.Empty(t, params.CacheControl.Type)
}

func TestApplyPromptCache_EnabledDefaultTTL(t *testing.T) {
	params := &anthropic.MessageNewParams{}
	applyPromptCache(params, types.PromptCacheConfig{Enabled: true, TTL: "5m"})
	require.Equal(t, "ephemeral", string(params.CacheControl.Type))
	require.Empty(t, params.CacheControl.TTL)
}

func TestApplyPromptCache_EnabledOneHourTTL(t *testing.T) {
	params := &anthropic.MessageNewParams{}
	applyPromptCache(params, types.PromptCacheConfig{Enabled: true, TTL: "1h"})
	require.Equal(t, anthropic.CacheControlEphemeralTTLTTL1h, params.CacheControl.TTL)
}

func TestBuildMessageParams_IncludesCacheControl(t *testing.T) {
	params := buildMessageParams(types.CompleteRequest{
		Model:        "claude-sonnet-4-5",
		SystemPrompt: "system",
		PromptCache:  types.PromptCacheConfig{Enabled: true},
	}, 1024)
	require.Equal(t, "ephemeral", string(params.CacheControl.Type))
}
