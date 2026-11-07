---
id: TASK-63
title: 'currency-rows: token row, drop line and bank recipe overlay'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-07T03:13:00Z'
labels:
  - d2d
  - feature
  - d2d-data
  - d2d-loot
  - metacurrency
  - T5
milestone: m-3
dependencies: []
priority: medium
ordinal: 61000
card: "currency-rows"
owns:
  - "data/overlay/didle-v1/"
  - "internal/overlay/didlev1_test.go"
  - "internal/loot/currencymark_test.go"
  - "internal/loot/pickit_starter.json"
  - "docs/research/currency-rows.md"
  - "docs/probes/currency1.md"
  - "docs/probes/currency1-drill.py"
  - "docs/overlay.md"
exclusive: []
contracts: []
result: "Offline proven 2026-11-06 (not yet verified live): fragments in data/overlay/didle-v1, go test -run Didle ./internal/overlay against the real tables (code free, Blood Raven class, one in 37.0 per kill, no earlier recipe takes mrk), pickit keep rule with a two-build test. Live drop count, d2d acceptance of the code and the bank recipe: docs/probes/currency1.md (ready-to-run drill, see its blocker)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

The token needs its data rows (item, drop line, bank recipe) as overlay fragments.

Done when: One Misc row with key art, one TC line, one CubeMain row; own-key prefix only; fork reads a new Misc code...; Offline probability test and stack-quantity roll recorded (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 One Misc row with key art, one TC line, one CubeMain row; own-key prefix only; fork reads a new Misc code from a d2s (probe)
- [ ] #2 Offline probability test and stack-quantity roll recorded
<!-- AC:END -->
