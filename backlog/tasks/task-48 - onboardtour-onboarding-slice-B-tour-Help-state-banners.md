---
id: TASK-48
title: 'onboardtour: onboarding slice B: tour, Help, state banners'
status: Proposed
assignee: []
created_date: '2026-11-06T16:20:00Z'
labels:
  - onboarding
  - watch
dependencies: []
priority: medium
ordinal: 48000
card: "onboardtour"
owns:
  - "internal/watch/web/onboarding.js"
  - "internal/watch/web/onboarding.css"
  - "docs/onboarding-tour.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice B of docs/research/onboarding-ux-design.md (approved 2026-11-06): guided tour with page-only sample thought and rule, Help sheet, state banners (hero died, engine restarted, upgrade), tour_seen flag, 44 px targets. About 8 h. Hooks into watch.html and app.js are one-line script/link tags: widen with a note in the watch-screen task that owns them (TASK-47 owns internal/watch/web/**).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Tour of five steps with a page-only sample thought, Back/Skip/resume, phone bottom sheet
- [ ] #2 Help sheet and state banners; tour_seen stored
- [ ] #3 Tests and a 390/1280 px check
<!-- AC:END -->
