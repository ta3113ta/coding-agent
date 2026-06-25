# ADR-0003: Streaming LLM responses via optional callback

**Status:** Accepted  
**Date:** 2026-06-17

## Context

The REPL runner waited for `provider.Complete` to return the full response before printing — slow UX and not interactive, especially for long answers.

Streaming must flow from the LLM API → provider → agent loop → runner, but the agent loop still needs to store history and tool calls the same way after each turn.

## Decision

Add an **optional `OnStream` callback** on `types.CompleteRequest`:

- When `OnStream != nil` — provider uses the SDK streaming API, emits `types.StreamEvent{TextDelta}` while receiving chunks, then returns a `CompleteResponse` accumulated fully like non-streaming
- When `OnStream == nil` — provider uses the original path (non-streaming)

The runner (REPL) passes a callback that `fmt.Print`s text deltas immediately; tool calls still display after the LLM turn ends via verbose mode in the agent loop.

Implementation:
- [`types/types.go`](../../types/types.go) — `StreamEvent`, `CompleteRequest.OnStream`
- [`agent/agent.go`](../../agent/agent.go) — forward callback
- [`plugins/providers/anthropic/anthropic.go`](../../plugins/providers/anthropic/anthropic.go) — `Messages.NewStreaming`
- [`plugins/providers/openrouter/openrouter.go`](../../plugins/providers/openrouter/openrouter.go) — `Stream: true` + EventStream
- [`plugins/runner/repl/repl.go`](../../plugins/runner/repl/repl.go) — print deltas live

## Alternatives Considered

| Approach | Reason not chosen |
|--------|-------------------|
| Separate `StreamingProvider` interface | Repeated type assertions in multiple places |
| `CompleteStream` method on `Provider` | Forces every provider to implement even when unused |
| Runner calls provider directly | Breaks layering, bypasses agent loop |
| Stream tool-call JSON | Out of scope v1 — display after turn like verbose |

## Consequences

**Pros**

- REPL responds token-by-token without changing the agent history model
- Backward compatible — `OnStream == nil` uses the original path
- Same provider serves both streaming and non-streaming callers

**Cons / trade-offs**

- Provider implementations are more complex (two paths + accumulation)
- Tool calls do not stream during generation (displayed after turn ends)
