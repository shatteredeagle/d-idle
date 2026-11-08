---
id: TASK-203
title: 'fid-load-gold-clamp: d2d keeps a saved purse or stash over its cap, game.exe zeroes it on load'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T17:15:00Z'
labels:
  - d2d
  - upstream-pr
  - d2d-items
  - fidelity
dependencies:
  - TASK-79
priority: low
ordinal: 156000
card: "fid-load-gold-clamp"
owns:
  - "docs/research/pr-load-gold-draft.md"
exclusive:
  - "fork-branch:local/fid-load-gold-clamp"
  - "upstream-pr:load-gold-clamp"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Found by TASK-79 (docs/probes/fid/fid-inv-limits.md): FUN_0056af80 puts gold and stash gold back through FUN_0053ff00, which sets a stat to 0 when it would pass its cap; d2d keeps it. Oracle fl_gold.py: 363 of 539 cases differ before, 0 after. Reachable only through an edited save (U). Done when the verifier agrees and the user decides to open or drop it.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Candidate branch local/fid-load-gold-clamp (fork remote, commit 42d40b09) re-verified by the independent Opus verifier
- [ ] #2 Draft with Claims table in docs/research/pr-load-gold-draft.md; opened only on the user's go
<!-- AC:END -->
