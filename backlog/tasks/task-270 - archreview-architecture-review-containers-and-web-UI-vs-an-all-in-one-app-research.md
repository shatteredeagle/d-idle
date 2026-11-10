---
id: TASK-270
title: >-
  archreview: architecture review, containers and web UI vs an all-in-one app
  (research)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T21:21:00Z'
updated_date: '2026-11-10T21:51:00Z'
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
result: "Docs only (docs/research/architecture-review-2026-11-10.md); nothing built, deployed or run live. Verified by commands: Go cross-builds per OS (one OS-specific package, internal/launch), d2d CI matrix and the Windows devctl stub at the shipped pin, sandbox image layers and resource use, Docker Desktop requirements. Recommendation: hybrid (one release layout, native per OS plus the container as one packaging), hosted beta first; open questions in section 1, proposed decision text in section 12. Whole-repo gate on the worker host exit 0."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the user asked whether the containerized path with a web UI should give way to an all-in-one app for an idle game, sunk cost ignored. Done when: the research doc answers with a matrix, costs, risks, migration plan and open questions, and the whole-repo gate passes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/research/architecture-review-2026-11-10.md gives the questions, the decision matrix (A status quo, B native all-in-one, C hybrid, D others), costs in PRs and agent-hours, risks, a migration plan and proposed decision text, every claim tagged V/S/U
- [x] #2 feasibility facts are tested, not guessed: Go cross-builds per OS, d2d platform support and control channel, container contents and resource use, Docker Desktop requirements
<!-- AC:END -->
