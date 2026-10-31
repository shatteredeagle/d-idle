## ADDED Requirements

### Requirement: Holding card without gameplay
When no gameplay source is live, the scene SHALL show a holding card with the current work (at most 4 sanitized items), the next-gameplay ETA, the time since the last gameplay and the rotating info cards. Source: user request 2026-10-30; baseline tools/publish/scene.py (work chips, ladder, ticker).

#### Scenario: Nothing live
- **WHEN** scene.json lists no live source
- **THEN** the page shows the holding card with work items, ETA line, "last gameplay" line and one info card

#### Scenario: Nothing being worked on
- **WHEN** the sanitized work list is empty
- **THEN** the card shows a neutral line ("between tasks") and not an empty box

### Requirement: Recent merges in plain words
The holding card and the recent-merges info card SHALL name recently merged changes and current work in plain words (user allowed it), passed through the same sanitizer (no internal names, people, machines or branch names). Source: user decision 2026-10-30.

#### Scenario: Merged change
- **WHEN** a change card was merged today
- **THEN** the card shows its plain title (for example "d2d default backend") and no agent or branch name

### Requirement: Current work from sanitized sources
The work list SHALL be derived only from sanitized public sources (active change card titles, recent merges, agent task names mapped through the publisher's work titles) and MUST NOT contain agent names, branch names or internal ids. Source: tools/publish/sanitize.py, work-titles.json.

#### Scenario: Agent task with an internal name
- **WHEN** a board task is named after an internal agent
- **THEN** the card shows its mapped plain title, or a generic phrase when no mapping exists

### Requirement: Next gameplay ETA
The holding card SHALL show the next gameplay as a countdown computed from `status/next-run.json`, which the dispatcher rewrites on every tick from the soak and test queue (`next_start` ISO time, `source`, `label`) and MUST show "not scheduled" when the file is absent, unparsable or `next_start` is in the past; an ETA is never estimated or invented. Default refresh of the file by the page: 30 s (default, tunable).

#### Scenario: Scheduled run
- **WHEN** next-run.json has `next_start` 20 minutes ahead and label "d2d run"
- **THEN** the card shows "Next gameplay in 20 min: d2d run" and counts down

#### Scenario: Nothing queued
- **WHEN** the dispatcher ticks with an empty queue
- **THEN** it writes no `next_start` (or removes the file) and the card shows "not scheduled"

#### Scenario: No schedule
- **WHEN** next-run.json is missing
- **THEN** the card shows "Next gameplay: not scheduled"

#### Scenario: Stale schedule
- **WHEN** `next_start` is 5 minutes in the past and no source is live
- **THEN** the card shows "not scheduled", not a negative countdown

### Requirement: Time since last gameplay
The card SHALL show how long ago gameplay was last on screen, taken from the latest end time of any source (kept per source), formatted in minutes under 2 h and hours above, and "no gameplay yet today" when none is known. Source: user request 2026-10-30.

#### Scenario: Run ended recently
- **WHEN** the last source ended 12 minutes ago
- **THEN** the card shows "Last gameplay 12 min ago"

#### Scenario: Gameplay returns
- **WHEN** a source comes online
- **THEN** the holding card is replaced and the timer restarts from that source's end time later

### Requirement: Rotating info cards
The holding card SHALL rotate at least these info cards every 15 seconds (default, tunable) with a progress hint (dots or a bar): project goal with the milestone ladder; recent merges; fidelity numbers d2d versus the original game; what d2d is; replay and LAN play as "coming"; credit to upstream psi29a/d2decomped (GPL-3.0); how it works. Source: user request 2026-10-30, docs/decisions.md (GPL fork rule, LAN decision).

#### Scenario: Rotation
- **WHEN** 15 seconds pass on a card
- **THEN** the next card shows and the progress hint advances; after the last card it wraps

#### Scenario: Missing data for a card
- **WHEN** the fidelity numbers are not available in scene.json
- **THEN** that card is skipped for the cycle and no placeholder numbers are shown

### Requirement: Grid layout by source count
When at least one gameplay source is live, the scene SHALL replace the holding card and tile the sources: 1 full frame, 2 side by side, 3 to 4 as 2x2, 5 to 6 as 3x2, 7 to 9 as 3x3, with at most 9 tiles (more sources: the 9 most recently started). Source: baseline stream.sh `single|grid2|grid3|grid4` (docs/streamctl.md), extended.

#### Scenario: One source
- **WHEN** exactly one source is live
- **THEN** its tile fills the frame

#### Scenario: Two sources
- **WHEN** two sources are live
- **THEN** two equal tiles sit side by side

#### Scenario: Three or four sources
- **WHEN** three or four sources are live
- **THEN** a 2x2 grid shows, the fourth cell empty (info card) at three

#### Scenario: Five or six sources
- **WHEN** five or six sources are live
- **THEN** a 3x2 grid shows

#### Scenario: Seven to nine sources
- **WHEN** seven to nine sources are live
- **THEN** a 3x3 grid shows and map panels are hidden

### Requirement: Sources join and leave automatically
A live source SHALL appear as a tile within 10 seconds of coming online and be removed, with the grid reflowing, within 10 seconds of ending (default, tunable). Sources are discovered, not configured: d2lod instances i1 to i5 (d2watch tiles), d2d runs (d2dview) and test runs.

#### Scenario: Source starts
- **WHEN** a d2d run starts while one d2lod tile is shown
- **THEN** within 10 seconds a second tile appears and the layout becomes side by side

#### Scenario: Last source ends
- **WHEN** the only live source ends
- **THEN** within 10 seconds the holding card returns and the last-gameplay timer starts

### Requirement: Tile contents and backend badge
Each tile SHALL carry a backend badge ("d2lod (original game)" or "d2d"), the character name, class and level; a d2d tile SHALL show the gameplay frame and, where the tile is at least 640 px wide (default, tunable), the map beside it. Source: docs/d2d-view.md (/map.png, /stream.mjpg), agent streamview /api/pub fields (backend, name, class, clvl).

#### Scenario: d2d tile in a wide cell
- **WHEN** a d2d run fills a half-width cell of 640 px or more
- **THEN** the tile shows badge "d2d", name, class, level, the frame and the map beside it

#### Scenario: Small cell
- **WHEN** the cell is under 640 px wide
- **THEN** the map is hidden and badge and fields stay

#### Scenario: Original game tile
- **WHEN** a d2lod instance tile shows
- **THEN** its badge reads "d2lod (original game)"

### Requirement: TESTING label
A tile fed by a test or demo run SHALL carry a visible "TESTING" label so it cannot be mistaken for the normal bot's gameplay. Source: user decision 2026-10-30.

#### Scenario: Test run tile
- **WHEN** a test run is live as a source
- **THEN** its tile shows the "TESTING" label next to the backend badge

### Requirement: Source health
A source whose frames stall for more than 8 seconds SHALL show a "reconnecting" tile and MUST be dropped after 60 seconds without frames (defaults, tunable); the grid MUST NOT show a black or frozen tile without that label.

#### Scenario: Stall
- **WHEN** a source delivers no new frame for 9 seconds
- **THEN** its tile shows "reconnecting" over the last frame, dimmed

#### Scenario: Recovery
- **WHEN** frames resume within 60 seconds
- **THEN** the label disappears and the tile keeps its place

#### Scenario: Dropped
- **WHEN** no frame arrives for 60 seconds
- **THEN** the tile is removed and the grid reflows

### Requirement: Stream-safe privacy
No element of the scene SHALL show hostnames, IPs, paths, tokens, emails, machine names or people's names; tiles show only sanitized fields, and scene.json MUST pass the publisher's sanitizer and leak grep before it is published. Source: tools/publish/sanitize.py, AGENTS.md (no secrets in the repo).

#### Scenario: Leak grep
- **WHEN** scene.html, scene.json and the rendered DOM text of every fixture layout are grepped for `.lan.`, IPv4 patterns, `/home/`, `token` and `@`, and the gate runs `sanitize.scan_file` plus the person-name list (real people and machine names; character names are public) over scene.html and scene.json for live, between-runs, stale-cache, d2d and original sources
- **THEN** there are no matches

#### Scenario: Source with an unsafe field
- **WHEN** a source reports a name field with a path or address in it
- **THEN** the tile shows a generic label and the field is not published

### Requirement: Stream off by default and user go
The public stream SHALL be off by default; starting, stopping or restarting it MUST need the user's explicit go in that session, and an encode MUST NOT start while `tools/capacity.sh` answers NO unless the user says go. Source: docs/streamctl.md (dry mode default, keepAlive), docs/process/worker-capacity.md.

#### Scenario: Capacity refusal
- **WHEN** capacity.sh answers NO for the encode
- **THEN** the start is refused with the reason

#### Scenario: Deploying a scene change
- **WHEN** a scene change merges
- **THEN** it ships through the existing publish path and the kiosk reloads it; no stream is started or stopped by the change

### Requirement: No new exposure
Scene changes SHALL NOT open new ports or public endpoints; the kiosk on the stream host reaches sources over loopback or LAN only, and viewers only see the encoded stream and the published scene page. Source: docs/streamctl.md (scene kiosk), docs/d2d-view.md (:8891 is LAN).

#### Scenario: Port check
- **WHEN** the change is deployed
- **THEN** the set of listening ports on the stream host is unchanged

### Requirement: Light scene page
The scene page SHALL stay light for a headless kiosk on a hidden display: at most 4 tiles animate at once (others update at 1 frame per second), live frames are capped at 10 fps per tile (default, tunable), and the page process stays within the existing kiosk limits (CPUQuota 35 percent, MemoryMax 700M). Source: docs/streamctl.md (d2-scene unit limits).

#### Scenario: Nine tiles
- **WHEN** nine sources are live
- **THEN** at most four tiles update above 1 fps and none above 10 fps

#### Scenario: Kiosk budget
- **WHEN** fixture mode with six tiles runs for 10 minutes in the kiosk
- **THEN** the unit stays under its CPU and memory caps (live check, U until run)

### Requirement: Fixture mode
The scene page SHALL render canned sources in fixture mode with `?demo=1` (and a count with `&n=1..9`, holding card with `&n=0`) without any live source or stream, so every layout can be screenshot-tested. Source: tools/publish/tests pattern.

#### Scenario: Layout screenshots
- **WHEN** the page loads with `?demo=1&n=` set to 0, 1, 2, 3, 4, 5, 6 and 9
- **THEN** the holding card, full frame, side by side, 2x2, 3x2 and 3x3 layouts render and match their stored screenshots

#### Scenario: Fixture tiles are labelled
- **WHEN** fixture mode is on
- **THEN** every tile carries the "TESTING" label so it cannot be mistaken for gameplay
