---
id: TASK-14
title: "d2dmesh: peer-to-peer d2d node mesh for the farm"
status: Deferred
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "parked"
dependencies: []
priority: low
ordinal: 14000
card: "d2dmesh"
owns:
  - "cmd/d2dnode/**"
  - "internal/d2dmesh/**"
  - "docs/d2d-mesh.md"
exclusive: []
branch: "agent/meshdesign"
agent: "meshdesign"
result: "prototype verified offline on loopback (go test, -race x3 on the worker host, whole-repo vet clean); not run against a real d2d; nothing deployed"
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/d2dmesh"
contracts: ["proposed/d2dmesh"]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The user wants every d2d container to be able to act as the farm server: nodes are meshed, a new one is added to the farm, and the farm is managed from any node (Tdarr nodes and forge runners, but peer-to-peer, no coordinator host). The v1 farm (change d2dfarm) is a push model driven from one machine over ssh; it needs that machine up and holds the only queue. A leaderless node mesh removes the single point and makes the NAS containers, the worker host slots and a WSL node interchangeable members. This card designs the mesh and proves the hard parts (membership convergence, job leases, partition recovery, authentication) in a stdlib-only prototype with an in-process cluster test; nothing is deployed.

What changes:
- Capability spec `d2d-mesh` (ADDED): membership, leases, admission, safety, security, hygiene, version skew.
- `docs/d2d-mesh.md`: the design and the honest limits (no Raft; at-least-once; double execution possible after a partition).
- `internal/d2dmesh`: state table + merge, lease logic, mutual-TLS HTTP transport with token-derived identity, join, anti-entropy loop, recipe allow-list, claim/heartbeat/expiry, dry-run runner.
- `cmd/d2dnode`: the node daemon (flags `-listen -seed -token-file -name -labels -capacity -dry-run`), embedded status page and JSON API on the same port.
- Tests: 3-node loopback cluster (join, convergence, cross-node claim, node death mid-job, partition heal, pause/drain, token rejection, recipe rejection) and a fuzz seed for the gossip decoder. No real d2d runs.

Parked: named in ADR 0015 item 10

Gates (from the card):
- go test ./internal/d2dmesh ./cmd/d2dnode passes (3-node loopback cluster: join, convergence, cross-node claim, lease re-queue after a node dies, partition heal, pause/drain, token rejection, recipe rejection)
- go test -race ./internal/d2dmesh ./cmd/d2dnode passes on the worker host
- tools/openspec.sh validate --all --strict passes; go run ./cmd/changecheck has 0 warnings
- no real hostname, IP or person in any committed file of this change

Source: git tag openspec-final, openspec/changes/d2dmesh/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 1.1 capability spec `d2d-mesh` (ADDED requirements with scenarios); `tools/openspec.sh validate --all --strict`
- [x] #2 1.2 `docs/d2d-mesh.md` (<= 150 lines): gossip membership, leases, conflict resolution, no Raft and the guarantees, discovery, UI/API on every node, how nodes run work, packaging, relation to d2dfarm and forge Actions, threat model and recipe allow-list, failure table
- [x] #3 2.1 `internal/d2dmesh`: state table and merge (LWW per node record, fenced leases per job, tombstones), lease claim/heartbeat/expiry/re-queue
- [x] #4 2.2 transport: HTTPS with mutual TLS, CA derived from the join token, join handshake, push-pull anti-entropy loop
- [x] #5 2.3 recipe allow-list with argument validation; admission (capacity, paused, draining, host guard, idle-only)
- [x] #6 2.4 `cmd/d2dnode`: flags, dry-run runner, embedded page and JSON API on the one listening port
- [x] #7 2.5 tests: 3-node loopback cluster scenarios; fuzz seed for the decoder; `-race` on the worker host via labrun
<!-- AC:END -->
