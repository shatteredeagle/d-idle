# Log fields for the per-skill soak report (change logfields)

Nothing here is verified live. All of it is V offline only (unit tests with the combat sim and fixture lines). Logging only: `TestCastLogDoesNotChangeActions` runs the same fight with the logging on and off and compares cast keys, moves, casts and virtual time (V offline).

| Line (level) | Fields | Written by | Read by | Tags |
|---|---|---|---|---|
| `cast result` (debug, as the cast decision) | `skill` (data name), `made`, `cut`, `cost` (whole mana), `ticks` | `castLogStart` in `Fighter.cast` (every attack and buff cast), closed by `castLogTick` per fight tick or by the next cast | `ParseSkills` | `made`: cast mode or a mana drop of the cost seen at the fight tick rate (U live: a short cast can read false; the mana drop is S). `cut`: hit recovery seen first and no made sign (U live). `cost`: skill table cost, 256ths not logged (V). `ticks`: modelled length from castgap.go for the configured FCR (S, not measured) |
| `combat kill` (info) | new key `last_skill` | `noteKills` | `ParseSkills` | the skill cast most recently, not a hit credit (S) |
| `minute` (info) | new key `armor`: `Frozen Armor`, `Chilling Armor`, `Shiver Armor`, `none`, `?` (state array empty) | `minuteLine` via `combat.ArmorState` | `ParseSkills` | state ids 10/20/88 from d2/states.go; 20 and 88 not seen live (U). One sample per minute line, so absent seconds are about 60 s per sample (S) |
| `build` (info, once, on the first cast) | `branch` (env `D2LOD_BRANCH`), `commit` (`buildinfo.Current`, left out for `dev`) | `castLogBuild` | `ParseSkills` (old `branch`/`commit` keys on any line still work) | n/a-safe: a missing key stays n/a |

The report prints: made effect and cut as "n of m", mana spent with the per-cast mean, kills as last-skill count, armor-absent seconds estimate on buff skills, and the Lightning cast length for every skill with "Lightning" in its name (13 on d2d, 19 on game.exe at FCR 0, from the model).

Still n/a: hits per cast, kills credited per skill by hit (no log source), real measured cast ticks (the model value is printed).

Old logs parse as before; every new field prints n/a when its line is absent.

Edits outside the card's owns (small, declared overlap): `internal/combat/fighter.go` (field `clog`, one call in `cast`), `internal/combat/fight.go` (one call per tick, `last_skill` on the kill line), `internal/bot/observe.go` (`armor` on the minute line).
