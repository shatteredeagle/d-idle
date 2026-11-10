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

## Pack fight: the Paladin that backs off (card packfight, 2026-11-09)

User report: the Paladin sits in a big group and clicks around without attacking, then starts. Evidence: container log of d2play-20261108, the last 60000 lines (about 3.7 h, read only). V = counted from that log.

- V: 669 kills, 421 by `Attack`, 248 by `Smite`. maxMana 30, `mp 0` and no mana potions in the belt for the whole window (poor: gold 26 to 36).
- V: 376 `out of mana: backing off` lines (375 `not safe to stand`) in 36 runs, 18 of them ended in `back-off cap, melee` after 8 steps, 39 s on average from the first step, at 14 to 120 hostiles (median about 77). The other 18 ended another way (not classified). 70 `stop casting` waits besides.
- Cause 1 (bug, fixed): `resource.AttackChoice` let only a weak pack, at most `meleeMaxPack` (3) hostiles and danger below 2, take the basic attack once Smite (a mana spell, cost above the pool) was unaffordable; everything else was `Retreat` (walk away, "clicking around") until `backoffMaxSteps` (8) steps passed. The Paladin's profile has the basic attack as its normal weapon, so the rule was meant for casters. Now `Input.MeleeClass` (from the combat profile's Melee block, data, not a class check) swings at once at danger below 3 and life at or above `meleeMinLifePct` (50). Test `TestMeleeClassSwingsInBigPackWithoutMana` (Paladin swings, Sorceress still backs off). Also `resSkills` now always offers the basic attack to a melee profile (before: only when the left skill was Attack or the pack weak).
- Cause 2 (by design, not changed): `combat crowd: stepping away` 81 times (crowd of 11 to 14, life 35 to 46 %) and `abort fight: flee/retreat` 83 times at low life with no potion: the poverty and crowd rules, driven by the empty belt, not by stance. Cautious/Balanced/Aggressive do not enter this path.
- Cause 3 (not changed, U): `swing held: the click would walk into a cave mouth` 126 times (swing suppressed near a cave mouth).
- Gear: no rule in internal/combat, resource or bot reads item durability as a reason to avoid melee (grep: durability appears only in the town repair code). U whether zero damage slowed the kills.
- Not live verified: the fix is tested offline only.

## Pack fight, second pass (card packfight2, 2026-11-09 evening)

Logs since the first fix (about 5 h, build 4273099 and later). V = counted in the container log.

- V: the first fix did not stop the back-offs: 665 `backing off` lines, 570 `back-off cap, melee`, mostly at 80 to 160 of about 176 life. Cause of that is U (inputs were not logged); the line now carries `meleeClass`, `life`, `danger`, `hostiles`, `hasMelee`, and every fight ends with a `fight.tick` line (ticks, attacks, moves, backoffs, maxHostiles, firstAttackSec, killsPerMin, retreat reason, lastNoCast). Not yet seen live: a later deploy by another agent replaced the build before it fought.
- V by log + code: `combat no attack skill bound` 397 times and `ignoring target` 398 times in one hour (23:00 UTC), `left 97` (Smite), `known` = Might, Prayer, Sacrifice, Smite (no Attack). Cause: `gearFilter` drops Smite when no shield is worn (gear.json: Smite needs a shield, V), and the skill dump does not list the basic attack, so the attack list was empty and every target was ignored (the hero walks around the pack). Fixed in `attackSkills`: the profile's `Attack` entry is synthesised when the dump omits it, and a swinger backend (d2d) no longer needs popup list rows to use a skill that is not on the left hand (the fight loop swings by skill id). Tests `TestAttackStaysWhenGearRulesOutTheBuildSkill` (Paladin, Barbarian; needs the gear tables) and `TestSwingerNeedsNoListRowsForOffHandSkill`.
- Not changed, U: why the shield was missing (broken or unequipped; poverty/town agents own that).
- After Zeal (clvl 12): 10 kills in 29 s in one fight, `killsPerMin` 16 in the health line (V, 01:35 UTC).
