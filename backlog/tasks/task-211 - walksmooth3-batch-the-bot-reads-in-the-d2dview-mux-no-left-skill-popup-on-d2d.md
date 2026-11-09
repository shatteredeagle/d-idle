---
id: TASK-211
title: 'walksmooth3: batch the bot reads in the d2dview mux, no left-skill popup on d2d'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T13:52:00Z'
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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Paladin still jerky after approachfast. Done when snapshot cost and pause causes are measured and the dominant one fixed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 mux batches pipelined bot lines (test)
- [ ] #2 swinger backend skips SetLeft
- [ ] #3 numbers in docs/walksmooth.md
<!-- AC:END -->
