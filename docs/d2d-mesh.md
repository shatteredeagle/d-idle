# d2d node mesh (d2dnode): design and prototype

> status: design + loopback prototype (change `d2dmesh`, 2026-10-30). Nothing deployed, no real d2d run. Contract: `openspec/changes/d2dmesh/specs/d2d-mesh/spec.md`. Code: `internal/d2dmesh`, `cmd/d2dnode`. Tags: V = proven by `go test ./internal/d2dmesh ./cmd/d2dnode`, U = not run.

User direction (2026-10-30): every container could be the server; nodes are meshed, you add one to the farm and manage the farm from any node (Tdarr nodes, forge runners, but peer-to-peer). This is the third transport behind the farm verbs (section 9); the v1 push farm (`tools/d2dfarm.sh`, change `d2dfarm`) stays the thing that works today.

## 1. Shape
- Leaderless. Every node holds the whole table (nodes + jobs), serves the same API and page on one TLS port, and runs work from the same queue. There is no coordinator process, no database, no election.
- One Go binary, stdlib only: `d2dnode -name N -token-file F [-listen 127.0.0.1:8470] [-seed a:p,b:p] [-labels d2d,fast] [-capacity 2] [-runner tools/d2dslot-recipe.sh | -dry-run]`.
- Small farm by assumption (3 to 10 nodes): the whole table (a few KB) travels on every exchange; no deltas, no Merkle trees.

## 2. Membership: push-pull gossip (V)
- Anti-entropy round every `-interval` (3 s default, 40 ms in tests): refresh own record, expire leases, then POST the whole table to `Fanout` random peers (`/mesh/v1/push`) and merge the reply. Peers = seeds + every address learned from node records. Join = start with one reachable seed; the seed's reply carries everyone (V: 3 nodes, B and C seeded with A only, all list a,b,c; tables `Equal` after a quiet period; a fresh change is on every node within 3 rounds).
- Node record: name, addr, labels, capacity, running, last-seen, version, Lamport clock, tombstone (`gone`). Liveness fields are written only by the node itself; merge = higher clock wins, ties by later last-seen (LWW register).
- Operator flags (`paused`, `draining`) are a SECOND LWW register inside the record with its own clock and writer name, merged independently, so a pause set on A is never overwritten by B's next self-refresh (V: `TestMergeRules`, `TestPauseAndDrain`).
- Leave = write `gone`, push once, exit (V). A crash leaves no tombstone: the record stays with an old `last_seen`; the UI greys it, the leases do the real work (section 3). Garbage collection of dead records: not done (open question 4).
- Lamport clock: incremented on every local write, bumped to the max seen on every merge.

## 3. Jobs and leases (V)
- Job record: id (`recipe-unixtime-rand`), recipe, validated args, required labels, submitter, state (`queued running done failed cancelled`), owner, lease expiry, **fence**, attempt, clock, result (attempt, node, exit, summary, out dir, pid).
- Claim = owner + lease + `fence+1` + `attempt+1` in one write under the local lock; the claiming node starts the run at once and heartbeats every TTL/3 (extends the lease only while it still owns that fence). Re-queue = any node that sees an expired lease writes `queued`, owner empty, `fence+1`. Cancel = `cancelled`, `fence+1`.
- Merge order of a job record is a total order on **(fence, terminal?, clock, owner)**: higher fence wins; at equal fence a terminal state (done/failed/cancelled) beats a live one (so a cancel racing a claim always wins, V); then clock; then the lexically smaller owner. Because it is one key comparison, every node converges whatever the merge order.
- A node whose run lost the fence (merge shows another owner or fence) cancels the run's context and DROPS its result (V: `TestOwnerDiesJobRequeued`: B stops mid-job, A re-queues after the 300 ms lease, C runs attempt 2 with fence 3; B and C each ran exactly once).
- Results are keyed by job id + attempt (`<out>/<job>/<attempt>/`), so a late stale attempt can never overwrite the winner's files.

## 4. Verbs from any node, results (V for verbs; U for file collection)
`GET /api/v1/{state,nodes,jobs}`, `POST /api/v1/jobs` (recipe, args, labels), `POST /api/v1/jobs/{id}/cancel`, `POST /api/v1/nodes/{name}/{pause,resume,drain,undrain}`, `GET /api/v1/jobs/{id}/attempts/{n}/tar` (the output dir of an attempt, served only by the node that ran it, only the dir recorded in the result), `GET /` the embedded status page (one HTML string, polls `/api/v1/state`, buttons are the same POSTs). A verb received on any node is a write into that node's table and gossips out; it does not matter which node you talk to. Collect-to-any-node = that node fetches the tar from the runner node (U, no test yet).

## 5. One port, loopback by default
Gossip, API and page share the one TLS listener. Default `-listen 127.0.0.1:8470`; a LAN address is an explicit flag. A non-TLS byte on the port closes the connection with no reply (V).

## 6. Running work and admission (V for admission; U for the real runner)
- A node claims only when: not paused, not draining, running < capacity, host guard says yes, job labels are a subset of its labels (V). Guard = any script, exit 0 = yes (`tools/capacity.sh --add 1` on the worker host; an idle-only node uses an idle check). Without `-runner` or `-dry-run` a node is manager-only and never claims (V).
- Runner = `tools/d2dslot.sh`-style script: takes a local slot, launches a hidden d2d (`-w -ns`, no X, no original game), records the pid in the attempt dir; the node stops a run only by that pid (SIGTERM via the process handle, V in `cmd/d2dnode` test). The script receives `RECIPE OUTDIR key=value...`; nothing from a job is ever interpolated into a shell line.
- Capacity on the worker host = the slot pool size (4); in a container = 1 (the container is the slot); a WSL node = however many slots its lab has.

