---
id: TASK-122
title: 'updclient: selfupdate package and watch update banner and API'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T23:28:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
dependencies:
  - TASK-25
  - TASK-120
  - TASK-33
priority: high
ordinal: 115000
card: "updclient"
owns:
  - "internal/selfupdate"
  - "internal/watch/update.go"
  - "internal/watch/web/update.js"
  - "internal/watch/update_test.go"
  - "internal/watch/serve.go"
  - "cmd/d2bot/update.go"
  - "cmd/d2bot/update_test.go"
  - "cmd/d2bot/ladder.go"
  - "docs/self-update-client.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Tier 2 part 2 of docs/research/self-update-design.md sections 3.5 to 3.7. Phase 1 (version display and check banner without apply) can ship first.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 internal/selfupdate: manifest verify, fetch with ETag and caps, safe unpack, state with downgrade floor, signed REVOKED file and minimum version, engine state machine; offline tests incl. hash mismatch, bad signature, crafted tar
- [x] #2 watch GET /api/update/status on the viewer and POST check/download/apply/cancel/rollback/inbox on the answer listener behind the token; banner with changelog and progress; never silent; offline switch D2BOT_UPDATE_CHECK=off
<!-- AC:END -->

result: "offline only, NOT verified against real GitHub or in a container. internal/selfupdate (fetch with ETag/caps/host allow-list, safe unpack, state with seq/version floors, REVOKED + min version, engine, inbox) tested against a fake GitHub (httptest): good update, hash mismatch, bad signature, tampered manifest, forged REVOKED, offline (zero calls), downgrade, revoked, needs-image, 304/rate limit, crafted tars, rollback marker. Watch: GET status on the viewer, POST routes on the answer listener behind token + origin, banner script injected into /watch pages, wired in cmd/d2bot (one line in ladder.go; owns widened to serve.go, cmd/d2bot update files, ladder.go, docs/self-update-client.md). IdleSafe is nil in d2bot (U what safe means). See docs/self-update-client.md. Status sync 2026-11-08 (stays open): AC left open per result (offline only, container/real GitHub not run)."
