---
id: TASK-247
title: 'firstminutes: welcome card and first-30-minutes text on /watch'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T16:55:00Z'
labels:
  - watch
  - feature
milestone: m-1
dependencies: []
priority: medium
ordinal: 185000
card: "firstminutes"
owns:
  - "docs/onboarding.md"
  - "data/onboarding"
  - "internal/watch/onboarding.go"
  - "internal/watch/web/panels/onboarding"
  - "internal/watch/testdata/onboarding_dom.js"
  - "internal/watch/onboarding_test.go"
exclusive: []
contracts: []
result: "BUILT offline 2026-11-11 (branch agent/onboarding): panel web/panels/onboarding, /api/onboarding, data/onboarding/en.json, docs/onboarding.md. Tests: Go state/endpoints, node DOM golden, pattern tests, uilint. Not verified live or in a browser."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: beta exit criterion 4 (docs/research/project-review-2026-11-10.md section 5): a friend must understand the first 30 minutes without us. Done when: the card and the checklist show on /watch for a new hero, dismissal survives a restart, texts live in one data file.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 welcome card (five short lines, three buttons, per-hero dismissal in state/onboarding.json, Help "show again") and the first-30-minutes checklist (five steps, auto-tick from observable state or manual, collapsible, hides itself) as one panel module
- [x] #2 all texts in data/onboarding/en.json; docs/onboarding.md and the friend-guide section "First 30 minutes"
- [x] #3 art "preparing" sentence shown in the guide while extraction runs; Go, node DOM golden, pattern tests and uilint pass
- [ ] #4 looked at in a real browser on a fresh hero (live check, not done)
<!-- AC:END -->
