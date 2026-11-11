# Agent brief template (12 lines)

> status: current (TASK-256, TASK-310; user approved the reading diet 2026-11-10; source: docs/research/project-review-2026-11-10.md section 6.2). The agent definition and [rules-short.md](rules-short.md) carry the standing rules (identity, trailers, never push, checks, models, sizing); the brief does not repeat them.

Fill one line each; leave out a line that does not apply.

1. Agent name and branch (`agent/<name>`), worktree path.
2. Goal in one sentence.
3. Task: `TASK-<n>` (the file carries owns, exclusive, contracts, acceptance).
4. Owns / exclusive resources (only if the task file does not say).
5. Read (three files at most): the task, the area doc (`docs/digest/<area>.md`), one design doc.
6. Already ruled out or already known (so the agent does not redo it).
7. Evidence to start from (paths, commits, logs).
8. Acceptance: "as in the task" or the one extra check.
9. Scope: "offline only" or the instance (`iN`) and character.
10. Size: one PR, one package family, about 400 lines of Go plus tests, one doc, 60 to 90 minutes (one agent = one PR family).
11. Report length: under 250 words unless stated.
12. Trailers: `Claude-Session: <url>`, `Change: <card>`. Rules: `~/code/d2lod-bot/docs/agents/rules-short.md`.

## Brief size budget
- The brief is at most 12 lines and about 250 words. Paths, not pasted content. If the brief needs more, the task file is missing detail: put it in the task.
- Name the model when it is not Sonnet (see the table in rules-short.md). Haiku for mechanical, Opus singly, Fable only on the user's word.
- Do not list the standing reading (START-HERE, backlog-howto, memory doc, glossary): AGENTS.md says what is mandatory and what is on demand.
- Reuse before spawning: a running agent that owns the area gets a SendMessage, not a new agent.
