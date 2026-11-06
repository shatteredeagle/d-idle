# Gate 2 g2f: why seed 17 failed the Andariel rule, and the other open items

> status: current. Read-only analysis of the merged g2f record (docs/soaks/gate2-multiseed-20261106-1626-g2f.md, the three per-seed records, the minute logs and the `*-g2f-extras.jsonl.gz` logs under docs/probes/soak-logs/) and the bot source of origin/main 35fe12b6; format of docs/research/g2e-diagnosis.md. No game, no contract change. Tags: V = read from a committed log line (wall clock -05:00, which is EST; the minute lines are VIRTUAL minutes) or from source; S = secondary; U = inference. Run: seeds 11, 13, 17, all clvl 16, writes after start 0; 11 and 13 PASS, 17 FAILS only the interim Andariel rule. Task: TASK-41. The code fix of part 2 is in the same branch (section 7).

## 1. Findings table

| id | seed | what | cause | evidence | fix | class |
|---|---|---|---|---|---|---|
| H1 | 17 | merc dead from clvl 12 to the end, 68 `revive skipped (gold)` lines | V: the bar (price + kit reserve) was never met; U: the price in the bar is an over-estimate | section 2 | F-A reserve cap, F-B real quote, F-C stash fetch (section 7) | bot defect |
| H2 | all | the revive pays from the pack, the bar counts pack + stash | V: `d2dmerc.Reviver` compares the quote with pack gold; banking keeps the pack at 300 + 50 x clvl (1100 at clvl 16), below every price from clvl 14 up | section 3 | F-C | bot defect |
| H3 | 13 | death at clvl 4 in Cold Plains (identical in g2d, g2e, g2f) | V: re-entered a 12-hostile pack with 0 healing in the belt | section 4.1 | TASK-39 | bot defect |
| H4 | all | `tp` is 0 on every minute line | V: `mercTown` (cmd/d2bot/mercwire.go) wraps the town and hides `TPCount` / `TakeTPBought` from the runner's type assertion | section 4.2 | fixed in this branch | bot defect (plain) |
| H5 | all | `mouthRef` 430/536/159 vs g2e 104/295/47 | V: different counters, not a regression and not a gain | section 4.3 | none | counting |
| H6 | 11 | "poverty windows of 8 and 6 minutes" | V: no BELT window over 4; the 8 is a NOKILL window | section 4.4 | none | no defect |
| H7 | all | `mercTown` hides every other optional town method too | V by source, not exercised | section 4.2 | decision needed | design flaw (U effect) |

## 2. Seed 17: the merc economy (H1)

### 2.1 Timeline (V)
- Merc died at 09:14:42 (`merc: died` in the minute log; minute line 476), clvl 12, pack gold 108, stash 0. It was never seen alive again: every one of the 68 refusals in the extras log is `merc: revive skipped (gold) price=P gold=G need=N` (G is pack + stash, see 2.4).
- Gold in the dead window (pack + stash, mean per wall hour): 09h 539, 10h 1077, 11h 566, 12h 555, 13h 1264, 14h 1418, 15h 1635, 16h 1578; max over the window 2234. For clvl >= 15 the mean is 1106 (seed 11: 1395, seed 13: 1498).
- Skipped refusals: first 09:14:54 (price 1080, gold 48, need 1390), last 16:19:48 (price 1920, gold 1701, need 2075). The closest misses: 16:14:13 gold 1932 vs need 2230; 15:46:15 gold 1928 vs 2075; 15:34:01 gold 1940 vs 2230; 13:19:49 gold 1725 vs 1835 (price 1680).

### 2.2 The price (V for the numbers, U for the cause)
- The price in the log is `ReviveCost(level)` = level^2 / 2 x 15 with `level` = the player level (`levelEstimate` in cmd/d2bot/mercwire.go; the d2d dump has no merc level, said in the code comment). Prices seen: clvl 12 1080, 13 1260, 14 1470, 15 1680, 16 1920 (exactly the formula). So the price rises about 200 per hero level and the hero must out-earn it.
- V: the estimate is an upper bound, not the quote. Seed 11 at 08:37:26: the revive was tried and the reviver read the real menu quote: `failed: d2dmerc: the purse is below the quoted resurrect price: 46 gold, quote 120` against `price=480` (hero clvl 8). 120 is the formula at merc level 4 (4^2 / 2 x 15 = 120), so that merc was 4 levels behind the hero. Seed 11 at clvl 12 (09:13:14) lost 620 gold in one minute, consistent with a merc level 9 quote (600) against an estimate of 1080. U: the ratio at clvl 16 (a quote near 1270 for merc level 13 is a guess, no clvl 16 quote was read because no revive was tried).

