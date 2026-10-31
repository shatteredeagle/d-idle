# Design gaps: underspecified or contradictory places, with proposed resolutions

Docs only; Go below is pseudocode for shape, not a patch. Tags (V)/(S)/(U) as in findings.md.

## D1. Process topology and ownership of the game (contradiction)
Gap: PLAN wants "no root, app is subreaper" (V docs/launch.md 1) and "an out-of-process sentinel that does not trust the bot"
(V PLAN.md Failsafes 2). A separate process cannot read a game it did not spawn without CAP_SYS_PTRACE (Yama scope 1). Today
lifecycle is split between `internal/launch`, `tools/game.sh` and the sentinel calling `game.sh` (findings F1).
Resolution:
```
d2bot --managed            (PID 1 or under tini in a container; subreaper; owns Wine)
 |- Fleet: per instance Supervisor{ Display, Prefix, Game, Helper, MemFD }
 |- per instance: Observer -> Runner (bot) -> goals.Manager
 |- Sentinel (in-process, own goroutines, reads Ticks AND raw memory via its own fd dup;
 |            acts only through launch + input lease; can kill the Runner goroutine tree)
 |- server (HTTP)
d2watchdog (tiny separate process, systemd/compose healthcheck): GET /healthz of d2bot;
            on failure restarts d2bot; reads no game memory.
```
- Trust boundary moves: the sentinel no longer protects against a crashed *process*, the external watchdog does; it protects against a
  stuck *runner*, which it can cancel. If WP-MEMFD proves fd passing, the sentinel may also be a separate process fed an fd by d2bot.
- `--attach` + setcap stays for development only and prints a warning.
- Ownership: only `launch.Supervisor` starts/stops Wine processes; everything else asks it (`Restart(ctx, id, reason)`).
- A game found running at startup that is not our descendant is adopted only with CAP_SYS_PTRACE; otherwise stopped (with Save and Exit through the helper first) and relaunched.

## D2. Event model
Gap: `goals.Event` is a flat union without version, lives in the top layer, is imported by `env` (V import graph); kinds are split
between goals and telemetry (V docs/display.md vs docs/api.md); duplicate emission is switched off by config (V docs/bot.md).
Resolution:
```go
package event
type Kind string // "kill","drop","death","level_up","area_change","potion","error","failsafe","lifecycle","cheat","goal.*"
type Event struct {
    V        int        // schema version of the envelope (1)
    Seq      uint64     // per instance, monotonic, assigned by the Bus
    Time     time.Time
    Instance string
    Run      string     // runner session id (process start)
    Game     int        // in-game session counter (each EnterGame)
    Kind     Kind
    Source   string     // "observer","combat","loot","sentinel","manager","launch"
    Data     any        // one of the typed payloads below; JSON tagged by Kind
}
type Kill struct{ Monster string; ClassID int; Boss bool; XP int64; Area int }
type Death struct{ Area int; Pos [2]int; Killer string; Affixes []string; LifeCurve []int; Difficulty int; CharLevel int }
type Failsafe struct{ Layer string; Rule string; Step int; Action string; Result string }
type Bus interface{ Publish(Event); Subscribe(name string, buf int) Sub } // never blocks Publish; per-sub drop-oldest + counter
```
Rules: the Observer derives state-change events (kill, level, area, death, potion from belt deltas) once; behaviors emit only what the
observer cannot see (intent: "chicken", "portal opened", "target skipped: immune"). The rule is a table in `event` (`DerivedKinds`), not a runtime flag.
goals.Manager consumes the bus through an adapter; API/SSE serialise `Event` directly; telemetry subscribes.

## D3. Error taxonomy
Gap: per-package sentinel errors matched by hand (findings F14).
Resolution:
```go
package fault // tiny leaf package, imported by nav/combat/town/loot/launch/bot
type Kind int
const ( Transient Kind = iota // retry same step
        Blocked             // path/target unreachable: skip target or replan
        NeedsTown           // out of potions/scrolls/space
        NeedsNewGame        // area exhausted, broken game state
        NeedsRestart        // helper/game/window gone
        Dead                // character dead: hand to recovery
        NotInGame           // menus/loading: wait or OOG
        Cancelled           // ctx: not a failure
        Fatal )             // stop the bot, alert
type Error interface{ error; Kind() Kind }
func KindOf(err error) Kind // errors.As; unknown -> Transient, logged once per error string as "unclassified"
```
Each package wraps its sentinels (`var ErrStuck = fault.New(fault.Blocked, "nav: stuck")`). The runner's ladder switches on `KindOf`.
Test: reflection over exported `Err*` vars of every behavior package asserts a Kind (same trick as the knob test, V docs/bot.md).

