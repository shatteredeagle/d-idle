## ADDED Requirements

### Requirement: A skipped move costs one poll tick
`Fighter.move` SHALL spend one poll tick when it skips a point inside a cave mouth, and away-point pickers (flee, kite, dodge, crowd step-away) SHALL NOT choose such a point, so no combat iteration returns without time passing or progress being made.

#### Scenario: Ten skipped moves
- **WHEN** ten moves to a point inside a cave mouth are skipped
- **THEN** the game clock has advanced by at least ten poll ticks and nothing was walked (TestSkippedMoveAdvancesTime)

#### Scenario: Flee with every point in a mouth
- **WHEN** a fight runs with life low, no portal, no potion and a Mover whose NearMouth is true everywhere
- **THEN** no point inside a mouth is offered to move and the no-progress watchdog does not fire (TestFleeIntoCaveMouthDoesNotSpin)
