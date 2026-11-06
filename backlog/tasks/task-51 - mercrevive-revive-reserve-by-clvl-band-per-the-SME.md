---
id: TASK-51
title: 'mercrevive: revive reserve by clvl band per the SME'
status: In Progress
assignee: []
created_date: '2026-11-06T21:58:00Z'
labels:
  - bot-defect
dependencies: []
ordinal: 51000
card: "mercrevive"
owns:
  - "internal/merc/knobs.go"
  - "internal/env/tuning_merc.go"
  - "cmd/d2bot/mercwire.go"
  - "cmd/d2bot/mercwire_test.go"
  - "docs/contracts/merc-policy.md"
  - "docs/merc.md"
  - "internal/merc/policy.go"
  - "internal/merc/policy_test.go"
  - "internal/merc/farm_test.go"
  - "internal/merc/logs_test.go"
  - "internal/env/tuning.go"
  - "docs/knobs.md"
  - "docs/research/g2f-diagnosis.md"
exclusive: []
contracts:
  - "merc-policy"
---

## Description
User 2026-11-06: defer to the SME on the revive reserve (follow-up of TASK-41 / #274, which capped it with merc.reviveReserveMax). SME rule: revive as soon as the real quote plus a small reserve is on hand (0 gold below clvl 13, 50 from clvl 13), before potions and repair.

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Revive reserve is 0 gold below clvl 13 and 50 gold from clvl 13 (knobs merc.reviveReserveLow/High/Clvl), merc.reviveReserveMax removed and retired; kitReservePct stays for hires only
- [ ] #2 Revive runs before potions/repair with pack plus stash gold and the probed real quote; an unreadable quote logs once per visit and retries next visit
- [ ] #3 Contract merc-policy.md, docs/merc.md, knobs.md updated (Source: user 2026-11-06 defer to the SME); two-build tests pass; ladder t2.mercenary parity tests unchanged and green
<!-- AC:END -->
