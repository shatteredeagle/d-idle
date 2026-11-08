# Senses tiers 2 and 3: enemy dots, target bar and name (TASK-151, sensesenemies)

Design: `docs/research/info-unlocks.md`; ladder rows: `docs/senses-unlock.md`. View only: nothing here reaches the bot. Tags: V offline test, U not run live. Nothing in this card ran against a live game.

## Data flow

`nav.observe` -> `Env.OnObserve` -> `sensesFeedUnits` (cmd/d2bot/sensesfeed.go) -> `unitSource.Units()` (cmd/d2bot/sensesunits.go, one `d2.StateReader.Snapshot()` on the nav goroutine, same 500 ms throttle as the hero) -> `Publisher.SetUnits` (`internal/senses/units.go`: copy, keep units within `senses.UnitRadius` (40 subtiles) of the hero, nearest first, at most `senses.MaxUnits` (120)) -> `Snapshot.Unit` -> `Senses.Reply` -> `publicUnits` (`internal/watch/senses_units.go`) -> `GET /api/senses` `units` -> `web/senses_units.js` (called by senses.js after the map).

- Source is `d2.StateReader`, so game.exe (`d2.Reader`) and the d2d backend's reader work the same; offline tests use `internal/d2/fake`. No new reader was needed.
- Kinds: `monster` (hostile, alive; red), `merc` (blue), `npc` (green), `player` (reserved, nothing fills it). Classification is `combat.FriendlyWhyOf` (NPC flag, hireling AI or class table). Summons and other friendlies are dropped, dead units (mode 0 or 12) are dropped. A new area starts with no units until the next feed.
- Target: the monster under the cursor (`Snapshot.Hovered`; the bot aims the mouse at what it attacks). U that this equals the attack target in every fight.

## What each tier sends (`publicUnits`; a locked field is absent from the bytes)

| tier | per unit |
|---|---|
| 2 | `id`, `kind`, `x`, `y` only. No class, no name. |
| 3 | plus `name`, `lifePct`, `target` for the target only, and only when its class is in the Codex (`LadderUnlocker.Met`); an unmet class stays a plain dot. |
| 4 | plus `elite`, `mods` and the own name of elites whose kind the Codex holds (TASK-152, section below). |

Without a ladder (`FixedTier`) the per-class and per-kind Codex rules do not apply: every class shows once the tier is open.

## Tests (V offline)

`go test -race ./internal/senses ./internal/watch ./cmd/d2bot ./internal/unlock`: golden JSON per tier 0 to 4 (`internal/watch/testdata/senses-units-tier0..4.json`), tier 2 and 3 privacy, an unmet class stays plain, merc present (Paladin) or not (Sorceress), the four-state ladder goldens, concurrent pollers, `SetUnits` radius/order/cap/copy isolation/area change, feed throttle.

## Live checklist (ready to run, not run)

1. Does `MonsterInfo.LifePct` fall when a monster is damaged on game.exe? (the reader comment says U). Fight one Fallen, watch `/api/senses` `lifePct` on the target.
2. Is `Snapshot.Hovered` the attacked monster during a fight, or does it flicker to none between swings?
3. Are units visible through walls? The client holds units of loaded rooms; the radius 40 and the fog are the only filters. If this looks like a cheat, also require the unit's block to be in the revealed set.
4. Performance with 50 units: one extra `Snapshot()` per 500 ms on the nav goroutine; measure the tick time.
5. Is the flag 0x02 next to the boss bit (`superFlag`) really the superunique marker (TASK-152)?

## Open choice

A rare unique (no superunique name) has no Codex row, so tier 4 shows it once the tier is open (U, `LadderUnlocker.EliteSeen`).