### 2.3 The reserve (V, source)
- `ReviveNeed` (internal/merc/policy.go) = price x `merc.reviveGoldMult` (1) + `Kit` x `merc.kitReservePct` / 100. `Kit` = `KitRefill` = 100 (scroll) + (`resource.kitMinHP` 4 + `kitMinMP` 3) x 30 = 310. With the kit already full (`KitFull`) the percent is `merc.kitReservePctFullKit` (50): 155. The `need=` values in the log are price + 155 or price + 310 on every line (V: 1415-1260, 1780-1470, 1835-1680, 1990-1680).
- What it protects: potions and a scroll before the merc ("potions first", docs/contracts/merc-policy.md line 126, a user decision of 2026-11-02; the percent was lowered from 150 to 100 on 2026-11-04 after a stall).
- The reserve is not the whole story: with the reserve at 0 the gold would still have been short of the price on 64 of the 68 lines (V, counted from the lines: 4 lines have gold >= price). With the reserve capped at 10 gold: 3 lines pass (13:19:49 at clvl 15, 15:34:01, 16:14:13). With 100: none. So the reserve alone does not explain the seven hours; the price over-estimate (2.2) and the pack/stash split (3) are as important.

### 2.4 Where the gold went (partly U)
- V: banking. `Town.bankKeep` = `town.stashGoldBase` 300 + `town.stashGoldPerLvl` 50 x clvl; every town visit deposits the pack above it. In the dead window the pack never exceeded 759 at clvl 12, 687 at clvl 13-14, 1441 at clvl 15 and 1100 at clvl 16 (the keep line), while the stash held up to 1134 (V, minute lines). Pack + stash is what the bar counts (`mercCtx`: `Gold = Player.Gold + StashGold`).
- V: purchases happen in town visits and are not in the WARN extras (the shop lines are INFO), so the per-item split is U. Visible drops of pack + stash in one minute: 1440 -> 804 (11:24:09, clvl 15) and 1235 -> 625 (11:35:38), both right after a `crowd: walk abandoned, retreated and opened a portal` line. U: kit refills and repairs (a clvl 15 kit is 310 plus repairs); there is no gambling in the bot (S: no gamble chore in internal/town).
- V: the hero was in poverty mode 47 times in the run (`poverty mode: on`), with the same count class as seeds 11 and 13 (37 and 37), so the purse was spent on the kit as designed. The merc revive was starved by the price, not by a purchase loop.

## 3. Seeds 11 and 13: how they afforded revives (H1, H2)

| item | seed 11 | seed 13 | seed 17 |
|---|---|---|---|
| merc deaths (V) | 11 | 9 | 1 |
| revives in the logs | at least 8 (U: the `revived` line is INFO and not in the extras; counted from `revive cap` refusals, 2 an hour, and gold drops) | at least 6 (same) | 0 |
| refusal reasons (V, extras) | `revive cap` 34, `gold` 30 | `revive cap` 25, `gold` 36 | `gold` 68 |
| mean gold at clvl >= 15 (V) | 1395 | 1498 | 1106 |
| max gold (V) | 2821 | 3250 | 2234 |

- V: seeds 11 and 13 were also refused on gold at times (30 and 36 lines) but their purse crossed the bar between refusals; their `revive cap` lines (the 2 an hour limit) carry gold values at or above the bar (for example seed 11, 12:23:10 gold 2360 vs need 1990). They revived because they were richer at the right moment and the merc died early (clvl 8-9, prices 480-750, well inside the purse), not because a rule differed. Seed 17's merc died once, at the one moment the price had already outgrown the income (clvl 12: 1080).
- V: the cap itself is a limit worth knowing: seeds 11 and 13 sat on `revive cap` for hours with more than enough gold (seed 11 15:00-15:50: seven lines). That is `merc.reviveMaxPerHour` (2) working as designed against a merc that dies every few minutes; not changed.
- Counterfactual for seed 17 (V arithmetic on the logged lines, U that a visit would have happened): with the reserve capped at 10 the first revive would have been at 13:19:49 (clvl 15), about 3 hours before the end of the run; with the real quote (lower than the estimate, U by how much) earlier still.

## 4. The other open g2f items

