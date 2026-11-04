## ADDED Requirements

### Requirement: Late baseline
The late baseline (`baselines-late.json`) SHALL hold at least 5 runs of a throwaway character cheat-levelled to clvl 12 and carrying the M1 kit on hidden d2d, with every cheat recorded in the cheat ledger, and SHALL gate only xp_per_hour and kills_per_hour; deaths and gold MUST NOT be judged against it. Source: user decision D5, 2026-10-31 (docs/roadmap.md); d2-sme 2026-10-31 (EXP: fair for XP and kills per hour, not for deaths or gold); docs/cheats.md.

#### Scenario: Clvl 12 run judged against the late baseline
- **WHEN** a clvl 12 d2d set is compared with the late baseline
- **THEN** xp_per_hour and kills_per_hour get a verdict and the report marks deaths and gold as not compared

#### Scenario: Unledgered cheat
- **WHEN** a baseline run used a cheat with no ledger entry
- **THEN** the run is not accepted into baselines-late.json
