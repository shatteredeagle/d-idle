# Proposal: tierbdesign

## Why
Real A:B testing of the d2d backend against the original game (same bot commit, same character, compare XP, kills, deaths, drops and time per area within a stated noise band) needs either deterministic stepping of game.exe 1.14d (Tier B) or a statistical protocol. The clock hook of Tier A scales time only (docs/timescale.md, docs/research/lockstep-p3.md, lockstep-p4.md), so a seed-for-seed comparison is impossible today. The user approved Tier B design work on 2026-11-02.

## What Changes
Docs only: `docs/research/tierb-design-20261102.md` with the time-source inventory, three stepping designs, the statistical alternative with sample sizes, a staged card list and the mapping onto `openspec/specs/fidelity-gate`, plus the reading of the kitb (k=1) against kitk2 (k=2) pair. Private hook details go to the private hook repository's docs only.

## Not in scope
No build, no Go change, no game start, no live run, no hook change, no spec edit, no hook release decision.
