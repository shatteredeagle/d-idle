# Early game policy specification

## Purpose
What a fresh character does from clvl 1 to about 12: take drops, Den of Evil first, no armor rebuy loops, a cap on rejuvenation potions. Source: docs/earlygame.md (audit, offline V), docs/inventory.md, docs/beltfill.md, docs/recovery.md, plan-v2 sections 10.1 and 11; merges earlygame 4553bae, earlyfix2 a7de502, invrejuv 40cb988, rejuvuse aec4ca1.

## Requirements

### Requirement: Den of Evil first
A fresh character SHALL do the Den of Evil before anything else in Act 1 (campaign entry floor clvl 1, kit exception only for a character without scroll and potions below clvl 5) and take the skill point reward; it is optional in the game but always done first. Source: plan-v2 section 10.1 (user, 2026-10-30); docs/earlygame.md section 2 (V offline TestFreshCharacterStartsWithTheDenAndNeverStallsInAct1).

#### Scenario: New Sorceress
- **WHEN** a clvl 1 Sorceress with scroll and potions starts the campaign
- **THEN** the first steps are talk to Akara, waypoint to Cold Plains side, Den clear, return, reward

### Requirement: Take drops
The bot SHALL pick up useful drops (potions, gold, equippable upgrades) while leveling and judge them by the loot rules, with no per-item cap below the pack limit; gold autoloot is off on d2d, so the pickup path must work there before a d2d run counts. Source: docs/town.md, docs/d2d-wire.md (loot not wired on d2d, U).

#### Scenario: Gold pile
- **WHEN** a gold pile lies within reach and no hostile is near
- **THEN** the bot picks it up

### Requirement: No vendor gear below the early gear cap
The bot MUST NOT buy vendor armor or weapons below `earlyGearMax` (the early-game cap) and MUST NOT rebuy armor it just sold or could not wear (unwearable-armor loop of G1 attempts 7 to 14). Source: merge 4553bae (V offline tests), docs/g1-log.md attempts 7 to 14.

#### Scenario: Unwearable armor
- **WHEN** a vendor armor needs more Strength than the character has
- **THEN** it is not bought and not bought again after a restart

### Requirement: Rejuvenation potion cap
The bot SHALL hold at most 3 rejuvenation potions across belt and pack, sell the surplus, move rejuvenation to the belt when a slot is free, and drink at the boss threshold only. Source: merge aec4ca1 and 40cb988 (rejuvuse, invrejuv; docs/inventory.md, V offline).

#### Scenario: Fifth rejuvenation potion
- **WHEN** 4 rejuvenation potions are held
- **THEN** the surplus is sold on the next town visit

### Requirement: Level where XP is
At clvl 12 or higher the level picker SHALL prefer Stony Field, Dark Wood and Black Marsh and never farm Den, Blood Moor or Cold Plains for XP (they pay under 3 XP per kill; recovery farming there earns gold only). Source: docs/d2d-fidelity.md (V-d2d, 7885 kills); plan-v2 section 11.5.

#### Scenario: Clvl 12
- **WHEN** a clvl 12 caster picks an XP area
- **THEN** it picks Stony Field or deeper, not Blood Moor

### Requirement: Belt before the field
The bot MUST refill the belt before leaving town and SHALL NOT start a field trip with fewer than 4 healing potions (kit gate also used by the route guard). Source: docs/beltfill.md, plan-v2 section 11.2.

#### Scenario: Low belt
- **WHEN** the belt holds 2 healing potions in town
- **THEN** the bot visits the vendor before the waypoint

### Requirement: Identify scrolls only for likely valuables
The town shop and the identify chore SHALL buy an identify scroll only while an unidentified set, rare, unique or crafted item is in the pack (knob town.idScrollNeed, default 1); an unidentified plain item is sold as it is when no identify tool is held. With town.idScrollNeed 0 the shop tops up to town.idScrollTarget as before. The rule is the d2-sme's EXP rule of thumb (UNVERIFIED on d2d). Source: docs/basics-audit.md rows 2, 3, 7.

#### Scenario: Scroll held, belt full, poor
- **WHEN** a character with gold 120, one portal scroll, a full belt and nothing to identify visits town
- **THEN** nothing is bought and 120 gold remain (TestAuditS2ScrollHeldBeltFullBuysNothing)

#### Scenario: Plain magic ring
- **WHEN** the pack holds an unidentified magic ring and no identify tool
- **THEN** the ring is sold as it is and no identify scroll is bought (TestAuditS7UnidentifiedRingIsSoldAsIs)

#### Scenario: Unidentified rare
- **WHEN** the pack holds an unidentified rare ring
- **THEN** an identify scroll is bought and the ring is identified (TestIDScrollBoughtForUnidentifiedRare)

### Requirement: A failed repair is not retried until something changes
After a repair that did not raise the lowest durability the repair chore SHALL stay quiet until that durability changed or gold grew by town.futileGoldStep (default 100); town.repairLatch 0 retries on every visit. Source: docs/basics-audit.md row 10 (d2d logged 5 failed repairs in one run).

#### Scenario: Two visits, then more gold
- **WHEN** the repair click changes nothing on visit 1, visit 2 has the same gold, and visit 3 has 100 more
- **THEN** repair is attempted on visits 1 and 3 only (TestAuditS10FailedRepairIsLatched)

### Requirement: Bank the surplus before the shop
With town.bankFirst 1 (default) a town visit SHALL deposit the gold above the keep line before the shop step. The keep line is town.stashGoldBase plus town.stashGoldPerLvl per level (300 + 50 x clvl) and never below the minimal kit (resource kitTPGold plus hpFirst times kitPotionGold, 190 by default). Source: docs/basics-audit.md row 11.

#### Scenario: Rich at clvl 5
- **WHEN** a clvl 5 character with 1500 gold visits town
- **THEN** 950 gold are banked before the shop spends (TestAuditS11KeepLineAtClvl5)

#### Scenario: Zero keep line
- **WHEN** the keep knobs are 0 and the character has 1000 gold
- **THEN** at least the minimal kit gold stays in the pack (TestBankFirstKeepsTheKit)

### Requirement: Potions before the first town portal scroll
With town.potionsFirst 1 (default) the shop SHALL buy the kit potions before the first town portal scroll, and that scroll only when the gold left covers the scroll price plus town.tpScrollPotions (default 2) healing potions. town.potionsFirst 0 buys the scroll first; town.tpScrollPotions 0 with town.tpFirstLevel 4 restores the old clvl 4 rule. Source: decision D3 2026-10-31; d2-sme order (UNVERIFIED on d2d prices).

#### Scenario: Poor at clvl 1
- **WHEN** a clvl 1 character with 120 gold and no scroll visits town
- **THEN** only potions are bought (TestPotionsFirstNoScrollBeforeKitLevel)

#### Scenario: Kit level with gold
- **WHEN** a clvl 8 character with 700 gold and no scroll visits town
- **THEN** potions are bought first, then one portal scroll (TestPotionsFirstThenScrollAtKitLevel)

#### Scenario: Kit level, little gold left
- **WHEN** a clvl 4 character has 130 gold
- **THEN** no portal scroll is bought (TestPotionsFirstSkipsScrollWhenPoor)

#### Scenario: Gold 100 to 190
- **WHEN** a character with 190 gold, an empty belt and no scroll visits town at any level
- **THEN** potions are bought first (at most 3) and no scroll (TestAuditPurchaseOrder)

#### Scenario: Old order
- **WHEN** town.potionsFirst is 0 and gold is 120
- **THEN** one portal scroll is bought first (TestShopBuysTownPortalBeforePotionsWhenPotionsFirstIsOff)
