## ADDED Requirements

### Requirement: Early baseline fields
The early baseline (`baselines-early.json`) SHALL carry, for at least 2 original-game fresh-character runs (clvl 1 to 7), the four gated metrics xp_per_hour, kills_per_hour, deaths_per_hour and deaths_per_1000_kills with a source line each, so a fresh d2d run can be judged on every gated metric; the verdict against it stays marked weak until it holds 5 runs. Source: card m2-earlybase; docs/research/per-kill-time-split.md (the inputs); fidelity-gate `Same character, same setup`.

#### Scenario: Fresh run judged on kills
- **WHEN** a fresh clvl 1 d2d set of 8 seeds is compared with the early baseline
- **THEN** kills_per_hour and deaths_per_1000_kills get a PASS, FAIL or INCONCLUSIVE like xp_per_hour, and the report says the baseline is weak (fewer than 5 runs)

#### Scenario: Early baseline values and the first d2d set against them
- **WHEN** `baselines-early.json` is read after card m2-earlybase
- **THEN** it holds Fresha (V: 11743 XP/h, 389 kills/h, 0.52 deaths/h, 1.34 deaths per 1000 kills, 115 play minutes, 746 kills) and Soaky (S: 6600 XP/h, 408 kills/h, 0.29 deaths/h, 0.70 per 1000 kills) with mean 9172 XP/h and 398 kills/h
- **AND** the 8-seed x 3-virtual-hour d2d set of 2026-10-31 on the current bot (V, docs/d2d-baselines-early.md) measures 4767 +- 230 XP/h and 192 +- 7.5 kills/h (mean +- SE), 3 deaths in 24 h, no excluded seed, and `d2dsoak compare` gives FAIL on xp_per_hour (-48 %) and kills_per_hour (-52 %)
