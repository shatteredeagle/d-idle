---
id: TASK-66
title: 'g2gfixmerc: merc caps and gold-farm need on the virtual clock and the read price'
status: Proposed
assignee: []
created_date: '2026-11-07T07:05:00Z'
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
- [ ] #1 mercWire clock is the env clock (virtual under lockstep); test with a stepped clock
- [ ] #2 gold farm need uses the last read quote when one exists; two builds tested
- [ ] #3 contracts merc-policy.md named in the card result if its text changes
<!-- AC:END -->
