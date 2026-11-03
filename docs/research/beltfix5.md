# BELTFIX5: Gate 1 20261103-0338, the second belt window (clvl 11) and the belt windows of four runs

> status: current. Follows docs/research/beltfix4.md. Run 0338: bot 30710e19, d2d lab2-freezeresume ee5e06d8ef99, seed 11.

Tags: V run = counted from the minute archive (`docs/probes/soak-logs/gate1-20261103-0338-minutes.jsonl.gz`) or from `logs/bot-0.log` of the run on the worker host (read only); V test = a unit test here that fails when the change is reverted; U = not verified live. Line numbers are lines of the archive file (the d2dsoak rule skips lines without fields, so its numbers are 3 lower after line 270).

## 1. Where the gold went (V run)

Window: minute lines 333-339 (04:06:04-04:06:29 wall), clvl 11, Cold Plains / Cave, hp min 25 to 90 percent, state `Recovery: farming gold`, packHP 0, belt = mana potions only, purse 3 to 11 gold.

The 60 lines before (03:57-04:05:53) were 17 shop visits, 3393 gold spent (V run, `shop: visit` lines, price list 30 healing, 60 mana, 100 portal scroll): 930 healing potions (27 percent), 1920 mana potions (57 percent), 500 portal scrolls, 43 other. 16 town trips, 5 of them emergency. Income was about the same (purse plus stash stayed at 1500 to 1800).

The two arrivals right before the window:
- 04:05:37 `town trip [0 healing potions] emergency true`, purse 318: bought 4 Minor Healing (120) and 1 Minor Mana (60), purse 138. Healing was bought before mana (potion plan wantHP 3, then wantHP 1 + wantMP 1).
- 04:05:53 `emergency town portal` (chicken from Black Marsh, 19 monsters at the arrival spot, `potion: not drinking because chose crowd-abandon`), purse 126: bought 4 Minor Healing (132), purse 6. Those 4 potions were drunk within about 10 s in the same crowd (potionsUsed 213 to 220 across lines 328-332).
- 04:05:56 `recovery: start (gold 6 < kit 500)`, `poverty mode: on (gold 6 below kit cost 190, 1 healing potions)`. 04:06:29 third arrival, purse 11: `shop: visit bought {} spent 8` (repair).

The emergency path (BELTFIX3/4: shop on every arrival without a healing potion, gold for one) ran both times and bought healing first. Nothing was spent on scrolls or mana ahead of healing in the last two visits. The purse was simply empty.

The stash was not: the merc line of every visit prints purse plus stash (`merc: revive skipped (revive cap) price=900 gold=1556` with purse 138 at 04:05:38, `gold=1424` with purse 6 at 04:05:53, `gold=1421` at 04:06:29): 1418 gold in the stash for the whole window (524 at 03:50, banked up through the `bank` chore at 03:50, 03:54, 03:56, 03:57 and 04:02). The town package only deposits (`needBank`: pack gold above 300 + 50 x clvl = 850 at clvl 11) and has no withdraw chore (docs/fixbank.md: "the town package has no withdraw chore"). The trip policy, `poverty mode`, `recovery` and the shop all read the pack gold, so a character with 1418 banked was in poverty mode and farmed gold in the field.

## 2. Classification

Defect (reserve), not low income and not an artefact. Poverty by purse was caused by the bot: it banks everything above the keep line and then cannot reach it. Spending order was right in the last two visits (healing first). Mana potions are 57 percent of the gold of the 60 lines before (a lightning sorceress pays 60 per Minor Mana Potion against 30 per healing potion); that is a design question (mana versus healing budget at clvl 9 to 11), not a defect, and is not changed here.

Second contributor, not fixed: the town portal returned the character into the crowd that had just chased it out (19 monsters at the arrival spot), and four fresh potions lasted 10 s.

## 3. Fix (V test, U live)

