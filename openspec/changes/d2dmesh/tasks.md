# Tasks

```yaml
change: d2dmesh
status: deferred
agent: meshdesign
branch: agent/meshdesign
worktree: ~/code/d2lod-wt/meshdesign
owns:
  - cmd/d2dnode/**
  - internal/d2dmesh/**
  - docs/d2d-mesh.md
  - openspec/changes/d2dmesh/**
exclusive: []
gates:
  - "go test ./internal/d2dmesh ./cmd/d2dnode passes (3-node loopback cluster: join, convergence, cross-node claim, lease re-queue after a node dies, partition heal, pause/drain, token rejection, recipe rejection)"
  - "go test -race ./internal/d2dmesh ./cmd/d2dnode passes on the worker host"
  - "tools/openspec.sh validate --all --strict passes; go run ./cmd/changecheck has 0 warnings"
  - "no real hostname, IP or person in any committed file of this change"
merge:
result: "prototype verified offline on loopback (go test, -race x3 on the worker host, whole-repo vet clean); not run against a real d2d; nothing deployed"
```

## 1. Design

- [x] 1.1 capability spec `d2d-mesh` (ADDED requirements with scenarios); `tools/openspec.sh validate --all --strict`
- [x] 1.2 `docs/d2d-mesh.md` (<= 150 lines): gossip membership, leases, conflict resolution, no Raft and the guarantees, discovery, UI/API on every node, how nodes run work, packaging, relation to d2dfarm and forge Actions, threat model and recipe allow-list, failure table

## 2. Prototype

- [x] 2.1 `internal/d2dmesh`: state table and merge (LWW per node record, fenced leases per job, tombstones), lease claim/heartbeat/expiry/re-queue
- [x] 2.2 transport: HTTPS with mutual TLS, CA derived from the join token, join handshake, push-pull anti-entropy loop
- [x] 2.3 recipe allow-list with argument validation; admission (capacity, paused, draining, host guard, idle-only)
- [x] 2.4 `cmd/d2dnode`: flags, dry-run runner, embedded page and JSON API on the one listening port
- [x] 2.5 tests: 3-node loopback cluster scenarios; fuzz seed for the decoder; `-race` on the worker host via labrun
