> CORRECTION (2026-10-28, verified by the user seeing the error dialog): a second Game.exe in the SAME Wine prefix fails with "Only one copy of Diablo II may run at a time". The guard is per wineserver, so use ONE PREFIX PER INSTANCE with a shared game directory (symlink) and one helper per prefix. Where this document says "one prefix, N Game.exe", read it as one prefix each.

# Design now: multi-instance readiness checklist for wave 2 (agent NET)

Principle: N bots = N copies of everything that was singular. Make "which game?" an explicit value passed down, never an ambient global. Context and reasoning: `multiplayer.md`. Tags V/S/R as there.

Cost rule of thumb: each item below costs minutes when written into a new package signature; retrofitting after nav/bot/goals are built touches every call site, test fixture and the helper protocol (days, with regressions).

## 0. Vocabulary

- `InstanceID` (string, e.g. "i1"): stable key of one running game plus its bot. Appears in logs, store paths, API routes, events.
- `Instance`: ID + character name + prefix config + process (pid, base, `mem.Source`) + window (hwnd) + `input.Controller` + runner + Tuning + Role.
- `Fleet`: owns all Instances; `Party`: orchestrator that assigns roles and coordinates runners across Instances.

## 1. Cross-cutting rules (apply to every package)

1. No package-level mutable state that is game-specific (current pid, hwnd, player pointer, caches, "the" character). Read-only static tables (`data`) are fine and shared.
2. Constructors take their dependencies (`mem.Source`, `input.Controller`, a logger carrying the InstanceID); nothing reaches for a default address or title.
3. Every log line, event, error and stats record carries `InstanceID` (use a `slog` logger with an `inst` attribute, or a field on `Event`).
4. Paths derive from the instance: `<state>/instances/<id>/{goals.json,logs/,snapshots/}`. Never write to a fixed path.
5. One character, one live instance: enforced in `Fleet.Start` (R: the game locks/keeps its `.d2s`; two clients on one character would corrupt it).
6. Tests: every layer already runs offline on `mem.Fake`; add one test per package that runs TWO instances side by side (two Fakes, no shared state) to catch globals.

## 2. Per package

### internal/mem
- Keep `Source` as is (V source.go: it is already a value, per-pid by construction). Add `mem.Proc` identity: `type Proc struct{ Pid int; Base uint32 }` so a `Source` can be created per instance, plus `Close()`.
- Do not add an `instance` parameter; instance identity lives above this layer.
- Cost now: ~0. Later: ~0 (already right).

### internal/d2 and internal/d2map
- All readers keep taking a `mem.Source` (V PLAN contracts). Add the check to review: no global singleton readers or `sync.Once` caches keyed on nothing. `d2map` per-level caches belong to a `World` value created per instance (`d2map.NewTracker(src)` already looks like that, V tracker.go existing; confirm no package var).
- Add to `d2`: `Roster(src) ([]RosterEntry, error)` reading RVA 0x3BB5C0 (V docs 2.5): `Name, UnitID, Class, Level, LevelID, X, Y, PartyID, PartyFlags` (PartyID/PartyFlags offsets UNVERIFIED, test T6; keep them behind a verified-flag like the OOG table). Constants: NoParty 65535, flags InParty 1, Accept 2, Invited 4 (V Kolbot SimpleParty.js:7-14).
- Add `d2.Hwnd(src)` only if the HWND global is found (D2GFX_GetHwnd RVA 0xF59A0 is a function, V D2Ptrs.h:382; the data global is unknown). Optional; the mouse handshake binds without it.
- Cost now: low. Later: low.

### cmd/d2input (in-prefix helper) and internal/input
- Protocol change (the important one). Today: fixed `127.0.0.1:47012`, window by `FindWindowW("Diablo II")` (V main.go:27,47-49). New:
  - Helper serves ONE port per prefix and many windows. Every command takes a target: `hwnd=0x1234 mousemove 400 300` (or a leading `@<hwnd>` token; keep old untargeted form working while exactly one window exists, so wave 1 still runs).
  - New command `list`: `EnumWindows`, match class/title "Diablo II", reply one line per window: `hwnd winpid w h visible`. Uses `GetWindowThreadProcessId` (WINDOWS pid; do not equate to the Linux pid, `multiplayer.md` section 2).
  - Per-command errors: `err no such window` when the hwnd is gone (game closed); never fall back to "any window" once a target is given.