### 4.1 Seed 13 early death (H3; TASK-39 earlydeath13 owns the fix)
Timeline (V, minute lines and extras, wall 08:26:39 to 08:28:08; the first virtual minutes run at 1 s of wall time each):
- 08:26:39 clvl 4, life 55/59, quest step `Level up to 5`; 08:27:04 town trip (`poverty: 4 items`), gold 82 -> 300 (U: sales), life 69/69, 6 potions.
- 08:27:43 area 3 (Cold Plains), life 55/69; 08:27:46 life 43/69, potions 4, `potionsUsed` 13; 08:27:48-08:28:00: potions drunk 15, 18, 21 in 14 s of wall time (11 in about 17 s), belt down to Minor Healing and mana only.
- 08:28:01 Save and Exit (chicken) at (34,111); re-entered 08:28:03, safety action `portal` failed (`d2d portal: no object after the click, the click was dropped`); 08:28:04 town chores (gold 500 -> 260, 5 potions bought); 08:28:06 back in area 3 with life 66/69; died 08:28:08 at (174,316), `killerType melee`, `Gargantuan Beast` x4 plus a Fallen within 2-3 tiles, 12 hostiles, `lastSafety drink-life`, `belt 0 hp 2 mp, 2 portal scrolls, 5 potions drunk in the last minute`, `0 casts in 5 s`, maxLife 69, `mlvl=2 gap=-2`, route guard blocked monastery and tower (no merc, clvl 4).
- V: the hero died 2 s after returning from the chores to the same pack (12 hostiles) with the cheapest potions (Minor Healing heals about 30 of 69). U: the Gargantuan Beasts of the lockstep seed stay on the same spot after a re-entry. What the logic did: drink-life fired repeatedly and the chicken fired once; the retreat did not leave the crowd and the return leg walked straight back into it.
- Smallest fix (for TASK-39, not duplicated here): after a chicken or a crowd retreat, do not return to the area for the same pack: let the `Quest: fighting (Level up to 5)` leg of the plan table pick Blood Moor or the Den at clvl <= 5 (the plan table in internal/plandata is the place, not Go), or add a minimum `flee` rule: with `maxLife` < 100 and 4 or more melee hostiles within 3 tiles, leave by portal at once instead of drinking. Offline test: replay the state above through the combat decision.

