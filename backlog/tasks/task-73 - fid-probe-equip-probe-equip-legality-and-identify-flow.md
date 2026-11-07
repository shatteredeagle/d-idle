---
id: TASK-73
title: 'fid-probe-equip: probe: equip legality and identify flow'
status: Deferred
assignee:
  - live-agent
created_date: '2026-11-07T18:03:00Z'
updated_date: '2026-11-07T21:55:00Z'
labels:
  - d2d
  - probe
  - d2d-items
  - d2d-ui
  - fidelity
  - parked
milestone: m-2
dependencies:
  - TASK-55
priority: high
ordinal: 71000
card: "fid-probe-equip"
owns:
  - "docs/probes/fid/fid-probe-equip.md"
exclusive: []
contracts: []
result: "paused by direction review D1/D4/D5, 2026-11-07 (docs/research/direction-review.md section 6)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fidelity matrix rows: IT03 IT07. Method/kind: probe. Live game (host time): yes (hidden game.exe instance plus a d2d slot, host time about 1 h). Size: M. Fix: upstream PR if d2d skips requirement checks (world.hpp:808, fight.hpp:819); else none; needs the proof standard docs (differential vs original code) before any upstream PR. Dependencies: TASK-55 (scenarios 3, 4). Source: docs/research/fidelity-matrix.md; map docs/research/fidelity-cards.md.

Done when: Scenario 4 save: equip an item above Str/Dex/level requirement on both sides; expected both refuse (code,...; Hand/slot rules: two-hand with shield, swap weapon set; expected equal accept/refuse on both sides; Identify: unid item in pack, ID scroll or tome use; expected same item state and same scroll count after... (and 1 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Scenario 4 save: equip an item above Str/Dex/level requirement on both sides; expected both refuse (code, slot, result); tolerance from internal/fidprobe/rules.json (exact; life/mana/stamina relative 0.5 %), explained differences only via known.json
- [ ] #2 Hand/slot rules: two-hand with shield, swap weapon set; expected equal accept/refuse on both sides
- [ ] #3 Identify: unid item in pack, ID scroll or tome use; expected same item state and same scroll count after (IT07), Cain not used
- [ ] #4 Each MISMATCH is written to docs/probes/fid/fid-probe-equip.md with the d2d source line; an upstream PR draft only with the differential proof
<!-- AC:END -->
