## ADDED Requirements

### Requirement: Early baseline fields
The early baseline (`baselines-early.json`) SHALL carry, for at least 2 original-game fresh-character runs (clvl 1 to 7), the four gated metrics xp_per_hour, kills_per_hour, deaths_per_hour and deaths_per_1000_kills with a source line each, so a fresh d2d run can be judged on every gated metric; the verdict against it stays marked weak until it holds 5 runs. Source: card m2-earlybase; docs/research/per-kill-time-split.md (the inputs); fidelity-gate `Same character, same setup`.

#### Scenario: Fresh run judged on kills
- **WHEN** a fresh clvl 1 d2d set of 8 seeds is compared with the early baseline
- **THEN** kills_per_hour and deaths_per_1000_kills get a PASS, FAIL or INCONCLUSIVE like xp_per_hour, and the report says the baseline is weak (fewer than 5 runs)
