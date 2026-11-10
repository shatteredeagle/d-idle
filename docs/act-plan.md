# Act plan: per-class recommended levels (TASK-250, card actplan)

> status: current. Replaces the hard-coded "level up to 17" Andariel hold (ADR 0011 interim 15 / 17).

## What changed
- The Andariel level gate is a per-class DATA row: `internal/plandata/boss_levels.go`, `plandata.BossLevels(gate, class)`. Each row has `Recommended` and `RecommendedMerc` (the level a careful player has) and `Floor` and `FloorMerc` (the lowest level the campaign starts the gate at), each without and with a living merc.
- Below the floor the gate still holds (a level step runs). Between floor and recommended the gate PASSES and the log carries `boss: andariel gate pass clvl=N kit=ok warning: going on below the recommended level X (clvl N, merc=none)`. At or over the recommendation there is no warning. Danger and chicken rules, the gear gate, the kit gate (potions, TP, antidotes), the route guards and the unattended hold (`boss.andarielHold`, ADR 0011) all still apply: only the level part is softened.
- The knobs `boss.andarielMinLevel` and `boss.andarielMinLevelMerc` are now overrides of the class floor (default 0 = the class row). `fillAndariel` (internal/bot/andarielfloor.go) resolves row and knob into the task World.
- An unknown class (or gate) is an error naming it (`plandata.BossLevels`); the bot logs it once and falls back to the old single-number defaults (17 / 15), never to another class's row.

## Data (EXP/S: nothing measured on the bot)
| class | recommended | with merc | floor | with merc |
|---|---|---|---|---|
| paladin | 17 | 15 | 16 | 14 |
| barbarian | 17 | 15 | 16 | 14 |
| sorceress | 18 | 16 | 17 | 15 |
| necromancer, druid, assassin | 18 | 16 | 17 | 15 |
| amazon | 16 | 14 | 15 | 13 |

Source: the Andariel readiness rows of `docs/sme/class-<class>.md` ("clvl a-b": merc = a, no merc = b; S, a veteran's table) and the SME consult of `docs/research/project-review-2026-11-10.md` section 4.4 (human Paladin with the Rogue at clvl 14 to 18; Andariel 14 with merc, 16 without; EXP). Paladin 15 / 17 is the user's starting data. Floor = recommended minus 1 (the SME's "not overcautious" numbers for the Paladin; the other classes copy the rule, U). `docs/sme/pace-and-formulas.md` (agent/smepace) was not in the tree: re-check the Sorceress row when it lands.

## Build-specific part
Only the data: one row per class, keyed by the lowercased `d2.Class` name. The gate code (`task.AndarielWarn`, `fillAndariel`) has no class literal. Builds of one class share its row (U: a Hammerdin and a Zealot would differ little here).

## Not done (open)
- Only the Andariel gate is data-driven. Blood Raven (class floors in `boss.br*` knobs), Barracks (8), Tower (14) and the Monastery route guard (`route.*` knobs, ADR 0011) keep their numbers; the table is keyed by gate so they can move in. The Andariel unattended hold and the route-guard floors are separate walls and unchanged.
- "Leave a zone six levels under the hero" and "wizard marks unproven classes" (the other items of the card line) are not part of this change.
- No live run (UNVERIFIED on the game; offline tests with fixtures only).
