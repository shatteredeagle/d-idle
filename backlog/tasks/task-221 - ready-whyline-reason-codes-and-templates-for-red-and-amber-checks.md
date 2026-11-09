---
id: TASK-221
title: 'ready-whyline: reason codes and templates for red and amber checks'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-09T14:27:00Z'
labels:
  - watch
  - feature
  - fidelity
dependencies:
  - TASK-214
priority: medium
ordinal: 170000
card: "ready-whyline"
owns:
  - "internal/bot/whyline.go"
  - "internal/bot/whyline_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Source: docs/research/player-power.md Part 3.4 (TASK-214 follow-up). Whyline explains why the planner holds back or retreats. Done when held-back steps show the reason.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Reason codes and templates (2 phrasings each) for red/amber checks; held-back planner steps show the reason
- [ ] #2 Build flexibility: checks, floors and boss rows are data (per class and difficulty), no class or skill literal in Go; tests run with Paladin and Sorceress; an unknown class or table is an error naming it
<!-- AC:END -->
