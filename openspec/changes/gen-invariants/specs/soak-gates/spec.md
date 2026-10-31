## ADDED Requirements

### Requirement: Per-seed invariants
`d2dsoak` SHALL report for every seed counters and a verdict for HANG (a pace interval without virtual advance, or `combat no progress, holding`), STALL (any `campaign stalled` rung), WEDGE (one `no route from level` or `no eligible reachable leveling area` storm longer than 15 virtual min with no non-town Entered), PROGRESS (no kill, xp, area, quest or spend for more than 30 min), DEATHS, RESTARTS and EXIT. Source: card gen-invariants; docs/geninv.md.

#### Scenario: Town wedge
- **WHEN** a log has 60 virtual minutes of `no eligible reachable leveling area` lines and no non-town Entered
- **THEN** the seed is VIOLATED with WEDGE and PROGRESS even though the process exited 0 at the time limit

#### Scenario: Frozen clock
- **WHEN** two pace lines are wall time apart and show the same virtual time
- **THEN** the seed is VIOLATED with HANG

### Requirement: Death and exit invariants
A seed SHALL violate DEATHS on more than 15 deaths per 1000 kills from 150 kills, on 3 deaths in 30 min without a clvl gain or 5 in 60 min, or on a death with no progress within 10 virtual min; RESTARTS on any bot restart; EXIT on an end under 95 percent of the requested hours, a non-zero exit or a recovered panic. Source: death-rule FAIL-1 and FAIL-3 (D); soakfix early end 95 percent (D); recovery window (J).

#### Scenario: Early end
- **WHEN** 3 virtual hours were requested and the command exited after 1
- **THEN** the seed is VIOLATED with EXIT; a log with no end marker and no requested length is `open`, not violated

### Requirement: Invariants verdict
The report SHALL end its invariants block with `INVARIANTS: PASS`, `VIOLATED(list)` or `NO-DATA`, separate from the fidelity verdict; the set is VIOLATED when any seed is, PASS only when every seed is PASS. The invariants verdict MUST NOT change the exit code of `run` or `compare`. Source: card gen-invariants; fidelity-gate Verdict exit codes.

#### Scenario: Violated but fidelity PASS
- **WHEN** the fidelity verdict is PASS and one seed has a WEDGE
- **THEN** the exit code is 0, the text and `-json` show `INVARIANTS: VIOLATED(seed N: WEDGE)` and OVERALL stays the last line

### Requirement: Holdout seeds
Seeds 16 to 23 SHALL be holdout seeds, never used for tuning, fitting or diagnosis; the report groups seeds as `tuning` and `holdout` (`-holdout` overrides, empty disables), gives each group its counters and invariants verdict, and prints a COVERAGE line naming seeds, class, act, difficulty, engine and clvl range. Source: docs/research/bot-generality.md sections 3 and 5 (experiment 1); docs/geninv.md.

#### Scenario: Gate run with holdout
- **WHEN** seeds 3 and 16 to 17 are compared
- **THEN** group tuning lists seed 3, group holdout lists seeds 16 to 17, each with its own verdict, and the COVERAGE line lists both seed sets

#### Scenario: No holdout ran
- **WHEN** only seeds 1 to 8 ran
- **THEN** the report says `group holdout: none ran` and the COVERAGE line shows holdout none