### 4.2 The `tp` field reads 0 (H4) and what else the wrapper hides (H7)
- V: `internal/bot/observe.go` `tpDiag` does `r.d.Town.(TPDiag)`. In the d2d stack the town is `*mercTown` (cmd/d2bot/mercwire.go), which embeds the `bot.Townsman` interface: only the five interface methods are promoted, so `TPCount` and `TakeTPBought` of `*town.Town` are invisible and `tp` stays 0 and `tpBuy` is never written (V: `tpBuy` is absent in all three minute logs, `tp` is 0 on all lines, while the death autopsy at 08:28:08 says `2 portal scrolls`). It is not a bug in tpfield (#256) or in `TPCount`.
- Fix (done): `mercTown.TPCount` and `TakeTPBought` forward to the inner town (test `TestMercTownForwardsTPDiag`).
- H7 (V by source, effect U): the same wrapper hides `BeltFiller` (`NeedsBeltFill`, `FillBelt`), `ForgetBelt`, `PlannedSpend` (the `town.tripMinSpend` floor) and `UseWeapon` from `r.d.Town.(...)` in internal/bot/resource.go, goal.go and task.go whenever `merc.enabled` is 1 (the default). Since those paths were never exercised with the wrapper, forwarding them would switch on unproven behaviour; left for a decision (section 6, question 4).

### 4.3 mouthRef against g2e (H5)
- V: g2e had no `mouthRef` field in its minute lines (0 lines carry it); the g2e figures 104/295/47 in docs/research/g2e-diagnosis.md are counts of one log message kind in the extras log. Like for like, the number of extras lines containing `unwanted exit` is 317/720/120 in g2e and 317/696/117 in g2f. The new `mouthRef` sum (430/536/159) counts gate refusals per minute line, a different counter, and cannot be compared with either.
- So the cave-mouth gate (F1 of g2e) refusals did not change between g2e and g2f (V, 317 vs 317, 720 vs 696, 120 vs 117), and no reduction can be claimed.

### 4.4 Poverty windows on seed 11 (H6)
- V: the longest BELT window on seed 11 is 4 minutes (limit 5). The longest NOKILL windows are 8 (15:20:55, clvl 16, `fsWhy guard_exit`) and 7 and 6 (10:50:02 and 09:35:01, `Returning to the field`, gold 133 and 539); none is over 10. The low-gold stretches (pack and stash both below the kit cost 190) are the first 25 minutes (clvl 1 to 6, 24 + 23 + 16 minute lines) and one 9 minute stretch at clvl 15 (11:23:58, gold 108).
- Why poor: V early game (no income); later poverty lines (12:33:15, 12:53:46, 15:03:52, 16:14:06) follow revives (gold 2037 -> 1337 at 15:03:56, 1744 -> 1004 at 16:14:13): a revive on this build spends the purse down to the reserve, which is the reserve doing its job. Does it matter: no, no gate criterion is near the limit and the merc was back each time. Nothing to fix.

## 5. Fixes of g2e (F1 to F7) against the g2f logs (AC2)
- F1 mouth gate: the refusals are unchanged (4.3). Recurred.
- F2 no-portal leave: BELT windows are 4/2/3 against limit 5 (seed 11 had 11 in g2e). Did not recur.
- F3 drought timing: NOKILL 8/9/6 against limit 10 (g2e 12 on seed 13). Did not recur.
- F5 revive first: never exercised (no refusal passed the bar on the pack-only test; section 2). Unproven.
- F6 route guard kit rule: seed 13 had no second death (g2e had one at clvl 16). Did not recur.
- F4, F7: no g2f evidence either way (U).

## 6. Needs the user
Nothing here is decided by the agent; contracts and the Numbers tables are not edited.
1. Seeds: 3 seeds are indicative only. For a real claim, 5 seeds (about 8 wall hours each in parallel on the worker host) or stay at 3 and call every pass indicative. Recommendation: 5, once the merc fix is merged, so the next Gate 2 is the proof.
2. Gate 3 (Gate 2 chain) needs the user's go; not requested here. Recommendation: not before a Gate 2 rerun shows the merc revive on a seed like 17.
3. The Andariel interim rule (clvl 17 with a dead merc, floor 15 with a live merc): keep it. It did what it should on seed 17: it showed that a hero at clvl 16 without a merc should not fight Andariel, and the fix goes to the economy, not to the floor. Do not lower the floor to pass the gate.
4. The reserve cap in this branch (section 7) deviates from the user decision of 2026-11-02 in docs/contracts/merc-policy.md line 126 ("plus `merc.kitReservePct` (100) percent of the full kit refill") for revives. Approve the cap (default 10) and a contract update, or revert `merc.reviveReserveMax` to a large value. Also: forward `BeltFiller`, `ForgetBelt`, `PlannedSpend` and `UseWeapon` through `mercTown` (H7)?
5. TASK-39 (the early death) is a separate bot-defect task; it needs a go for a fix.

## 7. Part 2: the fix in this branch (bot defect, offline tested only)
- F-A knob `merc.reviveReserveMax` (default 10 gold, range 0..2000, docs/knobs.md regenerated): the kit reserve of a REVIVE is `min(kitReservePct share, reviveReserveMax)`. A hire keeps the whole reserve. The kit itself is still required (scroll and potions in hand) in the normal step, and `ReviveFirst` (before the shop) keeps its waived check; gold 1500 against a price of 1920 still waits (test `TestReviveReserveCap`, two classes, both kit states).
- F-B `d2dmerc.Reviver.Probe` reads the real quote from the NPC menu without paying; `mercWire.probeQuote` calls it once per dead merc when the estimate says too poor and the purse is at least 100; the quote is kept until the merc is seen alive and also feeds the gold farm bar (test `TestReviveReadsTheRealQuote`). U live: the walk to the hireling for the probe was only ever tried inside a revive.
- F-C `Town.WithdrawGold` and `mercWire.fundRevive`: the part of the price that the pack lacks is taken from the stash before the payment (tests `TestReviveFetchesTheStashPart`, `TestWithdrawGoldForTheMerc`); `ReviveFirst` counts the stash when a bank is wired.
- F-D `mercTown` forwards `TPDiag` (H4).
- Not changed: the ladder (`internal/ladder`, `internal/decide`) tests pass unchanged; `merc.reviveMaxPerHour` and the stop rules.

### 7.1 Next Gate 2 checklist (live, not done)
1. Expect `merc: revive quote read price=...` lines and the `need=` of refusals using it; on a seed like 17, a `merc: revived` before clvl 16 and `merc: withdrew N gold for the revive`.
2. `tp` is non-zero and `tpBuy` appears in the minute lines.
3. No new BELT or NOKILL window above 5 / 10; poverty mode after a revive is expected.
4. Watch the probe walk (Greiz is beside the Sewers stairs): no death in the Act 2 town during a probe.

## Shared-file lines for the merger
See backlog/merge-lines/agent-g2fdiag.txt.
