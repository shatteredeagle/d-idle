---
id: TASK-359
title: 'cut010b: re-cut friends release 0.1.0 after the lock-ordering fix'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-12T13:58:00Z'
labels:
  - container
  - chore
dependencies: []
priority: medium
ordinal: 234000
card: "cut010b"
result: "0.1.0 re-cut from main 0f049a66 (lock-ordering fix) and verified, NOT published or sent. Pack sha256 393c62e174fba680f5d6910f97e74c403f139598eca9ad63a9bfac9288c0a36c, checklist 7/7 PASS, UPDATE_FEED set to the public https feed, signed beta set re-built. Verified live on the worker host: D1 install, D2 wizard and hero, stop/start, report scrub, 10 minute no-deadlock run (hero moving, 23 threads constant, 0 restarts). The first pack (0a6c2a13, deadlock) was deleted. Details docs/releases/0.1.0.md."
owns:
  - "docs/releases/0.1.0.md"
  - "docs/friend-checklist-0.1.0.md"
  - "tools/friendpack/checklist.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Image and pack rebuilt from main with the deadlock fix, version stamped
- [x] #2 Checklist 7 items PASS, D1/D2 drills and a 10 minute no-deadlock health run
- [x] #3 UPDATE_FEED set to the real feed
<!-- AC:END -->
