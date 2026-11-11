# Digest: watch, viewer and player UI

> status: current (TASK-310 digest; the area doc is the authority).

## Purpose
View-only web surfaces for the running instances: the d2watch tile viewer, the d2dview frame and map page, and the player watch screen (Thoughts feed, orders, rules, journal).

## Key files
- `cmd/d2watch/`, `cmd/d2dview/`, `cmd/d2dash/`, `internal/watch/` (hub, handlers, simulated runner, `web/`), `internal/viewport/` (capture and streaming)
- `docs/watch.md`, `docs/watch-screen.md`, `docs/d2d-view.md`, `docs/viewport.md`, `docs/viewer-fps.md`, `docs/watch-origin.md`
- `docs/contracts/dashboard-pub.md`, `docs/research/watch-screen-design.md`

## Invariants
- The viewer side is GET only (every other method is 405); every write goes to the separate answer listener with origin, content type and body-size checks (docs/watch-screen.md).
- Capture only the game window, never the root window (an instance can sit on a real desktop) (docs/watch.md).
- One capture in flight per instance; capture is lazy (docs/watch.md, Gotchas).
- The viewer is read-only and token-protected unless bound to loopback (docs/watch.md).
- The `/api/pub` fields and the stream recovery rules are fixed by docs/contracts/dashboard-pub.md.
- The currency screen is off in vanilla mode and `/api/currency` answers 404 (docs/currency.md).

## Gotchas
- A slow capture once queued a new capture per tick; the scheduler now keeps one in flight (docs/watch.md, Gotchas).
- A browser left open can hang the video (VIEWHANG) and an idle CPU burn existed (D2WATCHCPU); read those sections before changing streaming (docs/watch.md).
- The watch screen is verified on the simulated runner only, not live (docs/watch-screen.md status banner).
- The black rectangle in a frame comes from the game's own display output, not from the viewer (docs/watch.md).
- Pixel streaming from the tap is offline tested, not live (docs/d2d-view.md, last section).

## Read next
- [../watch-screen.md](../watch-screen.md)
- [../watch.md](../watch.md)
