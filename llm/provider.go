package llm

import (
	"context"
	"fmt"

	"coding-agent/config"
	"coding-agent/types"
)

type Provider interface {
	Complete(ctx context.Context, req types.CompleteRequest) (*types.CompleteResponse, error)
}

type Factory func(cfg config.Config) (Provider, error)

var registry = map[config.ProviderName]Factory{}

func RegisterProvider(name config.ProviderName, f Factory) {
	registry[name] = f
}

func NewProvider(cfg config.Config) (Provider, error) {
	f, ok := registry[cfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
	return f(cfg)
}
