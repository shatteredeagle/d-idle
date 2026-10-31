# Town basics specification

## Purpose
What the bot does in town with gold, scrolls, potions, repairs, the stash and the pack so that a poor early character spends on survival first and never burns time or gold on trips that pay nothing. Source: docs/basics-audit.md (the d2-sme's 15 scenarios, 2026-10-31), docs/BACKLOG.md section 0 (user milestone).

## ADDED Requirements

### Requirement: Sell trips only when worth it
A sell trip for a poor character SHALL start only when the sellable items reach the poverty sell count (4 without a portal scroll, 6 with one) on every trip trigger, so no second trigger reopens a trip below that count. Source: docs/basics-audit.md row 4 and G2; docs/d2d-economy.md Fixes (+28 percent XP/h from the first door).

#### Scenario: Five items with a scroll
- **WHEN** gold is under 150, a portal scroll is held and 5 sellable items are in the pack
- **THEN** no sell trip starts

### Requirement: Gold trip above the keep line
When carried gold exceeds the keep line (300 + 50 x clvl) by more than the trip margin the bot SHALL raise a town trip with reason `gold` at the next safe moment, so a death does not drop the surplus. Source: docs/basics-audit.md E6 and G8; death-rule `Gold carry and corpse`.

#### Scenario: Rich in the field
- **WHEN** a clvl 6 character carries 1100 gold (keep line 600) and no hostile is near
- **THEN** a trip with reason `gold` is due

### Requirement: Pack-full threshold
The pack SHALL count as full at 4 or fewer free cells (one knob in cells, default 4), which raises exactly one sell visit; fresh loot after the visit ends in the pack. Source: docs/basics-audit.md row 14 and G9; d2-sme checklist scenario 14.

#### Scenario: Four free cells
- **WHEN** the pack has 4 free cells
- **THEN** one sell visit is due and the next pickup after it lands in the pack
