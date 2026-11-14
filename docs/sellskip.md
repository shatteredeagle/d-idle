# sellskip: the sell chore that never ran, the Blood Moor recovery, the revive gold

> status: current (TASK-399). Source: the hero14 analysis (docs/probes/hero14-analysis-2026-11-14.md, branch agent/hero14) and the 15 h container log of the Paladin "Doofer". Tags: V verified (how), S secondary, U unverified.

## 1. Why the sell chore ran in 1 of 158 trips (V from the log, offline code read)

- V (log): every `chores done` line lists `sell` under `skipped`, meaning the step was enabled but its need check said "not needed": `needSell` = `len(t.sellable(...)) > 0` was false. The bot meanwhile logged `sellable:9` and a trip reason "[9 items to sell]" 22 times. Pack at 18:34Z: `lbl x4, isc x2, mp2 x2, skp x2, hla, hp2, lea, pa1, yps`; worn: Short Sword, Buckler, Quilted Armor (no helm, belt, boots).
- V (code): the bot counts with `Evaluate(u, clvl, gold)` (no gear context): a plain armor piece is "drop", and a poor character (gold under `loot.poorGoldBelow`) turns drop into "sell". The town judges with the full context (`loot.ctxFrom`): below `resource.earlyGearMax` (15) `WearBetter` is on, so a helm or belt for an empty slot, or armor better than the worn piece, is "keep: empty slot". The two disagree, so the trip fires and the chore finds nothing.
- V (code): the keep is a promise that the bot will wear it. With `resource.gearDecisions` 0 (the default: the player decides gear, `BotDecidesGear` false) `needEquipFound` never runs, so nothing wears it and nothing sells it. The pack filled to 95 % with unworn "upgrades".
- V (log): the one sale at 01:12Z came when the hero reached clvl 15 (minute row `clvl:15` at 01:12:13Z): `Early(15, 15)` is false, `WearBetter` off, all 8 items sold at once (gold in the pack was 39, so it is not the gold line). The 22:13Z sale was a Chipped Emerald (pickit sell rule, no armor).
- Fix: `loot.ctxFrom` sets `WearBetter` only when `BotDecidesGear()` (internal/loot/loot.go). Tests: `TestEarlyGearKeepNeedsTheBotToWearIt` (loot, Paladin and Sorceress), `TestPoorPackOfPlainArmorIsSoldWhenThePlayerDecidesGear` (town, both classes). Both tests run with the game tables (`D2_MPQ_DIR=$HOME/d2d-lab/data` on the worker host) and pass there (V); they skip without tables.
- U: with `gearDecisions` 1 the bot and the town can still disagree for an item the equip step refuses (`isRefused`); not changed.

## 2. Recovery sends a clvl 14 hero to Blood Moor (V code, V log)

- V (log): `merc`/`inv-tp-dump hastp:false` for the first 6.5 h, 47 "no town portal: taking the waypoint to town". V (code): without a town portal `recoveryArea` and `starveArea` fall to `starveStep`, the first plan step (Blood Moor), whatever the level; the recovery XP pick is only used with a portal.
- Fix: `starveStep` asks `homeBandArea` first: the plan area of the character's own level among the areas it can leave without a portal (the first field plus every area whose waypoint is active, `activeWaypoints`), not cleared, not banned, safe by `guardedAreas`. The early kit gate (clvl under `resource.kitGateLevel`) keeps the first-field rule. Data: the level band comes from the plan table (internal/plandata), no area literal in Go. Test: `TestNoPortalRecoveryFarmsTheLevelBandViaWaypoints` (bot rig, runs offline, passes).
- S: with the Blood Moor/Stony Field ban of the stale-target loop (agent/staletarget) removed the band area is reachable; this card does not touch loop.go.

## 3. Spend order: tome and scrolls before the merc revive (V log)

- V (log, 01:13Z): gold 1,050 after the sale; the shop bought 3 healing, 1 mana, a scroll, the tome (400) and 3 more scrolls (300): 950 spent, 100 left. The revive (quote 600, need 650) then ran in `mercChore` after the chores and was skipped.
- Fix: while a merc is listed dead the bot sets `ChorePlan.ReserveGold` to the revive price (`merc.ReviveCost` of the merc's level, else of clvl-5, which gives the quoted 600 at clvl 14); the shop buys the tome and the scrolls after the first only while gold minus the price stays at or above it. Healing potions and the first portal scroll are survival and are not gated. Knob `town.mercReserve` (1 default, 0 off). Tests: `TestMercReserveOnlyForADeadMerc` (bot), `TestShopKeepsTheMercReviveGold` (town, with tables); both pass (V).
- U: the revive price is an estimate when the unit's level is unread (stat 12 is U on a live merc); a wrong estimate only changes how much optional spend is held back.

## 4. `town budget` line (new)

One line per Chores visit: `in` (gold at the start), `sold` (the sell chore's change), `earned`, `spent`, `byChore` (gold change per chore that ran: sell, repair, shop, withdraw, ...), `left`, `mercReserve`. Not verified live (no game).

## 5. Live check (not run)

Pack with a helm, belt and armor in the pack, clvl 14, gold under 300, gearDecisions 0: the next vendor visit logs `sell: ... sold` and `town budget ... sold>0`; a dead merc with 1,000 gold buys potions and one scroll, then revives (600), the tome comes on a later visit.
