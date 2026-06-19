# Coding Agent

Minimal coding agent for learning purpose

Built with **minimal core + compile-time plugins** — see [AGENTS.md](AGENTS.md) for how to add new features.

## Structure

```
coding-agent/
├── main.go                          # bootstrap only
├── AGENTS.md                        # architecture guide
├── .cursor/rules/
│   └── agent-architecture.mdc       # Cursor rule
├── config/
│   └── config.go                    # env + flag config
├── plugin/
│   ├── plugin.go                    # Plugin interfaces + App
│   └── registry.go                  # Bootstrap
├── agent/
│   └── agent.go                     # agent loop (core)
├── types/
│   └── types.go                     # neutral shared types
├── llm/
│   └── provider.go                  # Provider interface + registry
├── tools/
│   └── tool.go                      # Tool interface + Registry
├── skills/
│   └── ...                          # skill discovery contract (parse, discover, registry)
└── plugins/
    ├── builtin/builtin.go           # default plugin registry
    ├── tools/                       # readfile, writefile, strreplace, listdir, runbash
    ├── providers/                   # anthropic, openrouter
    ├── skills/builtin/              # bundled SKILL.md files
    ├── prompt/coding/               # system prompt
    ├── session/filestore/           # JSON session persistence
    └── runner/repl/                 # stdin REPL (streams assistant text)
```

## How to run

```bash
cp .env-example .env   # then add your API key
go mod tidy
go run .
```

### Anthropic (default)

```bash
export ANTHROPIC_API_KEY=sk-ant-...
go run .
```

### OpenRouter via env

```bash
export LLM_PROVIDER=openrouter
export OPENROUTER_API_KEY=sk-or-...
go run .
```

### OpenRouter via CLI flags

```bash
go run . --provider openrouter --model openai/gpt-4o
```

### Configuration variables

| Variable / flag | Default | Description |
|---|---|---|
| `LLM_PROVIDER` / `--provider` | `anthropic` | `anthropic` or `openrouter` |
| `ANTHROPIC_API_KEY` | — | API key for Anthropic |
| `ANTHROPIC_MODEL` / `--model` | `claude-sonnet-4-5` | Anthropic model |
| `OPENROUTER_API_KEY` | — | API key for OpenRouter |
| `OPENROUTER_MODEL` / `--model` | `anthropic/claude-sonnet-4` | OpenRouter model |
| `SKILLS_ENABLE_PERSONAL` | `true` | Enable/disable discovery from `~/.cursor/skills/` (`false` to disable) |
| `PROMPT_CACHE_ENABLED` | `true` | Enable automatic prompt caching on LLM requests (`false` to disable) |
| `PROMPT_CACHE_TTL` | `5m` | Cache TTL: `5m` or `1h` (Anthropic/OpenRouter ephemeral caching) |
| `SESSION_SCOPE` / `--session-scope` | `project` | Session storage: `project` (`.coding-agent/sessions/` in cwd) or `global` (`~/.coding-agent/sessions/`) |
| `SESSION_DIR` / `--session-dir` | — | Override session storage directory |
| `--resume` | — | Resume session ID on startup |
| `--new-session` | — | Force new session (overrides `--resume`) |
| `-c` | — | Continue most recent session |
| `-r` | — | Browse and select a past session interactively |
| `--no-session` | — | Ephemeral mode; do not save sessions to disk |
| `--name` | — | Set session display name at startup |

CLI flags override env values.

Then try prompts such as:
- `create a fizzbuzz.go file that prints 1-20, then build it to verify it works`
- `read main.go and explain how it works`

The REPL streams assistant text token-by-token instead of waiting for the full response — see [ADR-0003](docs/adr/0003-streaming-llm-responses.md).

Prompt caching reuses stable prefixes (system prompt, tools, growing history) across tool-loop iterations — see [ADR-0004](docs/adr/0004-prompt-caching.md).

Sessions auto-save after each turn. Resume with `-c`, `-r`, `--resume <id>`, or REPL commands `/new`, `/sessions`, `/resume <id>`, `/session`, `/name <name>`. Use `--no-session` for ephemeral mode — see [ADR-0005](docs/adr/0005-session-management.md).

## Agent loop core (agent/agent.go)

```
loop:
  1. Call provider.Complete with messages + tool definitions
  2. Append assistant response (text + tool calls) to history
  3. If tool calls exist → run tools → append results as role=tool messages
  4. No tool calls → done, return text
  5. Send tool results back into history → loop again
```

## When adding new feature

1. Write an ADR in `docs/adr/` if the feature affects architecture (see [AGENTS.md](AGENTS.md#architecture-decision-records))
2. Add a plugin under `plugins/` and register it in `plugins/builtin/builtin.go`

## Done

- `str_replace` tool plugin — [ADR-0001](docs/adr/0001-str-replace-for-file-editing.md)
- Load skill (two-phase discovery) — [ADR-0002](docs/adr/0002-load-skill-two-phase-discovery.md)
- Streaming runner plugin — [ADR-0003](docs/adr/0003-streaming-llm-responses.md)
- Prompt caching — [ADR-0004](docs/adr/0004-prompt-caching.md)
- Session management — [ADR-0005](docs/adr/0005-session-management.md)

## Road map


- **Permission hook plugin**
- **Context compaction**
- **Sub-agents / task spawning**
- **External search: web fetch + web search**
- **Parallel tool execution**
- **internal search: grep*
- **Thinking level / reasoning tokens**
- **TODO / plan tracking**
- **Error recovery / retry policy**
- **Diff preview before apply**
- **Codebase indexing + vector db**
- **LSP integration**
- **Hashline edit** File state / staleness check 
- **Cost / token tracking**
- **MCP client**
- **File reference (@file)**
- **More tools: see ./tools.md**
- **Custom model and provider**
- **TUI implementation**
- **tool search** [tool search](https://code.visualstudio.com/blogs/2026/06/17/improving-token-efficiency-in-github-copilot#_tool-search)
- **Extension system and management**
- **Routing (auto model selection)**
- **Handoffs (share context between agents)**
- **Observability and logging**
- **full customize**

## Internal
- **prompt optimizer** eg. DSpy