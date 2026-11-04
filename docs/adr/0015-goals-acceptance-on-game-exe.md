---
status: accepted, amended 2026-11-04
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

## Amendment 2026-11-04 (user, dialog): d2d is the end product, game.exe is the oracle

The original text above stays as written. The user decided that the end product is d2d (native, no Wine layer). game.exe 1.14d is the oracle for fidelity. d2d still deviates from game.exe, so the bot must not be shaped by d2d-only quirks.

### What changes

* Decision Outcome item 1, "done means", now reads: the project is done when a fresh, uncheated character plays 24 h unattended on d2d and clears Act 1 (Andariel dead, Act 2 entered), `writes after start: 0`, AND the A:B comparison of G-D says d2d and game.exe are EQUIVALENT on XP/h and kills/h at the 20 percent tolerance.
* G-A is re-stated as "fidelity-validated acceptance": the 24 h unattended Act 1 clear runs on d2d; the game.exe lanes of G-D are the reference measurement, not the product. Gate A is renamed "Gate A (fidelity acceptance)": the 24 h d2d run plus the G-D verdict. The game.exe 2 h, 8 h, 24 h rungs of item 3 are no longer required for acceptance.
* G-D is unchanged and is now part of acceptance.
* The d2d container (item 5) stays deferred until the amended Gate A passes (native d2d, no Wine), not until a game.exe Gate A.

### What stays

Goals G-B to G-F, the cap of 12 open cards, the WIP limit, XP/h never a per-run pass or fail (seed-set judgement only), and the parked list of item 10.

### Rules

a. game.exe is the spec and d2d the implementation under test. A d2d deviation is a bug in d2d, fixed upstream (G-E), not something to design the bot around.
b. Any bot-side adaptation to a d2d deviation is a temporary SHIM recorded in `docs/d2d-deviations.md` with: the game.exe rule (V), the d2d behaviour, the code location, the upstream PR that removes it, and the removal condition.
c. One bot, one backend contract, the same drills and gates on both backends. Lab-fork debug verbs appear only in drills.

The fidelity loop that drives d2d improvements: decomp trace, emulator differential, independent verifier, upstream PR.

### Leaving d2d

The "leaving d2d is break glass" rule (ADR 0009 exit trigger) is replaced: if the G-D verdict is DIFFERENT and the gap needs a rewrite of the engine, the user decides whether to keep developing on d2d or move the work to game.exe. This is never automatic. The attribution rule stays: a replicated-soak FAIL counts against d2d only when the same bot build's game.exe baseline is inside the gate and the gap is traced to an engine difference.
