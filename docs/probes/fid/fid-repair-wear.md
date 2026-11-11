# fid-repair-wear (TASK-317, matrix row IT08): repair cost and durability loss, sized for the repair trip

> status: finished 2026-11-11 by agent decomp9 (Sonnet), offline. Result: **the 1.14d repair formula and the wear rates ARE verified (they were not collected in one place); nothing new deviates.** This card adds the join of the two existing proofs and the sizing table the repair trip needs. No PR candidate. Consumer: TASK-226 (repairmanual). Nothing was re-run for the formulas (the proofs below ran on the worker host and are cited, tag S); the NEW numbers here are D (tables) arithmetic on those formulas.

The question behind the card ("the SME said we have no verified 1.14d repair formula") is answered by two finished traces that the SME did not have in view: [price-trace.md](../../research/price-trace.md) (formula, 100,000 cases, V run) and [durability-trace.md](../../research/durability-trace.md) with [verifypaldur.md](../../research/verifypaldur.md) (wear, V run of the real wear function, independent check). Both are listed under "Existing evidence" so a consumer finds them from this page.

Tags: V run, V read, D (MPQ tables of the 1.14d stack, read with `tools/emu/mpq-table.sh`), S (existing proof), I, U.

## Q1. Repair cost formula (S, V run in price-trace.md and fid-price-pr.md)

- FUN_0062efb0 mode 3: for an identified, non-ethereal item with durability (FUN_00629930: durability > 0, nodurability 0, max > 0, stat 152 not set), **cost = ceil(R x (max - dur) / max) x rep mult / 1024**, R = the item's base cost (armor scaled by actual defence / maxac, a magic or rare item adds its affix costs; quiver and throwables: other rules in price-trace.md Q1). Whole items price 0 (Repair All skips them; a single repair of a whole item prices 1).
- **rep mult is 128 at every smith and every other NPC** (npc.txt `rep mult` column, D, read here: gheed, charsi, akara, lysander ... nihlathak all 128), and the quest-A discount touches the buy side only (npc.txt `questrepmult A` = 1024 = no change, D). So: **repair = cost x missing / max / 8** in every act (the price-proof cases cover the three difficulties and 17 vendors and d2d equals; I: no difficulty term in the formula, since none appears in its steps).
- Gate (FUN_0062e660, V run price-trace.md D10, fixed upstream): unidentified and ethereal items cannot be repaired (price 0, the server refuses).
- The oracle on current upstream main: 100,000 cases seed 7 repair differ 13 (all 32-bit overflow of price x missing above 2^31 at prices of 26 million gold and more, unreachable: I), seed 1: 16, edge cases 31; tomes 0 (fid-price-pr.md section 1). So d2d equals game.exe for every price a character can meet. S.

## Q2. Durability loss rules (S, V read + V run of the wear function in durability-trace.md / verifypaldur.md)

| Event | Rule | Tag |
|---|---|---|
| Your melee blow lands | `FUN_0057d4f0` (the melee hit-record handler) calls `FUN_0057d3d0` only when the record's hit bit is set; the active weapon (FUN_00535bc0) loses 1 durability with chance **4 percent** (normal weapon), 10 percent for a throwable used in melee (Expansion). Bows, crossbows and missile attacks: 0. A weapon that reaches 0 stays equipped; the "broken" flag (0x100) is set at the end of the attack animation (FUN_00580380). | V read + V run (the roll and ladder with 64,000 cases, 0 mismatch) |
| A monster's melee blow lands on you | one of your armour pieces is chosen and loses 1 durability with chance **10 percent**; pick = ring walk from rand(7) over the slots weighted **torso 5, head 3, shield 4 (a weapon in the off hand weighs 0), right arm 4 (if armour), belt 2, boots 2, gloves 2** among the worn pieces. The expected loss is therefore **0.10 durability per landed melee blow, split by weight over what you wear** (when nothing is worn the walk finds no piece). | V read + V run (34,000 armour cases) |
| You block, dodge, avoid or evade a blow, a blow misses, a missile or area hit lands | **no wear** (the hit bit is not set; missiles use FUN_005ad730 and never call the wear). d2d wore armour on monster missiles and explosions in the version the author read (his D1, only I); not re-checked on current main here. | V read (melee side), I (missile side) |
| Smite and Zeal strikes | each strike is a normal melee hit record (Smite sets the hit bit at 0x5cebb0): **every Zeal strike is its own 4 percent roll**, so a Zeal cast of 5 strikes wears the weapon like five attacks. Smite wears the weapon as well (the shield is the damage source, not the wearing item: durability-trace.md). | V read |

## Q3. Sizing for the repair trip (D arithmetic on Q1 and Q2)

