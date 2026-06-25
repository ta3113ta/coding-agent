# ADR-0002: Load Skill with two-phase discovery

**Status:** Accepted  
**Date:** 2026-06-14

## Context

The coding agent had a static system prompt via the `coding` prompt plugin injected only at bootstrap — no way to add procedural knowledge per project or user workflow.

Cursor Agent Skills solve this with `SKILL.md` (YAML frontmatter + markdown body) that the agent loads when a task is relevant, instead of putting all instructions in the system prompt for the entire session.

Phase 2 needed skill loading that:
- is compatible with the Cursor SKILL.md format
- saves tokens (index only in the system prompt)
- does not change the agent loop

## Decision

Use **two-phase lazy loading** with the **existing `read_file` tool**:

1. **Bootstrap** — discover `SKILL.md` from 3 sources, then merge by priority
2. **Index** — inject only `name` + `description` + `path` into the system prompt
3. **Full load** — agent reads `SKILL.md` with `read_file` when the task matches the description

### Discovery paths (priority: project > personal > bundled)

| Source | Path |
|--------|------|
| Project | `<workspace>/.cursor/skills/*/SKILL.md` |
| Personal | `~/.cursor/skills/*/SKILL.md` (disable with `SKILLS_ENABLE_PERSONAL=false`) |
| Bundled | `plugins/skills/builtin/*/SKILL.md` (embed + extract to temp dir for `read_file`) |

Implementation:
- Core contract: [`skills/`](../../skills/)
- Plugin: [`plugins/skills/skills.go`](../../plugins/skills/skills.go)

## Alternatives Considered

| Approach | Token efficiency | Reason not chosen for v1 |
|--------|------------------|--------------------------|
| Two-phase index + `read_file` | Highest | **Chosen** — no agent loop changes |
| Eager load all skill bodies into prompt | Low | High token cost with many skills |
| Separate `load_skill` tool | Medium | Requires injecting agent state into tool; `read_file` is enough |
| Compile-time prompt plugin only | High | Not dynamic per project/user |
| REPL `/skill-name` explicit invoke | Good for slash commands | Deferred v2 — still relies on model routing from index |
| Scan `~/.cursor/skills-cursor/` | — | Cursor internal — not used |

## Consequences

**Pros**

- No changes to [`agent/agent.go`](../../agent/agent.go) — `systemPrompt` is static per session but includes the skill index at bootstrap
- Compatible with Cursor SKILL.md format (`name`, `description`, `disable-model-invocation`)
- Token-efficient — descriptions only in the prompt for the whole session
- Bundled skills via `embed.FS` — ship with the binary

**Cons / trade-offs**

- Relies on the model deciding which skill is relevant from the description
- Bundled skills extract to a temp dir — path changes every session
- No `environments` / `metadata.surfaces` filtering in v1
- No explicit `/skill-name` in REPL v1

## Skill Index Format

```
## Available Skills

When the user's task matches a skill description:
1. Read SKILL.md with read_file before working.
2. Follow the instructions in the skill immediately.
3. If a skill has disable-model-invocation, do not load it unless the user explicitly names the skill.

<available_skills>
- commit-message (/path/to/SKILL.md): generate git commit message...
</available_skills>
```

## v2 (deferred)

- REPL `/skill-name` explicit invocation
- `environments` / `metadata.surfaces` filtering
- `embed.FS` path instead of temp dir extraction (if `read_file` supports virtual FS)
- Skill hot-reload during session

## References

- Cursor `create-skill` SKILL.md spec (`~/.cursor/skills-cursor/create-skill/SKILL.md`)
- Cursor two-phase skill index model (`<agent_skills>` block)
