---
id: TASK-1
title: "m3-act1run: fresh Sorceress clears Act 1 unattended on d2d"
status: Deferred
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-B"
  - "soak"
dependencies:
  - "TASK-2"
priority: high
ordinal: 1000
card: "m3-act1run"
owns:
  - "docs/act1-run.md"
  - "docs/probes/act1run/**"
exclusive:
  - "soak-slot:3"
branch: "agent/m3-act1run"
agent: "m3-act1run"
result: ""
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/m3-act1run"
contracts: ["proposed/m3-act1run"]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Milestone M3 (docs/roadmap.md) is done when a fresh Sorceress clears Act 1 unattended on d2d without cheats on repeated seeds, under the death rule, with the merc kept; every earlier M3 card proves one piece on a cheat-levelled character. This is the closing run and its evidence doc; it changes no code and files one finding per failure for a bug-fix card.

What changes:
- docs/act1-run.md: 3 fresh seeds (no cheats), wall-limited at the time cap of the act1-campaign spec, started through tools/d2dslot.sh under tools/agentrun.sh: per seed the virtual hour of each quest, the Andariel attempts, deaths and the d2dsoak death codes, merc uptime, XP/h by area.
- docs/probes/act1run/: minute lines and the d2dsoak reports.
- Delta: act1-campaign ADDED `Unattended Act 1 clear`.

Next step: not run live; absorbs the quest-chain and readiness floors text

Gates (from the card):
- 3 of 3 fresh Sorceress seeds, no cheats: Andariel dead and Act 2 entered (area 40 or the Act 2 quest word) before the time cap of act1-campaign `Unattended Act 1 clear`; d2dsoak reports no death-rule FAIL code (death_streak, death_recovery_stuck, death_rate); 0 panics; 0 manual interventions
- merc alive at least 70 percent of field time after the claim on every seed (minute lines)
- tools/agentreap.sh m3-act1run leaves nothing; changecheck -cli 0 warnings; leakcheck 0; validate --strict passes

Source: git tag openspec-final, openspec/changes/m3-act1run/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 1.1 Compute the cap from the M2 result (1.3 x 176,132 / the M2 XP/h) and write it in the delta with its source
- [ ] #2 1.2 3 seeds, one slot, each wall-limited; collect reports
- [ ] #3 1.3 docs/act1-run.md; a finding per failure for the next card; changecheck, leakcheck, validate; reap
<!-- AC:END -->