Hits to reach the bot's trip line (`tripRepairPct` 20 percent: 80 percent of the durability gone) and the gold to repair from that line; Normal difficulty, plain items, armour taken at its maximum defence (the price scales down with actual defence / maxac, so these are upper bounds). Armour set worn: Quilted Armor, Buckler, Leather Boots, Light Belt (weights 5 + 4 + 2 + 2 = 13), so a piece takes 10 percent x weight / 13 of the landed monster melee blows.

| Piece | cost | durability | loss per landed blow received | blows to reach 20 % | gold from 20 % | gold from 0 % |
|---|---:|---:|---:|---:|---:|---:|
| Quilted Armor | 140 | 20 | 3.85 % | 416 | 14 | 18 |
| Buckler | 68 | 12 | 3.08 % | 312 | 7 | 9 |
| Leather Boots | 80 | 12 | 1.54 % | 624 | 9 | 10 |
| Light Belt | 192 | 14 | 1.54 % | 728 | 21 | 24 |

Weapon (4 percent per landed blow of yours; every Zeal strike counts):

| Weapon | cost | durability | landed blows to reach 20 % | gold from 20 % | gold from 0 % |
|---|---:|---:|---:|---:|---:|
| Short Sword | 72 | 24 | 480 | 8 | 9 |
| Scimitar | 274 | 22 | 440 | 28 | 35 |
| Mace | 463 | 60 | 1,200 | 47 | 58 |
| Broad Sword | 1,029 | 32 | 640 | 105 | 129 |
| Flail | 1,412 | 30 | 600 | 142 | 177 |

Reading: with starting-level gear a full repair of everything costs **under 60 gold** and the first piece reaches the 20 percent line after about 300 landed blows received or 440 to 480 landed blows of yours. For a Paladin on Zeal (5 strikes a cast once the skill has 4 points) the weapon line comes about five times sooner per kill than with plain Attack. A trip for repair alone is not worth its time cost until the gear is worth hundreds of gold; repair is a by-product of the TP trip. The costs are base `cost` columns (armor.txt, weapons.txt, D); magic and rare gear raises R (price-trace.md Q1) and the bot's `data.RepairPrice` already adds it.

## Deviations (d2d against game.exe)

None new. Carried: overflow above 2^31 (unreachable, I); the armour wear on monster missiles (durability-trace.md D1, still I; differential: yes, drive d2d's `Fight::wear_armor` with a missile hit and compare with FUN_0057d4f0's gate: the game side never calls it). No PR candidate.

## Bot findings (described, not changed)

- B1 `internal/data/price.go:76-87` `RepairPrice` implements exactly Q1 (rep mult 128, missing / max, identified / ethereal / quiver gates, whole item prices 1). Equals the proof's table. The old finding "estimate = cost x missing% / 4" (price-trace.md B1) is fixed; `internal/town/chores.go:302-324` sums the equipment through it. No change.
- B2 `internal/env/tuning_town.go:67` `TripRepairPct: 20`. By Q3 the first piece reaches 20 percent only after hundreds of blows, and the total repair is tens of gold: 20 is a safe line, and the bot never needs to hurry for gold. The risk the line guards against is a weapon at 0 (it keeps swinging; whether a worn-out weapon still deals its damage was not traced: U).
- B3 `internal/town/shop.go:149-155` `durabilityPct` divides integer durability by max durability: a piece at 1 of 60 reads 1 percent, fine; an item whose max durability stat is absent is skipped (`!ok`), which is right for indestructible and no-durability items.
- B4 the bot does not distinguish a wear from landed-hit counts; nothing to change: Q2 only matters for planning (sizing), not for the check.

## Drill recipe (fork verbs; written, not run)

1. Wear rate: lab character in melee with a training dummy monster (`debug spawn` a Zombie), `debug stat` unused; read `dump equip` durability of the weapon and each piece after N landed blows: expect weapon 4 percent of landed blows (about N/25), armour total 0.10 x blows received split torso 5/13 etc. A 2,000-blow run gives the weapon loss 80 +- 9 (binomial).
2. Repair price: at Charsi, `debug equip` a worn item to 50 percent (`debug wear`), read the Repair quote in `menu`: expect cost x 0.5 / 8 (Short Sword 72 -> 4 or 5).
3. Zeal: the same weapon loss after 1,000 Zeal strikes: expect about 40.

## Matrix-cell updates (proposed text)

- IT08: append "UPDATE 2026-11-11 (decomp9, fid-repair-wear.md): the formula (rep mult 128 everywhere, cost x missing / max / 8) and the wear rates (weapon 4 %, armour 10 % split 5/3/4/4/2/2/2, melee hits that land only) are joined with sizing; nothing new deviates; the bot's RepairPrice equals the proof."
- ST05/IT07 untouched.

## Limits

- No emulator re-run: the formulas were run by price-trace, fid-price-pr and verifypaldur (S). The sizing table is arithmetic (D) on their numbers.
- Whether a worn-out weapon still deals damage, and the unit-level effect of the broken flag on stat sums, is U.
