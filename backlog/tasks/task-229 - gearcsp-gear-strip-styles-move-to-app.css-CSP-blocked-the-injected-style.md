---
id: TASK-229
title: 'gearcsp: gear strip styles move to app.css (CSP blocked the injected style)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-09T19:03:00Z'
updated_date: '2026-11-09T19:19:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 176000
card: "gearcsp"
owns:
  - "internal/watch/web"
  - "internal/watch/repairapi_test.go"
exclusive: []
contracts: []
result: "gear.js style moved to app.css; guard test added; go test internal/watch ok; not yet checked in a browser (sandbox preview pending)"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 gear.js injects no style element; strip styles live in app.css
- [x] #2 test guards embedded JS against injected styles
<!-- AC:END -->
