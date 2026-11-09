---
id: TASK-211
title: >-
  walksmooth3: batch the bot reads in the d2dview mux, no left-skill popup on
  d2d
status: Done
assignee:
  - credit-runner
created_date: '2026-11-09T13:52:00Z'
updated_date: '2026-11-09T14:29:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: medium
ordinal: 163000
card: "walksmooth3"
owns:
  - "docs/walksmooth.md"
  - "cmd/d2dview"
  - "internal/combat/fight.go"
exclusive: []
contracts: []
result: "mux batch fix measured live in the dev container (snapshot 170 to 15 ms); left-skill skip live-observed (no stall lines) but no discriminating unit test"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Paladin still jerky after approachfast. Done when snapshot cost and pause causes are measured and the dominant one fixed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 mux batches pipelined bot lines (test)
- [x] #2 swinger backend skips SetLeft
- [x] #3 numbers in docs/walksmooth.md
<!-- AC:END -->
