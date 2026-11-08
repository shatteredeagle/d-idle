---
id: TASK-192
title: 'devupdate: in-container dev-loop update path without an image rebuild'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T15:24:00Z'
updated_date: '2026-11-08T16:12:00Z'
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
result: "Dev loop works end to end: UPDATED in 5 to 6 s per update (warm cache), rollback 2 s, bad signature, missing sig, tamper, hash, downgrade all REJECTED, crashing build rolled back by the shim, volume survives. VERIFIED LIVE on a throwaway container (setup-wizard state, no game, no hero; a hero mid-fight is UNVERIFIED). Offline: internal/selfupdate/dev_test.go, cmd/didleboot/dev_test.go, real_test.go, deploy tests. Not covered: d2d binary and data via the dev script (format supports it), shim/Dockerfile changes need a rebuild. Docs: docs/dev-update.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Dev tooling on the designed updater; no registry, no CI key. Done when a bot change reaches a running container with one command.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Dev source (D2BOT_UPDATE_URL, D2BOT_UPDATE_PUBKEY, D2BOT_UPDATE_CHECK=on|off|manual) in the client and shim, gated by the DEV image; production defaults unchanged
- [x] #2 Shim reloads on SIGHUP and has check-now / rollback-now; dev probation
- [x] #3 tools/dev-update.sh builds, signs with the dev key, publishes to the LAN dir, updates a running container, rollback works
- [x] #4 Drill on a separate test container: two updates without rebuild, bad signature and version rollback rejected, volume survives
- [x] #5 docs/dev-update.md and first-session pointer
<!-- AC:END -->
