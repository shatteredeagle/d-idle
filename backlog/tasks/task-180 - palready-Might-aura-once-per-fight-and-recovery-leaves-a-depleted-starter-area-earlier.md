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
  - "internal/combat/bindleft.go"
  - "internal/combat/fighter.go"
  - "internal/combat/class_test.go"
  - "internal/bot/areaskip.go"
  - "internal/bot/nokill_test.go"
  - "internal/env/tuning_bot.go"
  - "cmd/d2bot/d2dwire.go"
  - "docs/knobs.md"
  - "docs/probes/paladinsmoke/"
  - "docs/probes/test-characters.md"
exclusive: []
contracts: []
result: "Might comes up through the right popup on d2d (0 failure lines vs 2485/140; verified live lockstep and realtime, writes after start 0); dry-recovery rule offline only (did not fire in the 22 min realtime drill); docs/palready.md"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Two Paladin readiness defects from docs/palmelee.md. Done when both are fixed and measured.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Might comes up once through the right popup on d2d and the check does not spam
- [x] #2 Recovery in a dry field moves on or starts a fresh game within bot.recoveryDryMin minutes
- [x] #3 Same lockstep and realtime drills rerun, before/after numbers in docs/palready.md
<!-- AC:END -->
