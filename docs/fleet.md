# Fleet supervisor and party orchestrator (agent FLEET)

> status: Wine-era fleet (N Game.exe per container); d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md), Wine container work paused (docs/decisions.md 2026-10-30). The d2d container is docs/d2d-container/ and cards d2d-container-*.

Code: `internal/fleet` (host supervisor), `internal/party` (party state machine), `cmd/d2fleet` (thin CLI + `serve`).
Context: one container or machine is ONE bot host with N instance slots, added and removed at runtime by an app call, never
through Docker (deploy/container/README.md "Decision", PLAN.md item 7, docs/archive/review/design-gaps.md D8).

Tags: everything below is VERIFIED OFFLINE ONLY (`go test -race ./internal/fleet ./internal/party ./cmd/d2fleet`, fake Runtime,
fake Driver). Nothing here has run against a live game; the live checks are listed at the end. Numbers taken from other docs keep
their tag there (V = docs/multi.md measurements on the worker host).

## Fleet (internal/fleet)

```go
f, _ := fleet.New(fleet.Config{Runtime: fleet.LifeRuntime{Life: launch.NewLife(w), Flow: ...}, Bus: bus, DataDir: root, StateFile: ...})
v, err := f.Add(ctx, fleet.Spec{Character: "Navi", Class: "sorceress", Role: party.Leader, Profile: "...", Build: "..."})
f.WaitState(ctx, v.ID, fleet.Running)
l, err := f.Lease(v.ID, "bot", time.Minute)   // *LeasedError if somebody else drives it
f.Remove(ctx, v.ID, fleet.RemoveOpts{})        // graceful drain
f.List(); f.Get(id); f.Status()                // views; Status adds the capacity checks
```

- **Slot ids and allocation.** `Add` picks the lowest free `i1..i9` (`launch.ValidID` only allows i0..i9; i0 is the desktop game
  and never a slot). Helper port `47012+N`, display `:10N` (`:100+N`), prefix dir from `Config.Prefix` (cmd uses
  `paths.Instance(id).Prefix()`). Freed numbers are reused. A caller may pin `Spec.ID`. Invariants refused with `ErrDuplicate`:
  same character twice (case-insensitive; one save dir per character), same slot id. At most one TCP host per machine is the
  party's job (one host per party), not the fleet's.
- **Admission** (`Limits.Check`, refusal is `*AdmissionError` wrapping `ErrAdmission`, with `Check`, `Need`, `Have`, and a sentence
  in `Reason`). Checks in order: slots (default max 9), cpu (`reserve 1 + (n+1) * 1.05` cores <= usable cores, usable = min of
  online cores and cgroup `cpu.max`), load (`load1 + 1.05 <= cores * 1.25`), ram (`1024 + 400` MB <= MemAvailable, also capped by
  cgroup `memory.max - memory.current`), disk (`1024` MB free at `DataDir`). Defaults come from V docs/multi.md (about 1.05 cores,
  265 MB PSS, 125 MB disk per instance); the RAM/disk/load headroom factors are our choice (U). The count includes slots that are
  starting or restarting; a `Failed` slot holds none. Check+register is under one lock, so two parallel Adds cannot both squeeze
  in. `Status().Capacity` shows the same numbers plus `Headroom` and whether another slot would be admitted. Note a loaded
  host (the worker host: load 35-40) is refused by the load check; relax `Limits.MaxLoadFactor` (<0 turns it off) there.
- **Leases.** In-process ownership of one slot, `Lease(id, owner, ttl)`; a second caller (even the same owner) gets
  `*LeasedError{Holder, Until}` (`errors.Is(err, ErrLeased)`). `Renew` extends a live lease, an expired one is dead for good,
  `Release` of a stale lease never frees a newer one (token check). Draining slots hand out no leases; `Remove` of a leased slot
  needs `Force` (the lease dies with the slot). This replaces the `/tmp` flock of the dev tools in product mode; dev tools keep
  flock (`launch.LockInstance`). The fleet only records ownership: the bot/input layer must hold a lease while it drives the instance (see requests).
- **Drain on remove.** `Remove`: mark `draining` (no new leases), cancel the supervisor and wait for it, `Runtime.LeaveGame`
  (OOG Esc menu, Save and Exit, `launch.Flow.LeaveGame`; budget `DrainTimeout` 30 s; only when the slot was starting or running),
  then `Runtime.Stop` (SIGTERM Game.exe, `wineserver -k`, Xvfb) and free the slot. A failed leave still stops the game and returns an
  error saying the stop was unclean. `Close` drains every slot (used on SIGTERM of `d2fleet serve`).
