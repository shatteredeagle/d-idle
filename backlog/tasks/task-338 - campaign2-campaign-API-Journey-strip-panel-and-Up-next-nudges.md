---
id: TASK-338
title: 'campaign2: campaign API, Journey strip panel and Up next nudges'
status: In Progress
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
  - "internal/watch/funwire.go"
  - "internal/watch/panels_test.go"
  - "internal/watch/web/layout.json"
  - "tools/uishots/layout-rows.json"
  - "cmd/d2bot/journeywire.go"
  - "cmd/d2bot/journeywire_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "docs/campaign.md"
exclusive: []
result: "GET /api/campaign (campaign/1, goldens Paladin and Sorceress), POST /api/campaign/intent (policy gate), journey panel v1 (path, Up next with meter and ETA, chapter detail with Go now/Skip/Hold/Ask), XP meter in d2bot. Open: return-screen question, top band tooltip, Chronicle and ladder hooks. Not verified live."
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: stage of the Guided journey (docs/campaign.md, TASK-335). Done when: the acceptance criteria below pass offline on Paladin and Sorceress.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GET /api/campaign schema campaign/1 with goldens
- [x] #2 POST /api/campaign/intent behind same-origin and session gate
- [x] #3 journey panel module (panel.json, tokens, compat promise)
- [x] #4 Up next card and Why line read the campaign state; ETA and progress meter
<!-- AC:END -->
