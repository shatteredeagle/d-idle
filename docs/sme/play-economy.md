# Play patterns 2: gold, shopping, inventory and town chores

Target: classic Diablo II LoD 1.14d, offline single player. Not D2R. Written 2026-10-29 by SMEKB-C.
Tags: **V** = fetched this session (URL given) or read in our own code/logs, **S** = snippet or secondary, **U** = memory or inference.
`[D2R]` = D2R-only or 1.14d status unknown.

Sources:
- VEN = Arreat Summit mirror, vendors: https://classic.hiveworkshop.com/diablo2exp/basics/vendors.shtml (V)
- CHR = Arreat Summit mirror, characters (death, gold loss, stash): https://classic.hiveworkshop.com/diablo2exp/basics/characters.shtml (V)
- GAM = Arreat Summit mirror, gambling: https://classic.hiveworkshop.com/diablo2exp/basics/gambling.shtml (V)
- POT = Arreat mirror, potions: https://classic.hiveworkshop.com/diablo2exp/items/potions.shtml (V; heal values are per class there, U for 1.14d)
- NPC = Arreat mirror, Act 1 NPCs: https://classic.hiveworkshop.com/diablo2exp/npcs/act1.shtml (V)
- PDM = PlanetDiablo misc items (archive): https://gamespy-archives.quaddicted.com/sites/www.planetdiablo.com/diablo2/items/misc/index.html (V)
- PDW = PureDiablo "Building Wealth on the Ladder" (v1.10, 2009): https://www.purediablo.com/?p=701 (V, old, realm trade advice)
- MT = Maxroll tips and tricks: https://maxroll.gg/d2/resources/tips-tricks (V, D2R)
- MM = Maxroll mercenaries: https://maxroll.gg/d2/resources/mercenary (V, D2R)
- MS = Maxroll sorceress leveling: https://maxroll.gg/d2/guides/sorceress-leveling (V, D2R)
- KB = Kolbot default config: https://raw.githubusercontent.com/blizzhackers/kolbot/master/d2bs/kolbot/libs/config/Sorceress.js (V, a bot's idea of "sane defaults")
- OUR = our docs: docs/items.md, docs/town.md, docs/resource.md, docs/knobs.md (V live numbers where stated there)

## 1. Gold flow, in the order a human experiences it

1. **Early game is poor.** In Normal Acts 1 the money is: gold piles (small), and selling magic/rare weapons and armor that drop (S: Fextralife/gamepressure snippets, "sell rare yellow gear for good gold"; our live logs: gold 0 to 3 in 29 minutes of Blood Moor because the bot skipped junk pickup and small piles, docs/g1-log.md attempt 5, V).
2. **Selling is the main income in Normal and Nightmare**, gold piles are secondary. Hell Council at Travincal (Act 3) was the classic gold farm: "four to five digit" drops (V PDW, v1.10; U whether still best in 1.14d).
3. **Gold per pile grows with monster level**; from Act 2 on piles of ~100 are common (S, gamepressure beginner snippet, D2R).
4. **Selling caps** (V VEN): one item sells for at most Normal A1 5,000, A2 10,000, A3 15,000, A4 20,000, A5 25,000; Nightmare 30,000 in every act; Hell 35,000. Holding a high-value item for a later act or difficulty pays more. All vendors pay the same price. Repairing before selling does not raise the sale price.
5. **What sells per inventory cell** (V PDW, old): class-specific items (staves of many mods up to ~35k, necro wands, sorc orbs, barb helms, druid pelts), good rare circlets/amulets, keys, chipped gems, throwing potions. A bot should sell by gold per cell, not per item (our `loot.minValuePerCell` 100 does this).
6. **Gold limits**: inventory holds 10,000 gold per character level, the stash up to 2.5 million (V PDW; D2R has the same number, U for 1.14d offline but widely quoted).
7. **Death costs gold** (V CHR): lose a share of carried gold equal to clvl % capped at 20%; "single player: stash gold is never lost, and the first 500 gold per character level is exempt" (page says single player; U whether that is exactly the 1.14d offline rule). Also 5% (NM) / 10% (Hell) of the XP gap (V A4). So a human banks gold in the stash above ~500 x clvl before a dangerous run; our `town.stashGoldPerLvl` 1000 deposits above clvl x 1000 (OUR knobs.md), double that exemption, which is fine.
  **Correction (V live, G1 attempt 12, Soaky clvl 11, i4, 22:56:47):** death took ALL 1125 carried gold (gold 1125 to 0 in the next minute line), far above the 'exempt 500/lvl' reading; the exemption is not applied in this offline game (the docs line above stays U). The bot now keeps only `town.stashGoldBase` 300 + `town.stashGoldPerLvl` 50 per clvl in the pack (850 at clvl 11) and banks the rest in the Chores stash step, which runs after the shop (kit first). Unverified: whether the gold lies on the ground at the corpse (the corpse retrieval walks onto a pile if one shows) or is gone.