- `input.Controller` gains `Window` (hwnd) at construction: `input.New(addr string, hwnd uint64, opts ...) (*Controller, error)`. Reconnect logic keeps the hwnd. `input.Discover(addr) ([]WindowInfo, error)` wraps `list`.
- `input.Bind(ctrl, src mem.Source) error`: the self-verifying handshake: post `mousemove` to hwnd at a random in-client point, read MouseX/MouseY (V RVA 0x3A6AB0/0x3A6AAC) from `src`, require equality, else `ErrWrongWindow`. Run on startup and after any reconnect. This is how a Linux pid is tied to an hwnd without any pid translation.
- Calibrated transforms (WorldToScreen) are per-instance state on the Controller (window size may differ); no package-level calibration variables.
- Add `char` (WM_CHAR/text) and `chat` helpers later; design the command as `type <text>` so chat and name entry share it (already wanted: OOG checklist 5).
- Cost now: small (a parameter and one new command). Later: every call site and every recorded test fixture changes.

### internal/launch
- `Config` becomes per-instance capable: add `InstanceID`, `Char string`, `Args []string` (per-instance extra args), keep `PrefixDir` shared by default. `HelperAddr` stays prefix-level (one helper, many windows). Add `Config.Instance(id, char) Config` copy-with-override.
- `Game.Launch` returns a `Game` per instance; launch serially with a stagger; bind = "exactly one new Game.exe pid in /proc (use `FindImage`, V proc.go) and exactly one new hwnd in helper `list`", then `input.Bind` proves it. Reject if the counts are not one.
- `Helper.Ensure` is already "adopt a listening helper else start"; it must be idempotent and shared among instances (refcount `Stop`). Subreaper is set once per app (V launch.md section 1); `Reap()` runs for all children.
- Monitor/Supervise: one `Monitor` per instance (it is a pure state machine, V monitor.go); crash of one instance must not restart the helper if others live.
- OOG (`Flow`): take a `mem.Source` + `input.Controller` (it does already, V launch.md section 8). Add flows: `HostTCPGame(difficulty)`, `JoinTCPGame(addr)` and screens `TcpIp`, `TcpIpEnterIp`, `TcpIpUnableToConnect` (Kolbot names, V Locations.js/sdk.js:5139-5143), flagged Verified=false. Add `Flow.Mode` (Single, TCPHost, TCPJoin; Realm later) in `Flow` config, not a global.
- Hard rule retained: no Battle.net code path (AGENTS.md); the Realm mode, if ever added, is behind an explicit config flag and a private realm address.
- Cost now: moderate (a few hundred lines in the same package we are already rewriting). Later: higher, touches lifecycle assumptions.

### internal/nav
- Executor takes its own `Source`, `Controller`, `World`. No shared "current level" cache. Add a goal primitive `GoToUnit(unitID)`/`FollowUnit(unitID, distance)` built on roster/unit positions, since follower logic needs it in phase A.
- Treat other players (roster entries and player units) as dynamic obstacles/targets that are NOT hostile; combat target selection must filter by the roster so a bot never attacks party members.
- Cost: low now, medium later.

### internal/bot
- `Tuning` and `Policy` per instance (already planned as one struct, PLAN design principle); add `Role` (leader, killer, teleporter, mule, follower, solo) as data in `Policy`, and `LootPolicy`/`PartyPolicy` fields: `AcceptInvitesFrom []string` (whitelist), `Hostile bool` (default false), `LeaveOnHostile bool` (default true), `PickupOnlyOwn bool`.
- `bot.Runner` stays the same (V goals/runner.go: Start/Pause/Resume/Stop/Status/Events) but `Status`/`Event` gain `Instance InstanceID` and, in Status, `Party{ID, Members, Leader}` and `Game{Name, Host}`.
- Add party behaviors as separate small behaviors (not in combat): `PartyJoin` (open panel `P`, click invite/accept, verify via roster flags), `Follow`, `Drop(items)`, `Pickup(owner)`. All side effects through `input.Controller`.
- Messaging: `bot` receives an `orch.Channel` interface (Send/Recv of typed messages like `Ready{}`, `PortalOpen{Area}`, `Arrived{}`) injected by the orchestrator; a no-op implementation for solo play. Do NOT use game chat for bot-to-bot.
- Cost: low now (interfaces + fields), high later (retrofitting roles into combat code).