## D4. Configuration model
Gap: six sources, no precedence (findings F13).
Resolution: layers, lowest first: built-in defaults < `config.json` (file, strict) < environment `D2BOT_*` < flags < runtime layer
(written only by the API, persisted to `instances/<id>/runtime.json`). Per-instance overlay: `config.json` has `instances: {id: {...}}`
merged over the top level. One schema (Go struct with JSON tags) generates the docs table and the dashboard form. Knob overrides use
dotted keys only (`tuning.combat.chickenLifePct`); `PUT /settings` writes those same keys (no separate `chickenLifePct` mapping).
`GET /api/v1/instances/{id}/config/effective` returns the merged value and which layer set each key. Sentinel thresholds become a
section of the same file. `tools/instances.conf` stays a dev-tool file and is generated from the fleet config.

## D5. Per-tick observation ownership
Gap: several readers per tick (findings F7); nav owns `Tracker.Update`.
Resolution:
```go
type Tick struct {
    Seq   uint64
    At    time.Time
    Snap  *d2.Snapshot       // immutable after publish
    World d2map.State        // tracker state after Update for this tick
    Err   error              // read failure (process gone) vs StateLoading in Snap
}
type Observer interface {
    Latest() Tick                        // never blocks
    Next(ctx context.Context, after uint64) (Tick, error) // wait for a newer tick
    Map() *d2map.Tracker                 // read-only view; only the Observer calls Update
}
```
Rate: one tick per `obs.tickMs` (knob, default 50-100 ms; survival needs <= 100 ms, V prior-art 07). Behaviors that need a fresher view
after an action call `Next(ctx, seqBefore)`. Event derivation lives here (D2). Recording for WP-REPLAY taps the Observer, so every
fixture is exactly what the bot saw. nav keeps its own field/grid caches but reads `Map()`.

## D6. API and instance addressing
Gap: single-instance API; three servers (findings F5); NET's Fleet not built (V docs/net/design-now.md).
Resolution: routes `/api/v1/instances/{id}/...` and `/api/v1/fleet`; existing routes alias the default instance; one token (cookie +
bearer) for all mounted handlers; `/healthz` (process), `/readyz` (all enabled instances in a known state). Instance ids are config keys,
stable, `[a-z0-9-]{1,16}`. The goals.Manager stays per instance; a thin `fleet.Manager` aggregates. SSE: one stream per instance plus a
fleet stream with `instance` on every event (D2 envelope already carries it).

## D7. Upgrade / meta data contracts (parked layer, contract now)
Gap: Knob metadata exists (V internal/env/tuning.go) but profiles, overrides, future upgrades and cheats each change the effective Tuning
through different paths; telemetry cannot tell which Tuning produced a number.
Resolution (contract only, nothing built):
- `TuningSource{Profile string; Overrides map[string]float64; Upgrades []string; Cheats []string}` and `EffectiveTuning = f(defaults, source)`; a
  stable `TuningHash` string is attached to every run (Event.Run metadata) so XP/h and deaths/h are always comparable per hash.
- `Upgrade{ID, Track, Tier, Cost, Requires[], Effects[]{Knob, Value|Delta}}` stays pure data (V PLAN.md Decisions); validation: every Effect key exists
  in `Knobs()`, every Value inside [Min, Max], monotonic toward "better".
- A rule from COMBAT's finding (V docs/combat.md): no profile may put more than K survival knobs past their midpoint; encode as a validator, not a comment.
- Cheats feed the same `TuningSource.Cheats` and the ledger flag (V docs/cheats.md); stats exclude cheated runs by default.

## D8. Multi-instance orchestration
Gap: instances exist as dev tooling (game.sh, instances.conf, `/tmp` locks; V tools/), not as product objects; locks are flock files shared with dev tools.
Resolution:
```go
type InstanceSpec struct { ID, Character, Class string; Difficulty string; Display DisplaySpec; Prefix PrefixSpec; HelperPort int; Enabled bool; CPUQuota float64 }
type Fleet interface {
    List() []InstanceView
    Apply(ctx context.Context, specs []InstanceSpec) error  // converge: start/stop/recreate, serial with stagger (V launch.StartInstance)
    Restart(ctx context.Context, id, reason string) error
    Lease(id string) *Lease                                  // in-process ownership; replaces /tmp flock in product mode
}
```
Invariants enforced in `Apply`: unique character per save dir, unique helper port, unique display, at most one TCP/IP host per machine (V docs/net/multiplayer.md).
Dev tools keep flock files; product mode uses in-process leases and a pid file per instance under the runtime dir. Resource policy: per-instance cgroup
(container: one container per bot or `cpu.max` per instance), staggered starts, and an admission check (free cores, RAM) before starting another instance.

