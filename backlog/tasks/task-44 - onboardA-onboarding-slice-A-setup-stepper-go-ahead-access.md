---
id: TASK-44
title: 'onboardA: onboarding slice A: setup stepper, go-ahead, access'
status: In Progress
assignee: []
created_date: '2026-11-06T16:02:00Z'
labels:
  - onboarding
  - tester
dependencies: []
priority: medium
ordinal: 44000
card: "onboardA"
owns:
  - "cmd/d2dsetup/**"
  - "internal/access/**"
  - "internal/watch/serve.go"
  - "internal/watch/gate.go"
  - "internal/watch/gate_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/ladder_test.go"
  - "docs/onboarding-slice-a.md"
exclusive: []
contracts: []
overlap: "cmd/d2dsetup/** is also owned by TASK-32 (testerd2dwiz, In Progress; merged code, this slice extends it) and cmd/d2bot/ladder.go by TASK-23/TASK-30 (watch token; this slice adds one access-file flag and keeps their tests unchanged). Widened 2026-11-06 with this note; TASK-32 and TASK-23 keep their own work."
result: "pending"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice A of docs/research/onboarding-ux-design.md (user approved the mock and its five recommendations 2026-11-06): six-screen stepper in cmd/d2dsetup (welcome, go-ahead, game files, hero, access, start), why boxes, Back, resume, plain-English errors, stored go-ahead tick with date, file verdict states, warm-up progress that survives a closed page, done screen; access by user-chosen passcode (PBKDF2-SHA256 hash in /data, 0600) or the generated token, with a passcode login and cookie in the watch gate, rate limit and a reset command. Slices B and C: TASK-48 (onboardtour), TASK-49 (onboardctl). Replaces the old placeholder.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Six-screen stepper with why boxes, Back, resumable state and plain errors with Retry; go-ahead tick stored with the date and enforced on Start
- [ ] #2 Game-file verdicts: missing, too small, wrong version, Resurrected folder, hash unconfirmed
- [ ] #3 Start screen with warm-up progress and estimate that survives a closed page; done screen
- [ ] #4 Passcode (6+ characters, typed twice, PBKDF2 hash 0600) or token; watch gate login form, cookie, rate limit, token ?t= unchanged; reset command
- [ ] #5 Go tests (state machine, go-ahead, passcode set/verify/lockout/reset, cookie flags, token kind, verdicts, httptest happy and failure paths, two profiles)
<!-- AC:END -->
