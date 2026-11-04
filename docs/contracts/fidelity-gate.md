# Fidelity gate specification

## Purpose
How far the d2d backend may differ from the original game before its statistics stop being trusted: the replicated soak compares the same character and bot on both backends. Source: plan-v2 section 11 and 11.5, docs/d2d-soak.md (harness and pass rule), docs/d2d-fidelity.md, docs/d2d-gaps.md.

> status: reconciled 2026-11-04 against docs/process/spec-ledger-20261104.md; pending deltas: none
>
> Death judging (user decision 2026-11-04): soaks judge deaths by the death-rule spec only; this spec's deaths metrics apply to the A:B backend comparison at the 20 percent tolerance. The 2026-10-30 gatedelta rules (fixed death budget, Poisson cap, three proxy rows) were dropped, not folded.

## Requirements

### Requirement: Twenty percent tolerance
The d2d mean of each gated metric (XP per hour, kills per hour, deaths per hour, deaths per 1000 kills) SHALL be within `max(20 % of the baseline mean, abs)` of the original-game baseline, with `abs` = 0.5 deaths/h and 1.0 deaths per 1000 kills near a zero baseline. An A:B report MAY state a tighter margin m but MUST name it. An unexplained gap above the tolerance is a fidelity bug. Source: plan-v2 s11; docs/d2d-soak.md s3 (V). Reconciled 2026-11-04 (source: 2026-11-02-abtool; decision 1).

#### Scenario: XP gap
- **WHEN** the baseline mean is 10.9k XP/h and the d2d mean is 13.5k XP/h
- **THEN** the gap is +24 percent and the metric is out of tolerance

#### Scenario: Named margin
- **WHEN** an A:B report uses 15 percent
- **THEN** the report states the margin next to every verdict

### Requirement: Verdict classes
Per metric: PASS when `|gap| <= tol` and the d2d 95 percent half width is `<= tol`, FAIL when `|gap| - half width > tol`, else INCONCLUSIVE (also under 5 seeds); overall FAIL if any gated metric fails. A cross-backend A:B (ab_compare.py) SHALL use the 95 percent ratio interval with both arms' error: PASS inside `[1 - m, 1 + m]`, FAIL wholly beyond, else INCONCLUSIVE with n per arm. Source: docs/d2d-soak.md s3 (V); tierb-design 3.3. Reconciled 2026-11-04 (source: 2026-11-02-abtool).

#### Scenario: Few seeds
- **WHEN** only 3 seeds ran
- **THEN** the verdict is INCONCLUSIVE and the tool says how many seeds would settle it

#### Scenario: Both arms' error counted
- **WHEN** the d2d mean is 10 percent above the game.exe mean and the ratio interval is 0.80 to 1.40
- **THEN** the verdict is INCONCLUSIVE, although the d2d half width alone would pass

### Requirement: Game.exe arm sample and regimes
A cross-backend A:B SHALL use at least 5 game.exe runs per arm, SHALL tag every run with its clock regime k and SHALL never pool or compare runs of different k; a lane marked weak is reported and kept out of the verdict. Source: docs/research/tierb-design-20261102.md 3.1, 4.1, 6; baseline-audit.md (5 runs, about 20 percent half width at CV 14 percent). Reconciled 2026-11-04 (source: 2026-11-02-abtool).

#### Scenario: Mixed k
- **WHEN** a set holds k=1 and k=2 runs
- **THEN** the tool prints one table per k and no combined row

### Requirement: Pair setup on both backends
A pair in a cross-backend A:B SHALL start from one `.d2s` (or the same pinned map seed) per seed on both backends with the arms interleaved; a setup-edited file MUST be declared as SETUP. Cross-backend sizing SHALL use the unpaired formula unless the measured pair correlation exceeds 0.5. Source: docs/research/tierb-design-20261102.md 3.1, 3.4. Reconciled 2026-11-04 (source: 2026-11-02-abtool).

#### Scenario: Low pair correlation
- **WHEN** the pilot measures a pair correlation of 0.2
- **THEN** n per arm is taken from the unpaired formula

