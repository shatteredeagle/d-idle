## Why

The NAS now serves the `linux` label. The worker host's runner should keep only `worker` so default CI goes to the NAS and the soak host is protected (user approved 2026-10-30).

## What Changes

- The worker host runner labels `worker:host,linux:host` -> `worker:host` (config.yml `runner.labels`)
- docs/process/ci-runners-worker.md (what was done, routing test, rollback)
