---
id: dashboard-pub
goal: rule
status: in-force
enforced_by:
  - cmd/d2dview/pub.go
---
# dashboard-pub contract

## Purpose
What the d2d viewer (cmd/d2dview) publishes for the public stream page: the /api/pub backend and engine fields, the status row, and how the stream image recovers after an error, a stall or a hidden tab.

> status: reconciled 2026-11-04 against docs/process/spec-ledger-20261104.md; pending deltas: none

## Requirements

### Requirement: Pub backend field follows the badge
The /api/pub `backend` field SHALL equal the header badge text: the `-badge` flag or the `D2DVIEW_BADGE` environment value, limited to 40 characters by safeText; an empty value falls back to the default `d2d (reimplementation)`, which is also the default when neither is set. Source: card streamfix; cmd/d2dview/pub.go and main.go; test TestPubIsSanitized.

#### Scenario: Badge set
- **WHEN** the viewer runs with badge `d.idle testing stream`
- **THEN** /api/pub reports `backend` as `d.idle testing stream`

#### Scenario: Default unchanged
- **WHEN** neither the flag nor the environment variable is set
- **THEN** /api/pub reports `backend` as `d2d (reimplementation)`

### Requirement: Pub engine field and status row
/api/pub SHALL include `engine` with the constant `d2d`, and the stream page MUST show it in the status row as `engine d2d` (the value stripped to word characters, space, dot and hyphen). Source: card streamfix; cmd/d2dview/pub.go and web/stream.html; tests TestPubIsSanitized and TestStreamPageIsStreamSafe.

#### Scenario: Engine in the feed
- **WHEN** /api/pub is read
- **THEN** `engine` is `d2d`

#### Scenario: Status row
- **WHEN** the stream page has loaded /api/pub
- **THEN** the element `engine` reads `engine d2d`

### Requirement: Stream image reconnects
The stream page SHALL start a fresh `/stream.mjpg` request, swapped in on its first frame, when the image errors (after 1500 ms), when a request shows no frame within 5000 ms, every 5000 ms while the shown image is complete with zero width, and on visibilitychange, focus or pageshow while not hidden. Timings are UNVERIFIED in browsers; only a live run is verified (pane non-black for 10 minutes). Source: card streamfix; web/stream.html.

#### Scenario: Image error
- **WHEN** the image request fails (for example a 503 before the first frame)
- **THEN** a new request starts 1500 ms later

#### Scenario: Stuck image
- **WHEN** a new request has delivered no frame after 5000 ms
- **THEN** it is dropped and another request starts

#### Scenario: Tab returns
- **WHEN** a hidden tab becomes visible, gets focus or is restored from the page cache
- **THEN** a new request starts at once

### Requirement: The stream map is contiguous across levels the game draws together
The map pane SHALL draw one map in the act frame: every visited level on the player's Levels.txt layer, placed by its act origin. A level with a layer of its own (cave, crypt, dungeon) SHALL be drawn alone. The player, monsters, exits, `MapRect` and the image share that frame, so a stitched crossing moves nothing on screen. Source: cmd/d2dview/stitch.go; docs/stream/map-contiguity.md; tests TestStitch*. Reconciled 2026-11-04 (source: 2026-11-04-mapitem).

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
The image SHALL be rebuilt (`mapVer` raised) only when a member grid grows or a level joins, not on a crossing between known levels. The page SHALL keep the old image until the new one has loaded, pairing each image with its `X-Map-Rect` header, and SHALL restart the trail only when `stitch` changes or the position jumps over 30 tiles. Source: web/stream.html; test TestStitchTransitionBloodMoorColdPlains. Reconciled 2026-11-04 (source: 2026-11-04-mapitem).

#### Scenario: Back and forth over a border
- **WHEN** the player crosses a border between two known stitched levels several times
- **THEN** `mapVer` and `mapRect` do not change

## Pending deltas (not in force)
None. No open card carries a dashboard-pub delta (a stream-grid / stream-scene delta targets the separate capability `stream-scene`).
