# Short live drills for the g2e/g2f fixes (card shortdrills)

> status: current (2026-11-06). Tags: V = run live on the worker host, S = secondary, U = unverified.

Build under test: bot = branch `agent/shortdrills` on `origin/main` 2a6cf2b9 (the fixes of g2e F2/F2c and g2f merc revive are in it), d2d = fork `d2d.lab3-labverbs-f75816c6-linuxbuild` (fork commit f75816c616c8, binary sha256 prefix 53217ea4ecc8a4d3), lockstep, `--lab-verbs`, Sorceress, seed 1, slots 4 and 5 (slot 3 was held by drill 1), hidden, niced, worker load 12 to 20 on 16 cores during the runs.

## How (V)
`cmd/d2drill -group E1` (`cmd/d2drill/e1_lab.go`, runner `docs/probes/shortdrills/run.sh SLOT IDS LABEL`): the harness sets the hero up with logged lab verbs, writes the ledger run marker, drops its control connection and starts the REAL `d2bot --backend d2d --attach --d2d-attach --goals farm:Blood Moor` on the same d2d. After the marker the bot acts through its own input channel only; every run below has `writes after start: 0` (`ledger.WritesAfterStart`). Logs: `docs/probes/shortdrills/2026-11-06-*.log.txt`, records `.txt`/`.json`.

Limit (V): the bot is attached, so d2bot cannot restart the d2d ("attached d2d cannot be restarted"); a leave is proven by the bot's own `leaving the game` WARN line and its NewGame call, not by a new process.

## Drill 2: merc revive (group shortdrill-merc)
Finding that shaped it (V): a `debug merc hire` mercenary is level 1 and the NPC menu says `Resurrect Aliza: 0` (hire types 0 to 4 alike); no lab verb sets a merc level, so a live quote of 1920 cannot be produced. The bot reads no quote then (`probeQuote` keeps only a price above 0) and decides on the level estimate (clvl^2/2 x 15: 1080 at clvl 12, 1470 at clvl 14). The drills therefore set the purse around the ESTIMATE; the real-quote path (probe above 0, stash fund) stays offline-tested only (U live).

| id | setup (before the start) | result |
|---|---|---|
| E1.merc-a (run E1.merc-a-20261106-231623) | clvl 12, merc dead, pack 1090 (estimate 1080 + 10), no potions | PASS: `merc: revived price=0 level=12 estimate=1080` (log line 19) comes before the first purchase `shop: bought Sash` (the first purchase of the visit); the bot's first revive try failed once (`npc 5 (Kashya): not reached in 22.36s`, with a menu left open by an earlier harness version) and the next try revived |
| E1.merc-b (E1.merc-b-20261106-231820) | clvl 12, pack 810 (`max(q-420, q*3/4)` of the estimate 1080) | PASS: no `merc: revived`; the potions are bought (`shop: bought Minor Healing Potion` x3, `Minor Mana Potion`, gold 750 down to 600): no starvation |
| E1.merc-c (E1.merc-c-20261106-231834) | clvl 14, pack 1480 (estimate 1470 + 10) | PASS: no revive; `merc: revive skipped (gold) price=1470 gold=440 need=1520` (need = 1470 + 50, the clvl 13+ reserve). The first refusal before the shop is silent by design (`ReviveFirst`), so the logged gold is the one left after the visit's purchases |

Virtual run time 127 to 159 s each, wall about 8 s of bot time (lockstep runs faster than real time).

## Drill 3: no portal, low life (group shortdrill-noportal)
Finding (V): area 11 (Hole Level 1) is no longer a no-portal area. g2e F2c added it to `resource.chainHome`, so with no scroll the bot logs `no town portal and no waypoint: walking the level chain to town` and walks (a first run at 47 percent life, 0 potions and monsters about died once on the way; run label E1.noportal-a of 17:51, FAIL "never reached the no-portal path", log not kept). The drill therefore uses Tamoe Highland (area 7), which is in no walk-home table and has no waypoint.

| id | setup | result |
|---|---|---|
| E1.noportal-a (E1.noportal-a-20261106-231911) | area 7, pack and belt empty (0 potions, no scroll or tome), life 18/40 (45 percent shown), monsters cleared | PASS: first `action failed ... town: no town portal available` then `low life, no potion and no town portal: leaving the game` with `waited` = 21 s (bot clock, virtual; knob 20 s) and then `new game` fails only because the d2d is attached. Time from the first no-portal line to the leave: 21 s virtual, 0.12 s wall (vs 240 s before F2). |
| E1.noportal-b control (E1.noportal-b-20261106-231922) | same with one `tsc` in the pack | PASS: no leave line; `d2d town portal attempt failed ... cancelled (safety town-portal)` then `zone entry area 1` (town) and a shop visit; life 40/40 afterwards |

