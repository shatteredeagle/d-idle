---
id: TASK-10
title: "fixbelt: low-kit belt window fix"
status: In Progress
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-C"
dependencies: []
priority: low
ordinal: 10000
card: "fixbelt"
owns:
  - "docs/fixbelt.md"
  - "internal/resource/recovery.go"
  - "internal/resource/recovery_test.go"
  - "internal/bot/broke.go"
  - "internal/bot/area_ban_test.go"
  - "openspec/changes/fixbelt/**"
exclusive: []
branch: "agent/fixbelt"
agent: "fixbelt"
result: "Unit tests only (mem.Fake bot rig); NOT verified live: needs a Gate 1 soak, recipe in docs/fixbelt.md."
live: "not run; recipe in docs/fixbelt.md"
merge_hash: ""
overlap: "Overlap (declared): FIXPROGRESS owns the town portal scroll and walk-to-town logic; this card does not touch them."
skip_specs: true
source: "openspec/changes/fixbelt"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Gate 1 rerun (bot f4ca264, d2d lab2-freezeresume) failed the belt criterion: a 6 virtual minute window with no healing potion outside town (limit 5). The window sits at clvl 7 in the 15 minute recovery cooldown; evidence and cause in docs/fixbelt.md.

What changes:
- internal/resource/recovery.go: `KitLow` and `RecoveryInput.LowKit`: healing potions under `recoveryPotionFloor` with an unaffordable kit start a recovery inside the cooldown.
- internal/bot/broke.go passes it; unit tests in internal/resource/recovery_test.go and internal/bot/area_ban_test.go.
- docs/fixbelt.md: evidence, cause, live recipe.

Next step: live proof in a seed-set record

Gates (from the card):
- go test -p 1 internal/resource internal/bot; changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict

Source: openspec/changes/fixbelt/ (kept in place; this task is a copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 locate the window in the minute log
- [x] #2 fix: low-kit override of the recovery cooldown
- [x] #3 unit tests
- [ ] #4 live proof (Gate 1 soak)
<!-- AC:END -->
