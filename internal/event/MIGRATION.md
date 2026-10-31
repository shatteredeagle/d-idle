# Migrating to internal/event

Status: package, Bus and goals adapters exist; nothing else imports them yet. The main agent executes this
after wave 3. Each step keeps the tree green; steps 1-4 are independent of the API shape.

Tags: V verified by reading the code at this commit, U unverified.

## Producers (V, current emitters)
| Emitter | Today | Becomes |
|---|---|---|
| `internal/bot/observe.go` (observer; kill, drop, death, level, area, potion from snapshot deltas) | `goals.Event` into `Runner.ch` via `Runner.emit` | `bus.Publish(event.New(KindX, event.SrcObserver, payload))` with real `Game`/`Run`. Fills Kill.ClassID, Death.Pos/Killer/Affixes/LifeCurve which the flat event cannot carry. |
| `internal/combat/fighter.go:120` `f.e.Emit(ev)` | `env.Env.Emit func(goals.Event)` | intent only: Chicken, Failsafe. Kill/Drop/Potion from combat/loot are duplicates of the observer (see DerivedKinds). |
| `internal/loot/pickup.go:141` Drop | `Emit(goals.Event{EvDrop})` | `ItemAcquired` (intent, SrcLoot); the observer keeps `Drop`. Removes the `disableObserve` hack. |
| `internal/town` (sim/env, townlive: Emit is a stub) | `Emit` unused in product | `TownTrip`, `Gold`, `PointsSpent`, `ItemEquipped` when tracking.md lands. |
| `internal/goals.Manager` (goal_*, control, info) | appends to own ring, stamps Seq/Time | publishes `Goal*` events with SrcManager; its ring becomes a subscriber (or the API reads a bus-backed ring). |
| `internal/telemetry/tracker.go` `emit` | own `Event{Kind,Text,Area,Value,Extra}`, kinds `levelup`, `area`, `activity`, `run` | subscriber. Kind map: levelup->level_up, area->area_change, activity->Activity (segments stay in telemetry), run->Lifecycle. `Text` is derived for display, not stored. |
| sentinel / launch (JSONL actions today, docs/sentinel.md) | own log format | `Failsafe`, `Lifecycle` events, SrcSentinel / SrcLaunch; JSONL log = a subscriber writing `json.Marshal(Event)` lines. |
| `internal/cheat` ledger | ledger only | additionally publishes `Cheat`; ledger stays source of truth. |

## Consumers (V)
| Consumer | Today | Becomes |
|---|---|---|
| `goals.Manager` (`manager.go:160` reads `Runner.Events()`) | `<-chan goals.Event` | step A: keep `Runner` interface, bot runner publishes to bus and a bridge goroutine calls `event.ToGoals` into the old channel. step B: Manager subscribes to the bus (`Subscribe("manager", 2048)`), reads `event.As[Kill]` etc. Counting of Kills/Drops/Deaths per goal is unchanged. |
| `internal/api` (`stream.go`, `handlers.go`: `/events`, SSE `log`) | serialises `goals.Event` | serialise `event.Event` (JSON already stable); `seq` is the same field name so `?since=`/Last-Event-ID keep working. Add `instance` filter for WP-SERVER fleet stream. The old flat JSON shape is lost: add a compat mapper (`ToGoals` then marshal) behind `/api/v1/events` for one release; new routes emit the envelope. |
| `internal/status` hub / display timeline | `telemetry.Event` over SSE `append` | unchanged until telemetry subscribes; then timeline items are built from bus events. |
| `cmd/d2bot/display.go`, `util.go` | construct/forward `goals.Event` | construct `event.Event`; drop the `goals` import for events. |

## Order
1. Create one `Bus` per instance in `cmd/d2bot` (instance id, run id = process start time). Bus.Publish assigns Seq: remove Seq stamping from `Runner.emit` and the Manager only after step 4.
2. Bot runner: `emit` becomes `bus.Publish(event.FromGoals(ev, SrcObserver))` (zero payload change; fields the flat event lacks stay empty). Add the bridge for the Manager (`ToGoals`). All existing tests keep passing.
3. Telemetry subscribes (`OnEvent` replaced by a bus subscription); keep its public `Event` JSON for the display until the dashboard (WP-DASH) reads the new envelope.
4. Manager subscribes directly; delete the bridge. `goals` no longer needs `Event` for runners: `goals.Runner.Events()` is replaced by an injected `Bus`.
5. Move behaviors off `env.Emit(goals.Event)` to `event.New(...)` intent events (Chicken, ItemAcquired, TownTrip, Failsafe). Delete `bot.Config.DisableObserve`; add a lint test that no package except the observer publishes a `DerivedKinds` kind.
6. API/SSE serialises the envelope; add compat route. Sentinel/launch/cheat publish their kinds; JSONL writer subscriber.
7. Delete `adapter.go` and the `goals` import from `internal/event` once `goals.Event` is gone (event must then be a leaf package that goals imports).

## Rules
- Buffers: Manager 2048 (must not lose goal-counted events; if `Dropped()>0` raise a warning event), API/SSE 256 per client, telemetry 1024, JSONL writer 4096.
- Seq is per Bus (per instance). Cross-instance ordering uses Time; the fleet stream tags `instance`.
- Adding a kind: add const, payload struct, `payloadType` case, a sample in `event_test.go` (TestPayloadForEveryKind fails otherwise). Adding a field to a payload is not a version bump; renaming/removing one or changing the envelope is (bump `Version`, readers return `ErrFutureVersion`).
- Goal type names and params never enter this package; goal.* events carry only the goal id.
