---
id: TASK-24
title: "testerimage: tester compose file, ffmpeg, watch ports"
status: Proposed
assignee: []
created_date: '2026-11-06T14:27:00Z'
labels:
  - G-F
dependencies: []
priority: high
ordinal: 24000
card: "testerimage"
owns:
  - "deploy/container/Dockerfile"
  - "deploy/container/d2start.sh"
  - "deploy/container/compose.tester.yml"
  - "deploy/container/image_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1. Depends on testerwatch. Build runs on the worker host, not the head box.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 image has ffmpeg, EXPOSE 8891 8892; compose.tester.yml sets D2BOT_LADDER, D2BOT_WATCH, D2BOT_WATCH_VIDEO, watch addrs, origin from an env var, D2FLEET_MAX_LOAD_FACTOR=-1, named volume, stop_grace_period 90s, ports bound to 127.0.0.1 by default
- [ ] #2 the static image test (go test ./deploy/container/...) pins these; the token file stays the single secret and is never in the compose file (estimate 4 h)
<!-- AC:END -->
