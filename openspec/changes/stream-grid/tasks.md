# Tasks

```yaml
change: stream-grid
status: deferred
agent: streambuild
branch: agent/streambuild
worktree: ~/code/d2lod-wt/streambuild
owns:
  - openspec/changes/stream-grid/**
  - tools/publish/**
  - docs/streamctl-grid.md
exclusive: []
gates:
  - "tools/openspec.sh validate --all --strict passes"
  - "python3 -m unittest in tools/publish/tests passes, including one fixture test per layout size (1, 2, 3-4, 5-6, 7-9 tiles)"
  - "?demo=1 screenshots of every layout size show no hostname, IP, path or token (leak grep over scene.html, scene.json and the DOM text)"
  - "go test ./cmd/d2dview passes (pub feed fields sanitized)"
merge: ""
result: "Built offline as a NEW page scene-grid.html + scene-grid.json (tools/publish/grid.py, scene-grid.js/css, info-cards.json, fixtures/grid-0..9.json, tests/test_grid.py + grid.test.js); scene.html/scene.json unchanged (byte-identical test). Not verified in a browser or on the kiosk. Open: kiosk d2d frame embed, live local feed for 8 s / 60 s health, kiosk CPU budget, screenshots, d2dview /api/pub feed (cmd/d2dview untouched). See docs/streamctl-grid.md."
```

Spec-only today. Implementation starts after agent/streamview is merged. Stream start, stop, restart and any encode need the user's explicit go; no stream is touched by this card until then (U: no instance needed, exclusive is empty).

## 1. Spec

- [x] 1.1 Write specs/stream-scene/spec.md (ADDED delta); `tools/openspec.sh validate --all --strict` passes
- [ ] 1.2 User answers the open questions in proposal.md

## 2. Implementation (later)

- [ ] 2.1 Sources feed: merge d2watch tiles, d2dview /api/pub and demo runs into scene.json with `last_seen`, backend, name, class, level (partly: sources come from the existing runs records into scene-grid.json with last_seen, backend, name, class, level, testing; d2watch and d2dview feeds open)
- [x] 2.2 Schedule reader: `status/next-run.json` to ETA text, "not scheduled" when absent or past
- [x] 2.3 Holding card, info card rotation and grid layouts (scene-grid.js, new page; `?demo=1` fixtures)
- [ ] 2.4 Health states (reconnecting, drop) and frame-rate caps (health done; fps cap is only a timing value until a frame feed exists)
- [x] 2.5 Fixture tests per layout size and a leak grep; docs/streamctl-grid.md; deploy through the existing publish path (no restart: user go still needed to point the kiosk)
