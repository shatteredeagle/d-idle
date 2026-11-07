---
id: TASK-56
title: 'fidprobe-vendor: vendor stock and quote probe, phase 2'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-06T22:43:00Z'
updated_date: '2026-11-07T18:04:00Z'
labels:
  - bot
  - probe
  - d2d-town
  - d2d-vendor
  - fidelity
  - gate
milestone: m-2
dependencies:
  - TASK-55
priority: medium
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
Size: M

Slice 2. After fidprobe-core (registers as a phase-2 scenario). Needs the user's answer to open question 4 (input on the hidden game.exe instance).

Done when: internal/fidvendor: vendor window reader on game.exe and d2d (stock list, per-item price), diff against...; run on saves 5 (poor/rich) and 1 with Akara, Charsi, Gheed, Kashya: input only through d2input, zero...; fidelity-vendor-probe.md: stock and quote differences per vendor, whether stock is seed-equal, matrix rows... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 internal/fidvendor: vendor window reader on game.exe and d2d (stock list, per-item price), diff against price_model quotes; offline tests
- [ ] #2 run on saves 5 (poor/rich) and 1 with Akara, Charsi, Gheed, Kashya: input only through d2input, zero memory writes, character restored from a save copy
- [ ] #3 fidelity-vendor-probe.md: stock and quote differences per vendor, whether stock is seed-equal, matrix rows VN01-VN05 updated
<!-- AC:END -->
