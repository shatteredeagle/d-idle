## MODIFIED Requirements

### Requirement: One d2d at a time
The d2d lab lock SHALL be a pool of 4 slots (slot 0 is the legacy `/tmp/d2d-lab.lock`); a run holds exactly one slot, only while a d2d runs, taken through `tools/d2dslot.sh run`; legacy single-lock holders occupy slot 0. Hidden d2d only; original-game rules are unchanged and the `pgrep -x d2d` refusal no longer applies. Source: user decision 2026-10-30; V live (tools/d2dslot-test.sh).

#### Scenario: Lock held
- **WHEN** another agent holds `/tmp/d2d-lab.lock` (slot 0)
- **THEN** the run takes the next free slot, or exits 75 when none is free within its wait

#### Scenario: Four runs
- **WHEN** four agents call `tools/d2dslot.sh run` at once
- **THEN** each gets a different slot, its own socket under `~/d2d-lab/slots/<n>/` and its own ports

#### Scenario: All slots held
- **WHEN** a fifth run asks for a slot and none frees within its wait time
- **THEN** it exits 75 and starts no d2d

#### Scenario: Legacy holder
- **WHEN** a script still runs `flock /tmp/d2d-lab.lock`
- **THEN** it occupies slot 0 and the pool hands out only slots 1 to 3
