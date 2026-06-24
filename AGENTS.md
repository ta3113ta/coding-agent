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

```go
package mytool

import (
    "encoding/json"
    "coding-agent/types"
    "coding-agent/plugin"
    "coding-agent/tools"
)

type MyTool struct{}

func (MyTool) Name() string { return "my_tool" }

func (MyTool) Definition() types.ToolDefinition {
    return types.ToolDefinition{
        Name:        "my_tool",
        Description: "...",
        Properties:  map[string]any{ /* JSON schema */ },
        Required:    []string{"field"},
    }
}

func (MyTool) Execute(input json.RawMessage) (string, error) {
    // ...
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/mytool" }

func (Plugin) Register(app *plugin.App) error {
    plugin.RegisterTools(app, MyTool{})
    return nil
}
```

## File editing tools

| Tool | ใช้เมื่อ |
|------|---------|
| `read_file` | อ่าน/สำรวจไฟล์ (พร้อมเลขบรรทัด) |
| `str_replace` | แก้ไฟล์ที่มีอยู่ (primary) |
| `write_file` | สร้างไฟล์ใหม่ หรือ fallback |

ดู rationale และ alternatives ใน [ADR-0001](docs/adr/0001-str-replace-for-file-editing.md)

## Skills

Agent discover `SKILL.md` ตอน bootstrap จาก project (`.cursor/skills/`), personal (`~/.cursor/skills/`), และ bundled (`plugins/skills/builtin/`) แล้ว inject skill index เข้า system prompt — agent โหลดเนื้อหาเต็มด้วย `read_file` เมื่อ task relevant

ดู rationale และ alternatives ใน [ADR-0002](docs/adr/0002-load-skill-two-phase-discovery.md)

## Streaming

REPL runner stream assistant text tokens ผ่าน optional `OnStream` callback บน `CompleteRequest` — ดู [ADR-0003](docs/adr/0003-streaming-llm-responses.md)

## Prompt caching

Provider ใส่ top-level automatic `cache_control` เมื่อ `CompleteRequest.PromptCache.Enabled` — ดู [ADR-0004](docs/adr/0004-prompt-caching.md)

## Session management

Persist conversation history เป็น JSON ผ่าน `session.Store` contract + filestore plugin — auto-save หลังแต่ละ turn, display name, ephemeral mode (`--no-session`), startup flags `-c`/`-r`, resume ด้วย CLI หรือ REPL slash commands — ดู [ADR-0005](docs/adr/0005-session-management.md)

## Permission hooks

ก่อน `registry.Dispatch` agent เรียก `permission.Chain` — script hooks จาก `.coding-agent/hooks.json` (`preToolUse`) แล้ว interactive REPL prompt สำหรับ risky tools — ดู [ADR-0006](docs/adr/0006-permission-hooks.md)

- Config: `PERMISSION_ENABLED`, `PERMISSION_HOOKS_FILE`, `--no-permission`
- Core: [`permission/permission.go`](../permission/permission.go)
- Plugins: [`plugins/permission/script/`](../plugins/permission/script/), [`plugins/permission/interactive/`](../plugins/permission/interactive/)

## Context compaction

ก่อน `provider.Complete` agent เรียก `compaction.Compactor` — auto-summarize เมื่อ projected context เกิน `contextWindow - reserveTokens` หรือ manual `/compact [instructions]` — ดู [ADR-0007](docs/adr/0007-context-compaction.md)

- Config: `COMPACTION_ENABLED`, `COMPACTION_RESERVE_TOKENS`, `COMPACTION_KEEP_RECENT_TOKENS`, `COMPACTION_CONTEXT_WINDOW`, `--no-compaction`
- Core: [`compaction/`](../compaction/) (serialize, project, split-by-tokens, file ops)
- Plugin: [`plugins/compaction/summarize/`](../plugins/compaction/summarize/)

## Sub-agent spawning

Parent agent เรียก `task` tool เพื่อ spawn sub-agent แบบ sync — sub-agent ใช้ session ชั่วคราวใน memory, tool set ตาม profile, ไม่แตะ parent archive — ดู [ADR-0008](docs/adr/0008-sub-agent-task-spawning.md)

- Config: `SPAWN_ENABLED`, `SPAWN_MAX_TURNS`, `--no-spawn`
- Core: [`spawn/`](../spawn/) (types, profiles, Runner contract)
- Plugins: [`plugins/spawn/runner/`](../plugins/spawn/runner/), [`plugins/tools/task/`](../plugins/tools/task/)

## Architecture Decision Records

Feature ใหม่ที่กระทบ architecture (tool contract, agent loop, bootstrap flow, discovery model ฯลฯ) ต้องมี ADR ใน `docs/adr/` ก่อน implement

รูปแบบ: `NNNN-short-title.md` — ดู [ADR-0001](docs/adr/0001-str-replace-for-file-editing.md)

Template:
- **Status** / **Date**
- **Context** — ปัญหาที่ต้องแก้
- **Decision** — สิ่งที่เลือกทำ
- **Alternatives Considered** — ทางเลือกอื่น + เหตุผลที่ไม่เลือก
- **Consequences** — ข้อดี/ข้อเสีย

## How to add a new LLM provider

1. Create `plugins/providers/myprovider/myprovider.go`
2. Implement `llm.Provider` (`Complete`)
3. Add `Plugin` with `Register()` calling `plugin.RegisterProvider()`
4. Add provider name constants to `config/config.go`
5. Append to `builtin.Default`

```go
type providerPlugin struct{}

func (providerPlugin) ProviderName() config.ProviderName { return "myprovider" }

func (providerPlugin) NewProvider(cfg config.Config) (llm.Provider, error) {
    return newProvider(cfg)
}

func (Plugin) Register(app *plugin.App) error {
    plugin.RegisterProvider(app, providerPlugin{})
    return nil
}
```

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
  → agent.New(provider, tools, model, prompt, cache, verbose, sessionStore, providerName, app.Permission, app.Compactor)
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
5. Feature ใหม่ที่มี architectural impact → เขียน ADR ใน `docs/adr/` แล้วลิงก์จาก AGENTS.md หรือ README.md
