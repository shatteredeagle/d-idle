# Agent brief template (12 lines)

> status: current (TASK-256, user approved the reading diet 2026-11-10; source: docs/research/project-review-2026-11-10.md section 6.2). The agent definition carries the standing rules (identity, trailers, never push, checks, report format); the brief does not repeat them.

Fill one line each; leave out a line that does not apply.

1. Agent name and branch (`agent/<name>`), worktree path.
2. Goal in one sentence.
3. Task: `TASK-<n>` (the file carries owns, exclusive, contracts, acceptance).
4. Owns / exclusive resources (only if the task file does not say).
5. Read (three files at most): the task, the area doc, one design doc.
6. Already ruled out or already known (so the agent does not redo it).
7. Evidence to start from (paths, commits, logs).
8. Acceptance: "as in the task" or the one extra check.
9. Scope: "offline only" or the instance (`iN`) and character.
10. Size: one PR, one package family, about 400 lines of Go plus tests, one doc, 60 to 90 minutes.
11. Report length (lines or words).
12. Trailers: `Claude-Session: <url>`, `Change: <card>`.

Do not ask for shared-file lines (BACKLOG, CHANGELOG, INDEX, OWNERS, STATUS): the merge generates them from the task card, so line 3 (the task) must name a card whose title, `result:` and acceptance boxes the agent keeps true. A line 12 `Change:` trailer is optional.

Do not list the standing reading (START-HERE, backlog-howto, memory doc, glossary): AGENTS.md says what is mandatory and what is on demand.
