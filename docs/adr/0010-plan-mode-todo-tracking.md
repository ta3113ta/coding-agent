# ADR-0010: Plan mode + todo tracking

**Status:** Accepted  
**Date:** 2026-06-28

## Context

The coding-agent roadmap listed **TODO / plan tracking** as a planned feature. Cursor exposes this via **Plan mode** (read-only research before implementation), a **CreatePlan** tool, and **TodoWrite** for structured task lists. Without plan mode, the agent may edit files before the user has reviewed an approach.

## Decision

Add **plan mode + todo tracking** via a core `plan` contract, shared session state, two LLM tools, and REPL slash commands:

1. **Core contract** — [`plan/`](../../plan/): `Mode`, `Plan`, `TodoItem`, `SessionState` with load/snapshot helpers and allowed-tool lists for plan mode
2. **Agent integration** — filter tool definitions in plan mode; dispatch guard for write tools; system prompt suffix; persist mode/todos/plan in session JSON
3. **`todo_write` tool** — [`plugins/tools/todowrite/`](../../plugins/tools/todowrite/): merge/replace in-session todos (available in both modes)
4. **`create_plan` tool** — [`plugins/tools/createplan/`](../../plugins/tools/createplan/): draft plan + write `.coding-agent/plans/<session-id>.md` (plan mode only)
5. **REPL commands** — `/plan [task]`, `/plan show`, `/approve [instructions]`, `/agent`, `/todos`; prompt shows `you (plan)>` in plan mode
6. **Approval gate** — `/agent` blocked while a draft plan exists; `/approve` approves and switches to agent mode (optional trailing text runs implementation)

### Plan mode allowed tools (v1)

`read_file`, `list_dir`, `grep`, `glob`, `todo_write`, `create_plan`

Write tools (`str_replace`, `write_file`, `run_bash`, `task`) are hidden from the LLM and denied at dispatch.

### Config

| Field | Env | Default |
|-------|-----|---------|
| `PlanEnabled` | `PLAN_ENABLED` | `true` |

CLI: `--no-plan` disables plan tools and slash commands; `--plan` starts in plan mode.

## Alternatives Considered

| Approach | Reason not chosen |
|----------|-------------------|
| Permission hook only for plan mode | Hiding tools from LLM is stronger than deny-at-dispatch |
| Plan as spawn sub-agent | No approval gate on parent; extra LLM cost |
| LLM-callable `switch_mode` tool | REPL slash commands are simpler for v1 |
| Todos in separate file store | Session JSON keeps resume simple |

## Consequences

**Pros:**
- User reviews plan before writes
- Todos persist across session resume
- Mirrors familiar Cursor workflow

**Cons:**
- Shared `SessionState` pointer couples tools to agent (same pattern as spawner)
- Plan file + session JSON can drift if edited manually
- No LLM-initiated mode switch until v2

## Deferred (v2+)

- LLM-callable mode switch
- Multiple plans per session
- TUI plan viewer
- Auto-spawn `explore` sub-agent in plan mode
