## Why

The user wants to watch the sped-up d2d bot (40 to 60x real time): a top-down map with trail and monsters, and real game frames. d2d's control socket takes one client, so the viewer needs its own read-only path that never slows the run.

## What Changes

- cmd/d2dview: a tap between bot and d2d (one upstream, two private sockets, the viewer's filtered to read verbs), a web page with map, bars, speed factor, goal text and sampled game frames, fast and watch modes
- docs/d2d-view.md: design, sharing choice, ports, measured cost, fork requests
