# Usage view (`tools/usage/usage.py`)

> status: current (TASK-309; source: docs/research/project-review-2026-11-10.md section 6.1 and 6.3 item 5).

One read-only command totals agent and token usage across all Claude Code project directories on this machine.

```
python3 tools/usage/usage.py                      # last 7 days, all projects
python3 tools/usage/usage.py --days 2 --project d2lod-bot
python3 tools/usage/usage.py --json               # machine-readable
```

Output: per day (agents, merges on main, fresh tokens by model, cache reads, fresh tokens per merge), per agent name (first word of the launch description), per project directory, per agent type, the top 10 most expensive agents, and the top coordinator sessions.

## How it counts (V: run on the worker's transcript set, 2026-11-11)
- Source: `~/.claude/projects/*/<session>/subagents/agent-*.jsonl` (one file per subagent, `.meta.json` beside it holds the description and agent type) and the main session `*.jsonl` files (shown as coordinator rows).
- Every assistant message carries the API `usage`; streamed chunks repeat the same message id, so each id is counted once.
- fresh = input + output + cache creation tokens. Cache reads are listed apart: they are about 98 % of all tokens by count but are billed far lower; no prices are quoted. The model comes from the message, so Opus/Fable/Sonnet/Haiku splits are exact, unlike the old ledger's guesses.
- Merges per day: first-parent `Merge agent/*` commits on `main` of the repo (`--repo`). "Fresh per merge" divides ALL projects' tokens of that day by the merges, so use `--project d2lod-bot` for a project figure.

## What it cannot see
- Credit/API runner jobs (they have their own ledgers; they stopped on 2026-11-05) and other machines.
- Transcripts removed by cleanup, and work whose transcript dir is not under the root (`--root`).
- Cost: tokens only. The harness-reported `subagent_tokens` of the older `usage-ledger.py` (ops repo) is a different number (it never matched this one); keep that tool for the per-class view, use this one for totals.
- Agent name is a convention (first word of the description); unnamed agents group under their first word.
