## Why

User decision 2026-11-02 ("go with the SME suggestions and what you see as the best path for the Merc"): the free Act 1 Rogue is a decoy to about clvl 12; Gate 2 showed a dead or missing merc closing the whole Monastery road (floors 16 and 17 for no merc, 14 and 15 only for a living one) and the merc policy was off by default, so nothing revived her. Sources: docs/sme/mercenary.md, openspec/specs/merc-policy/spec.md, docs/research/gate2-analysis-20261102.md, card mercfix (guard reads the dump merc, claim verified).

## What Changes

- `merc.enabled` default 1 (was 0). New knobs `merc.kitReservePct` 150 and `merc.goldFarmMaxLevel` 16; `merc.reviveGoldMult` 2 to 1, `merc.hireGoldMult` 3 to 1, `merc.potionLifePct` 40 to 50, `route.monasteryMinLevel` 16 to 14 (the full kit stays required on every guard; Andariel stays 17 and 15, the Forgotten Tower stays 16 and 14).
- Spending order, potions first: a revive or hire needs gold >= price x mult + 150 percent of the full kit refill (portal + `resource.kitMinHP` healing + `resource.kitMinMP` mana). Max 2 revives an hour (`merc.reviveMaxPerHour`, verified to bind by test).
- The revive price is never taken from the formula for the payment: the NPC takes what it takes; `merc: revived price=<paid> level=<n> estimate=<formula>` logs both. The formula is the pre-check estimate only (S, unconfirmed on 1.14d).
- Act 2: `mercWire.HireStep` (Hirer interface) swaps the Rogue for the Desert Mercenary with the best class aura (Defiance for the Sorceress), strips her gear first, only when affordable under the reserve; one-shot `merc: hire window` line per row (name, level, aura, price) for the live read. The d2d side of the window (open, read, click) does not exist: production wiring leaves the Hirer nil, so the step is unit-tested only.
- Gold farm: while the merc is dead or missing, gold is short and clvl < 16, the level picker keeps to Blood Moor, Cold Plains, Stony Field, Den, Caves, Burial Grounds and Crypt (`bot.MercGoldFarm`, line `merc: gold farm on|off`).
- Pure helpers for the field rules (`ShouldWaitForMerc`, `LeaveWithMercDown`) are written and tested but not wired into the fight loop.
- Delta spec merc-policy: MODIFIED Blood Raven kit gate, Revive and stop rule, Route guard, Merc gear and potions; ADDED spending order, Act 2 swap, gold farm, default on. Spec issue for the user: the Blood Raven kit gate says clvl 9 for casters and the code agrees, but Gate 2 claimed at clvl 7 (a kill the gate did not cover, U where from); the delta keeps 9 as the attempt gate and states that a kill below it still claims. Lowering to 7 is not recommended (G1 attempt 8: 14 portal-outs at clvl 9).

## Overlap (declared)

Card m3-merc owns `internal/merc/**` and carries a pending MODIFIED "Revive and stop rule" (multiplier 2 on gold minus kit). This card edits internal/merc for the kit reserve and the new lines and supersedes that delta: the coordinator applies this card's delta after (or instead of) it. `docs/knobs.md` is regenerated (generated block only).
