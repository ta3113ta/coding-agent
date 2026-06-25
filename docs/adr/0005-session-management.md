# ADR-0005: File-based session management

**Status:** Accepted  
**Date:** 2026-06-18

## Context

Conversation history lived only in `agent.Agent.messages` in RAM — REPL multi-turn worked within a single process, but history was lost on exit and could not be resumed across runs.

## Decision

Add **file-based JSON session persistence** via the core `session.Store` contract and `plugins/session/filestore` implementation:

- **Persist:** `id`, `created_at`, `updated_at`, `provider`, `model`, `messages[]`
- **Do not persist:** `systemPrompt` — re-resolve from bootstrap on resume (skills/prompt may change between runs)
- **Resume policy:** load messages; use provider/model from current config for subsequent LLM calls (metadata in store is for display in list)
- **Auto-save:** after each successful `Run()`
- **Storage:** default `{cwd}/.coding-agent/sessions/`; `SESSION_SCOPE=global` → `~/.coding-agent/sessions/`; `SESSION_DIR` overrides both
- **UX:** CLI `--resume`, `--new-session`, `--session-scope`, `--session-dir`; REPL `/new`, `/sessions`, `/resume <id>`, `/session`

Implementation:
- [`session/session.go`](../../session/session.go) — `Session`, `Meta`, `Store` interface
- [`config/config.go`](../../config/config.go) — `SESSION_SCOPE`, `SESSION_DIR`, `SessionDir()`
- [`agent/agent.go`](../../agent/agent.go) — persistence hooks around `Run()`
- [`plugins/session/filestore/filestore.go`](../../plugins/session/filestore/filestore.go) — JSON file store
- [`plugins/runner/repl/repl.go`](../../plugins/runner/repl/repl.go) — slash commands
- [`plugin/plugin.go`](../../plugin/plugin.go) — `App.SessionStore`, extended `AgentHandle`

## Alternatives Considered

| Approach | Reason not chosen for v1 |
|--------|--------------------------|
| File-based JSON (`session.Store` + filestore plugin) | **Chosen** — minimal deps, human-readable, fits plugin architecture |
| SQLite | Overkill for learning scope |
| Provider-native session APIs (OpenRouter `session_id`) | Coupled to provider; deferred |
| In-agent RAM only | Does not solve persistence |
| Persist `systemPrompt` | Stale when skills change |

## Consequences

**Pros**

- Resume conversations across process restarts
- Project-local sessions by default; global scope for cross-project resume
- Agent loop logic unchanged — hooks only around `Run()`
- Serialization DTO lives in filestore plugin — `types/` stays provider-neutral

**Cons / trade-offs**

- `--resume` on startup creates an empty session before load (minor orphan file) — **fixed in v2** with `InitNewSession` instead of auto-create in `New`
- No `/delete`, context compaction in v1
- `List` loads metadata from every file — fine for a small number of sessions

## v2 amendment (2026-06-18)

### Context

v1 lacked display name, ephemeral mode, and convenient startup UX (`-c` continue latest, `-r` interactive picker).

### Decision

- **Display name:** add `name` to session JSON; CLI `--name`, REPL `/name <name>`
- **Ephemeral mode:** `--no-session` uses `plugins/session/memory` (in-RAM `Store`, no disk writes)
- **Startup flags:** `-c` resume latest; `-r` interactive picker; flag precedence in `main.go`
- **Startup refactor:** `agent.New` does not auto-create session; `main` calls `InitNewSession` or `ResumeSession` per flags — fixes orphan file on resume paths
- **Default:** no flag → new empty session every time (same as before)

Flag precedence: `--no-session` > `--new-session` > `--resume` > `-c` > `-r`

Implementation additions:
- [`plugins/session/memory/memory.go`](../../plugins/session/memory/memory.go) — ephemeral store
- [`plugins/session/picker/picker.go`](../../plugins/session/picker/picker.go) — `-r` interactive selection
- [`session/session.go`](../../session/session.go) — `Name`, `Latest()`
