# ADR-0004: Prompt caching via top-level automatic cache_control

**Status:** Accepted  
**Date:** 2026-06-18

## Context

The agent loop calls `provider.Complete` multiple times per user turn (tool use loop), sending `system prompt + tool definitions + message history` where most of the prefix is identical — wasting tokens and adding redundant latency.

Anthropic and OpenRouter support prompt caching via `cache_control: { type: "ephemeral" }`, which reduces input token cost on cache hits for stable prefixes.

## Decision

Add **`PromptCacheConfig` on `types.CompleteRequest`** and have providers apply **top-level automatic `cache_control`** when enabled:

- Config: `PROMPT_CACHE_ENABLED` (default `true`), `PROMPT_CACHE_TTL` (`5m` or `1h`)
- Agent forwards config on every `Complete` call without changing loop logic
- Anthropic: `MessageNewParams.CacheControl`
- OpenRouter: `ChatRequest.CacheControl`
- OpenRouter: `ChatRequest.SessionID` from agent session UUID (sticky routing + observability grouping)

Top-level automatic caching places a breakpoint at the last cacheable block and advances as the conversation grows — suited to multi-step tool loops without manual breakpoint management.

Implementation:
- [`types/types.go`](../../types/types.go) — `PromptCacheConfig`, `CompleteRequest.PromptCache`
- [`config/config.go`](../../config/config.go) — env parsing + validation
- [`agent/agent.go`](../../agent/agent.go) — forward config
- [`plugins/providers/anthropic/anthropic.go`](../../plugins/providers/anthropic/anthropic.go) — `applyPromptCache`
- [`plugins/providers/openrouter/openrouter.go`](../../plugins/providers/openrouter/openrouter.go) — `applyPromptCache`

## Alternatives Considered

| Approach | Reason not chosen for v1 |
|--------|--------------------------|
| Top-level automatic `cache_control` | **Chosen** — minimal code, auto-advancing breakpoint |
| Explicit breakpoints (system, tools, messages) | Complex: 4-breakpoint limit, 20-block lookback, TTL ordering |
| Provider-internal always-on | Hides control, harder to debug |
| OpenRouter `session_id` sticky routing | Implemented — agent forwards local session UUID on OpenRouter requests |
| `CompleteResponse.Usage` cache stats | Deferred — roadmap item Cost / token tracking |

## Consequences

**Pros**

- Reduces cost/latency on tool-loop iterations without changing the agent history model
- Backward compatible — `PromptCache.Enabled == false` uses the original path
- Same pattern for both Anthropic direct and OpenRouter

**Cons / trade-offs**

- Cache hits require a minimum prefix size per model (e.g. 1,024 tokens for Sonnet) — short prompts may not cache
- Tool definitions must be stable during a session (order in `builtin.Default` must stay fixed)
- Long sessions (>20 content blocks) may need explicit breakpoints in the future
