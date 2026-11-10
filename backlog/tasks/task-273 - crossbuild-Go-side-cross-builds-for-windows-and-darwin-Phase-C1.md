---
id: TASK-273
title: 'crossbuild: Go side cross-builds for windows and darwin (Phase C1)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T21:58:00Z'
labels:
  - bot
  - chore
dependencies: []
priority: medium
ordinal: 199000
card: "crossbuild"
result: "Verified on the worker host (not live game): tools/ci/crossbuild.sh ok for linux/amd64,arm64, windows/amd64, darwin/arm64,amd64 (CGO off; d2bot d2dsetup d2dview didleboot d2dping apps/desktop); whole-repo vet+test exit 0; Linux behaviour unchanged, binary sizes unchanged. check.sh runs the quick matrix only with CROSSBUILD=1; full matrix is the ci.yml crossbuild job. Windows/darwin runtime behaviour UNVERIFIED (stubs return unsupported)."
owns:
  - "internal/launch"
  - "cmd/didleboot"
  - "tools/ci/crossbuild.sh"
  - "cmd/d2bot"
  - "internal/fleet"
  - ".github/workflows/ci.yml"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 launch/fleet/didleboot/d2bot vet and build for windows/amd64 and darwin/arm64,amd64 without CGO
- [x] #2 tools/ci/crossbuild.sh matrix passes on the worker host; whole-repo gate green
<!-- AC:END -->
