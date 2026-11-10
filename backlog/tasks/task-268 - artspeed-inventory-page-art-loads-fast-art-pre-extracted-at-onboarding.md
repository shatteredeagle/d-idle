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
  - "docs/dev-update.md"
  - "internal/data/invfiles_test.go"
  - "internal/watch/artembed_test.go"
  - "cmd/d2bot/artcmd_test.go"
  - "cmd/d2bot/d2dbackend.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/main.go"
  - "cmd/d2bot/invwire.go"
  - "internal/invdesk/desk.go"
  - "internal/invdesk/desk_test.go"
  - "internal/loot/loot.go"
  - "internal/loot/evaluatorfor_test.go"
  - "internal/watch/invapi.go"
  - "internal/watch/invapi_test.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: the inventory page was slow with art on. Measured causes: art decode is not it (open 7 ms, 30 cold icons 25 ms); the page waterfall (probe chain before the first load) and, on every poll, GET /api/inventory judging each item through a fresh game snapshot read (about 30 reads, state 15.7 ms + missiles 15.5 ms each on d2d). Done: load first, art beside it, background pre-extraction, d2bot art extract, state on /api/art. Result: measured V on the worker host with the real install; the page in a browser and the sandbox round trip time are not run (U); not deployed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Measured cause of the slow /watch/inventory recorded (V)
- [x] #2 Page never blocks first paint on art; art swaps in when ready
- [x] #3 d2bot art extract: idempotent one-shot extraction with progress and missing-MPQ message
- [x] #4 Background extraction at start; /api/art reports state and counts
- [x] #5 Docs: onboarding and dev-update warm path; regression tests and benchmark
<!-- AC:END -->
