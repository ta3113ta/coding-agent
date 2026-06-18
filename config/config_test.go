package config

import (
	"testing"
)

func TestParsePromptCacheTTL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", PromptCacheTTL5m},
		{"5m", PromptCacheTTL5m},
		{"1h", PromptCacheTTL1h},
		{"invalid", PromptCacheTTL5m},
	}
	for _, tc := range tests {
		if got := parsePromptCacheTTL(tc.in); got != tc.want {
			t.Errorf("parsePromptCacheTTL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestConfigPromptCache(t *testing.T) {
	cfg := Config{PromptCacheEnabled: true, PromptCacheTTL: PromptCacheTTL1h}
	pc := cfg.PromptCache()
	if !pc.Enabled || pc.TTL != PromptCacheTTL1h {
		t.Fatalf("PromptCache() = %+v, want enabled 1h", pc)
	}
}

func TestValidatePromptCacheTTL(t *testing.T) {
	cfg := Config{
		Provider:           ProviderAnthropic,
		AnthropicAPIKey:    "key",
		PromptCacheTTL:     "bad",
		PromptCacheEnabled: true,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid PROMPT_CACHE_TTL")
	}
}