## 7. Packaging (U)
- Container: `entrypoint.sh` starts `d2dnode` when `D2D_NODE=1` (env `D2D_MESH_NAME SEEDS TOKEN_FILE LABELS CAPACITY LISTEN`), the runner is the in-container d2d, capacity 1. The NAS: each container is a node, UI-visible template with those fields and one published port; the token file is a mounted secret, never an env value in the template.
- Worker host: a user service running `d2dnode -capacity 4 -runner tools/d2dslot-recipe.sh -guard 'tools/capacity.sh --add 1'`. WSL node: the same binary inside the distro, seeds = the LAN nodes.

## 8. Threat model and safety
- Shared join token: the mesh CA is derived from the token (sha256 -> ed25519 seed; the CA cert bytes are identical on every node, nothing else to distribute). Each node mints a fresh leaf signed by it; mutual TLS 1.3, the client verifies the CA, the server requires a client cert (V: wrong token = handshake fails, nothing learned either way; no client cert = refused).
- Token leak = anyone on the LAN can be a node and submit or claim jobs. So jobs are RECIPES from an allow-list with range-checked arguments (`sleep`, `d2d-soak`, `d2d-replay`; unknown recipe, unknown argument, out-of-range value, path-like name all refused, V), never shell. A rogue node can still run its own junk and claim real jobs; mitigation = rotate the token (restart all nodes with a new one; old certs stop verifying) and watch the node list. Rate limits: none yet (open question 3).
- Pinned identities instead of a token-derived CA (each node's cert fingerprint listed on every other node) is the stronger option when nodes are few and static; the transport supports it with a config change (RootCAs -> VerifyPeerCertificate), U.
- Safety: hidden d2d only; pid-recorded stop; the runner refuses anything but the allow-listed recipes; the mesh never touches a game instance of the original game (AGENTS.md rules hold on every node).

## 9. Relation to d2dfarm and forge Actions
The farm verbs are `probe start status collect stop`. v1 (`d2dfarm.sh`) pushes them over ssh from one machine; forge Actions would carry them as workflow jobs to labelled runners; the mesh carries them as table writes that any node applies. `d2dfarm.sh` can grow a `--transport mesh` that POSTs to the nearest node; the verdict and collect logic stay in one place.

## 10. Why no Raft; what is and is not guaranteed
- Raft/etcd would give one true queue but needs a quorum (half the farm down = no work), a consensus library (not stdlib) or 2000+ lines of our own, and operational care. The farm is small, jobs are idempotent soaks, and a double run costs a slot-hour, not correctness.
- Guaranteed: convergence (every connected node reaches the same table); at-least-once execution; exactly one recorded winner per job (fence order); cancel wins over a concurrent claim; a dead owner's job is re-queued within one lease after the next round; no plaintext, no unauthenticated peer.
- NOT guaranteed: at-most-once. After a partition both sides can claim the same job at the same fence; on heal one loses by (clock, owner) order, its run is cancelled and its result dropped, but it DID run for a while (double slot use). A node that is partitioned for longer than the lease keeps running its job while the other side re-queues it: same outcome. Wall clocks: lease expiry compares the OWNER's heartbeat time with the OBSERVER's clock; nodes more than about a lease apart in time will re-queue live jobs (keep NTP; open question 2).

## 11. Failure scenarios
| scenario | what happens | proven |
|---|---|---|
| node crashes mid-job | heartbeats stop; after the lease any node re-queues (fence+1); another node runs attempt 2 | V |
| node crashes and comes back with a stale run | its merge shows a higher fence; the run is cancelled, result dropped | V (fence check) |
| partition, submit on both sides | each side runs what it can; heal merges both; same-fence double claims resolve to one owner | V (submit both sides, heal); split-brain tie V on the merge function |
| wrong token / no client cert / plaintext | handshake refused, nothing learned / refused / closed silently | V |
| operator pauses or drains from another node | flag register wins over the node's own refresh; running job finishes on drain, nothing new | V |
| host guard says NO | node never claims; others do | V |
| unknown recipe or bad argument | 422 on every node, nothing queued | V |
| clean leave | tombstone gossiped, node disappears from lists | V |
| all seeds down at join | the new node waits alone until a seed answers; a LAN beacon would fix it | U (discovery v2) |
| clocks skewed by more than a lease | spurious re-queues, double runs | U, known limit |
| table grows without bound | finished jobs are never pruned | U, open question 4 |

## 12. Open questions for the user
1. Token-derived CA (one secret, easy) or pinned per-node certificates (stronger, more setup)?
2. Lease length for real soaks (30 s default; NTP on every node assumed) and whether a node may take over a job before the lease expires when the owner is tombstoned.
3. Rate limits and audit log for submissions: per node, per minute, or none (private LAN)?
4. Retention: prune done/failed jobs and dead node records after N days, or keep forever until a `gc` verb?
5. Discovery: explicit seeds only, or add a LAN multicast beacon (one more thing that can be spoofed)?
