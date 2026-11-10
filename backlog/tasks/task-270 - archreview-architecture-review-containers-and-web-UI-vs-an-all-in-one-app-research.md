---
id: TASK-270
title: 'archreview: architecture review, containers and web UI vs an all-in-one app (research)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T21:21:00Z'
labels:
  - docs
  - research
  - tester
  - onboarding
milestone: m-1
dependencies: []
priority: high
ordinal: 197000
card: "archreview"
owns:
  - "docs/research/architecture-review-2026-11-10.md"
  - "backlog/merge-lines/agent-archreview.txt"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Size: M. Why: the user asked whether the containerized path with a web UI should give way to an all-in-one app for an idle game, sunk cost ignored. Done when: the research doc answers with a matrix, costs, risks, migration plan and open questions, and the whole-repo gate passes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 docs/research/architecture-review-2026-11-10.md gives the questions, the decision matrix (A status quo, B native all-in-one, C hybrid, D others), costs in PRs and agent-hours, risks, a migration plan and proposed decision text, every claim tagged V/S/U
- [ ] #2 feasibility facts are tested, not guessed: Go cross-builds per OS, d2d platform support and control channel, container contents and resource use, Docker Desktop requirements
<!-- AC:END -->
