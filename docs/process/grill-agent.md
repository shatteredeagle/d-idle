# The grill agent: one command to be interviewed

Tags: V = read in the Claude Code docs today (fetched by a claude-code-guide agent, 2026-10-30; local CLI 2.1.314, `claude --help` shows `claude [options] [command] [prompt]` and `--agent <agent>`), U = not yet seen working in a live session.

## Command the user runs
```
tools/grill.sh 0001 d2d-primary-backend
```
It refuses if master is dirty or behind forge, if the worktree or branch exists, or if there is no prebrief. Otherwise it creates `~/code/d2lod-wt/grill-0001` on branch `grill/0001-d2d-primary-backend`, prints, and runs `claude --agent grill "grill topic 0001"` there. `--print-only` prints both commands and does nothing. Needs no user setting changes (U: the first start asks to trust the workspace, accept it; project-agent hooks are skipped otherwise).

## Verified mechanism
| question | answer | source |
|---|---|---|
| agent as main agent | `claude --agent <name>` or `"agent"` in settings; flag wins; the agent body replaces the default system prompt (V) | sub-agents.md, cli-reference.md, settings.md |
| frontmatter | required `name`, `description`; optional `tools`, `disallowedTools`, `model`, `permissionMode`, `skills`, `hooks`, `mcpServers`, `maxTurns`, `memory`, `initialPrompt` (main session only), ... (V) | sub-agents.md |
| preload skills | `skills:` injects full skill content (V), but not skills with `disable-model-invocation: true` (V). grill-me has that flag (V, SKILL.md), so it cannot be preloaded or called by the agent | skills.md |
| path-scoped writes | `tools` cannot scope by path; permission rules are session-wide, not per agent (V). Only a frontmatter `PreToolUse` hook is per agent (V) | permissions.md, sub-agents.md |
| `tools: Bash(git log *)` | not documented, so not used (V) | sub-agents.md |
| start in a directory with a first message | no cwd flag; `cd dir && claude --agent grill "prompt"` (V grammar, U live) | cli-reference.md |

## Decisions
- The grilling procedure is embedded in `.claude/agents/grill.md` (the `grilling` skill text, V from its SKILL.md in plugin 1.3.1), because grill-me cannot be invoked by an agent. If the plugin text changes, update the agent.
- `tools: Read, Grep, Glob, Write, Edit, Bash` plus a `PreToolUse` hook `tools/grill-guard.sh` that exits 2 unless: Write/Edit targets `docs/adr/NNNN-slug.md`, `docs/decisions.md` or `docs/grills/NNNN-slug/outcome.md`; Bash is `git log|status|diff|show|branch|remote`, `ls`, `cat`, `git add docs/...`, the project commit, or `git push forge grill/<name>`, with no shell operators. Tested offline with sample payloads (V); payload field names are from the docs only (U, check in the first real session: if the guard blocks everything or nothing, `claude --debug` shows the hook input).
- The prebrief's author recommendation is withheld until the key questions are answered (prompt rule, U).

## If it does not work
Fallback: the paste-in text in [grill-workflow.md](grill-workflow.md) section 3b. If `--agent` is unknown in your CLI version, upgrade Claude Code.
