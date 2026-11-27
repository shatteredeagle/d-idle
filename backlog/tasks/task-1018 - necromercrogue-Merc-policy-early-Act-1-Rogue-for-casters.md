---
id: TASK-1018
title: 'necromercrogue: Merc policy: early Act 1 Rogue for casters'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T19:14:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 489000
card: "necromercrogue"
lane: next
owns:
  - "internal/d2/classtraits.go"
  - "internal/d2/classtraits.json"
  - "internal/merc/policy.go"
  - "internal/merc/policy_test.go"
  - "cmd/d2bot/mercwire.go"
exclusive: []
contracts: []
result: "Policy and data only: class trait mercEarlyRogue (Necromancer true) lets merc.ShouldHire pay for the Act 1 Rogue before Blood Raven; the Act 2 Desert merc after Andariel already exists (ReplaceAtAct2, first Lut Gholein visit). NOT wired: no Act 1 paid hire step exists (HireStep runs in Lut Gholein only), so nothing hires at Kashya yet; follow-up card needs a live read of the Kashya window. Tests on three classes, offline."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 see docs/probes/necro-readiness-2026-11-26.md cards table
<!-- AC:END -->
