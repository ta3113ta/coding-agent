# ADR-0006: Permission hooks before tool execution

**Status:** Accepted  
**Date:** 2026-06-21

## Context

Tools such as `run_bash`, `write_file`, and `str_replace` execute immediately when the LLM requests them — no user consent, no policy layer, and no extension point for project-specific rules. For a coding agent that runs shell commands and writes files, this is a safety gap.

## Decision

Add a **permission hook chain** invoked in the agent loop before `registry.Dispatch`:

1. **Core contract** — [`permission/permission.go`](../../permission/permission.go): `Hook`, `Chain`, `ToolUseRequest`, `Result` with decisions `Allow`, `Deny`, `Ask`
2. **Script plugin** — [`plugins/permission/script/`](../../plugins/permission/script/): load `.coding-agent/hooks.json`, run `preToolUse` command hooks with JSON stdin/stdout (Cursor-compatible subset)
3. **Interactive plugin** — [`plugins/permission/interactive/`](../../plugins/permission/interactive/): REPL y/n prompt for risky tools when the chain reaches `Ask` or no prior hook decided

**Chain semantics:**

| Decision | Behavior |
|----------|----------|
| Allow | Continue to next hook; all Allow → execute tool |
| Deny | Stop; skip dispatch; append `IsError=true` tool message |
| Ask | Interactive plugin prompts user; approve → Allow, reject → Deny |

Hook errors fail closed (Deny). Optional `updated_input` from script hooks rewrites tool input before dispatch.

**Registration order:** script → interactive (policy first, user confirmation last).

**Config:** `PERMISSION_ENABLED` (default `true`), `PERMISSION_HOOKS_FILE` (default `.coding-agent/hooks.json`), `--no-permission` disables all hooks.

## Alternatives Considered

| Approach | Reason not chosen |
|----------|-------------------|
| Inline checks in each tool | Duplicated logic; hard to extend |
| Interactive only | User chose both script + interactive |
| Full Cursor hook event set v1 | Too broad; defer `postToolUse`, `beforeShellExecution`, MCP |
| Runtime `.so` plugins | Deferred per AGENTS.md |
| Deny stops agent loop | LLM should see denial and adapt (same as tool errors) |

## Script hook format (v1 subset)

Config: `.coding-agent/hooks.json`

```json
{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "command": ".coding-agent/hooks/block-rm.sh",
        "matcher": "run_bash",
        "timeout": 5,
        "failClosed": true
      }
    ]
  }
}
```

Hook stdin:

```json
{"tool_name":"run_bash","tool_input":{"command":"rm -rf /"},"tool_call_id":"tc_1"}
```

Hook stdout:

```json
{"permission":"allow"}
{"permission":"deny","agent_message":"blocked by policy"}
{"permission":"ask","user_message":"Network command detected"}
{"permission":"allow","updated_input":{"command":"curl --dry-run example.com"}}
```

**Matcher:** v1 matches `tool_name` via Go `regexp` (JavaScript-style patterns in config — document caveat). Command-level regex (`beforeShellExecution`) deferred to v2.

**Exit codes:** `0` = parse stdout JSON; `2` = deny; other non-zero + `failClosed` = deny.

## Consequences

**Pros**

- Single interception point before all tools
- Script hooks enable deterministic project policy (check into repo)
- Interactive layer catches anything scripts allow through
- Denied tools feed back to LLM like other tool errors

**Cons / trade-offs**

- One hook evaluation per tool call per loop iteration
- Interactive prompts block REPL during agent turn (stdin shared with runner)
- Non-REPL runners need injectable prompter later
- Matcher uses Go regexp, not full JS regex engine

## v2 (deferred)

- `postToolUse`, `beforeShellExecution`, MCP events
- Session-persistent "always allow" rules
- Prompt-type hooks (`type: "prompt"`)
- Injectable `Prompter` on `AgentHandle` for HTTP/non-REPL runners
