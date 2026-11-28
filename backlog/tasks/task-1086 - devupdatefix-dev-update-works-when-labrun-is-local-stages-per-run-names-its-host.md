---
id: TASK-1086
title: 'devupdatefix: dev-update works when labrun is local, stages per run, names its host'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T02:06:00Z'
labels:
  - ops
  - bug
dependencies: []
priority: medium
ordinal: 520000
card: "devupdatefix"
lane: next
owns:
  - "tools/dev-update.sh"
  - "tools/dev-update-clean-test.sh"
  - "tools/dev-update-local-test.sh"
  - ".gitignore"
  - "docs/dev-update.md"
exclusive: []
result: "Local-mode detection (hostname or D2_LABRUN_LOCAL), per-run stage dir with trap, host line printed, offline fake-labrun test; NOT verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Local mode skips clean+send and passes an absolute staged src
- [x] #2 Stage dir is per run and removed by a trap
- [x] #3 First line prints target and sandbox host; --serve uses D2_SANDBOX_HOST
- [x] #4 Fake labrun/docker shell test covers local, remote, first line
<!-- AC:END -->
