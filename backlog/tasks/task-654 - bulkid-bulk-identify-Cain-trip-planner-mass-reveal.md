---
id: TASK-654
title: 'bulkid: bulk identify, Cain trip planner, mass reveal'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-18T21:49:00Z'
updated_date: '2026-11-27T02:02:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 325000
card: "bulkid"
owns:
  - "internal/bulkid"
  - "data/bulkid"
  - "internal/env/tuning_identify.go"
  - "internal/watch/vaultmass_test.go"
  - "internal/env/tuning.go"
  - "docs/knobs.md"
  - "internal/vaultkeep/reveal.go"
  - "internal/vaultkeep/keep.go"
  - "internal/vaultkeep/note_test.go"
  - "internal/watch/vaultapi.go"
  - "internal/watch/web/vault.js"
  - "internal/watch/web/vault.json"
  - "cmd/d2bot/vaultkeep.go"
  - "cmd/d2bot/vaultpage.go"
exclusive: []
contracts: []
result: "B1-B5 built offline: identify.* knobs and filter table, planner, chunked stash-pack moves with journal, Cain trip orchestrator on fakes, mass reveal preview and Cain-not-rescued note in the vault view and page. NOT wired to the town click path and NOT run live (design section 9 drill). closecards3 Remaining (live): #5 need the live game, a browser or a worker-host run; not verified here."
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Slices of docs/design/bulk-identify-2026-11-18.md. Offline only; the live Cain drill is section 9 of the design.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 identify.* knobs, filter table and pure planner (B1)
- [x] #2 chunked stash-pack moves with journal and read-back (B2)
- [x] #3 Cain trip orchestrator with fakes (B3)
- [x] #4 mass reveal preview, Cain-not-rescued note, vault page button (B4, B5)
- [ ] #5 Town adapters for Mover, Lister and CainTalk wired into the town visit (live-dependent, left for the drill)
<!-- AC:END -->
