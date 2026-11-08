---
id: TASK-25
title: "testerlive: run the tester flow in the real d2d container"
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-06T14:27:00Z'
labels:
  - container
  - drill
  - tester
milestone: m-1
dependencies:
  - TASK-31
  - TASK-32
  - TASK-33
priority: high
ordinal: 25000
card: "testerlive"
owns:
  - "docs/tester-live.md"
exclusive: []
contracts: []
result: "Verified live 2026-11-07 (docs/tester-live.md): image built on the worker host from the PUBLIC fork tag didle-gate2-lab3 (f75816c616c8); wizard (data check, 1.14d ok, go-ahead, name, Sorceress, goals, token, Start) over the API from a second machine; bot plays; thought card, answer, order and video (113 frames in 30 s) over the LAN with the token; stop/start and same-volume recreate keep a clvl 3 hero and the ladder store. MPQ set d2data, d2exp, d2char, Patch_D2 confirmed required and sufficient. Defects D1-D6 fixed in deploy/d2d-container (blocker: ladder store name; high: no save on stop, tap stopped before the bot, hero recreated on every restart), D7-D9 cards TASK-137, 138, 139. Writes after start: 0. Not run in a browser (U). TASK-31 AC1: image pins the fork ref (V, built and run), wizard hook V; TASK-32 AC1: steps proven live over the API, the 1.14d hash matches the owner's install, page U in a browser; TASK-23 G3/G4: see report text."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Slice 1 (d2d route), reworked from the Wine version. Depends on testerd2dimage (TASK-31), testerd2dwiz (TASK-32) and testerrt (TASK-33). Uses the worker host Docker only, no lab game instance; throwaway data volume removed afterwards. The game data is a copy of the owner's own MPQs mounted read-only. Estimate 8 h.

Done when: image built on the worker host; wizard to character to the bot playing; watch screen opened from a second...; the minimal MPQ set (d2data, d2exp, d2char, Patch_D2) is tried and the answer recorded in...; every defect found is listed with a fix owner; the result says verified live or not, writes after start 0... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 image built on the worker host; wizard to character to the bot playing; watch screen opened from a second machine with the token; a card appears and an answer lands; an order is given; the video (tap source) shows the game; stop, start and a same-volume image upgrade keep the character and the ladder store
- [x] #2 the minimal MPQ set (d2data, d2exp, d2char, Patch_D2) is tried and the answer recorded in docs/tester-live.md (required set confirmed or widened)
- [x] #3 every defect found is listed with a fix owner; the result says verified live or not, writes after start 0 stated
<!-- AC:END -->
