---
id: TASK-23
title: "testerwatch: separate watch token, 832x640 slot display"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - G-F
dependencies: []
priority: high
ordinal: 23000
card: "testerwatch"
owns:
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/ladder_test.go"
  - "internal/launch/display.go"
  - "internal/launch/display_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 of docs/research/tester-build-plan.md. Depends on nothing.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 d2bot reads the watch token from D2BOT_WATCH_TOKEN or a token file (never --token on a command line) and the bot API token stays independent; tests prove the viewer is gated and the API is not
- [ ] #2 slot Xvfb in a container can be 832x640x24 (setting named in the card result) so the video shows the full 800x600 client; default stays 800x600
- [ ] #3 docs/research/tester-build-plan.md gap a, b, e updated with the result (estimate 4 h)
<!-- AC:END -->
