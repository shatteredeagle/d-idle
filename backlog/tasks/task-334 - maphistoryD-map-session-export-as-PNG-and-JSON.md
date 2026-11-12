---
id: TASK-334
title: 'maphistoryD: map session export as PNG and JSON'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:38:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 225000
card: "maphistoryD"
owns:
  - "internal/watch/web/panels/map"
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/testdata/map_panel.js"
  - "internal/watch/testdata/history-export.json"
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
- [ ] #1 GET /api/history/export JSON (trail, events, samples, heat, coverage, build; no credentials) and Export PNG/JSON buttons in the map panel
<!-- AC:END -->
