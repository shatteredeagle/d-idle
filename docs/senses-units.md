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

# Senses tier 4: elite markers with mod names (TASK-152, senseselites)

Tags: V offline test, U not run live, S secondary. Nothing here ran against a live game.

## What is drawn

Champion: gold ring. Rare unique: purple ring. Superunique: both rings and the monster's own name. Boss: a crown (the reader cannot tell a boss from a superunique yet, so `boss` is never sent today; U). The label is the name (own name for a superunique) and the mod names after a colon. Colours and shapes live in `web/senses_units.js`.

## Where each part lives

- Kind: `unitSource.elite` (cmd/d2bot/sensesunits.go) from `MonsterInfo.Kind()`: champion (flag 0x04), unique (boss bit 0x08 without 0x04), superunique when the 0x02 bit is also set (`superFlag`, 0x0a; U). Own name from `MonsterInfo.Name` (U: set for uniques and bosses per the reader comment).
- Mod ids: `MonsterInfo.EnchantIDs()` (the one place that reads the enchant array; an unknown kind or an id above 40 gives no mods at all, so a bad read never prints wrong names). Names: data table `internal/decide/champmods.json` via `decide.ModName(id)`; an id not in the table reads `unknown mod #N`.
- The table is S: the ids are the monumod.txt row ids the combat package read from the real file (internal/combat/target.go), the names are those of the Codex champion row (`firsts_act1.json`; a test keeps both in step). `Multiple Shots` (29) is U. monumod.txt itself is not in the repo or the extracted tables of this checkout, so no test compares against it; add one that skips without tables when the extract has the file.
- Gate (spoiler rule): `watch.applyElite` (internal/watch/senses_elites.go) with `LadderUnlocker.EliteSeen`: a champion is marked once any Codex champion entry exists, a superunique once its own Codex monster entry exists, a rare unique once tier 4 is open (U, open choice). A kind not met yet stays the plain red dot of tier 2: no ring, no name, no mods, not in the bytes.

## Tests (V offline)

`go test -race ./internal/watch ./internal/decide ./cmd/d2bot -run 'Senses|ChampMods|UnitSource'`: champion with two mods, superunique with its own name, rare unique with an unknown mod id, a plain monster unmarked, tier 3 shows none of it, first-encounter matrix (champion only, superunique only, both) for a Sorceress and a Paladin, mod table checks (unique ids, tags, Codex names covered, unknown id), garbage enchant bytes give a champion without mods, golden JSON of tier 4 (`senses-units-tier4.json`).

## Not done, live checklist

1. The `champion` Codex reader is still missing (docs/codex.md): nothing emits a champion first yet, so tier 4 opens today only through the superunique alt trigger. Wire that reader first.
2. Live: does a real champion read flags 0x0c with enchant ids at +0x1C, and are they the ids of the table? Is a superunique 0x0a with a name?
3. Check a Bishibosh/Corpsefire read: `Name` filled? If not, `Title` stays empty and the marker uses the class name, which the Codex gate would not match (the marker would stay hidden, never wrong).