- **Supervised restart.** One goroutine per slot: `Runtime.Start` (serialised across slots with `Stagger` 5 s after each, like
  `launch.StartInstance`), `Login` (OOG to in-game), then poll `Alive` every 5 s. When the game is gone or start/login failed:
  `Stop` leftovers, backoff `2 s, 4 s, ... 2 min`, start again. Backoff and the restart counter reset after the game ran
  `StableAfter` (5 min). After `MaxRestarts` consecutive restarts (0 = unlimited, `d2fleet serve` uses 10) the slot is `failed`;
  `Restart` revives it. This is the host-level restart of the whole instance (game process); the per-instance sentinel
  (`internal/sentinel`, rules like restart-game / helper-gone) stays the finer-grained watcher inside one instance. Both call
  `launch.Life`, so do not run both on the same slot without agreeing who restarts (request below).
- **Events.** `event.KindLifecycle`, `Source "fleet"`, `Instance` = slot id, `Data` = `Lifecycle{State: "slot.starting|running|restarting|draining|failed|removed", Reason, Message}`.
- **Persistence.** `Config.StateFile` (cmd: `<data>/fleet.json`) keeps the specs; `Restore` re-adds them through admission after a
  host restart (a smaller host takes fewer and reports the refusals).
- **Runtime.** `Start/Login/Alive/LeaveGame/Stop` per `Alloc`. `LifeRuntime` implements it over `launch.Life` (Start = Life.Start +
  EnsureHelper) and a caller-supplied `Flow(id)` factory for the OOG steps (Flow.Start with `CharSpec{Create: class != ""}`).
  Liveness is "Game.exe pid with the instance marker exists" (`Life.GamePIDs`); a frozen game is the sentinel's job.

