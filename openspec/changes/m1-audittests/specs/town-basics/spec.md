# Town basics specification

## Purpose
What the bot does in town with gold, scrolls, potions, repairs, the stash and the pack so that a poor early character spends on survival first and never burns time or gold on trips that pay nothing. Source: docs/basics-audit.md (the d2-sme's 15 scenarios, 2026-10-31), docs/BACKLOG.md section 0 (user milestone).

## ADDED Requirements

### Requirement: ID scrolls only on demand
The bot SHALL buy an identify scroll only when a pack item needs identifying and gold after the purchase stays above the gold reserve; an unidentified magic item whose estimated sale value is under 100 gold SHALL be sold unidentified. Source: docs/basics-audit.md rows 2, 3, 7 and G1; d2-sme checklist 2026-10-31.

#### Scenario: Scroll held, belt full, poor
- **WHEN** gold is 120, one portal scroll is held, the belt is full and no pack item needs identifying
- **THEN** no shop window is opened and the gold stays 120

#### Scenario: Cheap unidentified ring
- **WHEN** the pack holds an unidentified ring worth under 100 gold and gold is 80
- **THEN** the ring is sold unidentified and no identify scroll is bought

### Requirement: Purchase order when poor
Below `town.tpMinGold` gold the bot SHALL buy healing potions first (up to 3) and the portal scroll only once gold covers the scroll plus 2 potions; at or above the knob the scroll comes first. Source: user decision D3 2026-10-31 (docs/roadmap.md); d2-sme EXP; docs/basics-audit.md G5.

#### Scenario: Gold 150, empty belt, no scroll
- **WHEN** a clvl 4 character enters town with 150 gold, an empty belt and no portal scroll
- **THEN** healing potions are bought first (up to 3) and the scroll only if the gold left still covers scroll plus 2 potions

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
