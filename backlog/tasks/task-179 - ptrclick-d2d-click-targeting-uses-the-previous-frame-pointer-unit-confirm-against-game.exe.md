---
id: TASK-179
title: 'ptrclick: d2d click-targeting uses the previous frame pointer unit; confirm against game.exe'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-08T05:27:00Z'
labels:
  - d2d
  - probe
  - fidelity
  - d2d-combat
  - d2d-movement
milestone: m-2
dependencies: []
priority: medium
ordinal: 150000
card: "ptrclick"
owns:
  - "docs/probes/ptrclick.md"
exclusive: []
contracts: []
result: "plan and fork code reading only (S, fork c36b77b1); probe not run, no live game"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the Paladin melee fix (docs/palmelee.md, docs/palpace.md) showed that a bare click x y on d2d acted on the unit under the previous frame's pointer, so 15 swings at a Quill Rat 4 subtiles away never hit. Friend-readiness: if this is a d2d deviation, a real player is unaffected only if the pointer rests on the target for a frame. Done when the probe verdict and the route are in docs/probes/ptrclick.md. Plan and evidence: docs/probes/ptrclick.md. Needs a live game on the worker host (a user-agent run), so not run here.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Same-save probe scenario run on game.exe and d2d (docs/probes/ptrclick.md): Paladin next to one stationary monster; conditions A (click at the monster with no prior pointer move), B (pointer moved onto it 2+ frames before), C (pointer moved there the same frame); per condition record the targeted unit id (game.exe: hovered/selected unit read, GetSelectedUnit RVA 0x67A10 U for the backing variable; d2d: dump or log of the UseSkill unit) and whether life dropped within 3 s
- [ ] #2 d2d code path recorded with file:line against the fork commit on the private remote (read: Town::input in apps/d2d/town.cpp uses hovered_npc set by last frame's render; click verb in apps/d2d/devctl_verbs.hpp pushes motion+down+up in one frame)
- [ ] #3 Verdict: DEVIATION (d2d acts on a stale hover where game.exe resolves the unit under the cursor at click time) or HARNESS ONLY (a real mouse always has motion frames before the press; only the click verb teleports the pointer); matrix row text proposed (see the doc)
- [ ] #4 If DEVIATION: upstream PR route written as a differential against original game.exe code per docs/process/upstream-pr-standard.md and queued in docs/research/pr-bundle-queue.md by the coordinator; if HARNESS ONLY: the click verb change (hover motion frame before press) or the documented rule 'use cmd skill ... unit' is recorded
- [ ] #5 Bot answer recorded: done, Input.Swing in internal/d2dcombat/input.go sends cmd skill <id> x y <unit> left (palmelee, docs/palmelee.md); no bot change needed whatever the verdict
<!-- AC:END -->
