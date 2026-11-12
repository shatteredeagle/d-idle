---
id: TASK-344
title: 'frienddrills2: live drills D3/D4 with real game files, boot.pid guard'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T22:14:00Z'
updated_date: '2026-11-12T01:22:00Z'
labels:
  - container
  - drill
dependencies: []
priority: medium
ordinal: 229000
card: "frienddrills2"
owns:
  - "docs/friend-drills.md"
  - "cmd/didleboot"
exclusive: []
contracts: []
result: "Second drill run 2026-11-11 with the real game folder (read-only) and a fresh Paladin through the wizard (V, zero cheats): D3 (./didle update, update API behind the banner) and D4 (update while the hero fights, level 3 and pack intact, 14 s) PASS; bad v3 with a live hero rolled back in 4 s (D7 PASS). boot.pid clobber fixed offline (second shim refused, S). Banner not opened in a browser; non-DEV https feed not driven."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 D3 (./didle update, banner) and D4 (update while the hero fights, rollback with a live hero) run on a scratch install with real game files (read-only)
- [x] #2 boot.pid clobber refused by a second shim (offline test)
<!-- AC:END -->
