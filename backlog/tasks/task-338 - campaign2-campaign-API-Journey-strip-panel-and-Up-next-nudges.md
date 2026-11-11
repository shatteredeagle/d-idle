---
id: TASK-338
title: 'campaign2: campaign API, Journey strip panel and Up next nudges'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T20:15:00Z'
labels:
  - watch
  - feature
dependencies:
  - TASK-337
priority: high
ordinal: 227000
card: "campaign2"
owns:
  - "internal/watch/campaign.go"
  - "internal/watch/campaign_test.go"
  - "internal/watch/testdata/campaign/"
  - "internal/watch/web/panels/journey/"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: stage of the Guided journey (docs/campaign.md, TASK-335). Done when: the acceptance criteria below pass offline on Paladin and Sorceress.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 GET /api/campaign schema campaign/1 with goldens
- [ ] #2 POST /api/campaign/intent behind same-origin and session gate
- [ ] #3 journey panel module (panel.json, tokens, compat promise)
- [ ] #4 Up next card and Why line read the campaign state; ETA and progress meter
<!-- AC:END -->
