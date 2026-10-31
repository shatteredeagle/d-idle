# Town basics specification

## Purpose
What the bot does in town with gold, scrolls, potions, repairs, the stash and the pack so that a poor early character spends on survival first and never burns time or gold on trips that pay nothing. Source: docs/basics-audit.md (the d2-sme's 15 scenarios, 2026-10-31), docs/BACKLOG.md section 0 (user milestone).

## ADDED Requirements

### Requirement: Charms, runes and gems on a sell visit
A sell visit SHALL keep every rune, every unidentified small charm until it is identified, every identified charm with life, a resist, faster cast rate, faster hit recovery or magic find, and every gem better than flawed (chipped and flawed gems are sold until the Horadric Cube recipe is available, user decision D4, 2026-10-31); large and grand charms are picked up only with 3 or more free cells. Source: d2-sme 2026-10-31 (EXP); docs/basics-audit.md row 6 and G7; docs/roadmap.md decision D4 (docs/basics-audit.md row 6 changes from gem kept to chipped and flawed gems sold).

#### Scenario: Plain charm and rune in the pack
- **WHEN** the pack holds an unidentified small charm, a rune and a chipped gem and the bot sells junk at Akara before owning the Cube
- **THEN** the charm and the rune are still in the pack after the visit and the chipped gem is sold (a flawless gem would stay)

#### Scenario: Identified charm without a kept stat
- **WHEN** a small charm is identified and shows only a damage bonus
- **THEN** it is sold on the next visit
