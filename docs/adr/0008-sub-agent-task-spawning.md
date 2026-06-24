# ADR-0008: Sub-agent / task spawning

**Status:** Accepted  
**Date:** 2026-06-24

## Context

Long parent sessions accumulate context. Some work — codebase exploration, one-off shell commands, focused sub-tasks — benefits from an isolated agent with a fresh prompt and a restricted tool set. Cursor exposes this via a **Task** tool that spawns sub-agents (`explore`, `shell`, `generalPurpose`, etc.).

The coding-agent roadmap listed sub-agent spawning as a planned feature. Without it, the parent agent must do all work inline, growing context and mixing unrelated exploration with the main task.

## Decision

Add **synchronous sub-agent spawning** via a core `spawn` contract, a runner plugin, and a `task` tool:

1. **Core contract** — [`spawn/`](../../spawn/): `Type`, `Request`, `Result`, `Runner` interface, per-type `Profile` (allowed tools + system prompt suffix)
2. **Agent integration** — extract `runLoop` from `Run`; add `RunSubtask` with optional `maxTurns` safety limit
3. **Runner plugin** — [`plugins/spawn/runner/`](../../plugins/spawn/runner/): creates ephemeral child agents via `memory.Store`, filtered tool registry, no compaction
4. **Task tool** — [`plugins/tools/task/`](../../plugins/tools/task/): LLM-callable `task` tool delegating to `spawn.Runner`
5. **Context-aware tools** — `tools.Tool.Execute(ctx, input)` so sub-agents and `run_bash` respect cancellation

### Sub-agent types (v1)

| Type | Tools | Purpose |
|------|-------|---------|
| `explore` | `read_file`, `list_dir` | Read-only codebase exploration |
| `shell` | `run_bash` | Command execution specialist |
| `generalPurpose` | all tools except `task` | General autonomous sub-task |

**Recursion guard:** `task` is excluded from all sub-agent tool sets.

**Isolation:** child agents use in-memory sessions; parent `archive` is never mutated.

**Execution:** synchronous only — `task` blocks until the sub-agent finishes and returns final text.

### Config

| Field | Env | Default |
|-------|-----|---------|
| `SpawnEnabled` | `SPAWN_ENABLED` | `true` |
| `SpawnMaxTurns` | `SPAWN_MAX_TURNS` | `25` |

CLI: `--no-spawn` disables spawning (no `task` tool registered).

## Alternatives Considered

| Approach | Reason not chosen |
|----------|-------------------|
| Prompt-only delegation (no sub-agent) | No tool isolation; pollutes parent context |
| Background spawn + resume v1 | Needs Await tool + task registry; deferred |
| Specialized bugbot/security sub-agents | Defer; profiles can be added later |
| Sub-agents share parent session | Pollutes archive; breaks compaction projection |
| Sub-agent streaming to REPL | Noisy; v1 returns final text only |

## Consequences

**Pros:**
- Parent context stays focused; exploration/shell work is isolated
- Tool restrictions enforce read-only explore and bash-only shell agents
- Max-turn limit prevents runaway sub-agents

**Cons:**
- Extra LLM cost per spawn (full sub-agent loop)
- Permission prompts may fire inside sub-agents
- No background/resume until Await tool exists (separate roadmap item)
- Handoffs / shared context between agents deferred (separate roadmap item)

## Deferred (v2+)

- `run_in_background`, `resume`, agent IDs
- `bugbot`, `security-review`, and other specialized sub-agent types
- Handoffs (share context between agents)
- Sub-agent streaming to REPL
