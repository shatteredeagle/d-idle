---
id: TASK-44
title: 'onboardA: onboarding slice A: setup stepper, go-ahead, access'
status: Done
assignee: []
created_date: '2026-11-06T16:02:00Z'
updated_date: '2026-11-07T15:14:00Z'
labels:
  - container
  - feature
  - tester
  - onboarding
milestone: m-1
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
result: "Built and tested offline (go test -race of cmd/d2dsetup, internal/access, internal/watch, cmd/d2bot): six screens, go-ahead stored with date and enforced, verdicts, resumable progress, passcode (PBKDF2-SHA256 600000) or token, watch gate with cookie and lockout, reset-access. Not verified live: no real container, real MPQs or phone; headless browser only for welcome, hero and access screens. Build-specific data: classes and presets in cmd/d2dsetup/tables.json. Details docs/onboarding-slice-a.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Slice A of docs/research/onboarding-ux-design.md (user approved the mock and its five recommendations 2026-11-06): six-screen stepper in cmd/d2dsetup (welcome, go-ahead, game files, hero, access, start), why boxes, Back, resume, plain-English errors, stored go-ahead tick with date, file verdict states, warm-up progress that survives a closed page, done screen; access by user-chosen passcode (PBKDF2-SHA256 hash in /data, 0600) or the generated token, with a passcode login and cookie in the watch gate, rate limit and a reset command. Slices B and C: TASK-48 (onboardtour), TASK-49 (onboardctl). Replaces the old placeholder.

Done when: Six-screen stepper with why boxes, Back, resumable state and plain errors with Retry; go-ahead tick stored...; Game-file verdicts: missing, too small, wrong version, Resurrected folder, hash unconfirmed; Start screen with warm-up progress and estimate that survives a closed page; done screen (and 2 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Six-screen stepper with why boxes, Back, resumable state and plain errors with Retry; go-ahead tick stored with the date and enforced on Start
- [x] #2 Game-file verdicts: missing, too small, wrong version, Resurrected folder, hash unconfirmed
- [x] #3 Start screen with warm-up progress and estimate that survives a closed page; done screen
- [x] #4 Passcode (6+ characters, typed twice, PBKDF2 hash 0600) or token; watch gate login form, cookie, rate limit, token ?t= unchanged; reset command
- [x] #5 Go tests (state machine, go-ahead, passcode set/verify/lockout/reset, cookie flags, token kind, verdicts, httptest happy and failure paths, two profiles)
<!-- AC:END -->
