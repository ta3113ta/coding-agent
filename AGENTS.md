# Coding Agent Architecture

This project uses a **minimal core + compile-time plugins** design. The core defines contracts and the agent loop; all implementations live under `plugins/`.

## Documentation

- **AGENTS.md** — keep under **200 lines** (high-level index; link out for detail)
- **Every other `*.md`** — keep under **300 lines**
- Over budget → split into a new file or prune redundant content; do not compress prose to fit
- Cursor rule: [`.cursor/rules/documentation-limits.mdc`](.cursor/rules/documentation-limits.mdc)

## Core (do not add implementations here)

| Package | Purpose |
|---------|---------|
| [`agent/`](../agent/agent.go) | Agent loop only — LLM + tools until done |
| [`types/`](../types/types.go) | Neutral types: `Message`, `ToolDefinition`, `CompleteRequest/Response` |
| [`llm/provider.go`](../llm/provider.go) | `Provider` interface + provider registry |
| [`tools/tool.go`](../tools/tool.go) | `Tool` interface + `Registry` dispatch |
| [`config/`](../config/config.go) | Env/flag configuration |
| [`session/`](../session/session.go) | Session types + `Store` interface |
| [`permission/`](../permission/permission.go) | Permission hook contract + chain |
| [`compaction/`](../compaction/compaction.go) | Context compaction contract |
| [`spawn/`](../spawn/spawn.go) | Sub-agent spawning contract |
| [`plan/`](../plan/) | Plan mode + todo tracking types and session state |
| [`plugin/`](../plugin/) | Plugin interfaces + `Bootstrap()` |

**Rule:** If it talks to an external API, runs shell commands, or defines a persona — it is a plugin, not core.

## Plugins (add new features here)

```
plugins/
├── builtin/builtin.go       # single registration list
├── tools/                   # one package per tool
├── providers/               # one package per LLM provider
├── prompt/                  # system prompt contributors
├── skills/                  # skill discovery + index injection
└── runner/                  # REPL, one-shot, HTTP, etc.
```

Register every new plugin in [`plugins/builtin/builtin.go`](../plugins/builtin/builtin.go).

## How to add a new tool

1. Create `plugins/tools/mytool/my_tool.go`
2. Implement `tools.Tool` (`Name`, `Definition`, `Execute`)
3. Add a `Plugin` struct with `Register()` that calls `plugin.RegisterTools()`
4. Append `mytool.Plugin{}` to `builtin.Default`

## File editing tools

| Tool | Use when |
|------|----------|
| `read_file` | Read/explore files (with line numbers) |
| `grep` / `glob` | Search the codebase (ripgrep; requires `rg` on PATH) |
| `str_replace` | Edit existing files (primary) |
| `write_file` | Create new files, or fallback |

See rationale and alternatives in [ADR-0001](docs/adr/0001-str-replace-for-file-editing.md), [ADR-0009](docs/adr/0009-grep-glob-internal-search.md)

## Skills

At bootstrap the agent discovers `SKILL.md` from project (`.cursor/skills/`), personal (`~/.cursor/skills/`), and bundled (`plugins/skills/builtin/`) sources, then injects a skill index into the system prompt — the agent loads full content with `read_file` when a task is relevant.

See rationale and alternatives in [ADR-0002](docs/adr/0002-load-skill-two-phase-discovery.md)

## Streaming

The REPL runner streams assistant text tokens via an optional `OnStream` callback on `CompleteRequest` — see [ADR-0003](docs/adr/0003-streaming-llm-responses.md)

## Prompt caching

The provider applies top-level automatic `cache_control` when `CompleteRequest.PromptCache.Enabled` — see [ADR-0004](docs/adr/0004-prompt-caching.md)

## Session management

Conversation history is persisted as JSON via the `session.Store` contract and filestore plugin — auto-save after each turn, display name, ephemeral mode (`--no-session`), startup flags `-c`/`-r`, resume via CLI or REPL slash commands — see [ADR-0005](docs/adr/0005-session-management.md)

## Permission hooks

Before `registry.Dispatch` the agent calls `permission.Chain` — script hooks from `.coding-agent/hooks.json` (`preToolUse`) followed by an interactive REPL prompt for risky tools — see [ADR-0006](docs/adr/0006-permission-hooks.md)

## Context compaction

Before `provider.Complete` the agent calls `compaction.Compactor` — auto-summarize when projected context exceeds `contextWindow - reserveTokens`, or manual `/compact [instructions]` — see [ADR-0007](docs/adr/0007-context-compaction.md)

## Sub-agent spawning

The parent agent calls the `task` tool to spawn a sub-agent synchronously — the sub-agent uses an in-memory temporary session, a tool set per profile, and does not touch the parent archive — see [ADR-0008](docs/adr/0008-sub-agent-task-spawning.md)

## Plan mode + todo tracking

Plan mode restricts tools to read-only research; `create_plan` saves a draft for `/approve` (switches to agent mode; optional trailing text runs implementation). `/plan [task]` enters plan mode or plans in one shot. `todo_write` tracks in-session tasks persisted in session JSON — see [ADR-0010](docs/adr/0010-plan-mode-todo-tracking.md)

## Architecture Decision Records

New features that affect architecture (tool contract, agent loop, bootstrap flow, discovery model, etc.) must have an ADR in `docs/adr/` before implementation.

Format: `NNNN-short-title.md` — see [ADR-0001](docs/adr/0001-str-replace-for-file-editing.md)

Template:
- **Status** / **Date**
- **Context** — problem to solve
- **Decision** — what we chose to do
- **Alternatives Considered** — other options and why we did not choose them
- **Consequences** — pros and cons

## How to add a new LLM provider

1. Create `plugins/providers/myprovider/myprovider.go`
2. Implement `llm.Provider` (`Complete`)
3. Add `Plugin` with `Register()` calling `plugin.RegisterProvider()`
4. Add provider name constants to `config/config.go`
5. Append to `builtin.Default`

## How to add a prompt plugin

1. Create `plugins/prompt/myname/prompt.go`
2. Call `plugin.AppendPrompt(app, "...")` in `Register()`
3. Append to `builtin.Default`

Multiple prompt plugins are concatenated in registration order.

## How to add a runner plugin

1. Create `plugins/runner/myname/runner.go`
2. Implement `plugin.Runner` (`Run(ctx, plugin.AgentHandle)`)
3. Set `app.Runner` in `Register()`
4. Append to `builtin.Default` (only one runner should win — last one registered wins)

## Bootstrap flow

```
main.go
  → plugin.LoadConfigFromEnv()
  → plugin.Bootstrap(cfg, builtin.Default...)
      1. Each plugin Register(app)
      2. Tools collected into Registry
      3. Providers registered in llm registry
      4. Prompts concatenated
      5. llm.NewProvider(cfg) resolves active provider
  → agent.New(provider, tools, model, prompt, cache, verbose, sessionStore, providerName, app.Permission, app.Compactor, app.PlanState, cfg.PlanEnabled)
  → app.Runner.Run(ctx, agent)
```

## What we defer

- Runtime `.so` plugins
- `init()` auto-registration (explicit `builtin.Default` list is easier to debug)
- Additional hook events (`postToolUse`, `beforeShellExecution`, MCP) — see ADR-0006 v2

## Checklist before adding code

1. Is this a **contract** (interface, types, loop)? → core package
2. Is this an **implementation**? → `plugins/`
3. Did you register it in `plugins/builtin/builtin.go`?
4. Did you avoid importing `agent` from `plugin`? (use `plugin.AgentHandle` instead)
5. New feature with architectural impact → write an ADR in `docs/adr/` and link from AGENTS.md or README.md
