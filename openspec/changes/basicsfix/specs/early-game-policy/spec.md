## ADDED Requirements

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
With town.potionsFirst 1 (default) the shop SHALL buy the kit potions before the first town portal scroll, and that scroll only at clvl >= town.tpFirstLevel (default 4) with the scroll price (100) left. The numbers are the d2-sme's EXP rule of thumb (UNVERIFIED on d2d); town.potionsFirst 0 buys the scroll first. Source: user decision 2026-10-31.

#### Scenario: Poor at clvl 1
- **WHEN** a clvl 1 character with 120 gold and no scroll visits town
- **THEN** only potions are bought (TestPotionsFirstNoScrollBeforeKitLevel)

#### Scenario: Kit level with gold
- **WHEN** a clvl 8 character with 700 gold and no scroll visits town
- **THEN** potions are bought first, then one portal scroll (TestPotionsFirstThenScrollAtKitLevel)

#### Scenario: Kit level, little gold left
- **WHEN** a clvl 4 character has 130 gold
- **THEN** no portal scroll is bought (TestPotionsFirstSkipsScrollWhenPoor)

#### Scenario: Old order
- **WHEN** town.potionsFirst is 0 and gold is 120
- **THEN** one portal scroll is bought first (TestShopBuysTownPortalBeforePotionsWhenPotionsFirstIsOff)
