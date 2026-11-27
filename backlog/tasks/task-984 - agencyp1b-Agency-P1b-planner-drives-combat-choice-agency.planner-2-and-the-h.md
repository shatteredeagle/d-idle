---
id: TASK-984
title: 'agencyp1b: Agency P1b: planner drives combat choice (agency.planner 2) and the h'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-26T03:57:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 472000
card: "agencyp1b"
owns:
  - "docs/design/agency-p1b.md"
  - "internal/agency/drive.go"
  - "internal/agency/drive_test.go"
  - "cmd/d2bot/agencywire.go"
  - "cmd/d2bot/agencydrive_test.go"
  - "cmd/d2bot/d2dwire.go"
  - "cmd/d2bot/live.go"
  - "internal/combat/drive_test.go"
  - "internal/combat/fighter.go"
  - "internal/combat/aura.go"
  - "internal/env/tuning_agency.go"
exclusive: []
contracts: []
result: "Partial, not verified live. agency.DriveOf and the fighter wiring are built (Fighter.Drive seam, agencyWire.Drive, knob agency.planner Max 2, default stays 0, offline mode 0/1/2 test for Paladin and Sorceress). The shipped capability table has no proven attack or aura, so mode 2 changes nothing today; shadow review, owner go, the live drill and the flip are owner/live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Recommendation of the 2026-11-19 design review (06-roadmap R7; ADR 0020 item 2): after the capability table and planner in shadow (agencyp1a, merged), the planner drives (mode 2) and a viability line is shown; guide tracks as build files. No card exists past P1a. Knob agency.planner is default 0 (mode 1 shadow; ledger row says: shadow lines replayed and reviewed with no wrong pick; owner go for mode 2). Depends on dlv-data (one capability table) for the tier source. Risk: medium. Enable criteria: shadow lines replayed and reviewed, no wrong pick; owner go; then flip with a 24 h watch and auto-revert.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Review of the planner shadow lines (replay) recorded
- [x] #2 Mode 2 behind its knob with the viability line
<!-- AC:END -->
