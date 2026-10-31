## Why

Andariel has never been fought on either backend (docs/plan/STATUS.md), and the d2-sme calls her the biggest unknown of the Act 1 clear: a low-level Sorceress dies to poison nova and cold bolts at close range, to an empty belt, or to a dead merc that turns her onto the caster. The fight does not depend on the economy or walk work, so it runs in parallel with M1 and M2 on a cheat-levelled throwaway (ledgered) at the floor level.

## What Changes

- internal/boss/act1.go: Andariel script on d2d: approach from Catacombs 4, antidote before the fight and on poison, Static Field until the boss cap then Charged Bolt, keep range, retreat line, kill detect, log `boss andariel` lines with life minimum and potions used.
- Fights on a cheat-levelled throwaway at clvl boss.andarielMinLevel (docs/knobs.md) with the kit (6 healing, 2 antidotes, 3 mana, tome with 2 charges) and a living Rogue, and once at the same level without a merc.
- docs/andariel.md: the fight log, what killed her or us, Static Field's boss cap in Normal (U until read), her HP and resists from the tables.
- Delta: act1-campaign ADDED `Andariel fight rules`.

## Dependencies

Only after xpgap releases soak-slot:3 (no code dependency: runs in parallel with M1 and M2, d2-sme advice 2026-10-31).
