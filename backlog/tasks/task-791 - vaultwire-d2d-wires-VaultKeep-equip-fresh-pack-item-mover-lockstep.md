---
id: TASK-791
title: 'vaultwire: d2d wires VaultKeep, equip fresh pack, item mover lockstep'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T20:55:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 363000
card: "vaultwire"
owns:
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/d2dbackend_test.go"
  - "cmd/d2bot/behaviors.go"
  - "cmd/d2bot/invwire.go"
  - "cmd/d2bot/vaultequip.go"
  - "cmd/d2bot/vaultequip_test.go"
  - "cmd/d2bot/invverbs.go"
  - "cmd/d2bot/invverbs_test.go"
exclusive: []
result: "F1 d2dbackend never set deps.VaultKeep (fixed, must-fail test TestRunD2DWiresVaultKeep). F2 two causes: the desk read the runner's per-tick snapshot, and the Keep's created unit is listed by the game a few lockstep steps after the create answers; the equip hand now reads a fresh snapshot and waits on the bot clock (tests on Paladin and Sorceress). F3 not a fork bug: the queue-path item mover settled with a WALL-clock pause while d2d lockstep only advances when the bot clock steps, so identify (and move) always reported 'did not show the change'; the mover now pauses on the bot clock. Verified live (builder, devp17 lab d2d, CHEATED setups): Paladin 11 GREEN manage.equip-immediate, manage.identify-immediate, vault.equip-one-tap, -ring-left, -ring-right; Sorceress 3 GREEN manage.identify-immediate, vault.equip-one-tap, -ring-left, -ring-right. Still RED: vault.equip-swap (both), manage.recall-now (both), manage.equip-immediate Sorceress; see the report. Determinism pairs not run. closecards3 Remaining (live): #4 need the live game, a browser or a worker-host run; not verified here."
contracts: []
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Findings F1-F3 of the pgvault live runs.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 F1 d2d deps carry VaultKeep (must-fail test)
- [x] #2 F2 equip reads a fresh pack after Keep
- [x] #3 F3 item mover settles on the bot clock
- [ ] #4 Scenarios green Paladin 11 and Sorceress 3 (live, CHEATED setups)
<!-- AC:END -->