## 2. What a human buys first (in order)

Typical ordered list a human would follow in Normal Act 1, derived from the sources below; prices are U unless stated.
1. **Town Portal scrolls (2-4), then the Tome of Town Portal.** Tome: 1-20 charges, 1x2 cell, holds one spell type (V PDM). Real vendor price of the first tome was 450 gold at clvl 3 (V live, OUR items.md; the base cost is 250). Scroll base ~ 100 gold (U, OUR town.md says price equals base; docs say "price on live ~100 is U"). Maxroll tip: buy Tomes of Town Portal and Identify early because "inventory fills quickly" (S, search snippets, D2R beginner guides). Our `town.tpTomeMinGold` is 450 (OUR knobs.md), which agrees with the live price.
2. **Healing potions to fill 2 belt columns, then mana** (casters). Minor healing is the cheapest item in the game, 30 gold live (V OUR leveler.md: "Potion trips are not made with less than 30 gold (cheapest potion, VERIFIED LIVE)"). Maxroll sorceress route: "Life, lots of Mana, and Stamina potions" (V MS, D2R). Shift-right-click on a potion in the vendor fills empty belt slots; also works for TP and ID scrolls (V POT).
3. **A belt** that adds potion slots. Belt class "sash" 8 slots, "belt" 12 slots, plain no-belt = 4 slots in one row (V code comment internal/data/layout.go:75, from belts.txt; the item-to-class mapping per base is U, read `armor.txt` `belt` column). The Arreat page says shift-click moves a potion to its matching belt column (V POT). A human buys the first affordable belt at Charsi/Fara; Maxroll sorc: "Buy a belt from Farah" in Act 2 (V MS, D2R). A belt is also the cheapest big life/defense upgrade early.
4. **Weapon upgrade for the class** (e.g. +skill staff for a caster: 3,400-5,500 gold for +2/+3 Charged Bolt in D2R, V MS). Cheaper options: a Rare/Crafted weapon found on the ground.
5. **Charsi sockets and helm/armor bases** (MS: two 2-socket helmets and one 2-socket body armor, not Magic items, for runewords) is a build decision, not a first purchase.
6. **Antidotes (for Andariel and poison), Thawing (Duriel)**: ~30 each, 1-2 per boss fight (V MQ "Thawing potions add 50% Cold Resistance"; V POT).
7. **Identify**: humans buy a Tome of Identify only when they start using it; in Act 1 they walk to Cain (free once rescued) or sell unidentified items unidentified. Without the Cain rescue the Rogues rescue him, but Cain charges 100 gold per identify on that difficulty (V NPC).
8. **Mercenary** (V hirelings page https://classic.hiveworkshop.com/diablo2exp/basics/hirelings.shtml): Act 1 Rogue free after Blood Raven or at level 8-9 (the Arreat NPC page says level 9; the hirelings page and MM say 8; U which is right in 1.14d), hire cost from the hire dialog (our live reading: clvl 13 Desert row cost 360, V OUR items.md).

Never buy: Magic items from vendors at their expensive "Magic" price early (live: a magic Wand cost 1,669 vs 205 for a normal one, V OUR leveler.md); wait to find or gamble.

## 3. Selling, repair, stash and inventory rules

### Selling (what to sell)
- Sell by Ctrl+Left-click (no prompt) is the human accelerator (V VEN). Buy-back works sometimes (V VEN).
- Sell: all normal/superior weapons and armor not an upgrade, low-value magic items, unidentified items after a quick look when poor, extra potions of low tier, gems below the cube recipe needs.
- Keep (stash, do not sell): runes (all), gems for the Cube, rings/amulets/charms that might be useful, Horadric Cube (never sell), quest items, class charms.
- Item values: no formula verified for 1.14d in this session. OUR calibration logger idea (docs/items.md P9) is the V way to get real numbers.

### Repair
- Humans repair at the act's smith when any worn item is below ~20-40% durability: Kolbot defaults `RepairPercent` 40 (V KB), our `town.repairBelowPct` 30 (V knobs).
- Charsi A1, Fara A2, Hratli A3, Halbu A4, Larzuk A5 (V KB town task table from our prior-art 05-town.md). Hratli's repair is skipped by humans and bots in favour of Act 2 or 4 (V prior-art/05-town.md, KB behavior).
- Repair cost ("varies", S) rises with item level and missing durability; 1.14d formula U.
- Ethereal items cannot be repaired once at 0 (V MT). Throwing weapons restore quantity via repair.
- The Cube can repair/recharge: Ort rune repairs weapons, Ral armor (U/S in KB comment "CubeRepair, don't use if you don't understand the risk").

### Stash and inventory
- Backpack 10 x 4 = 40 cells, stash 6 x 8 = 48 cells (U for 1.14d; KB comments say "Stash container 6 columns", Inventory "4 rows", V KB, PDW says 10 columns for modded). Humans keep the backpack for: TP tome (1x2), ID tome (1x2), Cube, charms, and ~6-10 cells of loot space.
- Kolbot's default inventory lock grid has every cell "0 = locked": "If you don't change the default values, the bot won't stash items" (V KB). The lesson: the bot needs an explicit notion of which cells are keepers.
- Trip trigger: our `town.tripPackUsedPct` 80 and `loot.freeCellsKeep` 2 (V knobs). Human practice: leave town with 20-30 free cells; return when you cannot carry a 2x4 weapon/armor, not at 80% of cells.
- Charms live in the pack and give permanent bonuses; humans put them in a fixed corner and never sell them (S).

### Gold stash policy
- Deposit when carrying more than the death exemption; do not hoard in the pack (death loses a share, see section 1.7). Kolbot `StashGold` 100000 is a late-game number, not Normal advice (V KB).

## 4. Gamble

V GAM (Arreat): price rises with character level; item level of a gamble = clvl - 5 to clvl + 4 (min 5); odds: Magic 89.85%, Rare 10%, Set 0.10%, Unique 0.05%; Exceptional/Elite possibility depends on item level above base quality level, 1% base + 0.9%/0.33% per level of gap; a base appears on the gamble screen when clvl reaches its quality level - 4 (Exceptional -5, Elite -7); no effect from MF, players, act or difficulty. Stock refreshes each time you open the gamble window (V VEN).
When humans gamble (S/V mix):
- **Not while poor and low level.** PureDiablo (2009): gamble "when your stash is filled to exploding", with enough kept for the merc revive (V PDW). Gold-starved characters do not gamble.
- Preferred targets: amulets, rings, circlets, coronets (V PDW, V KB `GambleItems`); the high-ilvl thresholds (circlet ~clvl 92, amulet 95, ring 91) are from D2R sources (S InvenGlobal snippet) and match the item-level rule above.
- Gheed's Fortune charm reduces costs [D2R discussion; the same charm exists in 1.14d] (S).
- Bot rule of thumb: no gambling before clvl 40 and gold > 100k, or never (safe default). Kolbot starts at 1,000,000 gold (V KB `GambleGoldStart`).

## 5. Shop-run basics and vendor stock

- Vendor stock regenerates when all players have left town; leaving town to the wilderness resets it, moving between towns of one act does not; in a game with other players, after 4 minutes without trading (V VEN). In a solo offline game, a run to the field and back refreshes the stock. Humans use this ("shop run": leave, return, re-open, re-scan) to look for rares, particularly Charsi/Akara in Act 1 and Fara/Drognan in Act 2 (V VEN; the "shopbot" tools do it automatically, KB `MiniShopBot`, `Scripts.ShopBot`).
- Higher clvl, act and difficulty unlock better stock; in Hell, Act 1 and 2 vendors do not upgrade (except throwing weapons) (V VEN). Gambling is unaffected.
- Normal potions: Akara (A1), Lysander (A2), Ormus (A3, healer and shop), Jamella (A4), Malah (A5) are the potion sources (V from our prior-art/05-town.md KB table; Malah/Jamella carry the higher tiers). Humans buy potions for the next act in the act they currently are in only if stock is poor; many walk to Act 4 for Jamella (U).
- Healing potion tiers and per class heal amounts (V POT page; U that 1.14d keeps the per-class table): sorc/necro/druid Minor Healing 30, Light 60, Healing 100, Greater 180, Super 320; amazon/paladin/assassin x1.5, barbarian x2 (rounded). Mana: Minor Mana 40 for sorc, 20 for barb, 30 amazon/paladin/assassin. Rejuvenation restores 35% life and mana instantly; Full Rejuv 100% (V POT; "heals instantly", better than the over-time potion in a crisis).
- Potion over-time restore takes seconds; a human uses a quaff gap (~1 s for healing, Kolbot potion cooldown 1 s, Rejuv 0.3 s, V KB) and drinks a Rejuv at low life (Kolbot `UseRejuvHP` 40, `UseHP` 75, V KB). Ours: `combat.drinkLifePct` 55, `drinkRejuvPct` 35 (V knobs).

## 6. Order of town chores (a human)

KB order from the Kolbot Town module (our prior-art/05-town.md): heal (Akara free), identify, clear junk (sell), fill TP tome (and ID tome), buy potions, repair, gamble (only when rich), revive merc, cube, stash gold and items, sort inventory, return to field. Skip a step if its need check says no (needPotions, needRepair, needStash, needMerc). Our order: heal, identify, sell, shop (potions + scrolls), repair, stash/gold (V OUR town.md). No functional difference; the human difference is that **TP before everything else** and **sell before shop** (so shop has gold).

## 7. Where our bot's economy differs from what a human does

| Topic | Human | Our bot | Note |
|---|---|---|---|
| First purchase | TP scroll or tome, then potions | `buyFirstTP` then potions; tome at 450 gold (V OUR town.md, knobs.md) | Same |
| Belt filled | Always, 2 hp + 1 mp + rejuv columns | G1 attempt 6: 3779 gold, 0 potions used, no TP/potion purchase in the whole log (V docs/g1-log.md) | **Bug in bot, not strategy**: human would never hold 3.7k gold with an empty belt |
| Loot of low-value gear | Picks and sells everything above ~10-20 gold per cell when poor | Poor mode added (`loot.poorGoldBelow` 300, `poorMinSell` 6) after the gold-stuck run | Matches; the pickit mistake was found live (V g1-log.md) |
| Stash timing | Before dangerous runs; gold above exemption | `stashGoldPerLvl` 1000, `carryValuables` 0 | OK |
| Gamble | Rich only | Not implemented (Gold farm only) | OK |
| Shop for rares | Re-roll vendor stock | none | Optional improvement; not needed for survival |
| Merc revive | Always before next fight; pay up to ~25k at hlvl 57 | `town.Revive` gated behind unverified flag (V docs/progression/merc.md) | Gap, see play-safety |
| Repair threshold | 20-40% | 30 | Same |
| Pack-full trip | When an item no longer fits | 80% of cells | Bot is earlier; fine |
| Sell cap awareness | Holds items worth more than the act's cap | no | Minor |
| Identify | Cain (free) or tome | has Cain rescue off by default; ID scrolls 2 | Bot ID costs gold; ok |

## 8. Open questions

1. 1.14d vendor prices for TP/ID scroll, potions by tier, repair, hire cost (calibrate live, our docs/items.md P9 logger).
2. Whether the single-player death gold rule on the Arreat page (stash never lost, first 500/clvl exempt) holds in 1.14d offline: test on a throwaway with `cheat`-set gold and a deliberate death.
3. Belt capacities by belt type (U): read from items.txt/`belts.txt`.
4. Whether Kashya hires at level 8 or 9 in 1.14d (A NPC page 9, hirelings page 8).

## Correction: gold on death (GOLDSWEEP, S, source caveat)
From the d2decomped notes (docs/plan/plan-v2.md 10.6; secondary source, our live probe is the acceptance test and is still to do): on death the loss is min(clvl,20) percent (never below clvl x 500 in total) and the rest of the purse drops as a GROUND PILE at the death spot, the corpse holds only worn items, the stash is safe. This replaces "death takes all carried gold" (our logs showed 0 because nothing swept the pile). Consequences: a big purse makes the corpse trip worth it, keep the purse small (stash rule in section 3) because the loss percentage and the trip risk scale with it, and the bot sweeps the pile (docs/deathspiral.md "Gold on death").
