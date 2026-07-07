# ADR-0011: Parallel tool execution

**Status:** Accepted  
**Date:** 2026-07-05

## Context

When the LLM returns multiple tool calls in one assistant turn (e.g. several `read_file` or `grep` requests), the agent loop executed them **sequentially** — each `registry.Dispatch` blocked until the prior tool finished. Independent reads and searches are common; serial dispatch adds unnecessary latency per turn.

Permission hooks (script + interactive REPL) must remain predictable: interactive prompts read from stdin and cannot run concurrently.

## Decision

Run tool calls in **two phases** when the assistant turn has **more than one** tool call and parallel execution is enabled:

1. **Sequential preflight** — for each call in original order: verbose log, plan-mode guard, `permission.Chain.Evaluate`. Denied or blocked calls produce settled error tool messages immediately.
2. **Parallel dispatch** — allowed calls run concurrently via `golang.org/x/sync/errgroup`, each calling `registry.Dispatch` with the resolved input.
3. **Ordered archive append** — all tool result messages are appended in the **original call index order** (one batch `appendArchive`), regardless of which goroutine finished first.

**Fast paths:**

| Condition | Behavior |
|-----------|----------|
| Single tool call | Sequential path (no goroutines) |
| `PARALLEL_TOOLS_ENABLED=false` or `--no-parallel-tools` | Sequential path |

**Config:**

| Field | Env | Default |
|-------|-----|---------|
| `ParallelToolsEnabled` | `PARALLEL_TOOLS_ENABLED` | `true` |

CLI: `--no-parallel-tools` disables parallel dispatch.

Implementation lives in [`agent/agent.go`](../../agent/agent.go) (`executeToolCalls`).

## Alternatives Considered

| Approach | Reason not chosen |
|----------|-------------------|
| Parallel permission evaluation | Interactive stdin prompts and script `ask` hints require sequential preflight |
| Read-only parallel only | User chose full parallel; model is responsible for not emitting conflicting writes |
| Per-tool serial allowlist (`task`, `run_bash`, etc.) | Extra maintenance; same file races possible in sequential mode too |
| Global registry mutex | Would serialize all tools and defeat the purpose |
| Max concurrency cap | Deferred; typical N per turn is small |

## Consequences

**Pros**

- Faster multi-tool turns when the model batches independent operations
- Permission UX unchanged (one prompt at a time)
- Providers already key tool results by `tool_call_id`; call-order append is sufficient for Anthropic and OpenRouter

**Cons**

- Tools must tolerate concurrent `Execute` (shared state needs mutexes — `SessionRules` and `plan.SessionState` already have them)
- Concurrent writes to the same file can race; same class of risk as sequential conflicting edits
- No max-concurrency limit in v1
