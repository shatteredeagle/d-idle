# 05 Town

Sources: KB:core/Town.js (spots 17-100, doChores 155-230, initNPC 351-520, buyPotions 530-660, identify 796-910, repair 1390-1430, reviveMerc 1452-1505, stash 1531-1640, move 2054-2210, goToTown 2213-2275, visitTown 2276-2330), KB:core/Me.js (needPotions 345, needRepair 629, needHealing 568, needStash 588, needMerc 605, getTpTool 509), KB:core/Storage.js, KO:internal/action/pre_run.go, TY:docs/features/town-services.md.

## 1. Order of chores (`doChores`)
KB order (called after every run start/end and from `visitTown`):
1. goToTown if outside; require `gameReady` and in town within 2 s polls or fail.
2. Burst of Speed if assassin; switch to primary weapon.
3. `heal` (only if `needHealing`: hp% <= HealHP or mp% <= HealMP, or (HealStatus) poisoned/amplified/frozen/weakened/decrepified/lowered-res).
4. `identify` (Cain via "CainID" option, else shop NPC with scrolls/tome).
5. `clearInventory` (sell/drop junk not in the locked-slot grid).
6. pick up items on the ground in town (dropped).
7. `fillTome(TP tome)` (also keep tome of identify when field-ID enabled).
8. `buyPotions`.
9. `shopItems` (mini shopbot on the open shop), `buyKeys`.
10. `repair`.
11. `gamble` (if gold >= GambleGoldStart; stop at GambleGoldStop).
12. `reviveMerc`.
13. Cubing, runeword making.
14. `stash` (items then gold).
15. quest-item check, clear scrolls (remove excess), sort inventory.
16. return to the act we started in; close all UI; recast buffs.
KO order (`PreRun`/`InRunReturnTownRoutine`): reset stats, spend stat points, spend skill points, recover corpse, identify all, stash, vendor refill (potions/scrolls), gamble, stash again, ensure skill bindings, heal, revive merc, hire merc, repair, take portal back.
Rule of thumb: skip each step if the need check says no (avoids walking): needPotions, needRepair, needStash, needMerc, unidentified items.

