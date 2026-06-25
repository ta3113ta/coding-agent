# Cursor Tools

Tools available to the AI assistant in Cursor (not the `coding-agent` project tools like `read_file`, `str_replace`, etc.).

**Note:** This project implements its own `grep` and `glob` tools (via ripgrep) — see [ADR-0009](docs/adr/0009-grep-glob-internal-search.md).

## Core file & code tools

| Tool | What it does |
|------|----------------|
| **Read** | Read file contents (text and images: jpg, png, gif, webp) |
| **Write** | Create or overwrite files |
| **StrReplace** | Exact string replacements in files |
| **Delete** | Delete a file |
| **Glob** | Find files by glob pattern (e.g. `**/*.go`) |
| **Grep** | Ripgrep search (content, files, counts) |
| **SemanticSearch** | Meaning-based code search (“how does X work?”) |
| **ReadLints** | Read IDE/linter diagnostics for files |
| **EditNotebook** | Edit Jupyter notebook cells |

## Terminal & execution

| Tool | What it does |
|------|----------------|
| **Shell** | Run shell commands (with sandbox/permission options) |
| **Await** | Poll/wait on background shell tasks |

## Web & media

| Tool | What it does |
|------|----------------|
| **WebSearch** | Search the web for current information |
| **WebFetch** | Fetch and read a URL as markdown |
| **GenerateImage** | Generate an image from a text description (only when explicitly requested) |

## Workflow & interaction

| Tool | What it does |
|------|----------------|
| **TodoWrite** | Create/update a structured task list |
| **AskQuestion** | Ask structured multiple-choice questions |
| **Task** | Launch subagents (explore, shell, bugbot, security-review, etc.) |

## MCP (Model Context Protocol)

| Tool | What it does |
|------|----------------|
| **CallMcpTool** | Invoke tools from connected MCP servers |
| **FetchMcpResource** | Read resources from MCP servers |

### Connected MCP servers

- **GitLens / GitKraken** (`user-eamodio.gitlens-extension-GitKraken`) — Git-related MCP tools
- **Context7** (`plugin-context7-context7`) — Fetch up-to-date library/framework documentation

## Agent skills

Skills are instruction files the assistant reads when relevant (not callable tools):

- **Cursor built-in:** create-skill, create-rule, create-hook, canvas, SDK, split-to-prs, babysit, loop, statusline, review-bugbot, review-security, automate, update-cursor-settings
- **Understand Anything:** understand, understand-chat, understand-dashboard, understand-diff, understand-domain, understand-explain, understand-knowledge, understand-onboard
- **Other:** find-skills

## Not built-in

Unless exposed via MCP or shell, the assistant does not have direct tools for:

- Browser automation
- Database clients
- Email, calendar, or arbitrary third-party APIs
