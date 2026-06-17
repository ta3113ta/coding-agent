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

CLI flags override env values.

Then try prompts such as:
- `create a fizzbuzz.go file that prints 1-20, then build it to verify it works`
- `read main.go and explain how it works`

The REPL streams assistant text token-by-token instead of waiting for the full response — see [ADR-0003](docs/adr/0003-streaming-llm-responses.md).

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

## Road map


- **Prompt caching**
- **Session management**
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
- **full customize**
