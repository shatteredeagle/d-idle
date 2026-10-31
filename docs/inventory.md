# Inventory and stash policy (INVSTASH audit, 2026-10-30)

Goal: an 8 h unattended run never stalls on a full pack. Tags: V = verified (offline sim test or read in code), S = secondary, U = unverified live.
Grids: pack 10x4 = 40 cells (V, `town.invCells`, `data.Layout`), stash 6x8 = 48 cells (V, `fillStash` in tests; LoD size S). The bot also keeps a belt, equipment and the Cube (2x2, in the pack).

## Policy table (item class -> action)

| Class | Pickup | Town | Notes |
|---|---|---|---|
| Quest items, Cube | always (prio 100) | never sold or dropped (`neverSell`, `sellProtected`) | V `TestFullPackQuestItemSurvivesSelling`, cycle test |
| Runes | stash S prio 90 | stashed, never sold; protected in a full stash | V `TestFullStashOfRunesKeepsEverything` |
| Gems | top gems stash S; cube gems kept up to `GemKeep` 6 per code; surplus low gems sell | stash | cube chore consumes them |
| Unique/set | stash A prio 85 | stash, protected (prio >= valuableMinPrio 70) | non-upgrades are not sold yet (U, SME suggests selling later) |
| Unique charms / good charms | stash | stash; unidentified charm goes to the stash as is when the pack is >= 80 % (new) | V `TestInventoryCycles` |
| Jewelry, unidentified | keep until the identify step (isc/ibk) | identified: stash if good else sell. NEW: still unidentified in a pack >= 80 % -> sold as is (no ID tome early, SME) | V `TestFullPackOfUnidentifiedJewelryIsSold` |
| Magic/rare weapons and armor | pickup if value per cell >= `minValuePerCell` | sold unidentified | |
| White/low/superior junk | not picked (poor mode excepted) | sold when in the pack at >= 80 % | |
| Healing/mana potions | picked while the belt wants them | belted; surplus beyond `spareKeep` 4 per code sold when the pack is full |
| Rejuvenation | cap `loot.rejuvSpare` 3 in total, belt plus pack, at any level, Full preferred (REJUVUSE); pack ones are moved to a free belt slot in town so the hotkey drinks them | surplus sold when the pack is full (INVREJUV: was keep, tier S, prio 75, protected: hoarded to 2/3 of the pack) | V `TestFullPackOfSparesIsSoldDown` |
| TP/ID scrolls | with a tome: absorbed by the tome (S, Kolbot). NEW: without a tome only up to the shopping target (was 20 loose cells) | surplus beyond `spareKeep` sold when full | V cycle test |
| Tomes | the first of each kind (tier S) | extra tomes sold when full (NEW, allowance 1) | |
| Keys | not bought (`keyTarget` 0) | NEW: picked-up keys sold when the pack is full (allowance = keyTarget) | V `TestFullPackSellsKeysKeepsCube` |
| Gold | piles above `goldMinStack` | deposit above `stashGoldBase` 300 + 50 per clvl | docs/deathspiral.md |
| Ammo | quiver, not the pack | | |

## Full stash (V `TestInventoryCyclesStashOverflow`, 600 drops, 3 seeds)
`stashItems` stashes best-first; what does not fit goes to `relieveStash`: unprotected pack items are sold, and an unprotected stash item of lower priority than a protected pack item is moved to the pack and sold. Protected items (runes, uniques, gems, keep verdicts) that still do not fit stay in the pack and are not retried until the pack changes. Log: `town: stash full left=.. sell_from_pack=.. swap_from_stash=..` and `town: stash full sale sold=..`.
Remaining risk (U): 48+ protected items would fill the pack again. At about 1 rune per 30 minutes this does not happen in 8 h.

