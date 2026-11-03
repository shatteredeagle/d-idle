## MODIFIED Requirements

### Requirement: Verdict classes
Per metric: PASS when `|gap| <= tol` and the d2d 95 percent half width is `<= tol`, FAIL when `|gap| - half width > tol`, else INCONCLUSIVE (also under 5 seeds); overall FAIL if any gated metric fails. A cross-backend A:B (ab_compare.py) SHALL use the 95 percent ratio interval with both arms' error: PASS inside `[1 - m, 1 + m]`, FAIL wholly beyond, else INCONCLUSIVE with n per arm. Source: docs/d2d-soak.md section 3 (V); tierb-design 3.3.

#### Scenario: Few seeds
- **WHEN** only 3 seeds ran
- **THEN** the verdict is INCONCLUSIVE and the tool says how many seeds would settle it

#### Scenario: Both arms' error counted
- **WHEN** the d2d mean is 10 percent above the game.exe mean and the ratio interval is 0.80 to 1.40
- **THEN** the verdict is INCONCLUSIVE, although the d2d half width alone would pass

### Requirement: Twenty percent tolerance
The d2d mean of each gated metric (XP per hour, kills per hour, deaths per hour, deaths per 1000 kills) SHALL be within `max(20 % of the baseline mean, abs)` of the original-game baseline, with `abs` = 0.5 deaths/h and 1.0 deaths per 1000 kills where the baseline is near zero. An A:B report MAY state a tighter margin m but MUST name it. A gap above the tolerance that is not explained by a known gap is a fidelity bug. Source: plan-v2 section 11; docs/d2d-soak.md section 3 (V).

#### Scenario: XP gap
- **WHEN** the baseline mean is 10.9k XP/h and the d2d mean is 13.5k XP/h
- **THEN** the gap is +24 percent and the metric is out of tolerance

#### Scenario: Named margin
- **WHEN** an A:B report uses 15 percent
- **THEN** the report states the margin next to every verdict

## ADDED Requirements

### Requirement: Game.exe arm sample and regimes
A cross-backend A:B SHALL use at least 5 game.exe runs per arm, SHALL tag every run with its clock regime k and SHALL never pool or compare runs of different k; a lane marked weak is reported and kept out of the verdict. Source: docs/research/tierb-design-20261102.md 3.1, 4.1, 6; baseline-audit.md (5 runs, about 20 percent half width at CV 14 percent).

#### Scenario: Mixed k
- **WHEN** a set holds k=1 and k=2 runs
- **THEN** the tool prints one table per k and no combined row

### Requirement: Pair setup on both backends
A pair in a cross-backend A:B SHALL start from one `.d2s` (or the same pinned map seed) per seed on both backends with the arms interleaved; a setup-edited file MUST be declared as SETUP. Cross-backend sizing SHALL use the unpaired formula unless the measured pair correlation exceeds 0.5. Source: docs/research/tierb-design-20261102.md 3.1, 3.4.

#### Scenario: Low pair correlation
- **WHEN** the pilot measures a pair correlation of 0.2
- **THEN** n per arm is taken from the unpaired formula
