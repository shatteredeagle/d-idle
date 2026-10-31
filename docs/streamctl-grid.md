# Stream grid scene (scene-grid.html)

> status: built and tested offline (agent STREAMBUILD, change card `stream-grid`); NOT shown in a browser by a person, NOT on the stream. The running stream scene (`scene.html`) is unchanged.

Spec: `openspec/changes/stream-grid/specs/stream-scene/spec.md`. Code: `tools/publish/grid.py` (pure logic and builder), `scene-grid.js` and `scene-grid.css` (inlined into the page at build time), `info-cards.json` (hand-kept public-safe card text), `fixtures/grid-0..9.json` (layout fixtures, regenerate with `python3 tools/publish/fixtures/regen.py`).

## What it is
A NEW page published next to the others by the same publisher: `scene-grid.html` plus `scene-grid.json` (a superset of scene.json: sources, next_run, timing, card texts). `scene.html` and `scene.json` are byte-identical with and without it (test). Both new names are on the sanitizer and `verify_dir` allow-lists and pass `sanitize.scan_file` in the publisher's fail-closed gate, like the rest.
- No sources live: holding card (work chips, "Next gameplay in N min: label" or "not scheduled", "Last gameplay N min ago" or "No gameplay yet today", rotating info cards every 15 s with dots).
- Sources live: grid 1 / 2 / 2x2 / 3x2 / 3x3 (max 9, the 9 most recently started), each tile with backend badge ("d2lod (original game)" or "d2d"), a TESTING label when the run source is flagged, name, class, level, kills, deaths, area. An empty grid cell shows the info-card deck.
- Health: stall shows "reconnecting" dimmed; drop removes the tile and the grid reflows.
- Query flags: `?demo=1&n=0..9` fixture mode (all tiles TESTING), `?kiosk=1` (hides the online indicator and the cursor).
- Test/demo runs: set `"testing": true` on a source in `tools/publish/runs.json` (config, never published).
- ETA: the publisher reads `~/d2lod-status/next-run.json` (`--next-run PATH`); only `next_start`, `source`, `label` survive, cleaned; missing, broken, null or past = "not scheduled".

## Honest limits (open)
- The public feed is rebuilt every 5 minutes from log minutes, so on the public page "within 10 s" is bounded by the publish cadence. The page polls every 10 s and uses `PUBLIC_TIMING` (stall 150 s, drop 600 s); the spec defaults (8 s, 60 s) are in `grid.TIMING` and apply once a local live feed exists (U, not built).
- Public tiles show the sanitized card, never video: the public page has no video, image or iframe element and the sanitizer would block one.
- Kiosk embedding of d2d frames (`http://127.0.0.1:<d2dview port>/stream` per d2d tile, `D2S_VIEW_PORTS`): NOT implemented. It would need either a sanitizer relaxation for a public file (not done) or a locally generated, never published kiosk page served on loopback. Proposal: a `grid.py --kiosk-local DIR` writer plus `python3 -m http.server --bind 127.0.0.1`; needs a user decision and the stream host.
- Map beside the frame (cells of 640 px or more) is only a layout flag (`wide`) today; no map is drawn without the frame.
- Light page: only the first 4 tiles carry an animation class, nothing else animates; the 10 fps cap is a `timing.fps_cap` value for a future frame feed. The kiosk CPU and memory check is U until run.

## Kiosk start command (for the user to approve, NOT run)
The scene kiosk unit loads one URL (`tools/stream/scene.sh`, default scene.html). To point it at the grid page for a scene-only stream, on the stream host as the desktop session user:

    D2S_SCENE_URL='https://shatteredeagle.github.io/d-idle/scene-grid.html?kiosk=1' tools/stream/scene.sh start --mode full

`stream.sh` honours the same variable (`D2S_SCENE_URL`) when it starts the scene itself. Revert: `tools/stream/scene.sh stop full`, restart without the variable. Starting, stopping or restarting any stream needs the user's go; this card does none of it. A tile-mode kiosk (640x360) is not designed for the grid page (it appends `&tile=1`, which the grid page ignores).

## Deploy
The 5-minute `d2-publish.timer` publishes the two new files after merge. That is acceptable only because the page is new and leak-gated; nothing links to it and no existing page changed. Ports: none opened.
