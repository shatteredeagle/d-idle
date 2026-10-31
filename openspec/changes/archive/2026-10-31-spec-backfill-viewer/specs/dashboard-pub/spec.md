# Dashboard public data specification

## Purpose
What the d2d viewer (cmd/d2dview) publishes for the public stream page: the /api/pub backend and engine fields, the status row, and how the stream image recovers after an error, a stall or a hidden tab.

## ADDED Requirements

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
