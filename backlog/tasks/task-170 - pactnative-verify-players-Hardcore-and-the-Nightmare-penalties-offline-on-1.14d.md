---
id: TASK-170
title: 'pactnative: verify /players, Hardcore and the Nightmare penalties offline on 1.14d'
status: Proposed
assignee:
  - live-agent
created_date: '2026-11-08T03:34:00Z'
updated_date: '2026-11-08T04:02:00Z'
labels:
  - bot
  - research
milestone: m-4
dependencies: []
priority: low
ordinal: 146000
card: "pactnative"
owns:
  - "docs/research/pact-native.md"
  - "docs/probes/pactnative.md"
  - "docs/probes/pactnative/"
  - "cmd/pactprobe"
  - "cmd/mpqcat"
exclusive:
  - "instance:i5"
contracts: []
result: "2026-11-08, docs/probes/pactnative.md. VERIFIED LIVE (i5, game.exe 1.14d, zero writes): /players typed through the input helper works offline (game prints players set to N), monster hits-to-kill rise 1.00 / 1.36 / 1.75 / 2.25 for N=1/2/4/8 (Fallen, Blood Moor; capped below by one-hit kills, so the exact life factor stays S from the decompilation, 1.5/2.5/4.5); client life reads 128/128 so real max life is not readable; a hardcore hero is created and played offline (save status byte 0x24). READ FROM TABLES (T): DifficultyLevels.txt Nightmare -40 resist and 5 pct death XP, Hell -100 and 10; Levels.txt MonUMin/MonUMax (+N,H) drive the elite count (Blood Moor 0 / 4-5 / 7-9). NOT measured: damage scaling, drop and XP effect (U, S from fork notes); d2d has no /players command (S, fork source read). AC 1 left open."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Size: S. Needs the live agent. Not on the two-week playable path. Design note: docs/research/pact-system.md (a design note, not a decision).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Live: /players 1..8 offline on 1.14d works; monster life, damage, drops and XP effect measured
- [x] #2 Live or table check: Nightmare death XP penalty and resist penalty (-5 pct, -40) and Hell values
- [x] #3 Whether a Hardcore hero can be created and played offline; every number tagged V or U
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Decides whether /players Pacts exist (ADR 0019).
<!-- SECTION:NOTES:END -->
