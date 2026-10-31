# Wave 2: contracts between agents

Everything wave 1 produced is merged on master: `mem` (Source, Proc, Snapshot), `d2` (Reader, Snapshot, Player, Unit, Inventory, UI), `d2map` (Tracker, Grid, LevelMap),
`input` (Controller, Game, Fake, Reader, OOG), `data` (DB: monsters, levels, skills, items), `launch` (Config, Game, Helper, Monitor, OOG flow),
`goals` (Runner, Event, Manager), `api`, `env` (Env, Tuning). Read docs/mem.md, map.md, input.md, launch.md, data.md, api.md, display.md (if present) and docs/prior-art/*, docs/plan/*.

Shared conventions
- World coordinates are `input.Point` (subtiles). Convert from d2map/d2 types inside your package.
- Every behavior is a struct with `New(e *env.Env, ...)`, takes `ctx context.Context` first, never sleeps with `time.Sleep` (use `e.Clock`), never touches globals, logs through `e.Log`, reads knobs from `e.Tune`, emits events with `e.Emit`.
- Define the SMALL interfaces you consume in your own package (consumer-side), with a test fake, so you can work before the others land. The method sets below are the agreed contracts; an implementation may offer more, but must provide these.
- Tuning: add your knobs to your own `internal/env/tuning_<domain>.go` file (fields, defaults, Knob metadata with the range in which the bot still works). No magic numbers in behavior code.
- Offline first: develop and test against `mem.Fake`/recorded fixtures and `input.Fake`; use the live game only to verify. Live runs go through `tools/live.sh` and the game lock. Read `/tmp/d2lod-live.notice` on the worker host before a live run (`ssh <user>@worker.example cat /tmp/d2lod-live.notice`): if it exists, the OOG agent is restarting the game; do offline work and retry later.
- The throwaway character may die. Softcore rule: try to retrieve the corpse once, otherwise carry on. If the game process disappears, stop and report; do not restart it yourself.

Contracts (consumer-side interfaces; implementers in parentheses)
```go
// Mover walks the character. (nav.Navigator)
type Mover interface {
	GoTo(ctx context.Context, p input.Point) error           // path on the current level around obstacles; stuck recovery; returns when within tolerance
	GoToUnit(ctx context.Context, unitID uint32) error
	ExitTo(ctx context.Context, levelNo int) error            // leave through the border gap / exit tile into the adjacent level
	TakeWaypoint(ctx context.Context, levelNo int) error      // open the waypoint panel and select the area
	Explore(ctx context.Context, done func() bool) error      // frontier exploration of unknown regions until done()
	PathTo(p input.Point) ([]input.Point, error)              // pure planning on the current grid
}
// Fighter kills things and keeps the character alive. (combat.Fighter)
type Fighter interface {
	Clear(ctx context.Context, o ClearOpts) (ClearResult, error) // kill all hostile units within o.Radius of the player until none or ctx ends
	Kill(ctx context.Context, unitID uint32) error
	Buff(ctx context.Context) error                                // precast armors/buffs when missing
	Safety(snap *d2.Snapshot) SafetyAction                         // pure decision: None, DrinkLife, DrinkMana, DrinkRejuv, TownPortal, Chicken
	Act(ctx context.Context, a SafetyAction) error                 // carry it out
}
// Townsman does town chores and trips. (town.Town)
type Townsman interface {
	ToTown(ctx context.Context) error                              // portal scroll or waypoint back, then to a safe spot
	Chores(ctx context.Context, plan ChorePlan) (ChoreReport, error) // heal, potions, repair, identify, sell, stash; plan says what to do
	BackToField(ctx context.Context, levelNo int) error            // return to the area (portal or waypoint)
}
// Looter picks up and manages items. (loot.Looter)
type Looter interface {
	PickUp(ctx context.Context, o PickOpts) (PickReport, error)   // nearby ground items that pass the filter, hover-confirmed
	Evaluate(it d2.Item) Verdict                                    // keep / sell / drop / stash, from the pickit rules
}
```
Event names for `e.Emit` come from internal/goals (Kill, Drop, Death, LevelUp, AreaChange, Potion, Error).

Agents (wave 2) and ownership
| agent | owns | depends on (consumer-side fakes first) |
|---|---|---|
| BOT | internal/bot, cmd/d2bot, internal/env/tuning_bot.go | all contracts above; goals.Runner |
| NAV | internal/nav, internal/env/tuning_nav.go | env, d2map, input, data |
| COMBAT | internal/combat, internal/env/tuning_combat.go | env, d2, data, a Mover fake |
| TOWN | internal/town, internal/loot, internal/env/tuning_town.go, tuning_loot.go | env, d2, data, a Mover fake |
| OOG | internal/launch (oog*), tools/ for game restart | live game, exclusive while it works |
