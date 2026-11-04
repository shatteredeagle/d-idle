# Town basics specification

## Purpose
Delta of the goldfloor change (DEFERRED, not to be applied: the paired A/B of 2026-10-31 failed its gates, so the levers below are knobs that are OFF by default; this delta states their semantics when switched on). Trips and purchases of a poor early character. Source: docs/goldfloor.md.

## ADDED Requirements

### Requirement: Trips must buy something
With `town.tripMinSpend` 60 (default 0 = off) a non-emergency town trip SHALL start only when the planned purchase is worth at least 60 gold or the belt holds no healing potion, so that no trip burns a portal scroll for a purchase under 30 gold. Source: docs/goldfloor.md finding 2 (83 percent of NAVSOAK trips spent under 30 gold).

#### Scenario: Small plan
- **WHEN** the belt holds 2 healing potions, the plan is one 30-gold potion and a portal scroll is held
- **THEN** no trip is raised

#### Scenario: Empty belt
- **WHEN** the belt holds no healing potion
- **THEN** a trip is raised

### Requirement: No scroll rebuy while poor
With `town.tripMinSpend` above 0 (default 0 = off) a character with gold under the kit line and fewer than 2 healing potions SHALL spend on healing potions first and SHALL NOT buy a portal scroll in the same visit. Source: docs/goldfloor.md (two scrolls bought in 25 s at gold 209).

#### Scenario: Gold 209
- **WHEN** a visit starts with 209 gold and 1 healing potion
- **THEN** healing potions are bought and no town portal scroll is

### Requirement: Early mana cap
With `town.mpCapLevel` 10 and `town.mpCapEarly` 1 (defaults 0 = off, the old mana buying) a visit below clvl 10 SHALL buy at most `town.mpCapEarly` mana potions and a low mana belt SHALL NOT raise a trip. Source: docs/goldfloor.md finding 3 (mana 74 to 305 gold/h at 60 gold each).

#### Scenario: Clvl 6
- **WHEN** a clvl 6 character has an empty mana belt and 400 gold
- **THEN** one mana potion is bought and no `mp belt columns` trip is raised
