---
id: TASK-24
title: >-
  testerimage: Wine tester compose file, ffmpeg, watch ports (Deferred: Wine
  route)
status: Deferred
assignee: []
created_date: '2026-11-06T14:27:00Z'
updated_date: '2026-11-06T14:44:00Z'
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
DEFERRED 2026-11-06: the tester build is the d2d container (user decision, d2d first); this was the Wine-image version (appendix 'Wine route (later/optional)' of docs/research/tester-build-plan.md). Replaced by testerd2dimage (TASK-31). Kept for the Wine route; pick it up only if a Wine tester image is wanted later.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 image has ffmpeg, EXPOSE 8891 8892; compose.tester.yml sets D2BOT_LADDER, D2BOT_WATCH, D2BOT_WATCH_VIDEO, watch addrs, origin from an env var, D2FLEET_MAX_LOAD_FACTOR=-1, named volume, stop_grace_period 90s, ports bound to 127.0.0.1 by default
- [ ] #2 the static image test (go test ./deploy/container/...) pins these; the token file stays the single secret and is never in the compose file (estimate 4 h)
<!-- AC:END -->
