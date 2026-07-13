# ADR-0012: Error recovery / retry policy

**Status:** Accepted  
**Date:** 2026-07-13

## Context

LLM API calls fail transiently: rate limits (429), overloaded/5xx responses, network blips, and occasional empty model responses. The agent loop only retried empty responses / context cancel (no backoff), while Anthropic and OpenRouter SDKs each applied their own retry defaults. Nested, inconsistent policies made behavior hard to reason about and left 429/network failures to abort the turn after opaque SDK retries.

Tool failures already recover softly (`IsError` tool messages; the loop continues). That path stays unchanged.

## Decision

Own a **single LLM retry policy** in the agent loop around `provider.Complete`:

1. Core [`retry/`](../../retry/) package: `Policy`, `Do`, `Transient` / `IsTransient`, exponential backoff + jitter, optional `Retry-After`.
2. Providers **disable SDK retries** and mark known transient errors with `retry.Transient` (HTTP 408/409/429/≥500, connection errors, empty response).
3. Agent calls `retry.Do` with the configured policy; empty text + no tool calls counts as transient.
4. Do **not** retry `context.Canceled`.

**Config:**

| Field | Env | Default |
|-------|-----|---------|
| Max attempts (total) | `LLM_RETRY_MAX_ATTEMPTS` | `3` |
| Initial backoff | `LLM_RETRY_INITIAL_BACKOFF` | `1s` |
| Max backoff | `LLM_RETRY_MAX_BACKOFF` | `30s` |

`MaxAttempts=1` disables retries. Backoff: `min(initial * 2^attempt, max)` + ≤20% jitter; if `Retry-After` is present, use `max(parsed, computed)` capped by max backoff.

## Alternatives Considered

| Approach | Reason not chosen |
|----------|-------------------|
| Rely only on SDK `MaxRetries` | Different defaults per provider; no unified empty-response or config story |
| Nested agent + SDK retries | Unpredictable total wait; double-counting attempts |
| Tool-level retries | Soft `IsError` recovery already works; re-running shell/writes is riskier |
| REPL `/retry` after abort | UX deferred; policy alone covers most transient failures mid-turn |

## Consequences

**Pros**

- One configurable policy for Anthropic and OpenRouter
- Clear transient vs permanent classification
- Backoff + `Retry-After` reduces thundering herd on rate limits

**Cons**

- Providers must keep wrapping transient errors correctly when SDKs change
- Retried calls add latency and may increase cost (no separate accounting in v1)
