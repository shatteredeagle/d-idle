---
status: accepted
date: 2026-11-03
decision-makers: user
---
# 0015: Acceptance is game.exe, d2d is the lab; six goals G-A to G-F

## Context and Problem Statement

The project review of 2026-11-03 ([research/project-review-20261103.md](../research/project-review-20261103.md)) found two incompatible end goals on paper. ADR 0009 made the d2d container the v1 product ("users never run Game.exe"); the user's wish is a bot that plays the original game. Since ADR 0013 every gate runs on d2d, the container has had no work for a week, d2d still deviates from game.exe exactly where the gated metrics come from (rule-coverage), seed-for-seed comparison with game.exe is impossible (tierb-design), and every gate record was a single seed with 14 to 30 percent run-to-run noise.

## Decision Outcome

1. The project is done when a fresh, uncheated character plays 24 h unattended on game.exe 1.14d on a hidden lane and clears Act 1 (Andariel dead, Act 2 entered), `writes after start: 0`.
2. d2d stays the primary development backend and runs the gate ladder (Gate 0 to 3) for stability.
3. A final rung, Gate A, runs on game.exe: 2 h, then 8 h, then 24 h.
4. The game.exe freeze is lifted for fixes that Gate A finds; no new features that d2d does not get.
5. The d2d container is deferred until Gate A passes and the licence route is decided.
6. Transfer from d2d to game.exe is measured by the statistical A:B of the fidelity-gate spec, not assumed.
7. The six goals below replace the milestone ladder of ADR 0012 as the way work is ranked; ADR 0012's WIP limit (3 implementation agents) stays and is enforced.
8. XP/h is never a per-run pass or fail. Per-run gates judge stability, invariants and funded belt windows; XP/h is judged on a seed set (at least 5 seeds, 2 of them holdout seeds 16 to 23). The soak-gates delta for this is in the card specdelta. Single-seed Gate 1 re-runs on seed 11 are dropped.
9. The headline progress metric is milestone time: virtual hours from a fresh character to Andariel dead, to Act 2 entered, to Duriel dead, per seed. XP/h is a diagnostic.
10. Open-ended rule coverage is dropped (rows only for what the bot depends on). The d2d container, the fallback-lightning-v3 default switch, the idle layer, stream-grid and d2dmesh stay deferred or parked; Tier B stays held. Barbarian deviation work is dropped unless the class check shows a Barbarian gate character; until a check says otherwise the gate character is treated as a Sorceress (lightning build).
11. One page of truth: STATUS.md hand-written part is dated within 3 days, open cards are capped at 12, every card maps to one goal.

### The goals

| id | goal | done when |
|---|---|---|
| G-A | Acceptance on game.exe | one 24 h hidden game.exe run from a fresh uncheated character: Gate 1 stability criteria hold, Andariel dead, Act 2 entered |
| G-B | Act 1 clear on d2d (old M3) | 3 of 3 seeds (one holdout) fresh, uncheated, Den, free Rogue, Andariel, Warriv, inside the death rule |
| G-C | Stability on d2d | Gate 2 (8 h) PASS on 3 seeds incl. 1 holdout, then Gate 3 (24 h) PASS on 1 seed; XP/h is not a per-run criterion |
| G-D | Transfer is measured | statistical A:B in the fresh clvl 1 to 6 cell, 6 game.exe + 6 d2d lanes: verdict EQUIVALENT or DIFFERENT at 20 percent on XP/h and kills/h |
| G-E | Upstream: fix what the bot depends on, one batch at a time | each merged batch fixes a deviation on the bot's path; no new prep beyond the next batch while one is open |
| G-F | One page of truth | STATUS fresh, at most 12 open cards, each mapped to G-A..G-E, WIP limit honoured |

## Consequences

* Amends ADR 0009 items 1 and 5 (v1 product, retirement at the container soak) and the "Not changed" part of ADR 0013; supersedes the milestone ordering of ADR 0012 and the container end goal of ADR 0001. The older ADR files are left as written (guard rule: ADRs are not edited); this record is the pointer.
* game.exe memory reading stays maintained; Gate A lanes on the worker host are scheduled between d2d gates, not overlapped.
* The plan is [plan/plan-20261103.md](../plan/plan-20261103.md).
