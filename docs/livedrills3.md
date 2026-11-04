# livedrills3: DS, TM.a and D8.h/D8.h2 on d2d, first live runs (2026-11-04)

> status: current. Tags: V = ran live here, S = read from code, U = unverified. Lab fork d2d, slot 3, throwaway Drilly (Sorceress, clvl 1 per the 40 life), run recipe `docs/probes/livedrills3/run.sh` (same as livedrills2; `runtm.sh` = 170 s limit; `-v` added for logs). No drill or bot code edited.

## Results
| drill | verdict | numbers / reason | tag |
|---|---|---|---|
| DS.1 | PASS (2 runs, identical) | purse 600 -> 320, stash 1400 -> 1400, belt 0 -> 4, spent 280 (2 minor heal, 2 minor mana, 1 TP scroll) | V |
| DS.2 | FAIL (2 runs, identical) | purse 6 -> 6, stash 1400 -> 1400, belt 0 -> 0, withdrew 0 (wanted 344 = keep line 350 - 6). Cause is the drill plan, see below | V, S |
| DS.3 | BLOCKED | needs `TripLab.RunnerRounds` and a vendor-stock verb; not run | S |
| DS.4 | BLOCKED (2 runs) | `debug give rvs` put the potion straight into the belt (belt 1 -> 1, pack empty), the drill requires it in the pack | V |
| TM.a | BLOCKED | scrolls chore bought 1 tome (300) + 5 scrolls (100 each) with the drill's 1000 gold: 5 charges, recipe needs 13 (about 1600 gold) | V |
| TM.b, TM.c | BLOCKED | missing verbs: a way to force the stock `items` verb for one look; a fault injection that makes `dump items` fail | S |
| D8.h | FAIL (2 runs per repeat 2, identical) | no hire line after talk | V |
| D8.h2 | FAIL (2 runs per repeat 2, identical) | same menu, `ErrCantAfford` never reached | V |

## DS.2 cause (S, internal/town/chores.go:311, internal/drill/stash_codefix67.go)
The `withdraw` step is `plan.Potions && plan.Gold`. The drill runs the plan `potions stash`, which has no `gold`, so the step never runs; the log shows only the shop visit (`shop: visit ... broke=true gold=6`) and no stash open. This is a drill plan bug, not evidence about the bot's withdrawal. Fix to try: plan `potions gold stash` in `ds2` (and `ds4`). Not applied (no drill edits).

## TM.a (V)
Shop prices on d2d here: TP scroll 100, tome 300 (log `docs/probes/livedrills3/2026-11-04-tm1v.log.txt`). Setup gold of 1000 buys the tome and 5 scrolls only. Needs `debug stat 14` of at least 1600 in the TM.a setup. The run ended with the character in Blood Moor (`END ... in town: false`); the next D8 run's `debug goto 1` brought it back.

## D8.h / D8.h2: the real Kashya menus (V)
- First menu (livedrills2): header `Kashya`, `talk`, `cancel`.
- After `talk`: `talk`, `introduction`, `gossip`, `cancel` (4 lines, no `hire`, no priced row, no price text). Polled 8 x 400 ms by `hireFlow`.
- Quote: none read (0). Class code of the offered hireling: not observed (UNVERIFIED still; 0A and IW stay open). Hire did not happen: gold 5000 -> 5000, no merc. D8.h2 (gold 1): same menu.
- Reading: the Rogue hire is not behind talk/introduction/gossip in this menu. Next steps to try (U): click `introduction` or `gossip` and read the following menu; or the hire window is a separate UI the d2d `menu` verb does not list (a screenshot with `tools/shot.sh` at that point is needed); or d2d needs a quest/difficulty state. A merc-less character here shows no hire option at all.

## writes after start
The d2drill path does not call `d2cheat runstart`; every setup write (`debug clearinv`, `debug stat 14/15`, `debug give`, `debug goto 1`) is done by the drill before the bot code runs and is ledgered with the drill id. The bot code made no gameplay write. Number from the ledger: not queried (U), no run id exists to filter on. These are drill runs, not measured runs.

## Evidence
`docs/probes/livedrills3/`: DS1 json/txt, DS2v.log.txt, tm1.log.txt, tm1v.log.txt, d8a.log.txt, d8b.log.txt, d8b.json, run scripts. Character left alive in town, no panels (END line of the D8 run). Slot 3 released.
