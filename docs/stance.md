# Fight stance (cards stancefix, stancelever)

> status: built and tested offline; NOT run in a live game. Tags: V = proven by a test in this tree or by a log line, U = unverified.

## Why (user feedback 2026-11-08, container d2play-20261108, Paladin, realtime)

"The paladin isn't fighting enemies." The container log of that session (6 minutes, read only):

- 15:12:35 to 15:16:44 (4 min 10 s): the hero roamed Blood Moor over 160+ subtiles with `hostileNear 0` in every status line and 0 kills; the first kill came at 15:16:45 (`killtime` total 282.5 s, walk 92.8, wait 189.7). After that 7 kills in 37 s, 5 approaches (V: log lines). The notice radius `bot.clearRadius` is 30 subtiles (6 tiles) for every class; the roam (internal/nav, not changed here) walks past anything farther away. U: whether monsters stood farther than 6 tiles from the path.
- `clear killed nothing; moving on remaining=0` twice (14:59 at 269,36 and 15:15 at 215,31) with `combat fight kills=0 sec=0`: the sweep saw a hostile, the fighter found nobody in scope, and the sweep then ignores hostiles for 90 s or 40 subtiles (futile window). V by code: `hostileNear` measured a square (`max(|dx|,|dy|)`), `combat.inRange` a circle (Euclid), so a monster in the corner of the square (25,25 at radius 30) counted but was refused. Fixed in `internal/bot/observe.go` (test `TestHostileNearIsACircleLikeTheFighterScope`). U: that these two were corner cases (the log has no monster positions).
- `combat no bound attack skill, swinging the basic attack known:0 keys:0` at clvl 1 is expected (no skill point spent yet), not a bug.

## The stance row

Row `stance` ("How boldly the hero fights") on the policy sheet (`internal/decide/policy_chores.json`, `GET/POST /api/policy`). Options Cautious, Balanced (the row default, changes nothing, today's behaviour), Aggressive. Each option has `classKnobs`: knob values per catalogue class (a missing class is an error naming it, `PolicyOption.KnobsFor`). Values (subtiles, percent):

| knob | Balanced | Cautious paladin / sorceress | Aggressive paladin / sorceress |
|---|---|---|---|
| `bot.clearRadius` (how far a monster is noticed and fought) | 30 | 20 / 24 | 40 / 36 |
| `combat.drinkLifePct` | 55 | 65 / 65 | 50 / 50 |
| `combat.chickenPct` (leave the game) | 25 | 35 / 35 | 22 / 20 |
| `combat.championEarlyPct` | 10 | 20 / 20 | 8 / 5 |
| `combat.townPotionMin` | 2 | 3 / 3 | 1 / 1 |
| `combat.maxApproaches` (chase length) | 12 | 8 / 8 | 18 / 16 |
| `combat.kiteLifePct` (casters back away below this) | 70 | 85 / 85 | 70 / 55 |
| `combat.crowdAbandonN` | 15 | 10 / 10 | 18 / 20 |

`bot.clearRadius` max raised from 30 to 45 (Aggressive; survival at 40 U). Not in the table: how far to roam to hunt (internal/nav, other agent). The Paladin starts on Aggressive (`plandata/chores-defaults.json` `start`, also `docs/plan/data/`); the Sorceress on Balanced. A named run (`D2BOT_RUN`) starts nobody on a stance (`PolicySheet.SetStarts`), so gates and soaks keep today's knobs. The start is applied when the hero's class becomes known (`Hub.OnClass` -> `PolicySheet.EnsureStart`), not when the page is first opened. The choice is a `policy` line of the session record and a "Policy changes" card on the return screen with the plain words of the option (`says`). A Pact may set the row later through `SetByPact` (note only, nothing added).
