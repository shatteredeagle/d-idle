---
id: TASK-56
title: 'fidprobe-vendor: vendor stock and quote probe, phase 2'
status: Proposed
assignee: []
created_date: '2026-11-06T22:43:00Z'
updated_date: '2026-11-06T22:43:00Z'
labels:
  - gate
dependencies:
  - TASK-55
ordinal: 54000
card: "fidprobe-vendor"
owns:
  - "internal/fidvendor"
  - "docs/research/fidelity-vendor-probe.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Slice 2. After fidprobe-core (registers as a phase-2 scenario). Needs the user's answer to open question 4 (input on the hidden game.exe instance).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 internal/fidvendor: vendor window reader on game.exe and d2d (stock list, per-item price), diff against price_model quotes; offline tests
- [ ] #2 run on saves 5 (poor/rich) and 1 with Akara, Charsi, Gheed, Kashya: input only through d2input, zero memory writes, character restored from a save copy
- [ ] #3 fidelity-vendor-probe.md: stock and quote differences per vendor, whether stock is seed-equal, matrix rows VN01-VN05 updated
<!-- AC:END -->
