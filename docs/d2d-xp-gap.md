# d2d XP gap: where the time goes and what moved it (change xpgap, 2026-10-31)

> status: current. Tags: V = measured live on hidden d2d (lab6, lockstep, fresh uncheated Sorceress, 2 virtual hours per run, run policy on), U = inferred. Nothing was run on Game.exe. Data: the 12 soakfull logs (docs/d2d-soak-full-fix.md) for the diagnosis, new runs in `docs/probes/xpgap/` for the A/B. Tools: `tools/xpgap/{run,ab}.sh`, `tools/xpgap/an.py`.

## Diagnosis (V, 12 runs, 24 virtual hours, 4727 kills, 18.0 s per kill)
Minute lines split by the step the bot was in (kills and XP per virtual minute inside the state):

| state | share of time | kills/h | XP/h | note |
|---|---|---|---|---|
| "Recovery: farming gold" (clearing) | 54 % | 238 | 6027 | as productive as the quest step: the label is not the loss |
| quest "Level up to N" (clearing) | 17 % | 250 | 6248 | |
| going to town (walk or scroll) | 14.5 % | 168 | 4592 | |
| "Returning to the field" (walk out of town) | 14 % | 11 | 417 | pure loss, 2.4 min per trip |

Seconds per kill by gap since the previous kill: gaps <= 4 s are 50 % of the kills and 4 % of the seconds; **gaps over 30 s are 14 % of the kills and 70 % of the seconds** (walk 82 to 86 % of them). What precedes the 59 500 s of long gaps: town trips 40 % (128 gaps, mean 188 s), roaming a cleared area ("clear killed nothing") 28 %, chicken / leave-game restarts 21 % (95, mean 129 s), no-portal walks to town 5.5 %.

Economy (V): gold per kill 0.43 (piles only, 86 gold/h), income including sales about 170/h; 10 270 gold spent in 24 h: **64 town portal scrolls = 6400 gold (62 %)**, 93 healing potions 2790, 18 mana potions 1080. Gold never exceeded 412 (recovery target 900, kit 500): the character is in recovery for the whole 2 hours by construction (it starts with 0 gold). 85 town trips in 24 h (3.5 per hour): emergency (portal after a lost fight) 42, poverty sell trips 34 (mean net +23.5 gold for about 4 minutes), potion trips 24 (mean -46 gold). Each trip costs about 4 minutes (1.9 going, 0.5 shop, 2.4 back), 20 to 28 % of the time.

Area: at clvl 2 to 4 the bot farms Blood Moor: 21 XP per kill, 240 kills/h = 5200 XP/h. Cold Plains pays 32 to 35 XP per kill at clvl 4 to 5 (7700 to 9600 XP/h), Stony Field 38 (11 700, 15 minutes only). Plan enter levels: Cold Plains 5 (kit level 4), Stony Field 8.

## Top causes
1. Poverty trips: 20 to 28 % of the time goes into walking to and from town to spend about 400 gold an hour, of which 62 % buys portal scrolls the character then burns on the next emergency (portal, shop, walk, scroll again). Chicken restarts (14 % of the time) are the same disease: no potion, no scroll.
2. Roaming a spawn-poor first field (20 % of the time, U for the density share, docs/research/per-kill-time-split.md): Blood Moor holds about 70 monsters on d2d (90 on Game.exe), so after a sweep the bot walks empty ground.
Not causes: the recovery label itself (same XP/h as the quest step), casts (0.9 s), mana waits.

## Fixes (all behind knobs, default off, so Game.exe and the default d2d path are unchanged)
| knob | what | shared code (needs a G1/G2 check when the default changes) |
|---|---|---|
| `resource.recoveryXPEarly` (levels) | recovery XP pick may enter an area that many levels before its plan enter/kit level | `internal/bot/recoveryxp.go`: only when `recoveryXPPick` is on |
| `resource.povertySellMin` (items) | poor character walks to town to sell junk only with this many items | `internal/bot/goal.go` choresDue (both poverty sell branches) |
| `town.tpMinGold` (gold) | below this gold the first portal scroll is not bought and a missing scroll is no trip reason: potions first | `internal/town/shop.go`, `chores.go` (affects Game.exe if raised from 0) |
Offline tests: `TestRecoveryXPEarlyEntry`, `TestPovertySellMinKnob`, `TestTPMinGoldPotionsFirst` (needs the extracted tables; skipped without them), knob doc sync, whole-repo `go vet` and `go test -short` on the worker.

