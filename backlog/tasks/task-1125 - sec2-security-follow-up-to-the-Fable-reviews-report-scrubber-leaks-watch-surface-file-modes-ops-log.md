---
id: TASK-1125
title: 'sec2: security follow-up to the Fable reviews: report scrubber leaks, watch surface, file modes, ops log'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-28T06:46:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: high
ordinal: 534000
card: "sec2"
lane: next
owns:
  - "deploy/d2d-container/tester-report.sh"
  - "deploy/d2d-container/tester_report_sec_test.go"
  - "internal/support/"
  - "internal/watch/surface_sec_test.go"
  - "internal/watch/logtab.go"
  - "internal/logx/"
  - "internal/access/"
  - "internal/diffready/consent.go"
  - "internal/diffready/mode_sec_test.go"
  - "cmd/d2bot/opscmd.go"
  - "cmd/d2bot/opslog_sec_test.go"
exclusive: []
contracts: []
result: "Offline only, NOT verified live (box 8 is the live check). Report scrubber (tester-report.sh): the review 12 C4 leak rows are hidden, JSON lines stay JSON, labelled build ids and version numbers stay, control and NUL bytes are dropped, a leftover scan with its own patterns refuses the tarball (exit 3), umask 077. Go support redactor: passcode, pin, user and hostname labels, bare passcode phrases, loose keys; scan knows passcode. Watch: /api/log is GET only; surface test over real sockets and an AST walk that every AnswerOptions door judges the token. access.json and diffgo.json ignore a leftover or linked temp file. ops log: credentials hidden (support.RedactCredentials), control characters shown as hex, rotations are name.N only. Nothing is build-specific. Not done: rate-limit changes (limits already exist; review 4 finding 9 limiter key is open), review 4 findings 6, 10, 11 and 12 C2, low 7 (two-word home rule eats prose)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Follow-up to fable review 4 and review 12 C4 (docs/design/fable-review-2026-11-28). Done when the boxes hold with must-fail tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Report scrubber: the leak rows of review 12 C4 are hidden (lower-case and spaced keys, passcode phrases with and without a separator), JSON lines stay JSON, build ids and version numbers stay; adversarial corpus test
- [x] #2 Second line of defence: a known secret shape left in a bundled file or file name stops the tarball (exit 3); the bundle is owner-only
- [x] #3 Bounded time: long hostile lines through the shell scrubber and the Go redactor and scan (ratio test)
- [x] #4 Support bundle redactor (Go): passcode, pin, user labels, loose keys; corpus test is fail closed with the scan
- [x] #5 Watch surface: every route refuses without the token, the viewer listener takes no writing verb, every answer door judges the token (AST walk)
- [x] #6 Token and state files: access record and difficulty go are 0600 even over a leftover or linked temp file
- [x] #7 ops log: no credential, no control character, rotations are name.N only
- [ ] #8 Live check: run tester-report.sh in a tester container and read the bundle
<!-- AC:END -->
