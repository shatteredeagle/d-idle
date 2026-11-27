---
id: TASK-1075
title: 'nightcheck: preflight and corrected commands for the 2026-11-27 night run'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T22:49:00Z'
updated_date: '2026-11-27T23:18:00Z'
labels:
  - ops
  - chore
dependencies: []
priority: medium
ordinal: 516000
card: "nightcheck"
lane: next
owns:
  - "tools/provingground/nightcheck.sh"
  - "tools/provingground/nightcheck_test.sh"
  - "docs/probes/lab-night-commands-2026-11-27.md"
exclusive: []
result: "Offline only, no game: nightcheck.sh (ids, run flags, d2scenario plan, knob keys, no_writes assertion, ledger env, lab paths) and its test pass; commands doc lists three jobs. Finding: the long lane runs one job at a time with a 150 min limit. Not verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 nightcheck.sh checks ids, flags, knobs, no_writes assertion, ledger and lab paths without a game
- [x] #2 nightcheck_test.sh passes
- [x] #3 three corrected launch commands in docs/probes/lab-night-commands-2026-11-27.md
<!-- AC:END -->