### Admin HTTP API (`fleet.Handler(f, token)`)
Served by `d2fleet serve` through internal/server: `server.Config{FleetAdmin: fleet.Handler(f, ""), Slots: ...}` mounts it at
`/api/v1/fleet-admin/` (prefix stripped, the server's token applies; `token` here is only for standalone use). `Config.Slots`
adds every slot to `/api/v1/fleet` (`slots`, and as `instances` entries with `slot` = state, `runner.state` = state when the slot has
no instance on that server; a failed slot raises `failsafe`). The dashboard's chip click on such a slot selects an instance that
has no routes yet (U: nothing registers per-slot managers in d2fleet).

| Route | Meaning |
|---|---|
| `GET /status` | `HostStatus`: capacity checks and all slots |
| `GET /slots`, `GET /slots/{id}` | views |
| `POST /slots` | body `Spec`; 201 `View`, 409 duplicate, 422 invalid, 503 admission refused (body: `error`, `check`, `need`, `have`) |
| `DELETE /slots/{id}[?force=1]` | graceful drain, 204; 409 when leased |
| `POST /slots/{id}/restart` | `{"reason"}`, 202 |
| `POST /slots/{id}/lease`, `DELETE ...` | `{"owner","ttlSeconds"}`; 409 with `holder`. Over HTTP the release is advisory (no token) |

CLI: `d2fleet serve [-addr 127.0.0.1:8431] [-data DIR] [-token-file F]`, `d2fleet add CHAR [-class -role -profile -build -difficulty -hc]`,
`remove ID [-force]`, `list`, `status`, `restart ID [reason]` (`-url`/`$D2FLEET_URL`, default `http://127.0.0.1:8431/api/v1/fleet-admin`, `$D2FLEET_TOKEN`). `serve` needs memory
access for the OOG steps (ptrace rights or the subreaper design, docs/memfd.md and docs/launch.md); starting, stopping and
admission need nothing special.

## Party (internal/party)

Roles (`Leader, Killer, Teleporter, Mule, Follower, Solo`) with `PolicyFor(role)` (follows, assists, fights, picks up loot, can host).
Sizes 2 to 4 (`New` refuses others, Solo members, two leaders, a party with nobody who fights). Interfaces, all faked in tests:
`Driver{Create, Join, Leave}` (OOG; the real one is `party.FlowDriver` over `launch.Flow.HostGame/JoinGame/LeaveGame`, VERIFIED LIVE on i1+i2, see below),
`FollowLeader{Follow(leader), Stop}` and `Assist{Assist(leader), Stop}` (real implementations later: nav and combat).

### State machine
Party: `idle -> forming -> running -> (handoff -> running) -> disbanded`. Member: `out, in, dead, left, failed`.

| Call | Effect |
|---|---|
| `Start` | host = best of leader, mule, teleporter, killer, follower (the host's exit ends the game for everybody, V multiplayer.md, so the most stable one hosts). Host `Create`s, then joiners `Join` one by one (stagger 2 s, 3 tries, backoff 3 s doubling). A member that cannot join is `failed`; nobody joining makes the host leave and the party stay `idle` with an error |
| `Leave(id)` | member leaves via OOG and is not rejoined; the host leaving triggers a handoff (below), or a disband with fewer than 2 left |
| `MemberDied` / `MemberRevived` | member stays in the game (the bot revives itself in town), state `dead` but still counted for monster scaling; followers stop following a dead leader and resume on revive |
| `MemberLost(id)` | game or connection gone. Non-host: becomes `out`, `Rejoin(id)` brings it back (also accepts `left` and `failed`). Host: handoff |
| handoff | everybody still in leaves, old game name is revoked, next host (never the lost one) creates game `name+1`, the others join it, behaviours restart. The lost host comes back later with `Rejoin` into the new game |
| `Disband` | all leave, game revoked |

Followers follow the leader if it is in the game, else the host. Events: `KindLifecycle`, `Source "party"`, states `party.creating|running|join_failed|member_died|member_revived|member_lost|rejoined|handoff|left|disbanded`.

### Protocol for sharing the game name and password
Bots never use game chat (V multiplayer.md: chat is for humans). In-process `Directory` (mutex map party id to `Game{Name, Password, Host, Difficulty, MaxPlayers}`):
1. Orchestrator picks host, builds a fresh `Game` (`GameName(partyId, generation)`, at most 8 id chars + counter; password per generation), `Publish`es it.
2. `Driver.Create(game)` on the host; only after it returns nil (host inside, TCP 4000 bound) do members get `Driver.Join(game)`. VERIFIED LIVE: a TCP/IP game has NO name or password dialog, so `Game.Name/Password` only identify the generation (`FlowDriver.Join` refuses with `ErrStaleGame` when the Directory no longer holds that name); the only thing typed is `Game.Host` into the join IP box.
3. Handoff or end `Revoke`s the entry and publishes a new generation (new name), so a stale name never leads into a dead or foreign game.
4. Status output uses `Game.Public()` (no password). Across machines the same record would travel over the admin API (U, later). Game name and password length limits (15 chars) are U.
Host address defaults to `127.0.0.1` (all bots in one host; party traffic stays on loopback, so `network_mode: none` holds). Only one TCP host per machine can bind 4000: VERIFIED LIVE that the host prefix's wineserver binds 0.0.0.0:4000 while the host is in the game and the joiner (own prefix) binds nothing; a second simultaneous host is UNTESTED, so one party per host machine (U beyond that).

### Monster scaling notes (`party.ScalingFor(n)`)
Game-wide player count n: monster life and XP `x (n+1)/2` (V Arreat: 1.5x at 2, 2x at 3, 2.5x at 4), damage +6.25% per player above 1 (V D2R, S for 1.14d),
party XP share total about +34% per extra member in range (V), so each equal-level member gets about `factor*(1+0.34(n-1))/n` of a solo kill
(1.26x at n=4; arithmetic ours, U). Idle bots in town still raise n (U for 1.14d), mules at a 10 level gap get 5% shares, members must stay within
about 2 screens, CE/Death Sentry do not scale, Crushing Blow is divided by `0.5+0.5n`. Full table: docs/classes/party.md.

## Live checks for the main agent (not done, no game was touched)
1. `d2fleet serve` on the worker host against spare instances: `add` two characters (i1, i2 free), confirm Game.exe, helper `47013/47014`, Xvfb `:101/:102`, OOG login for both; `status` capacity numbers match `uptime`, `free -m`.
2. Kill one Game.exe (`pkill -f`): slot goes `restarting`, comes back with the same character, backoff visible in the event log; kill repeatedly to hit `failed`.
3. `d2fleet remove i1`: the game must show Save and Exit (OOG main menu) before SIGTERM; check the character is saved (reload it). Try `-force` with a lease.
4. Admission on the worker host under load 35-40: expect a `load` refusal; confirm the message, then relax `MaxLoadFactor`.
5. DONE (PARTYLIVE, V): `party.FlowDriver` over `launch.Flow`; host i1 created, i2 joined 127.0.0.1, both rosters list both players, 4000 bound by the host prefix only. Repeat with `oog "party i1:Navi:leader i2:Combi:killer" pstatus pplayers` via `tools/live.sh partylive oog ...` (needs both instances up and at the menus or in game).
6. PARTLY DONE (V): host Save and Exit drops the joiner to TCP/IP Options within ~5 s; `MemberLost`/`Rejoin` of the joiner worked; `Leave(host)` with two members disbands. NOT done: a three-member handoff where a new host creates the next game and the old host rejoins; measure the time.
7. Soak: 4 slots, run an hour, check CPU per instance (1.05 core claim, 4 on a 16 core box) and that stagger prevents overlapping launches.
8. Calibrate defaults: `Stagger`, `PollEvery`, join backoff, game name/password length limits (U).

## Not done / open
- Real `FollowLeader` and `Assist` (the `Driver` and the OOG TCP/IP screens are done, V; `FlowDriver.Create/Join` are slow because the join IP box needs 32 backspaces first, and one of about 8 joins sat minutes on the Loading screen, U); the fleet does not yet build parties from slot roles (the caller wires `Member{ID: slot, Driver: ...}`); no host-load guard that pauses fights; no teleporter/killer ready-arrived handshake (design in multiplayer.md, add as an orchestrator channel); no cross-container party.
- cgroup `cpu.max`/`memory.max` reading is for cgroup v2 only; per-instance `cpu.max` (D8 resource policy) is not applied.
