---
id: TASK-747
title: >-
  mgcache: cached whole-repo gate measured over 20 merges then default; nightly
  uncached race with red hold
status: Done
assignee:
  - credit-runner
created_date: '2026-11-21T04:49:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - ops
  - feature
dependencies: []
priority: medium
ordinal: 352000
card: "mgcache"
owns:
  - "docs/process/pg-ops-mgcache.md"
exclusive: []
contracts: []
result: "Gate stats per merge (whole-repo seconds, cached packages, count1 flag, race seconds, load) recorded in the queue result and summarised by gate_phases.py; replay measurement of the last 20 code merges without -count=1 on the builder long lane (own cache) scheduled nightly; MERGE_GATE_CACHED=1 switch (default off, -count=1 NOT dropped, AC 2 open until 20 merges are read); nightly uncached whole-repo -race on the second host in the 23:00-07:00 New York window (niced, host-idle check, flake re-run, reproduced red holds the queue). Offline tests pass; replay data not yet collected (first run at 01:30)."
lane: now
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design 4.2 and 4.3, 7 card 12. Private ops repo work. Design: docs/design/fable-review-2026-11-21/proving-ground-gate.md. Owner approved 2026-11-21 (all 10 questions as recommended).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Measure cached vs -count=1 gate over 20 merges
- [ ] #2 drop -count=1 from the merge gate afterwards (Q3)
- [x] #3 nightly uncached whole-repo -race with red hold
<!-- AC:END -->
