---
id: TASK-180
title: 'palready: Might aura once per fight and recovery leaves a depleted starter area earlier'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T05:37:00Z'
labels:
  - bot
  - bug
milestone: m-4
dependencies: []
priority: high
ordinal: 150000
card: "palready"
owns:
  - "docs/palready.md"
  - "internal/combat/aura.go"
  - "internal/bot/drought.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Two Paladin readiness defects from docs/palmelee.md. Done when both are fixed and measured.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Might comes up once through the select verb on d2d and the check does not spam
- [ ] #2 Recovery in a dry field moves on or starts a fresh game within bot.recoveryDryMin minutes
- [ ] #3 Same lockstep and realtime drills rerun, before/after numbers in docs/palready.md
<!-- AC:END -->
