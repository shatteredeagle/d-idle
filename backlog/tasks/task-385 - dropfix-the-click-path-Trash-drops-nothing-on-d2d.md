---
id: TASK-385
title: 'dropfix: the click-path Trash drops nothing on d2d'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T22:22:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: medium
ordinal: 244000
card: "dropfix"
owns:
  - "internal/d2dinput/controller.go"
  - "internal/d2dinput/controller_test.go"
  - "internal/input/controller.go"
  - "internal/d2dtown/adapter.go"
  - "internal/d2dtown/adapter_dropclick_test.go"
  - "internal/town/hand/drop.go"
exclusive: []
contracts: []
result: "Cause: on d2d ClickWorld is cmd move (a walk order), so the held item never dropped; all town.dropTries clicks failed. Fix: input.WorldDropper/DropClick presses the mouse at the hero-relative screen spot (hero at 400,292, V). Unit tests in d2dinput and d2dtown; verified live on throwaway Paladin and Sorceress: the click-path trash now drops (done, 25 s) where it failed before."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Found by the item verbs live comparison (TASK-383): the player's Trash by clicks failed on d2d on two classes (all ground clicks failed, the item went back to the pack). Cause: ClickWorld on d2d is cmd move, a walk order, not a mouse press, so a held item never dropped. Fix: input.WorldDropper / DropClick presses the mouse at the hero-relative spot. Done when the unit tests and the live drop on a throwaway hero pass.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 DropClick presses the mouse (unit tests: d2dinput, d2dtown wrapper)
- [x] #2 Live: the click-path trash drops the item on a throwaway Paladin and Sorceress
- [x] #3 Gate green
<!-- AC:END -->
