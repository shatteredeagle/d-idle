## Why

User decision 2026-10-30: pause Tdarr on the worker host while agent work settles, and let several hidden d2d runs go side by side instead of one global lock.

## What Changes

- lab-rules: the single d2d lock becomes a pool of 4 slots (`tools/d2dslot.sh`)
- `tools/capacity.sh` counts lab binaries (`d2d.lab5`), which `pgrep -x d2d` never matched
- `docs/process/worker-compute.md`: Tdarr pause and resume, slot rules
