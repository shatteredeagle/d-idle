---
id: TASK-148
title: 'sensesmap: side-panel minimap v1 from the nav map export'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T03:11:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies: []
priority: medium
ordinal: 131000
card: "sensesmap"
owns:
  - "internal/watch/senses.go"
  - "internal/watch/senses_test.go"
  - "internal/watch/web/senses.js"
  - "docs/senses.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/research/info-unlocks.md sections 3 and 5. Not on the two-week playable path; after codexfirsts/funwire. Done when the panel shows the live area map in a fake-source run and the live checklist is written.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Read-only snapshot of the d2map Grid (downsampled, versioned) and hero x/y exported to the hub; no change to bot behaviour (test: bot inputs identical with senses on or off)
- [ ] #2 /api/senses returns the map part only when tier 1 is unlocked (stub: profile flag), token-gated like the viewer
- [ ] #3 Canvas side panel draws explored cells, hero, waypoint, exits; offline test with a d2map fixture
- [ ] #4 docs/senses.md: data flow, schema, honesty tags; live check listed as a ready checklist
<!-- AC:END -->
