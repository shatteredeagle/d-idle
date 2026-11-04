# Act 1 campaign specification

## Purpose
What an unattended Normal Act 1 run on the d2d backend must achieve for a fresh character: the quest order, the evidence a quest is done, the Andariel fight rules and the time budget to reach Act 2. Source: docs/roadmap.md milestone M3, docs/sme/play-act-route.md, d2-sme consultation 2026-10-31 (EXP unless tagged).

## ADDED Requirements

### Requirement: Andariel fight rules
The Andariel attempt SHALL start only at the floor (merc-policy `Andariel floor`) with the kit (8 healing, 3 antidotes, 4 mana for a caster, tome with 2 or more charges) and a life floor (life floor = N from the diagnostic run (step 2/4)), drink an antidote when poisoned, keep range and retreat by portal at the escape or chicken line. Static Field opens only if a measured drill shows it helps. Source: docs/andarielsme.md (EXP).

#### Scenario: Poisoned in the fight
- **WHEN** the character is poisoned during the Andariel fight and holds an antidote
- **THEN** the antidote is drunk within 2 seconds

#### Scenario: Thin kit at the door
- **WHEN** the character reaches Catacomb 4 with 4 healing potions and no antidote (below the kit of 8 healing, 3 antidotes)
- **THEN** no attempt starts and a town trip for the kit is inserted
