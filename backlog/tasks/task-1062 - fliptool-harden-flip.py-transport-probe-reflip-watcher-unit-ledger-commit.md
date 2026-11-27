---
id: TASK-1062
title: >-
  fliptool: harden flip.py (transport probe, --reflip, watcher unit, ledger
  commit)
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T20:14:00Z'
updated_date: '2026-11-27T20:32:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: medium
ordinal: 512000
card: "fliptool"
lane: next
owns:
  - "tools/flip/"
  - "docs/process/dark-launch.md"
exclusive: []
contracts: []
result: "offline only, not verified against a real container: docker-inspect probe naming D2_FLIP_RUN, --reflip, heartbeat warning, watch --daemon and --install-unit, --commit-ledger; 34 fake-sandbox tests pass."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 default transport fails fast naming D2_FLIP_RUN
- [x] #2 --reflip for enabled rows
- [x] #3 watch --install-unit and heartbeat warning
- [x] #4 --commit-ledger branch
<!-- AC:END -->
