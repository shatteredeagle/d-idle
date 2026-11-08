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
  - "docs/senses-map.md"
  - "internal/senses/"
  - "internal/watch/serve.go"
  - "internal/watch/web/watch.html"
  - "internal/watch/web/app.css"
exclusive: []
contracts: []
result: "Built offline (fixtures only, no live run, no browser run): internal/senses, /api/senses, canvas panel, docs/senses-map.md. AC1 partly: snapshot and hero export exist (Publisher) but the bot tick is not wired (internal/bot not owned); AC1 stays open. Open choice: knob senses.tier registration with the wiring."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/research/info-unlocks.md sections 3 and 5. Not on the two-week playable path; after codexfirsts/funwire. Done when the panel shows the live area map in a fake-source run and the live checklist is written.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Read-only snapshot of the d2map Grid (downsampled, versioned) and hero x/y exported to the hub; no change to bot behaviour (test: bot inputs identical with senses on or off)
- [x] #2 /api/senses returns the map part only when tier 1 is unlocked (stub: profile flag), token-gated like the viewer
- [x] #3 Canvas side panel draws explored cells, hero, waypoint, exits; offline test with a d2map fixture
- [x] #4 docs/senses-map.md: data flow, schema, honesty tags; live check listed as a ready checklist
<!-- AC:END -->