### Requirement: Replicated soak is a fidelity check
The replicated soak SHALL NOT gate making d2d the primary backend; it MUST be used to find d2d gaps to fix and to fire the exit trigger, and only the four gated metrics (XP/h, kills/h, deaths/h, deaths per 1000 kills) are held to the 20 percent tolerance. Source: ADR 0009 items 2 and 3. Reconciled 2026-11-04 (source: 2026-11-04-d2d-replicated-soak).

#### Scenario: Price gap outside the four metrics
- **WHEN** a vendor price differs on d2d but the four metrics are in tolerance
- **THEN** it is an explained known gap, fixed on the fork when cheap

> OPEN QUESTION: "fire the exit trigger" is ADR 0009 wording; user decision 2026-11-04 ties leaving d2d to the A:B (G-D) verdict, which the user decides, never automatic. Wording to be confirmed with the user; text left as in the delta.

### Requirement: Soak FAIL attribution
A replicated-soak FAIL SHALL count against d2d only when the same bot build's original-game baseline is inside the gate and the gap is traced to an engine difference; otherwise it MUST be treated as a bot bug and fixed in the bot. Source: ADR 0009 attribution rule. Reconciled 2026-11-04 (source: 2026-11-04-d2d-replicated-soak).

#### Scenario: Baseline outside the gate
- **WHEN** the original-game baseline for the bot build is itself outside the gate
- **THEN** the FAIL is a bot bug, not an exit-trigger input

### Requirement: Late baseline
The late baseline (`baselines-late.json`) SHALL hold at least 5 runs of a throwaway character cheat-levelled to clvl 12 and carrying the M1 kit on hidden d2d, with every cheat recorded in the cheat ledger, and SHALL gate only xp_per_hour and kills_per_hour; deaths and gold MUST NOT be judged against it. Source: user decision D5, 2026-10-31 (docs/roadmap.md); d2-sme 2026-10-31; docs/cheats.md. Reconciled 2026-11-04 (source: 2026-11-04-m2-latebase).

#### Scenario: Clvl 12 run judged against the late baseline
- **WHEN** a clvl 12 d2d set is compared with the late baseline
- **THEN** xp_per_hour and kills_per_hour get a verdict and the report marks deaths and gold as not compared

#### Scenario: Unledgered cheat
- **WHEN** a baseline run used a cheat with no ledger entry
- **THEN** the run is not accepted into baselines-late.json

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
Wine, input injection, process memory reads, Game.exe crash and hang paths, the failsafe ladder, recycle paths, window focus, the 20 minute save point and OOM guards MUST stay game.exe topics, covered by Gate A (the game.exe acceptance run, ADR 0015; the old G1/G2 names are retired). Source: docs/d2d-soak.md section 4. Reconciled 2026-11-04 (source: Current truth gate names).

#### Scenario: Recovery drill
- **WHEN** a kill-recovery criterion needs verification
- **THEN** it runs on the original game, not in a d2d fidelity run

### Requirement: Fast d2d runs use the lab fork
Replicated soaks SHALL use the lockstep private fork (about 100 to 380 times real time, deterministic for equal seed and inputs) with autoloot and autoparty off, `--no-save`, nice 10, and refuse above load 45 or under 15 GB free. Source: docs/d2d-soak.md section 1 and docs/d2d-fork.md (V on the fork).

#### Scenario: Overloaded host
- **WHEN** load is 50 on the worker host
- **THEN** `d2dsoak run` refuses to start a seed

### Requirement: Verdict exit codes
`d2dsoak run` and `d2dsoak compare` MUST exit 0 only when the overall verdict is PASS, 2 on INCONCLUSIVE and 3 on FAIL or NO-DATA, so a script cannot read a non-pass as a pass. Source: docs/research/code-review-2026-10-30.md H1; docs/contracts/soak-gates.md (NO-DATA is exit 3); card m2-soakexit.

#### Scenario: Inconclusive verdict
- **WHEN** 3 seeds ran and the verdict is INCONCLUSIVE
- **THEN** the process exits 2 and the last line names the verdict and the code

