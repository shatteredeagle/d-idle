---
id: TASK-48
title: "onboardtour: first-30-minutes in-app tour, Help sheet and state banners (beta explain)"
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-06T16:20:00Z'
updated_date: '2026-11-10T17:11:00Z'
labels:
  - watch
  - feature
  - onboarding
milestone: m-1
dependencies: []
priority: medium
ordinal: 48000
card: "onboardtour"
owns:
  - "internal/watch/web/onboarding.js"
  - "internal/watch/web/onboarding.css"
  - "internal/watch/web/onboarding.json"
  - "internal/watch/web/watch.html"
  - "internal/watch/onboardingtour_test.go"
  - "internal/watch/testdata/onboarding_tour_test.cjs"
  - "docs/onboarding-tour.md"
exclusive: []
contracts: []
result: "Built offline 2026-11-27 (no sample thought, no browser check, so #1 and #3 stay open): tour of five state-tied tips, Now line from explain, Help sheet, seen in localStorage; node stub-DOM tests green; not looked at in a browser (checklist in docs/onboarding-tour.md), not verified live."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M
Why: beta exit criterion 4 (docs/research/project-review-2026-11-10.md section 5.1) is the explanation of the first 30 minutes. This card is the in-app half: the guided tour, the Help sheet and the state banners of docs/research/onboarding-ux-design.md (approved 2026-11-06). The welcome card and the first-30-minutes text stay in TASK-247 (firstminutes); the tour links to them. Absorbs nothing else. Pause, Stop and Reset are not in the beta exit criteria and are not carried here (TASK-49 closed).
Done when: tour of five steps with a page-only sample thought, Back/Skip/resume, phone bottom sheet; Help sheet and state banners (hero died, engine restarted, upgrade); tour_seen stored; tests and a 390/1280 px check. Hooks into watch.html and app.js are one-line tags (TASK-47 owns internal/watch/web/**).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Tour of five steps with a page-only sample thought, Back/Skip/resume, phone bottom sheet
- [x] #2 Help sheet and state banners; tour_seen stored
- [ ] #3 Tests and a 390/1280 px check
- [x] #4 the tour links to the welcome card and first-30-minutes text of TASK-247 (not built here)
<!-- AC:END -->
