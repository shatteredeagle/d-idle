---
id: TASK-1031
title: 'savefix: a managed d2d loads a saved hero, never creates over it'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T03:11:00Z'
updated_date: '2026-11-27T03:34:00Z'
labels:
  - bot
  - bot-fix
dependencies: []
priority: high
ordinal: 498000
card: "savefix"
result: "Fixed: d2dConfig loads (class -1) when the hero save exists, so a watchdog or supervisor restart no longer creates a new hero over it. Cause verified live on a lab slot; the fix is offline-tested (must-fail test), not re-run on a live container."
lane: next
owns:
  - "cmd/d2bot/d2dloadsave.go"
  - "cmd/d2bot/d2dloadsave_test.go"
  - "cmd/d2bot/d2dbackend.go"
  - "docs/probes/test-characters.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 A managed d2d started with a create class loads the save when it exists (must-fail test)
- [x] #2 No save: create still happens
<!-- AC:END -->