#### Scenario: No usable seed
- **WHEN** every seed is excluded (no minute lines)
- **THEN** the verdict is NO-DATA and the process exits 3, never 0 or 2

#### Scenario: Failed metric
- **WHEN** any gated metric is FAIL
- **THEN** the process exits 3

### Requirement: Excluded seeds
A seed with zero minute lines, or whose d2d process ended before the requested virtual hours, SHALL contribute zero exposure hours and zero kills, SHALL be listed under `excluded seeds` with the reason, and SHALL NOT count toward the minimum-seed rule. Source: docs/research/code-review-2026-10-30.md H1; card m2-soakexit.

#### Scenario: Crashed seed
- **WHEN** seed 11 of 8 hangs and produces 1.8 virtual hours of a 2 hour run
- **THEN** the report lists seed 11 as excluded, the death metrics use the other seeds' hours and the seed count is 7

### Requirement: A/B protocol for bot changes
A claim that a bot change moves XP per hour or kills per hour on d2d SHALL come from a paired same-seed before/after on one binary, at least 8 seeds per arm and 2 virtual hours per seed, and the effect is claimed only when the 95 percent interval of the paired mean difference excludes zero; anything less is reported as `no effect shown`. Source: docs/ab-design.md. Source: docs/d2d-xp-gap.md (xpgap 2026-10-31: SE 7 to 9 percent with 4 seeds per arm); card m2-abdesign.

#### Scenario: Four seeds per arm
- **WHEN** a card reports +8 percent XP/h from 4 seeds per arm
- **THEN** the report says `no effect shown` and names the seeds per arm the SD requires

#### Scenario: Paired result
- **WHEN** 8 paired seeds give a mean difference of +600 XP/h with a 95 percent interval of +150 to +1050
- **THEN** the change is reported as an improvement of about 12 percent

### Requirement: A/B sample size and interleaving
The arms of an A/B SHALL be interleaved per seed, and the seeds per arm for a planned relative effect e SHALL be at least ((1.96 + 0.84) * SD_diff / (e * mean_before))^2 from the measured SD of the per-seed differences, never below 8. Source: docs/ab-design.md; tools/ab/paired.py.

#### Scenario: Sample size from the measured SD
- **WHEN** a pilot gives an SD of the per-seed XP/h differences of 1052 on a mean of 5273 and the planned effect is 10 percent
- **THEN** the plan names about 34 seeds per arm, not 8

### Requirement: Early baseline fields
The early baseline (`baselines-early.json`) SHALL carry, for at least 2 original-game fresh-character runs (clvl 1 to 7), the four gated metrics xp_per_hour, kills_per_hour, deaths_per_hour and deaths_per_1000_kills with a source line each, so a fresh d2d run can be judged on every gated metric; the verdict against it stays marked weak until it holds 5 runs. Source: card m2-earlybase; docs/research/per-kill-time-split.md (the inputs); fidelity-gate `Same character, same setup`.

#### Scenario: Fresh run judged on kills
- **WHEN** a fresh clvl 1 d2d set of 8 seeds is compared with the early baseline
- **THEN** kills_per_hour and deaths_per_1000_kills get a PASS, FAIL or INCONCLUSIVE like xp_per_hour, and the report says the baseline is weak (fewer than 5 runs)

#### Scenario: Early baseline values and the first d2d set against them
- **WHEN** `baselines-early.json` is read after card m2-earlybase
- **THEN** it holds Fresha (V: 11743 XP/h, 389 kills/h, 0.52 deaths/h, 1.34 deaths per 1000 kills, 115 play minutes, 746 kills) and Soaky (S: 6600 XP/h, 408 kills/h, 0.29 deaths/h, 0.70 per 1000 kills) with mean 9172 XP/h and 398 kills/h
- **AND** the 8-seed x 3-virtual-hour d2d set of 2026-10-31 on the current bot (V, docs/d2d-baselines-early.md) measures 4767 +- 230 XP/h and 192 +- 7.5 kills/h (mean +- SE), 3 deaths in 24 h, no excluded seed, and `d2dsoak compare` gives FAIL on xp_per_hour (-48 %) and kills_per_hour (-52 %)
