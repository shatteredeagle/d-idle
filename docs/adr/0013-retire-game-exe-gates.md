---
status: accepted, amended by 0015 ("Not changed" part; amended 2026-11-04: d2d is the end product, game.exe the oracle)
date: 2026-10-31
decision-makers: user ("retire g1")
---
# 0013: Retire the Game.exe gate soaks (G1, G2); the same gates run on d2d as a ladder

## Context and Problem Statement

G1 (2 h) and G2 (8 h) were gate soaks on the original Game.exe under Wine (instance i4, character Soaky, an injected Game.exe kill). ADR 0009 made d2d primary and froze Game.exe, and said G2 and the Fresha soak finish as the last Game.exe baselines. Gating releases on a frozen backend that no user ever runs (v1 is the d2d container) tests the wrong thing, and its fault (Game.exe kill, winedevice orphans) does not exist on d2d.

## Decision Outcome

G1 and G2 are retired as release gates, effective 2026-10-31. They are replaced by the same gates on a hidden d2d slot, a gate ladder:

* Gate 0: the live drill suite (`cmd/d2drill`, docs/probes/drills): every S and A primitive passes.
* Gate 1: 2 h unattended soak, one injected fault at about 60 min (kill the slot's d2d only, through the process registry), translated G1 criteria.
* Gate 2: 8 h soak (profile m24).
* Gate 3: 24 h soak (profile m24), as in the strategy's "Next week" table.

Criteria: spec soak-gates (change `d2d-gate-ladder`). Procedure: docs/review-strategy.md section 3, "d2d gate ladder".

## Consequences

* Game.exe baselines (g1-log attempts, Soaky 6.6k XP/h, the Fresha run) stay as historical references; they are not rerun.
* The kill-recovery and winedevice criteria have no d2d counterpart other than the new fault-injection criterion; Game.exe-only paths stay deferred (card gameexe-retire-soak).
* A candidate drill D10 (d2d kill, restart, verify save and loop) should join Gate 0 so the fault is first proven in minutes, not at minute 60.

## Not changed

* ADR 0009 stands: d2d primary, Game.exe frozen (keeps building, nothing new added, Wine paused, not deleted), exit trigger as written.
* Game.exe stays the reference: oracle, emulator and decomp work and the fidelity gate (spec fidelity-gate) continue.
