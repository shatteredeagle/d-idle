---
id: TASK-297
title: 'maptrail2: coverage grid spans the whole zone, no truncation'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T13:42:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 208000
card: "maptrail2"
owns:
  - "internal/senses/coverage.go"
  - "internal/senses/trail_test.go"
  - "internal/watch/web/panels/map"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/testdata/map_panel.js"
  - "internal/watch/testdata/senses-coverage.json"
  - "docs/senses-map.md"
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
- [ ] #1 Coverage cells span the whole zone for a 1593-point level; status line and drawn count agree
<!-- AC:END -->
