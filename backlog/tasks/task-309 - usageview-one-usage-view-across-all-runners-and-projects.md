---
id: TASK-309
title: 'usageview: one usage view across all runners and projects'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T16:12:00Z'
updated_date: '2026-11-11T16:54:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: medium
ordinal: 212000
card: "usageview"
owns:
  - "tools/usage"
  - "docs/usage.md"
  - "docs/research/usage-2026-11-11.md"
exclusive: []
contracts: []
result: "tools/usage/usage.py (read-only, stdlib) counts per day/agent name/model/project over all ~/.claude/projects; 7-day run in docs/research/usage-2026-11-11.md: 498 subagents, 182 M fresh tokens, 53 percent in the Palimpsest project. Not verified against billing; credit-runner ledgers not included. Offline tool, not live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 tools/usage/usage.py totals per day, agent name and model across all projects
- [x] #2 docs/usage.md and the 7-day table with top offenders written
<!-- AC:END -->
