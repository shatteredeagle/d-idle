## Why

The d2d test farm should be a set of distributed nodes (like the Tdarr and the forge nodes): any machine with the private d2d lab can register, take d2d jobs by label and capacity, be paused, and return results that merge into one soak verdict. v1 transport is ssh push; the transport is swappable (forge Actions or a pull coordinator later, docs/process/d2dfarm.md).

## What Changes

- `tools/d2dfarm.sh` (nodes, queue, dispatcher, collect, verdict), `tools/d2dfarm-remote.sh` (node side), host registry example
- capability spec `d2d-farm`
- cross-host determinism check as the node qualification test
