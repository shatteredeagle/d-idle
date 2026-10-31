# d2d casts per kill, Charged Bolt Sorceress vs Fallen in the Blood Moor (agent PROBES-D2D, 2026-10-30)

> status: SUPERSEDED in part: this probe counted cast COMMANDS; real casts per kill are docs/d2d-casts-real.md and the gap was a probe artifact (docs/research/gap-casts-per-kill.md).

Tags: V-d2d = measured on the private lockstep fork (`d2d.lab5`, Normal, seeds 1 to 4, `--no-save`, throwaway Sorceress); U = not checked on Game.exe.
Probe: `cmd/d2dcasts` through `docs/probes/run-d2d.sh` (seeds 1 to 4 per level, 15 virtual minutes each). Raw per-kill rows: `docs/probes/d2d-casts.csv` (columns `area,seed,cb_level,clvl,monster,boss,xp,cast_cmds,mana_spent`).

## Method

Fork cheats only (logged `debug` verbs, throwaway character): clvl, `debug points 38 <n>` for Charged Bolt, `debug points 36 0` (a fresh Sorceress started with Fire Bolt 36 bound in the pilot, so it was cleared: only Charged Bolt exists), life 300 and mana 100 topped up between `Clear` calls so no cast is unaffordable and nobody dies. The bot's real `combat.Fighter.Clear` (radius 60, 30 s virtual per call) fights in the Blood Moor, wandering between calls. Every kill event (`goals.EvKill`, the same event the bot logs) records the monster name, the Charged Bolt cast commands and mana spent since the previous kill. A "cast command" is `cmd skill 38` issued by the fighter (`d2dcombat.Input.CastAt`); mana per command rises with skill level as the table's per-level cost does (1.6, 2.8, 4.1 at CB 1, 4, 8: mana spent divided by commands), so commands are about one per real cast (U: not counted from missile spawns).

## Result (V-d2d)

| Charged Bolt level (clvl) | Fallen kills | cast commands per kill, mean | median | p90 | note |
|---|---|---|---|---|---|
| 1 (clvl 2) | 183 | **2.59** (2.13 without one 86-cast outlier) | 2 | 4 | Fallen xp 18 each |
| 4 (clvl 5) | 177 | **1.18** | 1 | 2 | |
| 8 (clvl 9) | 237 | **0.97** | 1 | 2 | xp 7 each (level-gap penalty) |

Other Blood Moor monsters (same runs), mean cast commands per kill at CB 1 / 4 / 8: Quill Rat 3.27 / 2.21 / 1.81, Zombie 5.41 / 2.65 / 1.82.

Reading: a mean below 1 means one bolt (it splits into several charges that hit neighbours, and one cast often kills two Fallen of a pack) kills more than one Fallen, because the per-kill count is cast commands between kills, so multi-kills give zero to the second kill (13 to 36 % of Fallen kills show 0 commands for that reason). The number to compare against the original is therefore casts per kill as a mean over a whole fight, not per target.

Caveats: mana and life were topped up (a real early Sorceress runs dry, the bot's mana back-off is not exercised), the fighter is the bot's, so this includes its targeting and kiting; Normal only; one clvl per skill level (clvl = CB level + 1).

## XP-gain log line carries the monster name already

No patch is needed: `internal/combat/fight.go:411` logs `kill monster=<name> boss=<bool> xp=<share>` (and emits `goals.EvKill` with `Monster`, `XP`), and `internal/bot/observe.go` fills `Monster` the same way for the Game.exe path. Observed live on d2d: the names are right (Fallen, Quill Rat, Zombie). Neither file is owned by this card.
