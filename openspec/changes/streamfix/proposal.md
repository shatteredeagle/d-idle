## Why

The user asked (2026-10-31) for the stream page header to read "d.idle testing stream", for the engine to appear in the status row, and the gameplay pane went black twice: the page sets the MJPEG image once, so a first request before the first frame (503) or a background-tab cancel never recovers.

## What Changes

- d2dview: flag `-badge` / env `D2DVIEW_BADGE` (default unchanged) for the page header and the `/api/pub` `backend` field; new `/api/pub` `engine` field ("d2d") shown as "engine d2d" in the status row.
- stream page: image reconnect on error (1.5 s), on a stuck image (5 s), and on visibilitychange/focus/pageshow, swapping the new image in on its first frame.
- tools/d2dview-demo.sh: `--badge TEXT`; binaries are shipped under temp names and renamed.
