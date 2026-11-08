---
id: TASK-194
title: 'returnlayout: return page cards render as narrow columns'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T16:46:00Z'
updated_date: '2026-11-08T17:01:00Z'
labels:
  - watch
  - bug
milestone: m-4
dependencies: []
priority: high
ordinal: 157000
card: "returnlayout"
owns:
  - "internal/watch/returnscreen.go"
  - "internal/watch/returnscreen_test.go"
exclusive: []
contracts: []
result: "main.ret resets the global 12-column main grid (return and policy pages); Firefox headless screenshots at 1280 and 390 px checked on the worker host; live container not checked"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: app.css makes main a 12-column grid from 900 px, so the return page cards became ~25 px columns. Done when: main.ret is block, test pins it, screenshots checked.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 main.ret resets the global main grid
- [x] #2 test pins the rule
<!-- AC:END -->
