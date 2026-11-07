---
id: TASK-66
title: 'g2gfixmerc: merc caps and gold-farm need on the virtual clock and the read price'
status: In Progress
assignee: []
created_date: '2026-11-07T07:05:00Z'
updated_date: '2026-11-07T07:13:00Z'
labels:
  - bot-defect
dependencies: []
priority: high
ordinal: 64000
card: "g2gfixmerc"
owns:
  - "cmd/d2bot/mercwire.go"
  - "internal/merc/policy.go"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
docs/research/g2g-diagnosis.md findings M1 to M3: mercWire uses time.Now so merc.reviveMaxPerHour and stopDeathsPerHour count wall hours (about 4 game hours under lockstep); gold farm need and skip lines use the level^2 estimate (1470 to 2160) while the read quote is 60 to 270.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 mercWire clock is the env clock (virtual under lockstep); test with a stepped clock
- [x] #2 gold farm need uses the last read quote when one exists; two builds tested
- [x] #3 contracts merc-policy.md named in the card result if its text changes
<!-- AC:END -->

result: mercWire.now is the env clock (virtual under lockstep, wall fallback); the last real quote (readQuote, valid per merc level and merc.quoteMaxAgeHours (4) game hours) feeds GoldFarm, ShouldRevive and fundRevive, the formula only before the first read. Offline tests: TestMercWindowsFollowTheEnvClock, TestQuoteReplacesTheFormulaEverywhere (Sorceress, Paladin). Live drill E1.merc-quote BLOCKED (cheat-hired merc quotes 0), NOT verified live. merc-policy.md text unchanged. Also edited internal/env/tuning_merc.go (knob), cmd/d2drill/e1_lab.go.
