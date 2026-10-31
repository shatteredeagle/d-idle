## Why

Public page flicker (DISPATCH-39): duplicate Loopy from the `bot-*.log` glob, empty runs list when the ssh fetch times out, so the page flips to between runs.

## What Changes

- tools/publish/runs.py: dedupe per source and character; cache the last good fetch (publisher state dir, never published) and reuse it up to 15 min, marked "last update N min ago".
