---
id: TASK-67
title: 'g2gfixmouth: nav mouth gate no safe click loop at repeated spots'
status: In Progress
assignee: []
created_date: '2026-11-07T07:05:00Z'
updated_date: '2026-11-07T07:19:00Z'
labels:
  - bot-defect
dependencies: []
priority: high
ordinal: 65000
card: "g2gfixmouth"
owns:
  - "internal/nav/mouthgate.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
docs/research/g2g-diagnosis.md finding N1: 379/468/187 click held (no safe click) lines per seed, the same 3 spots repeated 38 to 56 times; detour fired 10/17/2 times.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 inside-disc and outside-disc holds at the recorded spots resolve or escalate within 3 holds
- [ ] #2 offline test from the recorded from/to pairs
<!-- AC:END -->
