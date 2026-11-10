---
id: TASK-263
title: 'devreset: reset the play sandbox to a fresh hero (archive, restore)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T18:33:00Z'
labels:
  - ops
  - feature
  - tester
dependencies: []
priority: medium
ordinal: 196000
card: "devreset"
owns:
  - "tools/dev-reset.sh"
  - "tools/dev-sandbox.sh"
  - "tools/dev-sandbox-host.sh"
  - "tools/dev-sandbox-test.sh"
  - "docs/dev-reset.md"
exclusive: []
contracts: []
result: "Scratch drill passed (V); not run on the real sandbox; VERSION and URL lines and the hero wait unverified live. CLI only."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User request: test the first hour from scratch. CLI only (no web UI: the container cannot reach docker). Done when merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 dev-sandbox.sh reset archives then fresh hero, refuses without --yes
- [x] #2 restore puts an archive back, list shows archives
- [x] #3 scratch drill passes; docs/dev-reset.md written
<!-- AC:END -->
