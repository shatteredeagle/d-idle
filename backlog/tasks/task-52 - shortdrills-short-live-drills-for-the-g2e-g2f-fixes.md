---
id: TASK-52
title: 'shortdrills: short live drills for the g2e/g2f fixes'
status: Done
assignee: []
created_date: '2026-11-06T22:34:00Z'
updated_date: '2026-11-07T15:14:00Z'
labels:
  - drills
dependencies: []
ordinal: 52000
card: "shortdrills"
owns:
  - "cmd/d2drill/e1_lab.go"
  - "cmd/d2drill/e1_lab_test.go"
  - "docs/probes/shortdrills.md"
  - "docs/probes/shortdrills"
exclusive: []
contracts: []
result: "VERIFIED LIVE 2026-11-06 (slots 4 and 5, fork f75816c6): merc-a, merc-b, merc-c, noportal-a, noportal-b all PASS, writes after start 0; merc drills set the purse around the level estimate because a cheat-hired merc revives for 0 (docs/probes/shortdrills.md). Edited 5 lines of cmd/d2drill/main.go (group E1 dispatch), owned by task m1-liveproof."
---

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Drill 2 (merc revive before potions, poor variant, clvl 14 variant) built and run live, result recorded
- [x] #2 Drill 3 (no-portal low life leaves after bot.noPortalWaitSec, scroll control) built and run live, result recorded
- [x] #3 docs/probes/shortdrills.md records run ids, build, binary sha and PASS/FAIL with log lines
<!-- AC:END -->
