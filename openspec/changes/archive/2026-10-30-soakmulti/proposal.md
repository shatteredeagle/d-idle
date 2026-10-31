## Why

Two soaks collided on the shared bot binary path, ports and log directory, so only one soak could run at a time.

## What Changes

- per-instance layout: bot binary, logs, unit, ports
- stop and status act on one instance only
