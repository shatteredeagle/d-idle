# Fidelity gate specification

## Purpose
How far the d2d backend may differ from the original game before its statistics stop being trusted: the replicated soak compares the same character and bot on both backends. Source: plan-v2 section 11 and 11.5, docs/d2d-soak.md (harness and pass rule), docs/d2d-fidelity.md, docs/d2d-gaps.md.

## Requirements

### Requirement: Twenty percent tolerance
The d2d mean of each gated metric (XP per hour, kills per hour, deaths per hour, deaths per 1000 kills) SHALL be within `max(20 % of the baseline mean, abs)` of the original-game baseline, with `abs` = 0.5 deaths/h and 1.0 deaths per 1000 kills where the baseline is near zero. A gap above 20 percent that is not explained by a known gap is a fidelity bug. Source: plan-v2 section 11; docs/d2d-soak.md section 3 (V, tool tested on real logs).

#### Scenario: XP gap
- **WHEN** the baseline mean is 10.9k XP/h and the d2d mean is 13.5k XP/h
- **THEN** the gap is +24 percent and the metric is out of tolerance

### Requirement: Verdict classes
The verdict per metric MUST be PASS when `|gap| <= tol` and the 95 percent half width of the d2d mean is `<= tol`, FAIL when `|gap| - half width > tol`, and INCONCLUSIVE otherwise or with fewer than 5 seeds; the overall verdict is FAIL if any gated metric fails. Source: docs/d2d-soak.md section 3 (V).

#### Scenario: Few seeds
- **WHEN** only 3 seeds ran
- **THEN** the verdict is INCONCLUSIVE and the tool says how many seeds would settle it

### Requirement: Minimum sample
A fidelity verdict MUST use at least 5 seeds and at least 2 virtual hours per seed (8 seeds by 2 h = 16 h recommended) and at least 10 virtual hours in total for the death metrics. Source: docs/d2d-soak.md section 3 (I, derived from the baseline spread of 10 to 15 percent).

#### Scenario: Short death sample
- **WHEN** a set has 6 virtual hours in total
- **THEN** the death metrics are reported but the tool warns the sample is too small

### Requirement: Same character, same setup
The d2d run MUST start the same character class, level range and kit as the baseline runs (Sorceress clvl 11 to 14 honest-play kit, profile m1) or use the early baseline (S quality); a fresh clvl 1 run is not comparable with the clvl 11 to 14 baseline. Source: docs/d2d-soak.md section 3.

#### Scenario: Fresh start
- **WHEN** a d2d run starts at clvl 1
- **THEN** it is compared with `baselines-early.json` only and marked weak

### Requirement: Known gaps are not bugs
A gap that matches a listed d2d deviation (Tome of Town Portal 250 gold vs 450, sell price ratio, cast period 14 ticks for every spell, missing potion double-fill, Den and Blood Moor XP, gold autoloot) SHALL be reported as explained and not counted as a failure. Source: docs/d2d-soak.md section 4, docs/d2d-fidelity.md, docs/d2d-gaps.md.

#### Scenario: Price gap
- **WHEN** gold per hour differs because the Tome costs 250 on d2d
- **THEN** the report lists it as an explained known gap

### Requirement: What d2d cannot validate
Wine, input injection, process memory reads, Game.exe crash and hang paths, the failsafe ladder, recycle paths, window focus, the 20 minute save point and OOM guards MUST stay original-game soak topics (G1, G2). Source: docs/d2d-soak.md section 4.

#### Scenario: Recovery drill
- **WHEN** a kill-recovery criterion needs verification
- **THEN** it runs on the original game, not in a d2d fidelity run

### Requirement: Fast d2d runs use the lab fork
Replicated soaks SHALL use the lockstep private fork (about 100 to 380 times real time, deterministic for equal seed and inputs) with autoloot and autoparty off, `--no-save`, nice 10, and refuse above load 45 or under 15 GB free. Source: docs/d2d-soak.md section 1 and docs/d2d-fork.md (V on the fork).

#### Scenario: Overloaded host
- **WHEN** load is 50 on the worker host
- **THEN** `d2dsoak run` refuses to start a seed
