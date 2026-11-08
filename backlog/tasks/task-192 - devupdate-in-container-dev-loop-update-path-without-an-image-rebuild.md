---
id: TASK-192
title: 'devupdate: in-container dev-loop update path without an image rebuild'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T15:24:00Z'
labels:
  - container
  - feature
milestone: m-1
dependencies:
  - TASK-122
  - TASK-123
priority: high
ordinal: 154000
card: "devupdate"
owns:
  - "cmd/didleboot"
  - "internal/selfupdate"
  - "cmd/d2bot/update.go"
  - "deploy/d2d-container"
  - "tools/dev-update.sh"
  - "tools/dev-update-host.sh"
  - "docs/dev-update.md"
  - "docs/first-session.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Dev tooling on the designed updater; no registry, no CI key. Done when a bot change reaches a running container with one command.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Dev source (D2BOT_UPDATE_URL, D2BOT_UPDATE_PUBKEY, D2BOT_UPDATE_CHECK=on|off|manual) in the client and shim, gated by the DEV image; production defaults unchanged
- [ ] #2 Shim reloads on SIGHUP and has check-now / rollback-now; dev probation
- [ ] #3 tools/dev-update.sh builds, signs with the dev key, publishes to the LAN dir, updates a running container, rollback works
- [ ] #4 Drill on a separate test container: two updates without rebuild, bad signature and version rollback rejected, volume survives
- [ ] #5 docs/dev-update.md and first-session pointer
<!-- AC:END -->