## Measured (V): before/after on one binary (`d2bot.xpgap`), knob toggled per arm, fresh Sorceress, 2 virtual hours, 0 deaths in all 24 runs
Arms interleaved per seed, 3 slots at once. Raw output: `docs/probes/xpgap/ab*.txt`. Mean +- SE over seeds, Welch p. The baseline arm (off) is 8 runs, seeds 1 to 8: XP/h 5043 +- 210 (SD 594), kills/h 197, 17.4 to 19.1 s per kill (walk 13.3 to 14.9), recovery 50 to 57 %, town 24 to 33 %. Seed 8 is the known poverty loop (3694 XP/h, 8.5 trips/h).

| arm (seeds) | knobs | XP/h | kills/h | s/kill (walk) | recovery / town share | trips/h | p (XP/h) |
|---|---|---|---|---|---|---|---|
| off (1-4) | none | 5274 +- 177 | 204 +- 2 | 17.4 (13.3) | 57 / 24 % | 3.9 | |
| on (1-4) | recoveryXPEarly 2 + povertySellMin 10 | 5268 +- 500 | 196 +- 8 | 18.4 (14.2) | 58 / 26 % | 2.8 | 0.99 |
| off (5-8) | none | 4812 +- 373 | 191 +- 16 | 19.1 (14.9) | 50 / 33 % | 4.1 | |
| sell (5-8) | povertySellMin 10 | 4988 +- 328 | 200 +- 16 | 18.1 (13.1) | 61 / 23 % | 3.3 | 0.73 |
| tp (5-8) | town.tpMinGold 150 | 4651 +- 373 | 186 +- 16 | 19.6 (15.4) | 48 / 35 % | 4.8 | 0.77 |
| norec (5-8) | resource.recoveryOn 0 | 4084 +- 285 | 168 +- 11 | 20.0 (15.4) | 0 / 41 % | 5.0 | 0.18 |

Reading: **no fix moved XP/h beyond noise** (SE is 7 to 9 % of the mean with 4 runs; an effect under about 15 % cannot be seen). Early Cold Plains entry and fewer sell trips cut the trip rate by 29 % but the town share did not fall (Cold Plains is a longer walk to town, U). The portal-scroll gate changed 1 purchase in 8 runs (scrolls are bought only once gold is over 150 anyway in this build, 3 per run here against 5 in the soakfull tree). Switching recovery off is the one visible effect and it is negative (-15 %, p 0.18): recovery mode is not the loss, it is what keeps the character fighting.

## What is still the gap, ranked next suspects
1. Walk 13 to 15 s of 18 s per kill, mostly in gaps over 30 s: trips (about 4 minutes each, 3.5 per hour), chicken restarts, roaming cleared ground. A trip fix has to shorten the trip (waypoint out of town to Cold Plains, no walk back to Blood Moor), not only count them (U: waypoint activation is not wired while leveling, BACKLOG LEVELWP).
2. Spawn-poor first field (70 vs 90 monsters, per-kill-time-split.md): the 8 to 20 % of time spent roaming an emptied Blood Moor; a bigger lever on d2d is leaving for the denser field sooner, which needs a cleared-area rule and the kit to survive Cold Plains (this run: 0 deaths either way).
3. Chicken / leave-game restarts (14 % of time, 129 s each): a potion-less character that loses a fight quits the game; cheaper than a death only if the walk back is short. Not tested: lower `combat.chickenPct` (floor 15) and `resource.fleeLifePct`.
4. Border ping-pong (335 Den of Evil entries in 24 h, "exploring crossed an area border"): small per event, U share about 4 %.
5. Sample size: to see a 5 % effect at SD 600 XP/h one needs about 35 runs per arm; the replicated recipe at 4 seeds is a screen for 15 %+ effects only.

## G1/G2 check list (shared code touched; all defaults 0, so nothing changes until a default is raised)
`internal/bot/goal.go` (poverty sell branches), `internal/bot/recoveryxp.go`, `internal/town/shop.go` and `chores.go` (tpMinGold), `internal/env/tuning_{resource,town}.go`, `internal/resource/tuning.go`. Before raising any default, run one Game.exe soak (G1) and the d2d soak (G2) on the three files above.
