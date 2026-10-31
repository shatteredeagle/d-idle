## ADDED Requirements

### Requirement: The Den floor does not yield far below the floor
In Normal, when the level planner finds no step and the character is below its Den of Evil floor (knobs campaign.denMinLevelCaster 5, campaign.denMinLevelMelee 3), the floor SHALL NOT be waived for a character more than one level below it (denYieldGap 1): a clvl 1 caster whose Blood Moor is cleared waits instead of entering the Den. Source: card d2dlevel; internal/bot/goal.go (denYieldGap); test TestDenFloorDoesNotYieldFarBelowFloor; docs/d2d-pace.md (a clvl 1 caster drank 4 potions in 12 s).

#### Scenario: Clvl 1 Sorceress, Blood Moor cleared
- **WHEN** a clvl 1 Sorceress with floor 5 has only Blood Moor in reach and it is marked cleared
- **THEN** the planner does not pick the Den of Evil

### Requirement: The Den floor yields only one level below with a kit
The Den floor SHALL be waived, with the log line `den: floor waived, nothing else to level in`, only when no other step fits, the character is at most one level below its floor, the Den is neither cleared nor banned, and the belt holds at least 2 healing potions (or its contents are unknown). Source: card d2dlevel; internal/bot/goal.go (the waive branch after pick). UNVERIFIED by test: only the far-below case is tested.

#### Scenario: One level below with potions
- **WHEN** a clvl 4 caster (floor 5) has Blood Moor cleared, nothing else fits and 2 healing potions are in the belt
- **THEN** the Den is allowed and the waive line is logged (UNVERIFIED, no test)

#### Scenario: Thin belt
- **WHEN** the same character holds fewer than 2 healing potions
- **THEN** the Den stays excluded (UNVERIFIED, no test)
