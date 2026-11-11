---
id: TASK-281
title: 'devupdateclean: dev-update mirrors deleted files on the worker copy'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T00:52:00Z'
labels:
  - ops
  - bug
dependencies: []
priority: medium
ordinal: 204000
card: "devupdateclean"
owns:
  - "tools/dev-update.sh"
  - "tools/dev-update-clean-test.sh"
  - "docs/dev-update.md"
exclusive: []
result: "Fixed in tools/dev-update.sh (rm -rf .devupdate-src on the host before the send); offline fake-labrun test passes and fails without the fix; NOT verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: labrun --send never deletes, so removed repo files lingered in the worker .devupdate-src and broke a build. Done when: the host copy is emptied before each send.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 dev-update empties the worker .devupdate-src before the send
- [x] #2 regression test with a fake labrun passes and fails without the fix
- [x] #3 documented in docs/dev-update.md
<!-- AC:END -->
