## ADDED Requirements

### Requirement: A failed walk does not condemn a route for good
A nav edge or gate marked bad after a failed walk SHALL expire after `nav.badEdgeTTLSec` seconds of game time (default 300, 0 restores the old permanent mark) and MUST be forgotten when the game is re-entered; an edge the level map itself shows MUST NOT be marked by a failed walk. Source: docs/townwedge.md; internal/nav/townwedge_test.go.

#### Scenario: Town route after a failed roam
- **WHEN** the link town to Blood Moor was marked bad and more than `nav.badEdgeTTLSec` seconds of game time pass
- **THEN** Route from the town to Cold Plains returns Blood Moor, Cold Plains again

#### Scenario: Save-and-exit re-entry
- **WHEN** a link is marked bad and the game is re-entered (the level maps are replaced)
- **THEN** the link and any bad gate of it are no longer bad

#### Scenario: Character cut off from a visible gate
- **WHEN** the character stands in a pocket of the known grid and the map shows the gate to the Blood Moor
- **THEN** it is walked to the nearest connected cell and leaves, and the link is not marked bad (and if it cannot get out, ToArea fails with an error that is not ErrNoExit)
