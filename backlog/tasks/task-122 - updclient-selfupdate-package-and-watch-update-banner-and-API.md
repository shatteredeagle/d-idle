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
- [ ] #1 internal/selfupdate: manifest verify, fetch with ETag and caps, safe unpack, state with downgrade floor, signed REVOKED file and minimum version, engine state machine; offline tests incl. hash mismatch, bad signature, crafted tar
- [ ] #2 watch GET /api/update/status on the viewer and POST check/download/apply/cancel/rollback/inbox on the answer listener behind the token; banner with changelog and progress; never silent; offline switch D2BOT_UPDATE_CHECK=off
<!-- AC:END -->
