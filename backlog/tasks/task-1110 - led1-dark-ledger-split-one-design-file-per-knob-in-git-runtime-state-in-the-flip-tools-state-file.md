---
id: TASK-1110
title: 'led1: dark ledger split, one design file per knob in git, runtime state in the flip tool''s state file'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T05:31:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: high
ordinal: 533000
card: "led1"
lane: next
owns:
  - "data/darklaunch/"
  - "data/darklaunch.json"
  - "data/darklaunch-cap.json"
  - "tools/darklaunch/"
  - "tools/flip/"
  - "docs/process/dark-launch.md"
  - "docs/process/autoflip.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Reviews 05 and 09 (docs/design/fable-review-2026-11-28): the ledger was one file edited from three places, the cause of merge conflicts and orphan trial rows.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 data/darklaunch/<knob>.json holds the design fields of one knob; a loader merges them; data/darklaunch.json is gone and check.py fails when it comes back
- [ ] #2 runtime state (state, trial_*, enabled_at, revert_*, criteria_met, window log) lives in the flip tool's state file outside git; flip, close and revert write only there
- [ ] #3 check.py (design only, same result on every machine), report.py, flip.py, autoflip.py, the cap and the new-row rules work on the split ledger; an old one-file ledger still loads
- [ ] #4 migrate.py splits a one-file ledger; golden test: the merged view of the split equals the old content including runtime fields
- [ ] #5 close and revert print the one-line design commit when the status word in git must change
- [ ] #6 Live: the watcher is restarted on the new code and the state file is seeded from the main checkout's ledger (coordinator)
<!-- AC:END -->
