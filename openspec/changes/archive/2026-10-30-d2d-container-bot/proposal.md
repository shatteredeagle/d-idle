## Why

Steps 5 to 8 of docs/d2d-container/risks.md under ADR 0009: the bot entry that drives d2d in the container, the d2fleet wizard for d2d, the viewer source and a soak driver without /proc/pid/mem. Blocked by d2d-container-build and d2d-default-backend.

## What Changes

- d2bot entry for d2d with per-slot sockets (step 5)
- d2fleet wizard for d2d (step 6)
- viewer source via the screenshot verb (step 7)
- soak driver sidecar (step 8)
