## Why

The user asked on 2026-10-30 that the public stream work the way it did before: a holding card saying what is being worked on, when gameplay returns and how long ago it last showed, rotating info cards, and a grid that grows as streams come online. The baseline (git history: tools/publish/scene.py, docs/streamctl.md) is a TV-style scene page (work chips, milestone ladder, shipped ticker, `?tile=1` 640x360 variant) plus stream.sh grids (`single|grid2|grid3|grid4`, scene always the last tile, picked from the number of healthy instances). Since then d2d runs (d2dview :8891) and backend badges were added, and nothing states the behaviour as a contract.

## What Changes

- New capability spec `stream-scene` (ADDED delta in this card): holding card, grid build, source health, stream-safe privacy, operations, performance, testability.
- Baseline described, not changed: one scene page, grids chosen by source count; the holding card is the scene page itself with an ETA, a "last gameplay" timer and rotating info cards added.
- A small schedule file `status/next-run.json` (`next_start`, `source`, `label`) written by the dispatcher or coordinator; absent means "not scheduled", never an invented ETA.
- A sanitized feed of live sources (d2watch tiles for i1..i5, d2dview runs, test runs) merged into scene.json; the page builds the grid from it, `?demo=1` (fixture mode) renders canned sources for screenshot tests.
- Implementation is a later card step; this card is spec only (status proposed).

## Impact

- Owned later: tools/publish/**, the streamed parts of cmd/d2dview (the combined /stream page by agent streamview, branch agent/streamview, not yet merged: rebase on it), tools/streamctl config, docs/streamctl*.md. No overlap with active cards found in docs/OWNERS (U: streamview owns cmd/d2dview now; sequence after its merge).
- A stream restart to deploy is a user-approved action; going live stays the user's decision.
- Evidence docs: docs/streamctl.md, docs/d2d-view.md, docs/watch.md, tools/publish/*.py.

## Decisions (user, 2026-10-30)

1. `status/next-run.json` is written by the dispatcher automatically on every tick, from the soak and test queue; "not scheduled" when nothing is queued; a time is never invented.
2. The map sits beside the frame only in wide cells (640 px or more).
3. The holding card MAY name recently merged changes and current work in plain words (same sanitizer: no internal names, people or machines).
4. Test and demo runs MAY appear as tiles, labelled "TESTING". The page's `?demo=1` mode is called fixture mode to avoid confusion.
5. Timing defaults kept: 15 s rotation, appear or leave within 10 s, stall 8 s, drop 60 s.

## Dependencies

PARKED until milestone M3 is done (user decision D2, 2026-10-31, docs/roadmap.md). Status `deferred`: the text, `owns:` and `exclusive:` are kept as intended, not claimed (no docs/OWNERS line). The dispatcher sets `proposed` and restores the OWNERS lines when M3 is archived.
