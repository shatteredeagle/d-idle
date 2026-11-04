## Why

Andariel has never been fought on either backend (docs/plan/STATUS.md), and the d2-sme calls her the biggest unknown of the Act 1 clear: a low-level Sorceress dies to poison nova and cold bolts at close range, to an empty belt, or to a dead merc that turns her onto the caster. The fight does not depend on the economy or walk work, so it runs in parallel with M1 and M2 on a cheat-levelled throwaway (ledgered) at the floor level.

## What Changes

- internal/boss/act1.go: Andariel script on d2d: approach from Catacombs 4, antidote before the fight and on poison, Charged Bolt at range (Static Field only if a measured drill shows it helps; the pinned no-opener script stays by default), keep range, retreat line, kill detect, log `boss andariel` lines with life minimum and potions used.
- Fights on a cheat-levelled throwaway at the fight floors (15 with a living merc, 17 without, plus a life floor: life floor = N from the diagnostic run (step 2/4)) with the kit (8 healing, 3 antidotes, 4 mana, tome with at least 2 charges). The declared setup may be goto Catacombs 4 plus a given merc and kit instead of the road. The series includes handicapped variants (merc at 50 percent life, kit at the gate minimum, tome at 2 charges) and the no-merc case at least 3 times. Series rule: 5 clean wins in a row (or 7 of 8 with a cause for the loss); retreat by portal at the escape line (30 percent) or chicken line (12 percent), one re-attempt after a refill, two retreats end the attempt, any death resets the series. A dead merc flips the floor between 15 and 17 mid-run: tested. Decisions and reasoning: docs/andarielsme.md (user decision 2026-11-04).
- docs/andariel.md: the fight log, what killed her or us, Static Field's boss cap in Normal (U until read from the tables first, step 2), her HP and resists from the tables.
- Delta: act1-campaign ADDED `Andariel fight rules`.

## Dependencies

Only after xpgap releases soak-slot:3 (no code dependency: runs in parallel with M1 and M2, d2-sme advice 2026-10-31).
