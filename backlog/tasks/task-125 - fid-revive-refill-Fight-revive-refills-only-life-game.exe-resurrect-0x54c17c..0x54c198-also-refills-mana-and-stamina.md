---
id: TASK-125
title: 'fid-revive-refill: Fight::revive refills only life; game.exe resurrect (0x54c17c..0x54c198) also refills mana and stamina'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T23:31:00Z'
labels:
  - d2d
  - upstream-pr
  - fidelity
  - d2d-combat
milestone: m-2
dependencies: []
priority: medium
ordinal: 112000
card: "fid-revive-refill"
owns:
  - "docs/research/fid-revive-refill.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

d2d Fight::revive heals life only. The independent verifier of TASK-101 (docs/research/review-load-life-mana-opus.md, finding F6, V read) found that game.exe's resurrect path (0x54c17c..0x54c198) also refills mana and stamina. Out of scope of upstream PR #242 (load life/mana). Evidence: review-load-life-mana-opus.md section 5 and finding F6. Done when the trace, the oracle and a draft fix exist and a verifier has checked the differential.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Trace game.exe resurrect (0x54c17c..0x54c198): which stats it refills (life, mana, stamina) and in which order, with a disassembly read
- [ ] #2 Oracle or harness that runs the game.exe resurrect on a dead unit and diffs life, mana and stamina against d2d Fight::revive
- [ ] #3 Fix draft (a test against two builds) for upstream Fight::revive, held in the bundle queue; no post without the user's go
<!-- AC:END -->
