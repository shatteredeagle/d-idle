# Town basics specification

## Purpose
What the bot does in town with gold, scrolls, potions, repairs, the stash and the pack so that a poor early character spends on survival first and never burns time or gold on trips that pay nothing. Source: docs/basics-audit.md (the d2-sme's 15 scenarios, 2026-10-31), docs/BACKLOG.md section 0 (user milestone).

## ADDED Requirements

### Requirement: Charms, runes and gems on a sell visit
A sell visit SHALL keep every rune, every unidentified small charm, every identified charm with life, a resist, FCR, FHR or MF, and every gem better than flawed; chipped and flawed gems are sold until the Horadric Cube recipe is available. Source: user decision D4 2026-10-31 (docs/roadmap.md); d2-sme EXP; docs/basics-audit.md row 6 and G7.

#### Scenario: Plain charm and rune in the pack
- **WHEN** the pack holds an unidentified small charm, a rune and a chipped gem and the bot sells junk at Akara before owning the Cube
- **THEN** the charm and the rune are still in the pack after the visit and the chipped gem is sold (a flawless gem would stay)

#### Scenario: Identified charm without a kept stat
- **WHEN** a small charm is identified and shows only a damage bonus
- **THEN** it is sold on the next visit
