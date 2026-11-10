---
id: TASK-268
title: 'artspeed: inventory page art loads fast, art pre-extracted at onboarding'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-10T21:01:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 197000
card: "artspeed"
owns:
  - "internal/art"
  - "internal/watch/artapi.go"
  - "internal/watch/artapi_test.go"
  - "internal/watch/web/inventory.js"
  - "internal/data/invfiles.go"
  - "cmd/d2bot/artwire.go"
  - "cmd/d2bot/artcmd.go"
  - "docs/paperdoll-art-build.md"
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
- [ ] #1 Measured cause of the slow /watch/inventory recorded (V)
- [ ] #2 Page never blocks first paint on art; art swaps in when ready
- [ ] #3 d2bot art extract: idempotent one-shot extraction with progress and missing-MPQ message
- [ ] #4 Background extraction at start; /api/art reports state and counts
- [ ] #5 Docs: onboarding and dev-update warm path; regression tests and benchmark
<!-- AC:END -->