## Drill 1 (seed 13 short run, by the drill 1 agent)
Run gate2-s13-20261106-1735-drill13, slot 3, bot 2a6cf2b9, lab3 binary (fork f75816c616c8), manual route (gate-run.sh refused a second run of the same seed). V: 24 wall minutes = 158 virtual minutes, deaths 0, writes after start 0. Progress: clvl 1 to 2 in Blood Moor, clvl 4 at about virtual minute 49 in the Den and Blood Moor, first Cold Plains entry at clvl 5 (about minute 80) and left alive, clvl 6 at about 110, clvl 7 at about 138. No `stale pack` or `potion burst` log lines (0; the grep wording may differ, so U whether the rules fired). Verdict: PASS on survival; the gate verdict FAIL is expected for a short run. Host load rose to about 22 late, so speed dropped.

## Needed for drills 4 and 5
- A merc with a real level (a quote above 0) would let the real-quote path run live: either a fork verb `debug merc level N` or a paid Kashya hire at clvl 12 (the D8.h hire flow still fails: no hire line after `talk`).
- The managed-bot variant (`d2bot --managed`) would exercise the real game restart after a leave, but a saved character is back in town, so the no-portal scenario needs the attach form used here.
- Slot 3 and a load under 1.1 per core; load stayed 0.8 to 1.3 per core here and admission waited about 5 minutes once.

## Drill 4: minute-line tp count (group shortdrill-tpcount, card tpcount)
Finding (V, g2g seed 13 bot log): 1488 minute lines, all `"tp":0`, no `tpBuy`, while 1115 of 1320 `inv-tp-dump` lines had `hastp` true (for example `tome class=518 ... counted=1`). Cause (V by code and by the live control below): `townAdapter` keeps the `*town.Town` in a named field, so neither it nor `d2dTown` (which embeds it) implemented `bot.TPDiag`; `mercTown` forwarded to an inner Townsman that had nothing to forward, and the runner's type assertion failed. Fix: `TPCount` and `TakeTPBought` on `townAdapter` (`cmd/d2bot/beltfill_adapter.go`), test `TestD2DTownChainIsTPDiag`. The count itself (`Town.scrollCount`) was right.

Run (V): slot 4, d2d lab3-labverbs f75816c6 (sha256 53217ea4ecc8...), `docs/probes/shortdrills/run.sh 4 shortdrill-tpcount shortdrill-tpcount`, writes after start 0 for all.

| id | setup | result |
|---|---|---|
| E1.tp-a (run E1.tp-a-20261107-015552) | clvl 12, 3 `tsc` in the pack | PASS: first minute line `tp=3` |
| E1.tp-b (E1.tp-b-20261107-015608) | a `tbk` in the pack | PASS: minute `tp=3`, equal to the quantity inv-tp-dump counted (a given tome starts with 3 charges) |
| E1.tp-c (E1.tp-c-20261107-015625) | no scroll, gold 600 | PASS: `tpBuy` on the first minute line, max tp 1 |
| control, label shortdrill-tpcount-prefix (E1.tp-a-20261107-015718) | the same as tp-a with d2bot built WITHOUT the fix | FAIL: first minute line `tp=0`, want 3 |

## E1.mouth-pocket (TASK-67, g2gfixmouth)

Run (V, 2026-11-07): slot 3, d2d lab3-labverbs f75816c6, `docs/probes/mouthfix-run.sh 3 E1.mouth-pocket shortdrill-mouth`, setup `debug goto 6` + `debug warp 17 57` (subtile 87,287), clvl 15, writes after start 0. Result: NOT REPRODUCED (inconclusive). On this slot's seed the hero is not boxed in at that spot: the bot walked on at once (zone entries 6, 5, 10, died to a Misshapen in the Underground Passage while walking to the Blood Moor on 40 life: the throwaway is a weak clvl 15 stat set), 0 `nav mouth gate` lines. The g2g pocket is seed dependent (U): reproducing it needs the seed of the g2g seed 17 run and a hero that stays alive; the slot runner takes no seed argument here. The escalation itself is covered by `TestRepeatedHoldInAPocketEscalates` (offline, assumed exit and walls, U). The first Eval printed FAIL for this run because only status lines counted as leaving; zone entries count now.
## Drill 6: merc quote in the farm trigger (E1.merc-quote, TASK-66)
Run E1.merc-quote-20261107-071629 (slot 4, same fork binary as above, lockstep): result BLOCKED, not PASS (V). Setup as drill 2 (clvl 12, dead cheat-hired merc, pack 300, stash 0); `writes after start: 0`, virtual 159 s. The cheat-hired merc is level 1 and the menu quotes 0, so the bot read no quote above 0 (`probeQuote` keeps only a price above 0; 0 also means "no price parsed") and only the formula path ran. The new rule (quote used by the farm trigger and the revive bar) is therefore proven offline only (`TestQuoteReplacesTheFormulaEverywhere`, two classes), U live. Unblock: a lab verb that sets the merc level, or a real hire via Kashya at a known low level.

