# Economy of a fresh character on d2d (agent ECONOMY, 2026-10-30)

Tags: V = verified on d2d (how stated), U = unverified. Card: `openspec/changes/economy`. Evidence: the bot logs of the D2DLOOP replicated runs
(seeds 11 to 14, 2 virtual hours, `~/d2d-lab/loop/bot-m1*.log` on the worker host) and the runs of this card (`bot-g*`, `bot-h*`). Scripts: `tools/d2dloop/`
(`econ.sh build|run`, `seeds.sh`, `m.sh`, `idle.py` time shares, `flow.py` gold flows).

## Diagnosis

1. **Gold income is what the original gives, not a bug (V).** Pile pickup works: 65 to 120 gold per 2 h run from 196 to 293 kills = 0.3 to 0.4 gold per kill,
   the same as the d2d probe (docs/d2d-probes.md: about 0.4 gold per kill) and the live Game.exe G1 log ("+2..+10 gold piles", docs/recovery.md). Sales add 110 to 290 gold per run
   (`flow.py`). So gross income is about 150 to 200 gold/h; the "20 gold/h" of the loop doc is NET after spending, not a pickup failure.
2. **Sell works.** Per visit the sell chore succeeded in 4 to 8 visits per run (gold +20 to +60 each); the failure `1 item(s) no vendor would buy` happened once in four runs (m11), `repair not confirmed (durability 14% unchanged)`
   5 times in one run (m13, seeds with gold 100 to 200, so not poverty; the other three runs repaired fine). U why: m13's lowest worn item did not repair; the town layer now logs every worn item's durability and the
   unsold items (`repair: not confirmed`, `sell: left unsold`), the next occurrence names it.
3. **The scroll loop (bot, V).** The bot is without a town portal scroll nearly all the time ("no town portal: walking back to town" on every trip of g11, 14 of 14). Cause: a poor character made a sell trip for ONE item
   whenever it held a scroll (`goal.go`: `sell >= povertyWalkMinItems || HasTP`); the trip burns the scroll (100 gold), the shop visit then buys the next scroll with the sale money (g11: 8 of 14 trips sold one item,
   gold 88 -> 8, 129 -> 69, ...). Gold never rose above about 300, the kit was never complete, every later trip was a walk of 1 to 3 virtual minutes each way (12.5 % "Returning to the field" plus 9 % "Going to town" in g11).
4. **Neighbour link wait burned 15 to 25 % of the virtual time (V, d2d plumbing, the largest single item).** `d2dworld.waitLinked` polls `debug nearby` after every level arrival with pending neighbours; under lockstep each poll
   stepped 500 ms of game time and returned after about 1 ms of wall time, but d2d links neighbours on a worker thread in WALL time (about 100 to 300 ms), so every arrival cost 80 to 300 virtual seconds
   (log: "link wait over after 167ms, 636 polls"; m11 11 waits, m13 14, e13 16 = 18 to 30 virtual minutes of 120). The wait also tripped the 60 s clearing watchdog ("no progress in state clearing", `area skipped ... nav-stuck`,
   failsafe ladder): 7 to 10 failsafe resets per run (8 to 11 % of the time in "Failsafe: resetting in town", each followed by a chicken restart and, after the skipped Blood Moor, a clvl 1 character sent to Cold Plains without potions).
5. **Recovery mode is not what costs XP (V).** Kills per minute while farming gold (2.5 to 3.5) are higher than during the quest fights (2.1 to 2.3), and Blood Moor pays the best XP per kill for clvl 1 to 5. About 30 to 50 % of the
   recovery minutes have 0 kills (roaming a cleared Blood Moor between restarts), which is the remaining kills/h gap (U: roam coverage over a cleared area, nav stuck 109 to 118 events per run; not in this card).

## Fixes

| commit | change | scope |
|---|---|---|
| link poll | `d2denv.LateClock.Poll` steps 2 ticks (80 ms) plus 10 ms wall per poll instead of 500 ms virtual | d2d only (the Poll is d2d's) |
| poverty trips | `povertySellWorth`: with a scroll a poor character needs 6 junk items (was 1), without one 4 (unchanged) | `internal/bot/goal.go`, shared with Game.exe: **needs a G1/G2 check** |
| diagnostics | `sell: left unsold`, `repair: not confirmed` log the items and durabilities | `internal/town/chores.go`, log only |

## Measurement (same seeds 11 to 14, 2 virtual hours, `d2dsoak extract -hours 2`; each run is a different game even with one seed, docs/d2d-loop.md)

| metric (mean of runs) | before (m11-m14, 4 seeds) | link fix only (g11, g12, g14: g13 lost its lock slot to D2DVIEW) | link fix + trip fix (h11-h14, 4 seeds) | early baseline |
|---|---|---|---|---|
| XP/h | 3013 | 3583 (3 seeds; before on those 3: 3301) | **3869** (+28 % vs before) | 6600 |
| kills/h | 132 | 150 (before on those 3: 144) | **167** (+26 %) | late baseline 500 |
| gold/h net | 21 | 6 | 39 (noisy: 5 to 120) | no value |
| recovery share | 63 % | 56 % | 67 % (unchanged: the mode itself was not the problem) | |
| town share | 19 % | 26 % | 17 % | |
| failsafe ladder fires per run (log) | 7 to 10 | 0 | 0 | |
| clvl after 2 h | 4 | 4 | 4.5 | |

XP/h is still about 41 % under the early baseline (gate 20 %): not met. Remaining causes (U, not fixed here): 30 to 50 % of the recovery minutes have 0 kills (roaming a cleared Blood Moor until the next
restart; without a portal only the first field is allowed), 2.5 to 3.5 kills per fighting minute against about 8 on Game.exe, and out-of-mana waits of a clvl 1 to 5 Sorceress. The limited unrepaired-item cause (item 2) is unexplained.

## Needs a G1/G2 check

`internal/bot/goal.go` `povertySellWorth` (shared with Game.exe): a poor character with a portal scroll now sells at 6 junk items instead of 1. Offline: `internal/bot` tests green (`go test -p 1 ./internal/bot`).
The `d2denv.LateClock.Poll` change is d2d only. Whole-repo vet/test was not run on the head box (memory rule); per-package tests of bot, town, d2denv, d2dworld pass.
