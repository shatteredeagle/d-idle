# Act 1 campaign specification

## Purpose
What an unattended Normal Act 1 run on the d2d backend must achieve for a fresh character: the quest order, the evidence a quest is done, the Andariel fight rules and the time budget to reach Act 2. Source: docs/roadmap.md milestone M3, docs/sme/play-act-route.md, d2-sme consultation 2026-10-31 (EXP unless tagged).

## ADDED Requirements

### Requirement: Unattended Act 1 clear
A fresh Sorceress on d2d without cheats SHALL kill Andariel and enter Act 2 on 3 of 3 seeds within a time cap of 1.3 x 176,132 XP (clvl 15, Arreat XP table via docs/sme/mechanics-progression.md) divided by the XP per hour measured at milestone M2, with no death-rule FAIL code, 0 panics and 0 manual interventions. The cap is U (set by the roadmap, d2-sme 2026-10-31) and is reviewed after the first run. Source: docs/roadmap.md M3; card m3-act1run.

#### Scenario: Cap at the baseline rate
- **WHEN** the M2 run measured 6600 XP/h
- **THEN** the cap is 35 virtual hours (1.3 x 176,132 / 6600 = 34.7, rounded up)

#### Scenario: One seed late
- **WHEN** two seeds enter Act 2 at hour 20 and the third is still in Act 1 at the cap
- **THEN** M3 is not done and the third seed's log yields a finding
