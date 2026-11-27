---
id: TASK-867
title: 'uifit1: fit suite triage and fixes (stale rules and phone/desk layout defects)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-23T07:10:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 390000
card: "uifit1"
owns:
  - "tools/uishots/"
  - "tools/uishots.sh"
  - "docs/ui/fit-triage-2026-11-23.md"
  - "internal/watch/web/utilities.css"
  - "internal/watch/web/shell.css"
  - "internal/watch/web/panels/statusline/panel.css"
  - "internal/watch/web/panels/quest-strip/panel.css"
exclusive: []
contracts: []
result: "Full fit on the worker host: 500 red runs before, 19 after (390x844 128 to 3, 360x640 80 to 11, 412x780 90 to 0, desk sizes 23 to 0-2). Triage in docs/ui/fit-triage-2026-11-23.md. Stale rules relaxed with unit rows, real defects fixed (hero-meters at 360, zone map under the bar, statusline, quest tile); 19 runs stay red and are listed (needs-you list design question, quest tile handing at two sizes, health, char-store hole, orders sheet at 360). Not seen on a real phone."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 triage doc groups every red fit run by rule with a verdict (real, stale, fixture)
- [x] #2 stale rules relaxed with must-pass and must-fail unit rows (fit_phone_test.go)
- [x] #3 real defects fixed and re-measured on the worker host; final red count per size in the doc
<!-- AC:END -->
