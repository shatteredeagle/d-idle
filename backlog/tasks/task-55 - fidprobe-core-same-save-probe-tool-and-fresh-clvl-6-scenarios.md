---
id: TASK-55
title: 'fidprobe-core: same-save probe tool and fresh/clvl 6 scenarios'
status: In Progress
assignee:
  - live-agent
created_date: '2026-11-06T22:43:00Z'
updated_date: '2026-11-07T18:04:00Z'
labels:
  - bot
  - probe
  - d2d-items
  - d2d-network
  - d2d-skills
  - fidelity
  - gate
milestone: m-2
dependencies: []
priority: high
ordinal: 53000
card: "fidprobe-core"
owns:
  - "cmd/d2fidprobe"
  - "internal/fidprobe"
  - "docs/research/fidelity-probe-run1.md"
exclusive: []
contracts: []
result: "first LIVE run 2026-11-07 (i5 + d2d slot 3, lab verbs off): scenarios 1-4 read-only, writes after start 0, all MISMATCH. Run 2 same day: d2s file reader (third reference, verified live on 3 saves), reader-scope normalisation, scenario 15 load-life-mana: d2d heals life/mana to full at load (12 of 12 loads, game.exe keeps saved value minus fraction), Paladin max mana 19.5 vs 19.0, poison res 6 vs 0 unresolved; docs/research/fidelity-probe-run1.md. Missing: true fresh Paladin and separate sorc-6-gear fixtures; items UNREADABLE on d2d. Verified live, writes after start 0."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Slice 1 of docs/research/fidelity-matrix.md. Needs a coordinator step to start and log in one hidden instance. Design: docs/research/fidelity-probe-design.md.

Done when: internal/fidprobe: Sheet, three readers (d2s file, game.exe mem, d2d dump), Diff, Expected table,...; cmd/d2fidprobe wired; scenarios 1-4 of docs/research/fidelity-probe-design.md section 5 run read-only, no...; fidelity-probe-run1.md: mismatches and UNREADABLE fields listed against matrix rows; matrix rows updated... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 internal/fidprobe: Sheet, three readers (d2s file, game.exe mem, d2d dump), Diff, Expected table, Markdown/JSON report; offline tests on recorded snapshots
- [x] #2 cmd/d2fidprobe wired; scenarios 1-4 of docs/research/fidelity-probe-design.md section 5 run read-only, no input, writes after start 0
- [x] #3 fidelity-probe-run1.md: mismatches and UNREADABLE fields listed against matrix rows; matrix rows updated by the coordinator
<!-- AC:END -->
