---
id: TASK-216
title: 'ready-audit: live read of Player.Resist on Nightmare and Hell characters'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-09T14:27:00Z'
labels:
  - docs
  - probe
  - fidelity
dependencies:
  - TASK-214
priority: medium
ordinal: 165000
card: "ready-audit"
owns:
  - "docs/probes/ready-audit.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). Does Player.Resist include the difficulty penalty? Done when the live read is recorded and the TODO in internal/combat/hard.go is resolved.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Live read of Player.Resist on a Nightmare and a Hell character (zero setup writes) settles whether it includes the penalty; result in docs/probes/; resolves the TODO in internal/combat/hard.go
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
<!-- AC:END -->
