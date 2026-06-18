package anthropic

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"

	"coding-agent/types"
)

func TestApplyPromptCache_Disabled(t *testing.T) {
	params := &anthropic.MessageNewParams{}
	applyPromptCache(params, types.PromptCacheConfig{Enabled: false})
	if params.CacheControl.Type != "" {
		t.Fatalf("CacheControl should be zero when disabled, got %+v", params.CacheControl)
	}
}

func TestApplyPromptCache_EnabledDefaultTTL(t *testing.T) {
	params := &anthropic.MessageNewParams{}
	applyPromptCache(params, types.PromptCacheConfig{Enabled: true, TTL: "5m"})
	if params.CacheControl.Type != "ephemeral" {
		t.Fatalf("CacheControl.Type = %q, want ephemeral", params.CacheControl.Type)
	}
	if params.CacheControl.TTL != "" {
		t.Fatalf("CacheControl.TTL = %q, want empty (default 5m)", params.CacheControl.TTL)
	}
}

func TestApplyPromptCache_EnabledOneHourTTL(t *testing.T) {
	params := &anthropic.MessageNewParams{}
	applyPromptCache(params, types.PromptCacheConfig{Enabled: true, TTL: "1h"})
	if params.CacheControl.TTL != anthropic.CacheControlEphemeralTTLTTL1h {
		t.Fatalf("CacheControl.TTL = %q, want 1h", params.CacheControl.TTL)
	}
}

func TestBuildMessageParams_IncludesCacheControl(t *testing.T) {
	params := buildMessageParams(types.CompleteRequest{
		Model:        "claude-sonnet-4-5",
		SystemPrompt: "system",
		PromptCache:  types.PromptCacheConfig{Enabled: true},
	}, 1024)
	if params.CacheControl.Type != "ephemeral" {
		t.Fatalf("expected cache control on built params")
	}
}
