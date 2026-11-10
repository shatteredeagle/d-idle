---
id: TASK-203
title: "fid-load-gold-clamp: d2d keeps a saved purse or stash over its cap, game.exe zeroes it on load"
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T17:15:00Z'
updated_date: '2026-11-10T16:54:00Z'
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
result: "Candidate prepared (2026-11-08, agent decomp5): local/fid-load-gold-clamp commit 42d40b09 on the private fork remote, oracle 363 of 539 before, 0 after; waits for the independent verifier and the user's go; docs/research/pr-load-gold-draft.md."
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

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
