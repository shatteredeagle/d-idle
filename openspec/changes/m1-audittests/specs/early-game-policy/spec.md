# Early game policy specification

## MODIFIED Requirements

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
