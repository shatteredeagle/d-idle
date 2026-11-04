# Andariel fight: the SME version and the user decisions (2026-11-04)

> status: decisions recorded offline, nothing verified live. Tags: experience = SME judgement from play, V = verified (source named), U = unverified. Plan: docs/andarielplan.md; card text: openspec/changes/m3-andariel. Numbers with an owner (knobs, floors) stay in docs/knobs.md and the merc-policy spec.

## Decisions (answers to plan section 6)

1. **Static Field is not an opener by default.** The script and `TestAndarielScriptHasNoStaticFieldOpener` stay. Card wording: "Static Field only if a measured drill shows it helps". The Normal boss cap is read from the tables first (plan step 2).
2. **Floors.** Fight floor 15 with a living merc, 17 without, plus a life floor next to the level floor. The life floor number comes from the diagnostic run (plan steps 2 and 4), not from this doc. Kit minimum at the gate: 8 healing, 3 antidotes, 4 mana, tome with at least 2 charges.
3. **Setup.** The declared cheat setup (goto Catacombs 4, given merc, given kit) may replace the road. The series includes handicapped variants (merc at 50 percent life, kit at the gate minimum, tome at 2 charges) and the no-merc case at least 3 times.
4. **Hold lift.** A default change in a separate small commit by the coordinator after 5 clean wins in a row (or 7 of 8 with a cause for the loss), `writes after start: 0`, run at the lowest floors that will ship. Retreat rules inside the series: portal at the escape line (30 percent) or the chicken line (12 percent), one re-attempt after a refill, two retreats end the attempt, any death resets the series. The user decides at lift time.
5. **Gate 2 interim wording (a)** is already merged (delta spec andarielspec).
6. **Merc revive: knob first** (separate card mercrevive): `merc.goldFarmMaxLevel` 18, reserve 100 percent of a kit refill, log the bar.

## SME reasoning

- A Static Field opener puts a clvl 15 caster in melee and poison range (experience). Static Field takes a fixed share of life with a cap per difficulty; the Normal cap is unknown (U). Only 33 percent Nightmare and 50 percent Hell are V in docs/sme/mechanics-combat.md. So no opener until a drill measures it, and the cap is read from the tables first.
- Life is the bar, not clvl (experience): a clvl 15 with low vitality dies to the poison nova where a better-built clvl 15 does not. Hence the life floor beside the level floor; its value is U until the diagnostic run (plan step 5) gives hits per nova and per cold bolt.
- The cheat setup hides what the real road costs (experience): attrition on the road, the Catacombs 3 and 4 packs, a half-dead merc, a depleted tome. The handicapped variants model these. The model stays an approximation (I): it never replaces a road run.
- A dead merc flips the floor between 15 and 17 mid-run (V: `andarielFloor` prefers a living merc, plan section 4). Test that: a fake run where the merc dies during the approach must re-read the floor and either continue or return to town.
- Kit: more healing and antidotes than the old card (6/2/3) because poison and cold bolts drain the belt faster at low life; 4 mana for a caster; tome 2 charges stays.
- Separate unverified cards, not part of this decision: Blood Raven, the free Rogue claim, the Cain chain (docs/research/andariel-chain-needs.md).

## Research gaps (U)

- Minimum-life table: life needed at the Andariel gate by class and build. Source: the plan step 5 diagnostic run (monster damage reads), then a table for plan data.
- Normal Static Field cap: read from the skill and monster tables (plan step 2, `tools/emu` table read).

## Dead ends

None yet.
