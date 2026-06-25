---
name: coding-agent
description: Teaches the agent how to add tools, providers, prompts, or runners following this coding-agent project's architecture. Use when the user asks to add a new feature to the repo.
---

# Coding Agent Architecture Guide

Read [AGENTS.md](../../../AGENTS.md) before working.

## Core rules

1. **Contract** → core package (`agent/`, `types/`, `llm/`, `tools/`, `config/`, `plugin/`)
2. **Implementation** → `plugins/` only
3. Register every plugin in `plugins/builtin/builtin.go`
4. Features that affect architecture → write an ADR in `docs/adr/` first

## Add a tool

1. Create `plugins/tools/mytool/my_tool.go`
2. Implement `tools.Tool` (`Name`, `Definition`, `Execute`)
3. Add a `Plugin` struct + `Register()` calling `plugin.RegisterTools()`
4. Append to `builtin.Default`

## Add a provider

1. Create `plugins/providers/myprovider/myprovider.go`
2. Implement `llm.Provider` (`Complete`)
3. Add a constant in `config/config.go`
4. Register in `builtin.Default`

## Edit files

- Read before editing with `read_file`
- Edit existing files with `str_replace` (primary)
- Create new files with `write_file`
