---
id: TASK-42
title: 'fork-labverbs: gate the mutating debug verbs behind a fork flag'
status: Proposed
assignee: []
created_date: '2026-11-06T16:02:00Z'
updated_date: '2026-11-06T22:05:00Z'
labels:
  - tester
dependencies: []
priority: medium
ordinal: 42000
card: "fork-labverbs"
owns:
  - "docs/lab-verbs.md"
  - "deploy/d2d-container/"
  - "cmd/d2dsetup/"
  - "cmd/d2drill/"
  - "cmd/d2dsoak/lab.go"
  - "tools/d2dview-demo.sh"
  - "tools/d2dslot-test.sh"
  - "docs/d2d-container/README.md"
exclusive:
  - "fork-branch:d2d.lab3-labverbs"
contracts: []
result: "Verified live on the worker host 2026-11-06 (V): fork d2d.lab3-labverbs f75816c616c8 (on the private remote) built (Release with tests, and on the NAS builder as d2d.lab3-labverbs-f75816c6-linuxbuild, sha256 53217ea4ecc8a4d3...; the Gate 2 binary is untouched). ctest lab_verbs_d2d, coop_d2d, coop_lobby_d2d, pad_use_d2d pass; smoke_d2d fails at cmd skill didnt hurt on the old Gate 2 binary too (pre-existing, not the gate). Bot play path without the flag: d2bot managed lockstep, 6 min, 294 kills, 0 errors, 0 lab verbs off; dump and the read-only debug verbs answer; clearinv, durability, stat and give reply err lab verbs off without the flag and work with it. Pin moved in context.sh; --lab-verbs passed by d2drill, d2dsoak lab, d2dview-demo, d2dslot-test and the ops gate supervisor (only when the binary knows it). d2dsetup hook in the tester entrypoint; 1.14d Patch_D2 hash pinned (V: the owner's install, Game.exe 1.14.3.71). Tester image built and tester smoke PASS (408 MB, about 6 min). There is no env twin D2D_LAB_VERBS (CLI flag only)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
docs/d2d-container/README.md: at the pinned fork ref the debug verb group answers unconditionally on the container-internal socket. Proposed: --lab-verbs (env D2D_LAB_VERBS=1), passed by the lab and gate harness, never by the tester entrypoint. Work is in the private fork, committed and pushed the same day.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Fork: build and ctest pass with the flag off and on; read-only verbs (dump, level, players, warps, portals, presets, grid, nearby, itemat, missiles, states, blocked) stay ungated; commit is on the fork remote
- [x] #2 Bot play path unchanged without the flag (a short bot run on the gated build); the doc lists every gated verb
- [x] #3 Pin moved in the tester image; the bot-repo gate and drill launch flags updated (the private ops side is reported, not committed here); says verified live or not
<!-- AC:END -->
