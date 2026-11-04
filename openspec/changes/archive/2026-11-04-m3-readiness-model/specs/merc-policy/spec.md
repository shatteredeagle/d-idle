## ADDED Requirements

### Requirement: Readiness model
Once the readiness model is enabled the route guard SHALL allow an area only when character power is at least zone danger times the safety margin, SHALL never allow an area more than 2 clvl below the merc-policy floors, and the model MUST be enabled only after its offline replay rates the 9 Monastery road deaths (clvl 11-12, 2-potion kit) as not ready. Source: ADR 0011 (grill 0003, 2026-10-31); card m3-readiness-model.

#### Scenario: Road death replay
- **WHEN** the model replays a clvl 11 character with a 2-potion kit entering the Monastery road against a Dark Stalker pack
- **THEN** the verdict is not ready

#### Scenario: Floor safety net
- **WHEN** the model rates a clvl 12 character ready for the Monastery road with no merc and the floor is 16
- **THEN** the guard still blocks the road because the model may go at most 2 clvl below the floor, so 14 is the lowest allowed level

#### Scenario: Model disabled
- **WHEN** the enable knob is 0
- **THEN** only the clvl floors decide
