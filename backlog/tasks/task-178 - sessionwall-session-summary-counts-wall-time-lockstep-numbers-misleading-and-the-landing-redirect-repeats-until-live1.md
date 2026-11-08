---
id: TASK-178
title: 'sessionwall: session summary counts wall time (lockstep numbers misleading) and the landing redirect repeats until live=1'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T04:19:00Z'
labels:
  - watch
  - bug
milestone: m-4
dependencies: []
priority: low
ordinal: 145000
card: "sessionwall"
owns:
  - "docs/sessionwall.md"
  - "internal/watch/sessionrec.go"
  - "internal/watch/sessionrec_test.go"
  - "internal/watch/testdata/session-summary.json.golden"
  - "internal/watch/returnscreen.go"
  - "internal/watch/returnscreen_test.go"
  - "cmd/d2bot/class.go"
  - "cmd/d2bot/play.go"
  - "docs/session-record.md"
  - "docs/return-screen.md"
exclusive: []
contracts: []
result: "Offline V: summary reports wall and game clocks, rates on the game clock (TestSummaryBothClocksLockstep, two classes); identity line carries class and build; landing redirect once per absence (TestReturnLandingAfterAbsence); record size measured from the smoke run and doc corrected, not reduced. Not verified live. Owns widened to the watch/session code and docs."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Found by paladinsmoke (TASK-142, docs/paladinsmoke.md). (1) summary.json of a lockstep play session derives per-hour and per-10-minute numbers from wall time: 36 wall minutes covered 707 virtual minutes, so questions_per_10min is [28,1,1,0] and decision_bytes_per_h is 1.6 MB; per virtual hour the dataset was about 83 KB against the 30 KB expected in docs/session-record.md, and 27,857 default fires were counted (2,035 logged, 25,822 suppressed). (2) GET /watch after 20 minutes away answers 302 to /watch/return every time until /watch?live=1 is requested; docs/return-screen.md says 'redirects once'. (3) the session start record has class and build empty and the summary build is empty. Done when: the summary uses the game clock under lockstep (or says wall), the redirect text and behaviour agree, and the start line carries the class once known.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Summary per-hour numbers use the injected game clock; test with a fake clock
- [x] #2 Redirect behaviour and docs/return-screen.md agree
<!-- AC:END -->
