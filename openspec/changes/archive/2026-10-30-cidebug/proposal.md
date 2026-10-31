## Why

The forge run 397 on the NAS runner failed the `test` job although the same steps pass on the worker host.

## What Changes

- tools/ci/scripts.sh runs container-checklist.sh with D2FLEET_MAX_LOAD_FACTOR=-1
- docs/ci.md gets "the NAS runner notes"
