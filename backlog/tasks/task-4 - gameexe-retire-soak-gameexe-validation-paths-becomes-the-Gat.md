---
id: TASK-4
title: "gameexe-retire-soak: game.exe validation paths; becomes the Gate A card"
status: Deferred
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-A"
dependencies: []
priority: high
ordinal: 4000
card: "gameexe-retire-soak"
owns:
  - "docs/gameexe-retire.md"
exclusive: []
branch: "agent/gameexe-retire-soak"
agent: "gameexe-retire-soak"
result: ""
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/gameexe-retire-soak"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Retirement of Game.exe and the validation of Game.exe-only paths (kill recovery, failsafe ladder, recycle, Wine/OOM) that d2d cannot test, deferred by the ADR 0009 addendum until original-game work resumes. G2 and the Fresha soak finish as the last baselines. Deferred: user: original-game work is much further down the line.

What changes:
- list the Game.exe-only criteria no d2d run covers
- regression soak on Game.exe when original-game work resumes

Next step: rewrite the text for ADR 0015 (acceptance, not retirement)

Gates (from the card):
- Game.exe G1/G2 criteria re-verified once work resumes

Source: git tag openspec-final, openspec/changes/gameexe-retire-soak/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 1.1 list the Game.exe-only criteria no d2d run covers
- [ ] #2 1.2 regression soak on Game.exe when original-game work resumes
<!-- AC:END -->
