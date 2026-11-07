---
id: TASK-57
title: 'tpcount: minute-line tp reads 0 in the g2g runs'
status: Done
assignee: []
created_date: '2026-11-07T01:55:00Z'
updated_date: '2026-11-07T15:14:00Z'
labels:
  - bot-defect
dependencies: []
ordinal: 55000
card: "tpcount"
owns:
  - "cmd/d2bot/beltfill_adapter.go"
  - "cmd/d2bot/d2dwire_test.go"
  - "cmd/d2drill/e1_lab.go"
  - "cmd/d2drill/e1_lab_test.go"
  - "docs/probes/shortdrills.md"
exclusive: []
contracts: []
result: "VERIFIED LIVE 2026-11-06 (slot 4, fork f75816c6): E1.tp-a, tp-b, tp-c PASS with writes after start 0; the pre-fix d2bot FAILs tp-a (tp=0). Cause: townAdapter hid bot.TPDiag; fixed. Count was right (docs/probes/shortdrills.md, drill 4)."
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Cause found and fixed with a unit test per wrapper
- [x] #2 Drill E1.tp-a/b/c run live, result recorded
<!-- AC:END -->
