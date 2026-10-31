# Early gear policy, clvl 1-15

Tags: SRC = the user (an experienced D2 player) and the d2-sme veteran agent (agreed about 85%), EXP = needs a live check, V = read from code or logs.
Code: `internal/resource/earlygear.go` (needed-weapon table, `Early`), `internal/resource/gear.go` (gate), `internal/town/rebuy.go` (`slotWanted`, `foundGear`), `internal/loot/wear.go` + `loot.go` (`wearsBetter`), knob `resource.earlyGearMax` (default 15, 0 = old rules).

## Why
V (docs/g1-log.md attempts 6-11): the bot spent hours trying to buy Quilted Armor for a clvl 9-11 Sorceress and gated quests on "no body armor". SRC: in the first 10-15 levels nobody targets item bases. Armor is a few defense points; the gold and the walking cost far more than they return. What kills a low-level character is no potions, no portal and a missing skill weapon, not a missing Quilted Armor.

## The policy (SRC)
- Wear what drops if it is better and wearable; sell the rest. Never make a trip or gate a quest for armor.
- Required, per class (data: `resource.neededWeapon`, tested):

| class | clvl 1-15 needs |
|---|---|
| Sorceress | nothing (ignore all armor; a wand or shield is worn if it drops) |
| Necromancer | nothing (take a wand or shield if one drops) |
| Druid | nothing |
| Amazon | any bow, crossbow or javelins; buy the cheapest if none dropped by about clvl 3 |
| Paladin, Barbarian | any usable melee weapon; Paladin takes any shield |
| Assassin | any usable melee weapon (claws or katar) |

- Helm, armor, gloves, boots, belt, rings, amulets: equip-if-better only.
- Gold order: Tome of Town Portal, then potions (Akara health and mana), then a belt (Fara, Act 2). Never buy vendor magic items early.
- Targeted gear starts at Nightmare entry (resists, raw 60 target, the -40 penalty), the Act 2-3 belt, and life. Those rules are unchanged and live in docs/gear-gate.md and the leveler.

## What the code does
- Gate (`resource.GearGate`): below `earlyGearMax` armor adds no risk points and a missing weapon counts only for a class in the table; a caster in Normal never counts armor at any level. "No potions and no gold" stays HIGH. Missing weapon of a needing class stays HIGH and a rebuy reason. Quest blocking for a naked Sorceress in Normal is gone (tests: `gear_test.go`, `bot/gear_gate_test.go`).
- Rebuy (`town.Rebuy`): below the limit only the weapon slot of a needing class is bought, potions as before. An Amazon buys only a bow, crossbow or javelin (a melee weapon is no kit). At or above the limit the old tiers apply (`town.rebuyGearTier`), so the armor path stays reachable.
- Equip-if-better: loot keeps an armor piece when its slot is empty or its `WearScore` beats the worn item (defense, +10 per +all skills, resists, max life; weights are U); town wears it (`EquipFound`, the swapped-out piece goes to the pack and is sold by the normal chore). Requirements (Str, Dex, clvl, class) must be met; no shield with a two-hander. Weapons are only taken for an empty weapon slot.
- Trips: the "wear" trip reason fires only for a needed weapon in the pack with no weapon worn ("found weapon to wear"). Armor is worn when the bot is in town anyway (the chore step runs in every chore visit).

## Unverified (U / EXP)
- Item stat ids used by `WearScore` (armorclass 31, allskills 127, poison res 45) and whether item max life is 24.8 fixed point: U, EXP (read a worn magic item's stat list live).
- Rings and amulets are not worn automatically yet (no empty-slot rule for them).
- Live: a naked Sorceress at clvl 9-11 now proceeds with quests; not run live.