## Gaps found and fixed in this audit
1. Scrolls without a tome were picked up to 20 loose cells (`loot.needed("scroll")`): now capped at the shopping target.
2. Picked-up keys and extra tomes were tier C "town supplies" and never sold (found by the cycle sim: 85 % pack after every trip): now surplus when the pack is full.
3. Unidentified jewelry waiting for an ID that could not happen (no gold/scroll) was never sold: sold at >= 80 %; charms stashed unidentified.
4. Valuables left on the ground ("loot: pack full, item left on the ground") had no consequence: `Looter.LeftBehind()` counts them (prio >= `loot.LeftBehindMinPrio` 40, last full pickup pass) and the runner raises a trip "loot: pack full, N valuable item(s) left on the ground" when the pack is >= 80 %; it rides the normal trip gates (boss run, no-portal step, cooldown, `packFullQuiet`). After the trip the count is cleared; the bot returns to its task, and the items are picked up by the normal pickup passes if they lie near the return point (U live: does the return land near the loot?).

## Logs
- `inventory pct=.. cells=used/total stashPct=.. stashCells=..` once per chores run (town). The stash figure counts only stash items the snapshot lists (U: whether memory lists the stash while it is closed; 0 would then mean "unknown").
- Minute line gets `invPct` (backpack percent, -1 unknown).

## Live checklist (nothing here ran live)
1. Fill a throwaway pack to 85 % with keys and unidentified rings, go to town: expect `inventory pct` below 70 after the chores and gold up.
2. Drop a unique body armor (2x3) with a fragmented 78 % pack: expect `loot: pack full, item left on the ground`, then the trip reason, then the pickup after the return.
3. Confirm the minute line shows `invPct` and the stash figure in `inventory` reads non-zero with items in the stash while the stash is closed.
4. Confirm a tome absorbs a picked-up TP scroll (S today).

## INVREJUV (2026-10-30)
Soaky clvl 14 G2: pack 87-92 %, trips every 5 min, stash swap "pack has no room". V (read of a Soaky.d2s copy): the pack held 13 identical 1x1 simple items plus 5 of another code and 2 more, i.e. potions/scrolls by the thousand-cell count, no junk; item names not decoded (no Huffman table in the repo, U). Root cause V (code): pickit rule "rejuvenation" matched every rpot unconditionally at prio 75, tier S: always picked (even into the last cells), protected from selling (sellProtected, Keep) and never belted below clvl 31 (belt plan has no RV column) nor drunk from the pack. Fix: rule needs `needed: potion` (prio 65), rpot joins "spare supplies" (tier C), town sells rejuv surplus, `inventory contents` log line names pack and stash per code.

## REJUVUSE (2026-10-30)
SME (EXP, veteran judgment, not sourced): the pack filled because pickup was unconditional; the defect is that only belt potions are drunk. Design chosen: belt route, no pack right-click mid-fight (needs the inventory panel open, costs frames, blocks the screen; U cost). V (code + tests):
- Carry: `loot.rejuvSpare` is now the total cap (belt plus pack), default 3. At the cap a Full is still wanted while a regular one is in the pack. `TestRejuvIsCapped`.
- Town: the sell step sells pack rejuvenations over the cap, regular first, even when the pack is not full (log `rejuvenation sold` reason over cap). `TestFullPackOfRejuvsIsSoldDown`.
- Belt: `FillBelt` moves pack rejuvenations (Full first) into free belt slots after the plan's own candidates, also below clvl 31 (log `belt: moved N potions ... rv a->b`). `TestPackRejuvIsBelted`. U: shift-click puts the potion in the first free slot; if that is an HP column the column is Mixed (BeltOf already flags it, the front potion is drunk by the hotkey), not verified live.
- Drink: belt rejuvenation at life < `combat.drinkRejuvPct` 35 (existing), and new `combat.drinkRejuvBossPct` 50 against danger 1+ packs in a fight (0 = off). Log `potion kind=rejuv`. `TestBossRejuvLine`.
- Restock under 50 %: already `town.tripBeltColumnPct` 50.
