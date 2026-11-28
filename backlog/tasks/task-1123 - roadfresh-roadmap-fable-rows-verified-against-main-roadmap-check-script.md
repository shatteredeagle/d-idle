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
- [ ] #1 every row of docs/roadmap-fable.md checked against git log origin/main, the code and backlog/tasks; states, next-up and what-remains text fixed
- [ ] #2 tools/roadmap/check.py lists rows whose card is Done or merged while the STATE says not started (warning, exit 0), with a test
- [ ] #3 rule in the doc names the check; proposal for the CI warning line in docs/probes/roadmap-refresh-2026-11-28.md
<!-- AC:END -->