- `internal/town`: chore `withdraw` between `bank` and `shop` (`needWithdraw`, `withdraw` in basicsfix.go): no healing or rejuvenation potion in belt or pack, stash gold above 0, pack gold under the keep line (`bankKeep`): `MoveGold(keep - gold, false)` on the open stash, confirmed by the pack gold. Only the d2d controller (`goldMover`) is supported; the non-d2d path would need the typed prompt, which is unverified, and is left out. The withdraw direction itself is V-d2d (docs/d2d-town.md: withdraw 2000, 20000/0 to 17000/3000).
- `internal/bot`: `resInput` adds the stash gold to `Gold` while no healing or rejuvenation potion is in belt or pack, and `choresDue` does the same for its two purse tests (`broke` and `minPotionPrice`). With a potion in hand nothing changes.
- Tests: `TestBankedGoldCountsWithNoHealingPotion` (replay of the window: purse 126, 6, 3 with 1418 banked, in town and in the field: emergency potion chore; 5 failures when reverted), `TestBankedGoldIgnoredWithPotionOrNothingBanked`, `TestWithdrawForTheKitWhenNoHealingPotion` (purse 6, bank 1418, clvl 11: one withdrawal of 844, potions bought; fails with the chore off), `TestNoWithdrawWhenPotionsOrPurseOrStash`. The town tests need the extracted tables (skip here, run on the worker host).

Known limits (U): a recovery that started with one potion left still runs until that potion is drunk (the recovery start reads the purse); the next evaluation, with 0 potions, sends the emergency trip. A failed withdrawal leaves the trip policy believing in gold it cannot spend; the trip budget and the emergency cooldown bound the repeats, and the chore logs the error. The withdrawal brings the purse up to the keep line (850 at clvl 11), which is the amount a death takes; it is the line banking already maintains.

## 4. Not done: the repair chore runs before the shop

`repair` precedes `shop` in the chore list and has no potion reserve: with 0 healing potions a 300-gold repair could take the potion gold. It did not happen in this window (8 gold). Candidate for a later card.

## 5. Belt windows of the four Gate 1 runs of 2026-11-02/03 (V run)

The d2dsoak rule replayed over the four minute archives (consecutive lines outside town with no healing potion in pack or belt; limit 5): every window longer than 3 lines, gold = pack gold of the minute lines (the stash is not in the minute line).

| run | lines | max window | windows over 5 | gold in the window | clvl | cause |
|---|---|---|---|---|---|---|
| 1939 | 1182 | 12 | 1 (lines 446-457) | 30 to 38 | 12 | gold covered one potion; reason not analysed here |
| 2210 | 1147 | 37 | 1 (218-254) | 332 to 366 | 9 | BELT A, no trip home (BELTFIX3) |
| 0053 | 1239 | 7 | 1 (6-12) | 10 to 16 | 2 | fresh character, poverty |
| 0338 | 1205 | 7 | 2 (6-12, 330-336) | 10 to 16; 3 to 11 | 2; 11 | poverty; stash defect (this card) |

Five windows over 5 in four runs. Pack gold under 30 (poverty by the pack): 3 of 5 (60 percent); 30 or more: 2 of 5 (40 percent), and both of those were cases where a potion was affordable. Windows of 4 and 5 lines (runs 1939 and 2210) had 25 to 28 gold.

Reading for the proposed delta "count a window only once gold covers one healing potion (30)": it would drop the two clvl 2 windows (identical in two runs, same seed) and the clvl 11 one, and it keeps 1939 (12 min) and 2210 (37 min), the two the bot could have avoided. But the clvl 11 window was a defect with 1418 banked: the rule must read pack plus stash gold, or it would have excused it. The minute line has no stash field; adding `stashGold` to the minute line (internal/bot/observe.go, internal/d2dsoakstat/minute.go) is the prerequisite and is not part of this card. With pack plus stash the count is 3 of 5 that must be explained (1939, 2210, 0338 L2) and 2 of 5 excused (the clvl 2 windows).

## 6. Live check (not run)

Gate 1 re-run or a short run on a throwaway character: put 1000+ gold in the stash, empty belt and pack of healing potions, pack gold under 100, clvl 6+. Expect in `bot-0.log`: `town trip ... emergency true`, `town: withdrew gold for the kit`, `shop: bought ... Healing`, `chores done` listing `withdraw`, before `wp: used`. Also expect no `poverty mode: on` while the stash covers the kit and the belt is empty.
