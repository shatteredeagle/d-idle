---
id: TASK-32
title: 'testerd2dwiz: first-run wizard for the d2d tester container'
status: Proposed
assignee: []
created_date: '2026-11-06T14:44:00Z'
labels:
  - G-F
dependencies: []
priority: high
ordinal: 32000
card: "testerd2dwiz"
owns:
  - "cmd/d2dsetup/**"
exclusive: []
contracts: []
result: "Offline only (fake MPQs, fake extractor, httptest): cmd/d2dsetup, own listener on 8891 before the bot, watch token gate and d2dview cookie, wizard.json plus wizard.env, resumable. AC1 open: the 1.14d proof is an EMPTY patch_sha256 list (verdict unverified until pinned from a real install), real MPQ sizes and data.Open on real data unproven, entrypoint hook and context.sh build line not applied (deploy/ not owned). AC2 page marked not run in a browser (node --check only)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 1 (d2d route), gap G2. Standalone small wizard mirroring cmd/d2fleet/internal/setup, no Wine steps. No dependency for the code; joins the image at testerd2dimage. Estimate 8 h.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 steps: data check of /game (required d2data, d2exp, Patch_D2; 1.14d check; plain-language misses), table extract, character name and class (class to d2d UI index from a data table, unknown class is an error naming it), one goals preset, Start; state in /data/wizard.json, resumable
- [x] #2 token gate and cookie, no field for keys or free text beyond the name, key-shaped input refused; tests with a fake /game folder and a fake backend; page checked in a browser or marked not run
<!-- AC:END -->
