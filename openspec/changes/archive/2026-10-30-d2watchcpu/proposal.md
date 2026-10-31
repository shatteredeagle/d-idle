## Why

The live d2watch burns about 70% of a core continuously: serveStream spins on a closed instance stream while a browser keeps the connection open.

## What Changes

- serveStream stops when its stream is closed; Stream.Close wakes waiters; real idle timeouts told apart correctly
- opt-in -pprof flag; docs/watch.md records evidence and redeploy steps
