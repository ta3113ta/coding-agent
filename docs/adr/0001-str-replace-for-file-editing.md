# ADR-0001: Use str_replace for file edits instead of write_file

**Status:** Accepted  
**Date:** 2026-06-14

## Context

The coding agent already had `read_file` and `write_file`, but `write_file` overwrites the entire file on every edit — forcing the LLM to send output tokens equal to the full file size even when only a few lines change.

Phase 2 needed an edit tool that saves tokens for typical file edits in the agent loop.

## Decision

Use **`str_replace`** (exact match + uniqueness check + optional `replace_all`) as the **primary edit tool**, alongside:

- `read_file` — inspect before editing (with line numbers)
- `write_file` — create new files, or fallback after multiple str_replace failures

Implementation: [`plugins/tools/strreplace/str_replace.go`](../../plugins/tools/strreplace/str_replace.go)

## Alternatives Considered

| Approach | Token efficiency | Reason not chosen for v1 |
|--------|------------------|--------------------------|
| `str_replace` | Highest (~1-5% of file) | **Chosen** |
| `apply_patch` (unified diff) | Good (~5-15%) | High retry rate, fragile line offsets |
| line-range edit | Medium | line numbers go stale after multiple edits in one session |
| fuzzy matching (Aider/Cline) | Good (fewer retries) | Risk of silent wrong edits in Go (imports, struct tags, string literals) |
| `write_file` (full overwrite) | Lowest (100%) | Kept as fallback only |

## Consequences

**Pros**

- Saves ~90-95% output tokens for typical edits (e.g. editing 8 lines in a 400-line file: ~80-120 tokens instead of ~1,600)
- Matches industry practice: Cursor `StrReplace`, Anthropic `str_replace_based_edit_tool`, OpenHands `str_replace_editor`
- Clear separation of concerns in the plugin architecture — no monolithic editor needed

**Cons / trade-offs**

- Model must `read_file` first and copy `old_string` with exact whitespace/indentation
- No multi-hunk atomic edit in one tool (deferred to `apply_patch` v2)
- Tab/space mismatch handled by `expandTabs` only — no fuzzy cascade in v1

## Tool Contract

### Schema

```json
{
  "name": "str_replace",
  "required": ["path", "old_string", "new_string"],
  "properties": {
    "path": "string — file path",
    "old_string": "string — exact match text",
    "new_string": "string — replacement text (empty = delete)",
    "replace_all": "boolean — default false"
  }
}
```

### Semantics

| `replace_all` | matches | behavior |
|---------------|---------|----------|
| `false` | 0 | error + hint to re-read |
| `false` | 1 | replace |
| `false` | >1 | error + line numbers |
| `true` | ≥1 | replace all |
| `true` | 0 | error |

### Implementation details

1. Read file from disk immediately in `Execute()` — do not trust conversation context
2. Normalize: `\r\n` → `\n`, `\t` → 4 spaces (before match)
3. Preserve line ending style and file mode when writing back
4. Success response: snippet ±4 lines around the edit with line numbers (same format as `read_file`)

### Error examples

```
# 0 matches
old_string not found in main.go. Re-read with read_file before trying again.

# >1 matches
old_string matched 3 times at lines [14 52 103]. Add context to old_string or set replace_all=true
```

## Token analysis (example)

Editing a nil-check in a 400-line Go file (~8 lines changed):

| Tool | Output tokens (approx.) |
|------|-------------------------|
| `write_file` | ~1,600 |
| `str_replace` | ~80-120 |

Input tokens are similar (model still reads the file), but **output tokens in tool-calling agents are more expensive** — `str_replace` reduces that cost significantly.

## v2 (deferred)

- Line-trimmed fallback (2nd pass if exact match fails)
- `insert` at line (e.g. add import at top)
- `apply_patch` (multi-file atomic refactor)

## References

- Cursor `StrReplace` tool
- Anthropic `str_replace_based_edit_tool` (computer-use demo)
- OpenHands `str_replace_editor`
- SWE-Edit / AdaEdit papers — search-replace accuracy > unified diff for LLM code editing
