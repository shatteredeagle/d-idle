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

## Measured (V) RESULTS_PLACEHOLDER
