# Agent rules, short (stable path; every brief links this file)

> status: current (TASK-310). Replaces the per-session "overnight-rules" scratchpad file that was missing in later sessions. The binding text is AGENTS.md; this is the page a brief points to.

## Sizing
1. One agent = one PR = one package family: about 400 lines of Go plus tests, one doc, 60 to 90 minutes. Above about 400 k tokens an agent is in rollover and its summary loses detail: split it.
2. Shared registry/data files get one owner per wave; feature agents run after, never in parallel on the same file.
3. Docs-only work: batch three or more docs into one branch; a research doc needs a named consumer card, else it is a section of an existing doc.
4. Report under 250 words unless the brief says otherwise; no logs pasted.

## Reading (no standing re-reading)
5. Read: START-HERE, your task, the area doc (docs/digest/<area>.md first). On demand only: memory doc, GLOSSARY, contracts README, the rest of backlog-howto. Never re-read a file you already read this run.
6. Brief size budget: at most 12 lines (docs/agents/brief-template.md); the agent definition and this file carry the standing rules, a brief never repeats them.

## Model table (review 2026-11-10 section 6.4)
| Work | Model |
|---|---|
| status, drift, merge-lines, cards, archive stubs, link fixes, running a given command | Haiku |
| implementation, bot fixes, live checks, oracles, most docs | Sonnet (default) |
| review of a merged family, merge-risk read, hard bug after two Sonnet failures | Opus, singly |
| direction and whole-project reviews, design partner rounds | Fable, only when the user asks |

## When NOT to spawn an agent
7. A lookup or edit of a few tool calls; a check you can run inline; a question an existing running agent can answer (SendMessage it); a fix inside the current agent's branch; a second agent on a symptom another card owns; "while we wait" work; research with no consumer card.

## Always (details in AGENTS.md)
8. Own worktree and branch `agent/<name>`; commit often; never push, never `git stash`, never pkill, never edit docs/adr, GLOSSARY.md, docs/decisions.md unless named.
9. Commit trailers: Co-Authored-By, Claude-Session, Change. Shared-file lines go in `backlog/merge-lines/<branch>.txt`.
10. Processes only through `tools/agentrun.sh`; reap before the report. Whole-repo go test/vet only via `labrun orion`.
11. Tag claims V (say how), S or U. No live game unless the brief names the instance.
12. Cap: at most 6 agents run at once (docs/process/coordinator-efficiency.md).
