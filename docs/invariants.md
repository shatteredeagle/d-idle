# Invariant monitor (knob `bot.invariants`)

> status: current (card invmon, 2026-11-25). Logging only. Source: P1 of `docs/research/fable-logic-review-2026-11-25.md` (section 3.1, 3.3).

## What it does
- Once per tick `Runner.invTick` (called from `step` right after the poll) builds an `invFacts` value from the snapshot and loop state and runs the table `invRows` (`internal/bot/invariants.go`). A hit logs ONE WARN line `invariant <name>` with `detail=...`, rate limited per name (60 s, `GapSec` in the row), and counts per name.
- The minute line gains `inv=name=n,name=n` (hits since the last minute line); nothing when none, so the default line is unchanged. `Runner.InvariantHits()` returns lifetime counts per name.
- Nothing is acted on. Knob `bot.invariants` 1 (default) logs, 0 is off.
- Thresholds are rows of the table (Secs, Count, Pct). The per-hero numbers (drink line, chicken line, kit minimum, belt) come from the facts and the tuning, so no row names a class or build.

## Rows
| name | fires when |
|---|---|
| `potion-not-drunk` | in the field under the drink line, healing potion in the belt, none drunk for 8 s |
| `guard-stall` | no log line for 60 s while 2 or more hostiles are near (soft, diagnostic: `stallguard.go` logs one `stall: goroutine stack` per 5 min with the bot goroutines' stacks, the action name and its age; the minute line carries `stalls`) |
| `no-earn` | in a clearing, traveling or resupplying state for 5 min with no change of xp, kills or gold, not paused or parked (soft; area changes do not count) |
| `fight-stall` | a Fighting action older than 120 s with no kill, xp or target life change for 60 s (soft): the guard breaks the action and keeps the target out for 5 min (`stallguard.go`, `fightStallCheck`) |
| `stuck-in-pack` | in the field with 5 or more hostiles near and life under 60% for 8 s (a crowd fight that neither kills nor leaves) |
| `portal-not-cast` | in the field at or under the chicken line, hostiles near, a town portal scroll, same area after 6 s |
| `field-entry-hurt` | first tick outside town with life under 50 percent |
| `field-entry-no-kit` | first tick outside town while `kit.missing()` (portal, healing, mana minimum) is not empty |
| `resupply-in-field` | state `resupplying` outside town for 10 s |
| `refusal-as-failure` | a refusal (`not ready`, `errAreaGated`) reached `fail` |
| `refusal-storm` | 10 refusals (`noteStepRefused`) with no kill, xp or area change |
| `life-zero-alive` | alive (not dying) with life 0 for 3 s |
| `xp-frozen` | in the field, not paused or parked, no xp change for 30 min |
| `mode-no-exit` | the same state and step for 20 min with none of xp, kills, gold or area moving (a levelling or farming mode with rising xp never fires; a town chore run, state resupplying, is capped at 10 min), not paused, parked or idle. Per-state rules: `invModes` in `internal/bot/invariants.go` |
| `corpse-unresolved` | `corpseTown` pending for 15 min |
| `town-thrash` | 17 field entries within an hour |
| `arbiter-starved` | an arbiter intent outranked for 600 s or more (soft; only with the arbiter knobs on) |
| `aura-off` | soft: in the field with an aura skill learned and the d2d state line reading `aura=0` for 10 s (V live: a pick sets it at once, docs/probes/paladin-aura-live-2026-11-26.md) |
| `summon-missing` | soft: in the field, hostiles near, a summon skill learned (skills.txt `summon` column, any class) with its mana in hand, and the d2d state line reading `pets=0` for 10 s. The minute line carries `minions=N` (S: `pets=` documented in docs/d2d-backend.md, never read on a live line, so the row stays soft: U whether the count includes the mercenary) |
| `economy-stall` | purse and stash gold under 100 for 2 h while kills continue |

Not yet rows (need the reflex lane, P3): safety latency (verdict to potion) and cancellation latency.

## Hard counters (card autorevert1)
Rows with `Hard: true` (potion-not-drunk, life-zero-alive, refusal-storm, xp-frozen, mode-no-exit, corpse-unresolved) are what the flip watcher reverts on. `GET /api/explain` and `d2bot ops status --why` carry `invariants`: per row `name`, `hard`, `count` (lifetime of the process), `since` (first tick of this process, so a restart shows as a new since), `first`, `last`. V: `go test -run InvariantCounters ./internal/bot`. Tool and rule: docs/process/dark-launch.md.

## Oracle use
`Deps.OnInvariant(name, detail)` is called for every logged hit. A lockstep or Proving Ground runner sets it (or reads `InvariantHits()` at the end) and fails the run when anything fired. The hook point is in place; wiring it into `cmd/d2scenario` is a follow-up in the ops-side runner (U: not run).

## Tags
- V: unit tests `go test -run 'Invariant|InvTick' ./internal/bot` (two heroes with different maxLife, drink and chicken lines, kit lists).
- U: threshold values are the review's first numbers (section 7.1) and have not been tuned on live logs; `resupply-in-field` may fire on the portal-back leg of a trip (state is resupplying until the portal is taken). Read the counts for a few days before trusting a row.

False-positive review of the hard rows (2026-11-26, S: from reading the code, no live run):
- `potion-not-drunk`: a drink is counted from the potion events. While an earlier healing potion still works, the fighter holds a same or lower tier drink above the emergency line (`combat.potionEmergencyPct`, `healAllowed`), so a legitimate skip can last up to the effect time and the row needs 8 s. Risk is small; raise `Secs` or read `healNow` into the facts if it shows up.
- `life-zero-alive`: dead is read from the player flags; a hero at life 0 for 3 s before the flag flips would hit. Not seen.
- `refusal-storm`: a hero parked or paused no longer counts refusals (fixed here); idle in town on a hold without being marked parked still counts one per loop and fires after ten. Mark every deliberate hold as parked.
- `xp-frozen`: parked no longer counts (fixed here). Town holds (Andariel hold) are outside the field and reset the stopwatch.
- `corpse-unresolved`: fires after 15 min when the corpse run is queued behind a closed guard or a gold wait (recovery farm). That is a real stall of the corpse, but it can be a deliberate wait.