### internal/goals
- `Manager` per instance already works as is; add an `Orchestrator` (new file, same package or `internal/party`) above managers:
  ```go
  type Orchestrator interface {
      Instances() []InstanceView            // ID, Char, Role, Status
      SetRole(InstanceID, Role) error
      StartParty(PartySpec) error           // game name/host, difficulty, members, plan
      StopParty() error
      Events() <-chan Event                 // merged, each Event has Instance
  }
  type PartySpec struct { Mode GameMode; Host InstanceID; Addr string; Difficulty string; Members []Member; Plan RunPlan }
  ```
- `GameMode`: `solo`, `tcp-host`, `tcp-join`, later `realm`. Goal types gain an optional `Instance` or `Role` target, and a goal may be `party`-scoped (e.g. "Baal runs with roles T+K").
- `Store` path per instance (rule 4); stats keyed by (Instance, Goal). The Manager's persistence and `Snapshot` get an `Instance` field. Keep the single-bot dashboard working by treating a lone instance as a fleet of one.
- Settings gain a `Fleet` section: list of `{id, char, role, prefix?}`; `Configure(Settings)` stays per runner.
- Cost: small now (struct fields, one extra layer that defaults to fleet of one); large later (store migration, API shape change).

### internal/api and web/
- Routes by instance from the start: `/api/instances`, `/api/instances/{id}/status`, `/api/instances/{id}/events` (SSE), plus `/api/party`. Keep the current routes as aliases to the single/first instance (V docs/api.md describes the current ones) to avoid breaking the dashboard.
- Dashboard: instance switcher + a party strip (members, roles, who hosts, who is where). A global kill button and an "only whitelisted humans" indicator.
- Cost: low now, medium later (API versioning).

### cmd/d2bot
- `d2bot run --fleet fleet.json`: builds Fleet, launches Instances, wires Orchestrator and API. Single character `d2bot run --char Testy` equals a fleet of one.
- One subreaper and one process for all instances.

## 3. Minimal decision list to commit in wave 2 (each a one-line decision)

| # | Decision | Where | Cost now | Cost later |
|---|---|---|---|---|
| 1 | `InstanceID` in logs, events, paths, API | all | hours | days (every call site) |
| 2 | Helper commands target an hwnd; `list` command; server one-per-prefix | cmd/d2input, input | ~half a day | protocol break plus all fixtures |
| 3 | `input.Controller` constructed with hwnd; `Bind` handshake via MouseX/Y | input | ~half a day | days; wrong-window bugs are silent and dangerous |
| 4 | `mem.Source` per pid only; no shared state | mem, d2, d2map | 0 | 0 |
| 5 | Per-instance `Tuning`/`Policy`, with `Role` and party policy fields | bot | hours | days (combat/loot assumptions) |
| 6 | `Fleet` + `Orchestrator` interfaces with fleet-of-one default | goals, cmd | ~1 day | week (refactor Manager singleton) |
| 7 | Launch serially, bind by new-pid and new-hwnd, handshake verifies | launch | ~1 day | rework of lifecycle |
| 8 | `Flow.Mode` and TCP/IP host/join screens (unverified table entries) | launch/oog | ~half a day | low |
| 9 | Roster reader with party fields (flagged unverified) | d2 | hours | low |
| 10 | `orch.Channel` interface for bot-to-bot messages; chat is human-facing only | bot, goals | hours | medium |
| 11 | Hard rules in code: no Battle.net path; non-hostile default; whitelist for invites | bot, launch | hours | n/a (safety) |
| 12 | Config: character, prefix and port per instance; `fleet.json` | launch, cmd | hours | low |

## 4. Do not do yet
- No packet injection, no D2DX for extra instances, no realm/PvPGN code, no hardcore-specific party logic. No attempt to make several instances share one process address space.

## 5. Open items for other agents (not editable from here)
- INPUT agent: helper `list` and hwnd-targeted commands, `type` command (shared with OOG name entry).
- LAUNCH agent: `Config` per-instance copy, serial launch with binding, TCP/IP screens in `oog_table.go`.
- MEM agent: `Roster` reader and the unit table filter for players.
- GOALS agent: Orchestrator and `Instance` fields in Status/Event/Snapshot; fleet-of-one default.
