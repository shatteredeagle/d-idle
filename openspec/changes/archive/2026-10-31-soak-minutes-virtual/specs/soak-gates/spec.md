## ADDED Requirements

### Requirement: Gate 1 minutes are virtual minutes
Every minute of the Gate 1 criteria (and of Gates 2 and 3 over their length) SHALL be a virtual (game) minute, not a wall minute, because the lockstep d2d clock runs about 10 times faster. One `minute` log line is one virtual minute; `d2dsoak` MUST print the longest belt window and no kill and no XP stretch with their limits (5 and 10). Source: user decision 2026-11-01; docs/probes/postfix-soak2.md.

#### Scenario: Belt window short in wall time, long in virtual time
- **WHEN** no healing potion is in the pack or any belt box outside town for 27 consecutive minute lines that took 87 s of wall time
- **THEN** the belt criterion FAILS (27 virtual minutes is over 5) and `d2dsoak` reports the violation code BELT

#### Scenario: Belt window at the limit
- **WHEN** the empty window outside town is exactly 5 consecutive minute lines
- **THEN** the belt criterion passes

#### Scenario: No kill and no XP in virtual time
- **WHEN** 26 consecutive minute lines show neither a new kill nor new XP, although they took 2.5 wall minutes
- **THEN** the no-progress criterion FAILS (26 virtual minutes is over 10) and `d2dsoak` reports NOKILL

#### Scenario: Silent stretch at the limit
- **WHEN** exactly 10 consecutive minute lines show neither a new kill nor new XP
- **THEN** the no-progress criterion passes

#### Scenario: Wall limits unchanged
- **WHEN** the injected d2d kill is restarted
- **THEN** the 3 minute restart limit is read in wall minutes, as are the start guards and the m24 monitors

#### Scenario: Wider invariant unchanged
- **WHEN** the same log has area changes, quest steps or gold spent inside that 26 minute stretch so the longest stretch without any progress event is 16 virtual minutes
- **THEN** the PROGRESS invariant (limit 30) still passes and the Gate 1 NOKILL criterion still FAILS

## MODIFIED Requirements

### Requirement: Gate 1 deaths and progression
A Gate 1 run MUST follow the death rule (death-rule spec, `d2dsoak report`) and SHALL earn true XP per hour at or above the d2d baseline at the same clvl (Early baseline fields of the fidelity-gate spec, numbers not copied here). No single state may last more than 10 virtual minutes without a kill or XP gain. Source: G1 criteria 2 and 3; ADR 0009 item 2; user decision 2026-11-01.

#### Scenario: XP below the d2d baseline
- **WHEN** true XP/h is under the d2d baseline at the same clvl
- **THEN** the run is a FAIL even with no crash

#### Scenario: Long silent stretch
- **WHEN** a Gate 1 log has 11 or more consecutive minute lines with no new kill and no new XP
- **THEN** the run is a FAIL on the progression criterion

### Requirement: Gate 1 belt and portal
A Gate 1 run SHALL have no window longer than 5 virtual minutes with zero healing potions outside town, measured from the minute-line fields packHP and beltBoxes (pack plus belt; one minute line is one virtual minute), and at least one portal trip completed by town portal, not by the waypoint fallback. Source: translated from G1 criterion 4 (the old "bought and used" is now measurable); docs/beltfill.md; user decision 2026-11-01.

#### Scenario: Empty field window
- **WHEN** packHP is 0 and no belt box holds a healing potion outside town for 6 consecutive minute lines
- **THEN** the run is a FAIL

#### Scenario: Waypoint only
- **WHEN** every town return in the run used the waypoint fallback
- **THEN** the run is a FAIL on the portal criterion
