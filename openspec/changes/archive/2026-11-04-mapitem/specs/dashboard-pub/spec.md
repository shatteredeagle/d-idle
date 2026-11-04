# dashboard-pub Specification

## ADDED Requirements

### Requirement: The stream map is contiguous across levels the game draws together
The map pane SHALL draw one map in the act frame: every visited level on the player's Levels.txt layer, placed by its act origin. A level with a layer of its own (cave, crypt, dungeon) SHALL be drawn alone. The player, monsters, exits, `MapRect` and the image share that frame, so a stitched crossing moves nothing on screen. Source: cmd/d2dview/stitch.go; docs/stream/map-contiguity.md; tests TestStitch*.

#### Scenario: Blood Moor to Cold Plains
- **WHEN** the player steps from Blood Moor into Cold Plains
- **THEN** both maps are in the image, the last Blood Moor column and the first Cold Plains column are neighbours, and the player position changes only by the step

#### Scenario: Field to cave
- **WHEN** the player enters the Den of Evil
- **THEN** the map image holds only the cave, and Blood Moor is drawn again on the way out

#### Scenario: Town and field
- **WHEN** the player is in the Rogue Encampment after visiting Blood Moor
- **THEN** the image holds both, joined at the town gate

### Requirement: A crossing does not blank or rebuild the map
The image SHALL be rebuilt (`mapVer` raised) only when a member grid grows or a level joins, not on a crossing between known levels. The page SHALL keep the old image until the new one has loaded, pairing each image with its `X-Map-Rect` header, and SHALL restart the trail only when `stitch` changes or the position jumps over 30 tiles. Source: web/stream.html; test TestStitchTransitionBloodMoorColdPlains.

#### Scenario: Back and forth over a border
- **WHEN** the player crosses a border between two known stitched levels several times
- **THEN** `mapVer` and `mapRect` do not change
