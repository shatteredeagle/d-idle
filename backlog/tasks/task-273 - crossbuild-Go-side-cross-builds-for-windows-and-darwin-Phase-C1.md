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
- [ ] #1 launch/fleet/didleboot/d2bot vet and build for windows/amd64 and darwin/arm64,amd64 without CGO
- [ ] #2 tools/ci/crossbuild.sh matrix passes on the worker host; whole-repo gate green
<!-- AC:END -->
