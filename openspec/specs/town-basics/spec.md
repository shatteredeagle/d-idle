# town-basics Specification

## Purpose
What the bot does in town with gold, scrolls, potions, repairs, the stash and the pack so that a poor early character spends on survival first and never burns time or gold on trips that pay nothing. Source: docs/basics-audit.md (the d2-sme's 15 scenarios, 2026-10-31), docs/BACKLOG.md section 0 (user milestone).

## Requirements

### Requirement: ID scrolls only on demand
The bot SHALL buy an identify scroll only when a pack item needs identifying and gold after the purchase stays above the gold reserve; an unidentified magic item whose estimated sale value is under 100 gold SHALL be sold unidentified. Source: docs/basics-audit.md rows 2, 3, 7 and G1; d2-sme checklist 2026-10-31.

#### Scenario: Scroll held, belt full, poor
- **WHEN** gold is 120, one portal scroll is held, the belt is full and no pack item needs identifying
- **THEN** no shop window is opened and the gold stays 120

#### Scenario: Cheap unidentified ring
- **WHEN** the pack holds an unidentified ring worth under 100 gold and gold is 80
- **THEN** the ring is sold unidentified and no identify scroll is bought

### Requirement: Purchase order when poor
With town.potionsFirst 1 (default) the shop SHALL buy healing potions first (up to 3) and the first Town Portal scroll only when the gold left covers the scroll price plus town.tpScrollPotions (default 2) healing potions, with no level rule. The same gold test decides whether a missing scroll raises a town trip. Source: decision D3 2026-10-31; docs/basics-audit.md G5. Shared town code; d2d prices UNVERIFIED.

#### Scenario: Gold 150, empty belt, no scroll
- **WHEN** a clvl 4 character enters town with 150 gold, an empty belt and no portal scroll
- **THEN** healing potions are bought first (up to 3) and no scroll is bought, because the gold left does not cover scroll plus 2 potions (TestAuditPurchaseOrder)

#### Scenario: Rich enough for both
- **WHEN** a clvl 1 character enters town with 900 gold, an empty belt and no portal scroll
- **THEN** potions are bought first and a portal scroll after them (TestAuditPurchaseOrder)

#### Scenario: Old rule restored by knobs
- **WHEN** town.tpScrollPotions is 0 and town.tpFirstLevel is 4 at 150 gold and clvl 4
- **THEN** a missing scroll is a trip reason again (TestAuditPurchaseOrderOldRuleKnobs)

### Requirement: Bank before the shop
When carried gold exceeds the keep line (300 + 50 x clvl, death-rule spec) the stash deposit SHALL run before the shop, so the shop spends only from the kit budget and the surplus is banked. Source: docs/basics-audit.md row 11 and G6.

#### Scenario: Gold 1500 at clvl 5
- **WHEN** a clvl 5 character enters town with 1500 gold
- **THEN** 950 gold is banked and the pack keeps 550 before any purchase

### Requirement: Repair failure latch
After a repair attempt fails the bot SHALL log it once and MUST NOT retry the repair until carried gold or the lowest worn durability has changed. Source: docs/basics-audit.md row 10 and G4; docs/d2d-economy.md item 2 (V d2d: five retries in one run); card basicsfix.

#### Scenario: Second visit, nothing changed
- **WHEN** a repair failed on the previous visit and gold and durability are unchanged
- **THEN** the repair step is skipped and no second log line is written

### Requirement: Charms, runes and gems on a sell visit
A sell visit SHALL keep every rune, every unidentified small charm, identified charms with life, a resist, FCR, FHR, MF or skills (large and grand ones only with 3 free pack cells) and every gem of standard grade or better; chipped and flawed gems SHALL be sold until a Horadric Cube is held (pack or stash; UNVERIFIED as the stand-in for "recipe available"). town.sellLowGems 0 restores the old keeping. Source: decision D4 2026-10-31; docs/basics-audit.md row 6, G7. Shared loot code.

#### Scenario: Plain charm and rune in the pack
- **WHEN** the pack holds an unidentified small charm, a rune, a chipped gem and a flawless gem and the bot sells junk before owning the Cube
- **THEN** the charm, the rune and the flawless gem are kept and the chipped gem is sold (TestCharmsRunesGemsOnSellVisit, TestAuditS6RuneCharmAndGems)

#### Scenario: Identified charm without a kept stat
- **WHEN** a small charm is identified and shows only a damage bonus
- **THEN** it is sold on the next visit (TestCharmsRunesGemsOnSellVisit)

#### Scenario: Full pack and a grand charm
- **WHEN** a large charm with life is found and fewer than 3 pack cells are free
- **THEN** it is sold, not kept (TestBigCharmsNeedThreeFreeCells)

### Requirement: Stash gold cap
When the stash refuses a deposit because it is full the bot SHALL deposit only the difference to the cap (stash step and bank-before-shop alike), log it once, and MUST NOT fail the stash step on every later visit. Source: docs/basics-audit.md E4 and G10; the cap constant (2,500,000) is UNVERIFIED until card m1-liveproof measures it; a refused deposit into a stash holding 100,000 gold or more is taken as the cap.

#### Scenario: Full stash
- **WHEN** the stash gold is at the cap and the pack holds 2000 gold above the keep line
- **THEN** the deposit is skipped with one log line and the next visit does not retry it while the stash is still full

#### Scenario: Partial deposit
- **WHEN** the stash has room for 100 gold and the pack holds gold above the keep line
- **THEN** exactly 100 gold are deposited and the stash step does not fail
