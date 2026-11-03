## Why

The fidelity gate asks whether d2d plays like game.exe at bot level (XP per hour, kills, deaths, potions, town trips, time per area). Records of both arms already exist (game.exe late set and early baselines, five d2d gate soaks). Nobody had put them side by side with intervals that carry both arms' error, nor listed which comparisons are apples to apples. This card does that offline, from committed records only, and sizes the new runs needed to answer "is d2d within 15 percent of game.exe on XP/h".

## What Changes

- docs/research/botab-20261102.md: the per-metric table (d2d, game.exe, sample sizes, spread, ratio with a 95 percent interval), the comparability table, the gap list and the minimum new runs.
- docs/research/botab-trace.md: Q, D (d2d vs game.exe), B (our bot) lists, limits and matrix proposals.
- tools/baseline/botab_extract.py and tools/baseline/botab_run.py: read-only extractor and driver that reuse tools/baseline/ab_compare.py.

## Impact

Docs and offline tools only. No game, no Go change, no spec change, no ledger row (nothing here proves a rule; the ledger needs case counts of rule tests).

## Overlap (declared)

Reads the records of the gate1run, gate2run and baselineruns cards and the design of tierbdesign; edits none of them.
