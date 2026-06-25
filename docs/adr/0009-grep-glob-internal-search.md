# ADR-0009: Grep + Glob internal search

**Status:** Accepted  
**Date:** 2026-06-25

## Context

The agent had `list_dir` and `read_file` for exploration but no first-class search tools. Codebase search went through `run_bash` (`grep`, `find`, `rg` ad hoc) — unstructured params, inconsistent output, no caps, and easy to search ignored paths like `node_modules`.

Cursor exposes dedicated **Grep** (ripgrep) and **Glob** tools with structured schemas and `.gitignore`-aware results.

## Decision

Add **`grep`** and **`glob`** tool plugins that shell out to **`rg`** (ripgrep) on PATH:

1. **Shared helper** — [`plugins/tools/rghelper/`](../../plugins/tools/rghelper/): resolve `rg`, run with context, truncate output
2. **`grep` tool** — [`plugins/tools/grep/`](../../plugins/tools/grep/): content / files_with_matches / count modes; Cursor-aligned params (`pattern`, `path`, `glob`, `-i`, context lines, `type`, `head_limit`, `offset`, `multiline`)
3. **`glob` tool** — [`plugins/tools/glob/`](../../plugins/tools/glob/): `rg --files -g '<pattern>'`; auto-prepend `**/` when pattern is not recursive; sort by mtime descending
4. **Integrations** — register in builtin; add to explore sub-agent profile; auto-allow in interactive permission hook; steer prompt away from `run_bash` for search

**Prerequisite:** `ripgrep` installed (`brew install ripgrep`, `apt install ripgrep`, etc.)

## Alternatives Considered

| Approach | Reason not chosen |
|----------|-------------------|
| `run_bash` only (status quo) | Unstructured; poor token efficiency; no schema for LLM |
| Pure-Go directory walk + glob | Does not respect `.gitignore` without extra logic |
| Embed `rg` binary | Larger binary; deferred |
| Single combined `search` tool | Cursor separates grep vs glob; clearer LLM tool choice |

## Consequences

**Pros:**
- Fast, `.gitignore`-aware search matching Cursor behavior
- Read-only tools — no permission prompts
- Explore sub-agent can search without `run_bash`

**Cons:**
- Requires `rg` on PATH; clear error if missing
- No `.cursorignore` support in v1
- Glob mtime sort requires post-processing (rg does not sort by mtime)

## Deferred (v2+)

- `.cursorignore` / custom ignore files
- Semantic search / vector index
- Bundled ripgrep binary
