## Why

The user wants every d2d container to be able to act as the farm server: nodes are meshed, a new one is added to the farm, and the farm is managed from any node (Tdarr nodes and forge runners, but peer-to-peer, no coordinator host). The v1 farm (change d2dfarm) is a push model driven from one machine over ssh; it needs that machine up and holds the only queue. A leaderless node mesh removes the single point and makes the NAS containers, the worker host slots and a WSL node interchangeable members. This card designs the mesh and proves the hard parts (membership convergence, job leases, partition recovery, authentication) in a stdlib-only prototype with an in-process cluster test; nothing is deployed.

## What Changes

- Capability spec `d2d-mesh` (ADDED): membership, leases, admission, safety, security, hygiene, version skew.
- `docs/d2d-mesh.md`: the design and the honest limits (no Raft; at-least-once; double execution possible after a partition).
- `internal/d2dmesh`: state table + merge, lease logic, mutual-TLS HTTP transport with token-derived identity, join, anti-entropy loop, recipe allow-list, claim/heartbeat/expiry, dry-run runner.
- `cmd/d2dnode`: the node daemon (flags `-listen -seed -token-file -name -labels -capacity -dry-run`), embedded status page and JSON API on the same port.
- Tests: 3-node loopback cluster (join, convergence, cross-node claim, node death mid-job, partition heal, pause/drain, token rejection, recipe rejection) and a fuzz seed for the gossip decoder. No real d2d runs.
