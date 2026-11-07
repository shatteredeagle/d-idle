---
id: TASK-106
title: 'sessionrec: play-session command plus a session record (questions per 10 minutes)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T21:56:00Z'
labels:
  - watch
  - feature
  - ladder
  - T0
milestone: m-4
dependencies:
  - TASK-43
  - TASK-33
priority: high
ordinal: 99000
card: "sessionrec"
owns:
  - "internal/watch/sessionrec.go"
  - "internal/watch/sessionrec_test.go"
  - "docs/session-record.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: direction review section 7 step 1 (user decision D1, 2026-11-07). Done when: the criteria below hold.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A session writes one file with start/end, hero level span, questions asked, answered, faded, rules learned and questions per 10 minutes; offline test with a fake clock
- [ ] #2 Serves friends too: realtime play through the tester container (TASK-33 realtime check is a dependency). Relates to TASK-43 (one command, phone URL, realtime hero): the record is written by that command's run; if realtime trips the lockstep-only clocks (tester plan G6) say which and how it was handled
- [ ] #3 docs/session-record.md says built vs verified live; go vet and go test -p 1 of internal/watch; leakcheck 0
<!-- AC:END -->
