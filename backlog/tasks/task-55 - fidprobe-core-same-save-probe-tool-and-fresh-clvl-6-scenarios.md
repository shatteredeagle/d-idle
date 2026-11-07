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
result: "offline: engine, tool and tests built (two readers + replay; d2s file reader, merc and affix fields not yet); NOT run live, AC all open"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Slice 1 of docs/research/fidelity-matrix.md. Needs a coordinator step to start and log in one hidden instance. Design: docs/research/fidelity-probe-design.md.

Done when: internal/fidprobe: Sheet, three readers (d2s file, game.exe mem, d2d dump), Diff, Expected table,...; cmd/d2fidprobe wired; scenarios 1-4 of docs/research/fidelity-probe-design.md section 5 run read-only, no...; fidelity-probe-run1.md: mismatches and UNREADABLE fields listed against matrix rows; matrix rows updated... (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 internal/fidprobe: Sheet, three readers (d2s file, game.exe mem, d2d dump), Diff, Expected table, Markdown/JSON report; offline tests on recorded snapshots
- [ ] #2 cmd/d2fidprobe wired; scenarios 1-4 of docs/research/fidelity-probe-design.md section 5 run read-only, no input, writes after start 0
- [ ] #3 fidelity-probe-run1.md: mismatches and UNREADABLE fields listed against matrix rows; matrix rows updated by the coordinator
<!-- AC:END -->