## D9. Input ownership and arbitration
Gap: concurrent writers to one window, no cancellation (findings F6).
Resolution:
```go
type Owner struct{ Name string; Prio int } // bot=10, sentinel=50, user/debug=90
type Lease interface{ Release() }
type Arbiter interface {
    Acquire(ctx context.Context, o Owner, timeout time.Duration) (Lease, error) // preempts lower prio; returns ErrPreempted to the holder's next call
}
type Controller interface { // every method gains ctx; Game checks ctx and lease before each helper round trip
    Click(ctx context.Context, b Button, x, y int, mods ...Mod) error
    ...
}
```
The arbiter lives in-process (one per instance, inside the Supervisor). If a separate process must send input (dev tools), it goes through the
d2bot API (`POST /instances/{id}/input` debug-only, disabled by default) instead of the helper port. Optionally the helper accepts a session token
so only the arbiter's connection may post (cheap hardening; the helper listens on 127.0.0.1 and any local process can drive it today, V docs/input.md).

## D10. Versioned snapshots and fixtures
Gap: three fixture formats without headers (findings F12).
Resolution: one envelope (WP-REPLAY) wrapping existing payloads:
```
magic "D2FX" | u16 schema | u32 headerLen | header JSON {kind: "pages"|"mapimage"|"stream"|"trace", build:{ts,size}, runner, recorder,
instance, char, class, difficulty, seed, area, createdAt, verified:[...], notes} | gzip payload (existing D2SN / d2mi / JSONL)
```
Loaders accept old files (no magic) and treat them as schema 0. CI runs `canary.Check` over every fixture; a fixture whose build differs from
the supported build is rejected. A fixture index (`testdata/INDEX.json`) lists what each one is for, so obsolete ones can be deleted.

## D11. Persistence ownership and lifecycle of state files
Gap: goals store (V docs/api.md), death log (V docs/bot.md), cheat ledger, sentinel log each pick paths and formats; no migration story except
goals' "refuse future version".
Resolution: WP-STATE `store.File[T]` for documents and `store.Log` for JSONL; every file registers `{name, version, migrations}`; a startup
`d2bot migrate --dry-run` prints planned migrations; backups of state before migrating. Layout (all under `D2BOT_DATA`):
```
config.json
instances/<id>/{runtime.json, goals.json, deaths.json, routes.json, saves/<rotation>, logs/, fixtures/}
cache/data/<schema>-<stamp>/   (extracted tables)
prefixes/<id>/                 (Wine prefixes; never on NTFS)
game/                          (shared, read-only game dir, hardlink source)
ledger/cheats.jsonl  logs/  run/ (pids, sockets)
```

## D12. Plan and task layer (state-machine explosion)
Gap: bot.Runner already has 8 states and per-state timeouts (V docs/bot.md); quests, merc, act travel and bosses would multiply them.
Resolution: keep Runner states coarse (Idle, Working, Recovering, Paused, Error); add a task layer:
```go
type Step interface {
    Name() string
    Need(w WorldView) bool          // should it run now (skip predicates, prior-art lessons A4)
    Ready(w WorldView) error        // preconditions (gear, potions, waypoint, quest state); returns a fault Kind if not
    Run(ctx context.Context, d Deps) error
    Budget() time.Duration          // per-step watchdog
}
type Plan []Step // built by the planner from goal + progress (WP-SAVES) + plan data
```
Run scripts, quests, boss fights and town chores become Steps; the runner executes plans, the watchdog and ladder stay in the runner, and every
Step is testable with fakes. This is where RESEARCH-GOALS' goal types should land.

## D13. Progress state source
Gap: quest/waypoint/difficulty-unlocked state has no source (findings F2).
Resolution: `progress.State{Quests[diff][q], Waypoints[diff][level], Unlocked[diff], MapID[diff]}` produced by (1) the save file at each
menu phase (WP-SAVES; authoritative after Save and Exit), (2) in-game deltas observed by the Observer (quest NPC dialogs, area reached,
waypoint panel rows read when opened), (3) later a verified memory reader. The planner reads only `progress.State`; the source can improve
without planner changes.

## D14. Time
Gap: `env.Clock` exists but `input.Game`, `d2.Reader` retries, launch and sentinel use real time (V grep of `time.Sleep`).
Resolution: one `clock` leaf package (Now, Sleep(ctx), After, NewTicker) injected into input, launch, sentinel, observer; a fake clock that
advances only when all registered sleepers are blocked (the bot tests' flakiness, V run here, points at mixed clocks).

## D15. Character identity and allowlists
Gap: the cheat allowlist is a hard-coded name list (V docs/cheats.md); the product will have user characters with any name.
Resolution: `CharacterPolicy{Name, Mode (softcore|hardcore), Throwaway bool, CheatsAllowed bool}` in the instance config; the cheat gate reads
`CheatsAllowed`; hardcore characters can never have cheats or save edits; the bot refuses to drive a character whose save header disagrees with the policy (mode bit, expansion bit).
