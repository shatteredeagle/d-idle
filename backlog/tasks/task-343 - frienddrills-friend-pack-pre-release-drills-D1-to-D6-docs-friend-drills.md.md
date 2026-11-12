---
id: TASK-343
title: 'frienddrills: friend pack pre-release drills D1 to D6 (docs/friend-drills.md)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T21:42:00Z'
labels:
  - container
  - drill
dependencies: []
priority: medium
ordinal: 228000
card: "frienddrills"
owns:
  - "docs/friend-drills.md"
  - "tools/friendpack/didle"
exclusive: []
contracts: []
result: "Ran 2026-11-11 on the worker host with a throwaway key, scratch feed and DEV image (V): D1 pass, D2 pass, D3 engine and shim pass (page/CLI U), D4 blocked (no real game data), D5 pass, D6 pass. Fixed ./didle update silence before the bot runs. Findings in docs/friend-drills.md. Not run: non-DEV https feed path, banner, hero fight."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 D1 to D6 run on the worker host with a throwaway key and a scratch feed; results in docs/friend-drills.md
- [x] #2 D3 page/CLI and D4 (hero fighting) need real game data: not run, listed
<!-- AC:END -->
