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
