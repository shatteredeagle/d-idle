---
id: TASK-15
title: "stream-grid: public stream scene with a holding card and a source grid"
status: Deferred
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "parked"
dependencies: []
priority: low
ordinal: 15000
card: "stream-grid"
owns:
  - "tools/publish/**"
  - "docs/streamctl-grid.md"
exclusive: []
branch: "agent/streambuild"
agent: "streambuild"
result: "Built offline as a NEW page scene-grid.html + scene-grid.json (tools/publish/grid.py, scene-grid.js/css, info-cards.json, fixtures/grid-0..9.json, tests/test_grid.py + grid.test.js); scene.html/scene.json unchanged (byte-identical test). Not verified in a browser or on the kiosk. Open: kiosk d2d frame embed, live local feed for 8 s / 60 s health, kiosk CPU budget, screenshots, d2dview /api/pub feed (cmd/d2dview untouched). See docs/streamctl-grid.md."
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/stream-grid"
contracts: ["proposed/stream-grid"]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The user asked on 2026-10-30 that the public stream work the way it did before: a holding card saying what is being worked on, when gameplay returns and how long ago it last showed, rotating info cards, and a grid that grows as streams come online. The baseline (git history: tools/publish/scene.py, docs/streamctl.md) is a TV-style scene page (work chips, milestone ladder, shipped ticker, `?tile=1` 640x360 variant) plus stream.sh grids (`single|grid2|grid3|grid4`, scene always the last tile, picked from the number of healthy instances). Since then d2d runs (d2dview :8891) and backend badges were added, and nothing states the behaviour as a contract.

What changes:
- New capability spec `stream-scene` (ADDED delta in this card): holding card, grid build, source health, stream-safe privacy, operations, performance, testability.
- Baseline described, not changed: one scene page, grids chosen by source count; the holding card is the scene page itself with an ETA, a "last gameplay" timer and rotating info cards added.
- A small schedule file `status/next-run.json` (`next_start`, `source`, `label`) written by the dispatcher or coordinator; absent means "not scheduled", never an invented ETA.
- A sanitized feed of live sources (d2watch tiles for i1..i5, d2dview runs, test runs) merged into scene.json; the page builds the grid from it, `?demo=1` (fixture mode) renders canned sources for screenshot tests.
- Implementation is a later card step; this card is spec only (status proposed).

Parked: named in ADR 0015 item 10

Gates (from the card):
- python3 -m unittest in tools/publish/tests passes, including one fixture test per layout size (1, 2, 3-4, 5-6, 7-9 tiles)
- ?demo=1 screenshots of every layout size show no hostname, IP, path or token (leak grep over scene.html, scene.json and the DOM text)
- go test ./cmd/d2dview passes (pub feed fields sanitized)

Source: git tag openspec-final, openspec/changes/stream-grid/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 1.1 Write specs/stream-scene/spec.md (ADDED delta); `tools/openspec.sh validate --all --strict` passes
- [ ] #2 1.2 User answers the open questions in proposal.md
- [ ] #3 2.1 Sources feed: merge d2watch tiles, d2dview /api/pub and demo runs into scene.json with `last_seen`, backend, name, class, level (partly: sources come from the existing runs records into scene-grid.json with last_seen, backend, name, class, level, testing; d2watch and d2dview feeds open)
- [x] #4 2.2 Schedule reader: `status/next-run.json` to ETA text, "not scheduled" when absent or past
- [x] #5 2.3 Holding card, info card rotation and grid layouts (scene-grid.js, new page; `?demo=1` fixtures)
- [ ] #6 2.4 Health states (reconnecting, drop) and frame-rate caps (health done; fps cap is only a timing value until a frame feed exists)
- [x] #7 2.5 Fixture tests per layout size and a leak grep; docs/streamctl-grid.md; deploy through the existing publish path (no restart: user go still needed to point the kiosk)
<!-- AC:END -->