## 2. NPCs per act (KB `Town.tasks`)
| act | heal | shop (potions, scrolls, tomes) | gamble | repair | merc | keys | identify |
|---|---|---|---|---|---|---|---|
| 1 | Akara | Akara | Gheed | Charsi | Kashya | Akara | Cain |
| 2 | Fara (Atma if closer) | Drognan | Elzix | Fara | Greiz | Lysander | Cain |
| 3 | Ormus | Ormus | Alkor | Hratli | Asheara | Hratli | Cain |
| 4 | Jamella | Jamella | Jamella | Halbu | Tyrael(no hire; A4 has none) | Jamella | Cain |
| 5 | Malah | Malah | Anya | Larzuk | Qual-Kehk | Malah | Cain |
Policy details: potions in Normal when Act 4 reachable: go to Act 4 (Jamella sells better potions); repair in Act 3 -> walk to Act 4 or 2 (avoids Hratli's travel); reviving in Act 3 -> go to 4 or 2 (avoid Asheara); gambling in Act 4 not at Jamella ("Jamella gamble fix"). Rule `justUseClosest`: when only selling/clearing inventory, use whichever of key/repair/gamble/shop NPC is closest (choose set by which needs exist).
Failure handling: if NPC unit not found ("getUnit bug"/NPC not loaded), move to its spot and retry; if still failing, travel to a random other act town (<= highest act) and retry.

## 3. Town coordinates (KB `Town.act[n].spot`, world coords)
| act | spots |
|---|---|
| 1 | derived from the town fire preset (A1TownFire at (fx,fy)): stash (fx-7, fy-12); Warriv (fx-5, fy-2); Cain (fx+6, fy-5); Kashya (fx+14, fy-4); Akara (fx+56, fy-30); Charsi (fx-39, fy-25); Gheed (fx-34, fy+36); portal spot (fx+10, fy+18); waypoint = preset A1Waypoint (roomx*5+x, roomy*5+y). Note: Act 1 town layout is one of 4 presets: offsets above are relative to the fire and apply to all; TY found exit side differs per game |
| 2 | Fara/Cain (5124,5082); Lysander (5118,5104); Greiz (5033,5053); Elzix (5032,5102); Jerhyn (5088,5153); Meshif (5205,5058); Drognan (5097,5035); Atma (5137,5060); Warriv (5152,5201); palace (5088,5153); sewers (5221,5181); portal (5168,5060); stash (5124,5076); waypoint (5070,5083) |
| 3 | Meshif (5118,5168); Hratli (5223,5048) or after quest (5127,5172); Ormus (5129,5093); Asheara (5043,5093); Alkor (5083,5016); Cain (5148,5066); stash (5144,5059); portal (5150,5063); waypoint (5158,5050) |
| 4 | Cain/Tyrael (5027,5027); Halbu (5089,5031); Jamella (5088,5054); stash (5022,5040); portal (5045,5042); waypoint (5043,5018) |
| 5 | Larzuk (5141,5045); Malah (5078,5029); Cain (5119,5061); Qual-Kehk (5066,5083); Anya (5112,5120); Nihlathak (5071,5111); stash (5129,5061); portal spot (5098,5019); portal (5118,5120); waypoint (5113,5068) |
Hard-coded Act 5 paths (A* misbehaves in A5 town): WP -> portal spot: (5113,5068),(5108,5051),(5106,5046),(5104,5041),(5102,5027),(5098,5018); Malah -> stash/WP: (5089,5029),(5093,5021),(5101,5027),(5107,5043),(5108,5052); portal spot -> stash/WP: (5102,5031),(5107,5042),(5108,5052).
Act 2: if inside Atma's house (x>5122,y<=5049) and Atma stands at x 5136..5137,y 5048..5051 she blocks the door: use side door nodes (5140,5038),(5148,5031),(5154,5025),(5161,5030).
`moveToSpot`: target within 5 of spot already = done; for NPCs stop when the NPC unit exists within 5; for stash stop when the stash object is visible; waypoint: "long range" when no telekinesis: first A* node of an 8-spaced path; up to 3 attempts; third attempt disables telekinesis.
Mapping: town NPC and object ids come from `internal/data` (monstats.txt / objects.txt; Kolbot resolves NPCs by localized name, we should use txt ids). UNVERIFIED: stash and waypoint object ids per act (Kolbot constants `sdk.objects.Stash`, A1Waypoint etc.; read them from objects.txt).

## 4. NPC interaction protocol
`npc.openMenu()` = approach to within ~5 (KB `moveToSpot` stops at NPC distance < 5), flash (self-select) + 40 ms, click NPC until the NPC dialog UI flag opens. Then `startTrade(task)` clicks the "Trade" (or "Repair") entry; UI flag `Shop`. Tasks: Shop/Repair/Gamble -> trade menu; Key -> same (Hratli A3 uses the Repair trade menu); CainID -> "Identify Items" menu entry then close; Heal -> opening the dialog heals instantly on contact; Atma is not a shop NPC (closing dialog right away avoids a crash).
KO (D2R) uses keyboard menu sequences ("home, down, enter") after hover-confirmed NPC click: `step.InteractNPC` hovers NPC body (screen projection + hover verify), clicks, waits >= 1 s for UI flag `NPCInteract`; Tyrael hit-box special (aim 20 px above); fails if NPC > 15 away.
Our `input`: NPC menu entries are text lines at fixed 800x600 positions: first entry y ~ 2xx; calibrate live per NPC (UNVERIFIED).

## 5. Potions and belt (`buyPotions`)
Belt layout: 4 columns x (1..4 rows by belt: sash/light belt 2 rows? (KB `BeltSize` 2 for "lbl,vbl", 3 for "mbl,tbl", 4 for others, 1 none). Config `BeltColumn` = ["hp","hp","mp","rv"] (SP default after build: ["hp","mp","mp","rv"] or ["hp","hp","mp","mp"]), `MinColumn` = [3,3,3,0] per column (refill when column below).
Algorithm:
1. skip if gold < 1000.
2. `clearBelt`: move wrong-type potions (and stamina/antidote/thaw) out of belt (columns that don't match config) to inventory.
3. compute per-column missing count; belt need if any hp/mp column has more than `beltSize - MinColumn[i]` missing; buffer need if inventory has fewer than `HPBuffer`/`MPBuffer` potions of that type.
4. at an Akara-type potion vendor: for each column with missing > 0, pick the best potion in the shop (highest tier first: `hp5..hp1`, `mp5..mp1` names) and buy: if shifting is allowed (no empty rejuv column and only one type being filled) shift+click buy fills the whole belt row-wise ("shift buy"), else buy one at a time `missing` times. Recompute columns after each since one shift-buy can fill several columns.
5. buffer: buy `HPBuffer - have` and `MPBuffer - have` into inventory if they fit.
6. rejuv slot empty and buffer needed: fill the rv column with thawing/antidote/stamina (single-slot trick).
Which tier to buy: highest tier available (KB). SP notes: Normal chars buy at Akara/Fara; early levels buy only HP (BeltColumn all "hp" at level < 12), mana later.
`needPotions` (field check): only if MinColumn configured; level > 2 and gold > 1000; true when no hp or no mp potion in belt (if configured) or a configured column (x position) has zero potions.
TP scrolls: `fillTome`: need gold >= 450; if tome quantity < 13 buy scrolls `buy(true)` (shift = full stack up to 20); buy a tome (Tome of Town Portal) if none; keep tome (identify) too if FieldID. Scroll auto-pickup only if a tome exists with quantity < 20.
KO belt rules: potion keys 1..4 correspond to belt columns, drink the first potion of the type in the column list; refill when current healing < 75 percent of target (columns x rows) or mana < 75 percent; merc potions: shift+key.
Mapping: belt = item list at location "belt" with x = slot index (column = x % 4, row = x / 4); `internal/d2` inventory page enumerates by `ItemData` location (mem 2.9).

## 6. Repair, merc, stash, identify
- `needRepair`: affordable (gold >= repair cost) and any equipped non-ethereal, non-indestructible item with durability% <= `RepairPercent` (config; SP ~ 25? UNVERIFIED), charged/quantity items (javelins, throwing weapons) below %, or arrows/bolts missing -> buy quiver. Repair via NPC Trade window "repair all" button (KB `me.repair()`), Act 3 -> go to 4 or 2 first. Cube repair alternative (Ort+Ral+item) exists.
- Merc revive: needMerc = expansion, `UseMerc`, gold >= revive cost, and no living merc (3 checks 100 ms apart). Click dialog lines containing ":" (menu entries) up to 3 rounds; after each wait max(750, 2*ping); if the "not enough gold" text appears -> give up; then wait up to 2 s for merc unit to exist. After revive, re-precast to buff merc if MercWatch.
- Stash (`stash`): move to stash object (telekinesis allowed), click it, wait <= 5 s for UI flag Stash, +100ms+ping*(i+1) for UI init (5 tries). Items eligible = inventory items outside the lock grid (`Config.Inventory` grid of 0/1 per slot: 1 = free to be touched, 0 = locked: charms, keys, tomes) and not type in ignoreTypes (quiver, tome/books, scrolls, keys, potions, stamina/antidote/thawing). Stash only items the pickit says WANTED/CUBING/RUNEWORD/CRAFTING (result between UNWANTED and TRASH) or kept by systems. Gold: stash when gold >= `StashGold` (SP: level*1000) and bank < 2.5M (25e5): click gold button (gold(amount,3)), wait 1 s.
 Stash layout: classic 6x4, expansion 6x8 (PlugY 10x10); inventory 10x4; cube 3x4; belt 4x(1..4). Container placement uses first-fit scanning (`Container.FindSpot`) with sort modes (`SortItems`) - not read in detail.
- Identify: skip unless a wanted-unid or sellable trash item exists (or LowGold). Preferred: Cain (when `CainID.Enable`, gold >= MinGold, >= MinUnids unid items, no kept unid items): talk to Cain, "identify items", then drop unwanted ones on the ground (not sell). Else with tome of identify (right-click tome -> identify cursor (cursor type 6) -> click item; 3 tries each, 270 ms delay after cursor appears) or scroll bought from NPC. After ID: kept (WANTED) logged, CUBING/CRAFTING updates lists, everything else sold (sell timing: `sellTimer` guard ensures >= 500 ms between sells).
- Sell: shop window, ctrl? click item with right-click? `item.sell()` = click in shop with the sell cursor; UNVERIFIED mechanics for our input (in D2 1.14d: right-click item in inventory while Shop open sells it? left-click item, then click NPC area? Actually alt: with shop open, left-click on inventory item picks it and a second click in the shop area sells; right-click sells directly in 1.14d? UNVERIFIED).

## 7. Gamble
Config `Gamble`, `GambleItems` list (Amulet, Ring, Circlet, Coronet, ...), start at gold >= `GambleGoldStart` (SP 1,250,000; stop at 750,000 `GambleGoldStop`). Loop: open NPC gamble, refresh by leaving/re-entering? (KB loops items), buy items from the list, identify result via the gamble purchase (gambled items come identified), keep if pickit WANTED else sell. KO has `gambling.go`. Skip in MVP; plan: later with refined pickit.

## 8. Inventory management
- Lock grid `Config.Inventory[4][10]` marks slots reserved for TP tome, ID tome, keys, charms (SP sets 1 everywhere = all free, charms managed by CharmEquip).
- `clearInventory`: drop/sell items in free slots not wanted; `Pickit.canMakeRoom` triggers a town visit when a wanted item does not fit and junk exists (see 06).
- Item on cursor: dropped on the ground in `Pather.move` and `Cubing.cursorCheck` returns it to inventory.
- `Storage.Inventory.UsedSpacePercent`, `IsPossibleToFit(item)` (checks if any empty rectangle exists for size w x h in empty grid).

## 9. TP and portal safety in town
- Always carry >= 1 TP; low threshold: tome < 13 scrolls triggers buy; if no TP tool and no way to town: quit game (KB `goToTown` error path: "no tps available. Restart" -> `scriptBroadcast("quit")` when alone).
- Town visit from field timing: do chores, return to portal spot, wait until 4.5 s since leaving field, then use own portal (fall back to any portal to the previous area).

## 10. Mapping
| step | package | data |
|---|---|---|
| NPC location | `bot/town` | NPC unit by txt id (monster list in town), fallback hardcoded spots above; stash/WP objects |
| UI flags | `d2` | NPCMenu, Shop, Stash, Cube, Inventory, Waypoint (mem 2.11) |
| inventory grid | `d2` | items with location, x, y (mem 2.7/2.9); `internal/data` base item sizes (invwidth/invheight from armor/weapons/misc.txt) and costs |
| buy/sell | `input` | shop item slot -> screen coordinates (shop grid 10 wide, 800x600 classic coordinates; calibrate live) |
| gold | `d2` | stats Gold (id 14), GoldBank (15) |
| repair cost | `d2` | `getRepairCost` is a client function (not memory): approximate via UI text? alternatively always attempt repair when durability% <= threshold and gold > estimate |
