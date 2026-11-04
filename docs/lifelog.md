# Life log fields

Why: the route-floor design (docs/floormeasure.md) needs max life, life and potion state at zone entry, at each retreat and at each death. Before this change no log carried max life. Logging only; nothing here feeds a decision (V: unit tests in internal/bot/lifelog_test.go; the full `internal/bot` short suite passes unchanged).

## Lines (slog, text or JSON)

| msg | writer | keys | status |
|---|---|---|---|
| `minute` | `minuteLine`, internal/bot/observe.go | existing keys kept; new `life`, `maxLife`, `mp`, `maxMana` | V (test) |
| `zone entry` | `zoneEntry`, internal/bot/lifelog.go, on every area change (and the first snapshot) | `area clvl life=<n>/<n> heal=<n> mana=<n>/<n> tome=<n> merc=<alive\|dead\|none>` | V (test) |
| `safety decision` | `carryOut`, internal/bot/loop.go, for every non-drink safety action (chicken, town-portal, crowd-retreat, crowd-abandon) | `action area life maxLife heal beltMP rejuv tome` | V (test of fields) |
| `character died` | `recordDeath`, internal/bot/observe.go | existing keys + `life maxLife heal beltMP rejuv tome` (kit from the last living snapshot; `life` is 0) | U (no test of this line) |

Notes:
- The minute key `mana` already meant belt mana potions, so the player's mana is `mp`/`maxMana`; `mana=<n>/<n>` only appears on `zone entry`.
- `heal` = belt + pack healing potions; `tome` = town portals in the kit (Kit.TownPortals); `merc` comes from `d2.FindMerc` (none = no merc unit listed, which is also what a hired-but-away merc shows; S).
- A chicken decided inside `combat.Fighter.Fight` (not through the bot loop) is not logged by `carryOut`; only decisions that reach the bot loop are covered (U: how often that is). A later card can add the same line in internal/combat.
- Classes and skills are data; nothing is class specific.

## Who parses them

`internal/d2dsoakstat` `ParseLife` / `LifeText`, printed by `cmd/d2dsoak report` as the "life per area" table: entries, min life percent (lowest life/maxLife over zone entry and safety lines; the death line has life 0 and is not used), chickens, deaths. Old logs without these lines print `n/a`. `minuteFromKV` ignores the new minute keys.
