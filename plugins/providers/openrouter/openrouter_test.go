package openrouter

import (
	"testing"

	"github.com/OpenRouterTeam/go-sdk/models/components"

	"coding-agent/types"
)

func TestApplyPromptCache_Disabled(t *testing.T) {
	chatReq := &components.ChatRequest{}
	applyPromptCache(chatReq, types.PromptCacheConfig{Enabled: false})
	if chatReq.CacheControl != nil {
		t.Fatalf("CacheControl should be nil when disabled")
	}
}

func TestApplyPromptCache_EnabledDefaultTTL(t *testing.T) {
	chatReq := &components.ChatRequest{}
	applyPromptCache(chatReq, types.PromptCacheConfig{Enabled: true, TTL: "5m"})
	if chatReq.CacheControl == nil {
		t.Fatal("CacheControl should be set when enabled")
	}
	if chatReq.CacheControl.Type != components.AnthropicCacheControlDirectiveTypeEphemeral {
		t.Fatalf("CacheControl.Type = %q, want ephemeral", chatReq.CacheControl.Type)
	}
	if chatReq.CacheControl.TTL != nil {
		t.Fatalf("CacheControl.TTL = %v, want nil (default 5m)", chatReq.CacheControl.TTL)
	}
}

func TestApplyPromptCache_EnabledOneHourTTL(t *testing.T) {
	chatReq := &components.ChatRequest{}
	applyPromptCache(chatReq, types.PromptCacheConfig{Enabled: true, TTL: "1h"})
	if chatReq.CacheControl.TTL == nil || *chatReq.CacheControl.TTL != components.AnthropicCacheControlTTLOneh {
		t.Fatalf("CacheControl.TTL = %v, want 1h", chatReq.CacheControl.TTL)
	}
}
