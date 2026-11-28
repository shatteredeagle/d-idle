---
id: TASK-1123
title: 'roadfresh: roadmap-fable rows verified against main, roadmap check script'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T06:38:00Z'
labels:
  - docs
  - chore
dependencies: []
priority: medium
ordinal: 533000
card: "roadfresh"
result: "Docs and one read-only script, no game run, nothing verified live. All 84 rows of docs/roadmap-fable.md checked against origin/main 01b93bd88, backlog/tasks and the ledger: 15 rows changed STATE (12 of them were not started with merged code), stale text fixed in about 20 more, Next up, Flips ready, Built but dark, Order of work and the plan snapshot rewritten; knob citations by name. tools/roadmap/check.py (warning only, exit 0, with a test) lists rows whose cards are Done or merged while the STATE says not started or in flight; rule in the page; the check.sh line is a proposal in docs/probes/roadmap-refresh-2026-11-28.md (check.sh is not owned). Not build-specific. Open: the review of 2026-11-28 has no rows yet."
lane: now
owns:
  - "docs/roadmap-fable.md"
  - "docs/probes/roadmap-refresh-2026-11-28.md"
  - "tools/roadmap/"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: docs/roadmap-fable.md marked merged work as not started for the third time. Done when: rows match origin/main and a read-only check warns on the next drift.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 every row of docs/roadmap-fable.md checked against git log origin/main, the code and backlog/tasks; states, next-up and what-remains text fixed
- [x] #2 tools/roadmap/check.py lists rows whose card is Done or merged while the STATE says not started (warning, exit 0), with a test
- [x] #3 rule in the doc names the check; proposal for the CI warning line in docs/probes/roadmap-refresh-2026-11-28.md
<!-- AC:END -->
