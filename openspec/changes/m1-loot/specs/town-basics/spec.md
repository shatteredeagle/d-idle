# Town basics specification

## Purpose
What the bot does in town with gold, scrolls, potions, repairs, the stash and the pack so that a poor early character spends on survival first and never burns time or gold on trips that pay nothing. Source: docs/basics-audit.md (the d2-sme's 15 scenarios, 2026-10-31), docs/BACKLOG.md section 0 (user milestone).

## ADDED Requirements

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
