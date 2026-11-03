# Soak-gates specification

## MODIFIED Requirements

### Requirement: Gate 1 deaths and progression
A Gate 1 run MUST follow the death rule (death-rule spec, `d2dsoak report`) and SHALL earn true XP per hour at or above the d2d baseline at the same clvl (baseline file named in the fidelity-gate spec), or, for a single seed, below it but within the tolerance while d2dsoak says INCONCLUSIVE (pass with the note "one seed, inconclusive"). No single state may last more than 10 minutes without progress. Source: G1 criteria 2 and 3; ADR 0009 item 2; user decision 2026-11-03.

#### Scenario: XP below the d2d baseline
- **WHEN** true XP/h is under the d2d baseline by more than the tolerance
- **THEN** the run is a FAIL even with no crash

#### Scenario: One seed inside the tolerance
- **WHEN** true XP/h is under the d2d baseline, by no more than the tolerance, and d2dsoak says INCONCLUSIVE
- **THEN** the XP criterion PASSES with the note "one seed, inconclusive"; only the 5-seed d2dsoak verdict supports a claim of "at or above the baseline"

#### Scenario: At or above the baseline
- **WHEN** true XP/h is at or above the d2d baseline
- **THEN** the XP criterion PASSES

### Requirement: Gate 1 belt and portal
A Gate 1 run SHALL have no funded window longer than 5 minutes with zero healing potions outside town (fields packHP, packMP, beltBoxes), and at least one town-portal trip, not the waypoint fallback. A window is funded when pack plus stash gold (`gold` + `stashGold`) reaches one Minor Healing Potion (30 gold) at some minute; others are reported poverty, not failing. No `stashGold` (old archive) means stash 0. Source: G1 criterion 4; docs/beltfill.md; user decision 2026-11-03.

#### Scenario: Empty field window
- **WHEN** no healing potion is held outside town for over 5 minutes and pack plus stash gold reaches 30 at some minute of the window (for example pack 6, stash 1418)
- **THEN** the run is a FAIL

#### Scenario: Poverty window
- **WHEN** no healing potion is held outside town for over 5 minutes and pack plus stash gold stays under 30 for the whole window
- **THEN** the window is reported as poverty and does not fail the run

#### Scenario: Gold crosses the price mid-window
- **WHEN** pack plus stash gold is under 30 at first and reaches 30 later in the same window
- **THEN** the window is funded and the run is a FAIL

#### Scenario: Old archive without stashGold
- **WHEN** the minute lines carry no `stashGold`
- **THEN** the stash counts as 0: a window with pack gold of 30 or more fails, one with less is poverty, and the report tag stays "stash unknown (old archive)"

#### Scenario: Waypoint only
- **WHEN** every town return in the run used the waypoint fallback
- **THEN** the run is a FAIL on the portal criterion
