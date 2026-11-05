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
  - "internal/town/shop.go"
  - "internal/town/chores.go"
  - "internal/town/town.go"
  - "internal/town/healbelt_test.go"
  - "internal/env/tuning_town.go"
  - "docs/knobs.md"
exclusive: []
branch: "agent/fixbelt"
agent: "fixbelt"
result: "Unit tests only (mem.Fake bot rig); NOT verified live: needs a Gate 1 soak, recipe in docs/fixbelt.md."
live: "not run; recipe in docs/fixbelt.md"
merge_hash: ""
overlap: "Overlap (declared): FIXPROGRESS owns the town portal scroll and walk-to-town logic; this card does not touch them."
source: "openspec-final:openspec/changes/fixbelt"
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
- go test -p 1 internal/resource internal/bot; leakcheck -mode fail 0; tools/ci/check.sh

Source: git tag openspec-final, openspec/changes/fixbelt/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 locate the window in the minute log
- [x] #2 fix: low-kit override of the recovery cooldown
- [x] #3 unit tests
- [x] #5 Gate 2 g2b #1/#4: shop buys a healing potion when belt and pack hold none (Rejuvenation does not count, as in the BELT gate judge; judge unchanged); town.tripEmptyHPBelt (default 1) raises a trip when no healing potion is held and gold >= KitPotionGold, one log line when gold is short. Offline tested (internal/town/healbelt_test.go, Sorceress + Barbarian); not run live.
- [ ] #4 live proof (Gate 1 soak)
- [x] #5 fixcursor2 (G-C, Gate 2 g2b fix #3): DropCursor (internal/town/hand/drop.go) clicks up to town.dropTries (6) different cells beside the character, falls back to a free pack cell (ClearCursor), else ErrCursor + failsafe; log lines `cursor: dropped item= try= offset=`, `cursor: could not clear after N tries items=`; a launcher the class cannot use is not picked up without a pack cell to take it off (internal/loot/pickup.go takenOffAgain). Offline tests only (Sorceress + Barbarian); live proof pending
- [x] #6 fixcursor3 (G-C, Gate 2 g2c seed 13): ClearCursor tries up to town.cursorStuckMax (3) distinct pack spots (a spot whose put is not confirmed is blocked for the next try, log `cursor: pack spot refused`); PickUp clears a held item before any pickup (pack, else DropCursor) and after cursorStuckMax failed pack clears in a row goes straight to the ground; Town.ClearCursor (nav waypoint hook) and the leveler displaced-item clear use Hand.ClearOrDrop; log `cursor: cleared before pickup item= how=pack|ground`, `cursor: stuck item= cause=`. Offline tests only (Sorceress + Barbarian); live proof pending
<!-- AC:END -->
