# Coordinator efficiency checklist

> status: current (TASK-310; source: docs/research/project-review-2026-11-10.md section 6 and docs/research/usage-2026-11-11.md). For the main (coordinator) session. Tools: `python3 tools/usage/usage.py` shows where tokens went.

1. **Batch deploys.** Deploy the sandbox in windows: at least 2 merged PRs or a request (docs/dev-update.md, `tools/play-sandbox/deploy-window.sh`). Deploy on the user's request at any time.
2. **Fewer hand-offs.** One agent = one PR family (docs/agents/rules-short.md). Do not spawn for a lookup, a check you can run inline, or a fix inside a running agent's branch. Docs-only work is batched, three docs per branch or more.
3. **Reuse running agents.** Send a follow-up with SendMessage to the agent that owns the area instead of a new spawn: it keeps its context (a new agent re-reads about 35 k tokens of standing docs).
4. **No polling loops.** No `until ... sleep` waiters, no repeated status calls to see whether something finished: agents notify on completion. A wait is one bounded foreground command.
5. **Terse replies.** One line per event to the user; reports from agents are under 250 words; never paste logs; link the file.
6. **Cap concurrent agents at 6.** The small coordinator box has 4 cores and 4 GB; more agents (each with its own tools and test runs) thrashed it into a reboot. Heavy work goes to the worker host via labrun; refill an agent slot only when one finishes.
7. **Right model.** Haiku for mechanical, Sonnet default, Opus singly, Fable only when the user asks (table in rules-short.md).
8. **Research has a consumer.** A research card names the card that will use it (`Consumer:` in the description; `tools/backlog/check.py` warns). One doc per card family; run records under `docs/probes/<family>/`; research with no inbound link after 7 days is archived (`git mv` to docs/archive/ with a one-line stub).
9. **Look at the numbers weekly.** `python3 tools/usage/usage.py --days 7`: top 10 agents, tokens per merge. Any single agent above about 2 M fresh tokens gets a look at why.