## Drill 7: scroll before gear, equip failure (groups shortdrill-tpfirst, shortdrill-equip; task g2gfixscroll)

> Tags: V = run live on the worker host (2026-11-07), U = unverified. Bot = branch `agent/tpfirst` on `origin/main` 23ccaf64, d2d = fork `d2d.lab3-labverbs-f75816c6-linuxbuild`, slot 4, runner `docs/probes/shortdrills/run-tpfirst.sh SLOT GROUP LABEL`, drill code `cmd/d2drill/e1_tp_first.go` and `e1_equip.go`. Writes after start: 0 in every run.

Finding behind the fix (V, bot log of Gate 2 g2g seed 17, minute 1035): the hero entered town with ONE scroll (`inv-tp-dump` counted=1), the town trip raises "no town portal scroll" only at 0, so no chores ran; the gear step (`internal/bot/leveling.go gearStep`) spent 156 gold (Cap, Sash, Leather Gloves), the hero left with one scroll and 304 gold, used it for the pocket escape and had none 15 virtual minutes later. The gear shop's own kit reserve (`leveler.kitReserve`) counts a tome or ONE scroll as "has a portal", so it reserved nothing. The minute line `tp` read 0 there (pre-#286).

| id | setup (before the start) | result |
|---|---|---|
| E1.tp-first-a (E1.tp-first-a-20261107-074523) | clvl 20, 1 `tsc`, 4 healing in the belt, spare potions in the pack, purse 700 | PASS: `gear: scroll before gear` (town restock run by the gear step) and a Tome of Town Portal bought (the town code buys the tome from 450 gold) before the first gear purchase; the gear budget is then recomputed from the gold left |
| E1.tp-first-b control (E1.tp-first-b-20261107-074536) | the same with 2 scrolls and purse 400 | PASS: the gear pass runs, no `scroll before gear` line |
| E1.equip-fail (E1.equip-fail-20261107-074309) | clvl 20, two Caps in the pack | INCONCLUSIVE: the d2d accepted every swap, so no `equip failed` line appeared; the "no leave after a failed equip" assertion is NOT proven live (offline: leveler test only) |

Runs that failed on the way and shaped the setup (V): the belt of a fresh character has 4 boxes (`no belt box` above 3); a potion trip (`[2 healing potions]`) ran before the gear step until the belt held 4 healing potions; with purse 400 a `[heal]` trip sometimes ran first and bought the scroll through the normal chores (run 2026-11-07 074323: FAIL "no restock line", the scroll was bought anyway), so the purse of the main scenario is 700.

Not proven (U): a control run WITHOUT the fix (prefix build) was not made; the first live run of the group (before the potion-trip fix of the setup) is not a control.

### Equip failure vs leave-game (g2g seed 13 E1) (V by log, code reading)
- No code path leaves the game on an equip failure. In all 20 `leveler: equip failed` lines of seed 13 the next game-leaving line is one of two independent things: `save point: saving and re-entering the game` (xp at risk, `savePointStep`, runs right after `gearStep` by design) or `leveling: everything worth farming is cleared, starting a fresh game ... reason no area left to pick` (`freshGameToLevel`; area 7 unreachable and Forgotten Tower closed by the route guard on 2 mana potions). The sequence in the same second is coincidence of ordering, not cause.
- Real defect found: the failed-swap memory was keyed by unit id (`badKey{id, slot}`), and ids change with every game, so after each fresh game the same Cap to head swap was tried again (Cap to head failed in several games). Fix: a second memory by item kind (base class, slot) that survives the new ids, `town.equipKindFails` (2), the kind is skipped until the character gains the levels the backoff asks for (one per failure, at most 4). Retry per swap was already bounded (2 clicks, `swapAttempts`, 2 failed swaps per visit).
- Cause of the refusal itself (U): "slot click refused ... stillOnCursor: true" on swapping over an occupied slot and `hand: unexpected cursor state: cursor never held item N` on picking from the pack. Not reproduced with lab verbs (d2d accepts the swap in the drill). Fidelity-matrix row candidate: IT03 (equip rules, swap over a worn item): add "swap onto an occupied slot leaves the item on the cursor in d2d (g2g seed 13, 20 times)" as the observed difference to check against game.exe.
