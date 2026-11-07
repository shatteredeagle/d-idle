---
id: TASK-32
title: 'testerd2dwiz: first-run wizard for the d2d tester container'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-06T14:44:00Z'
updated_date: '2026-11-06T16:00:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
dependencies: []
priority: high
ordinal: 32000
card: "testerd2dwiz"
owns:
  - "cmd/d2dsetup/**"
overlap: "TASK-44 (onboardA, 2026-11-06) extends cmd/d2dsetup/** with the six-screen stepper, go-ahead and access; merged testerd2dwiz code is edited there."
exclusive: []
contracts: []
result: "Offline only (fake MPQs, fake extractor, httptest): cmd/d2dsetup, own listener on 8891 before the bot, watch token gate and d2dview cookie, wizard.json plus wizard.env, resumable. AC1 open: the 1.14d proof is an EMPTY patch_sha256 list (verdict unverified until pinned from a real install), real MPQ sizes and data.Open on real data unproven, entrypoint hook and context.sh build line not applied (deploy/ not owned). AC2 page marked not run in a browser (node --check only). Status sync 2026-11-07: stays open, AC 1 needs the entrypoint hook and a pinned 1.14d hash from a real install; tester live run TASK-25 pending."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Slice 1 (d2d route), gap G2. Standalone small wizard mirroring cmd/d2fleet/internal/setup, no Wine steps. No dependency for the code; joins the image at testerd2dimage. Estimate 8 h.

Done when: steps: data check of /game (required d2data, d2exp, Patch_D2; 1.14d check; plain-language misses), table...; token gate and cookie, no field for keys or free text beyond the name, key-shaped input refused; tests... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 steps: data check of /game (required d2data, d2exp, Patch_D2; 1.14d check; plain-language misses), table extract, character name and class (class to d2d UI index from a data table, unknown class is an error naming it), one goals preset, Start; state in /data/wizard.json, resumable
- [x] #2 token gate and cookie, no field for keys or free text beyond the name, key-shaped input refused; tests with a fake /game folder and a fake backend; page checked in a browser or marked not run
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Groom 2026-11-06: merged on main (offline). Status set to In Progress, not Done: AC 1 is open (1.14d proof needs a pinned Patch_D2 hash from a real install; entrypoint hook and context.sh build line not applied, see TASK-31 and the backlog-groom doc).
<!-- SECTION:NOTES:END -->
