# SME review: Sorceress (cold default) and Paladin (hammerdin, hammerdin_zeal), 2026-11-27

> status: advice only, no code or data changed. Tags: SRC = read in this checkout this turn (path given), EXP = veteran 1.14d experience, not looked up, UNSURE. No web fetch this turn; skill numbers are the repo's skills.txt-derived tables (`docs/sme/class-paladin.md`, `docs/sme/class-sorceress.md`).

## Verdicts
- Sorceress cold (`build-sorceress_normal30.json`, profile `sorceress/cold`): sensible to play, two defects (Glacial Spike points never cast; escape trigger too late for a 150-250 life caster).
- Paladin `hammerdin` table (`build-paladin_hammerdin_endgame90.json`): NOT sensible clvl 3-17. It banks 14 points (rows 4-11 empty, SRC) and fights with Attack plus Smite 1 and Might 1. Playable only because BH comes at 18.
- Paladin `hammerdin_zeal` (`build-paladin_hammerdin_zeal_endgame90.json`, profile `paladin/hammerdin_zeal`): the right idea (Zeal 12-17, BH from 18), wrong filler: 9 points of Smite, Concentration at clvl ~50, Holy Shield at clvl 36.

## The Smite 0 dps concern (classkeys)
- Smite adds no damage in the repo table: Smite is a shield bash, weapon damage plus knockback/stun (SRC `docs/sme/class-paladin.md` section 2 shows no added damage; UNSURE exact bash numbers). The planner's 0 dps is correct, not a bug.
- A live soak agrees: Zeal 506 of 652 kills, Smite 0 casts in the log (SRC `docs/probes/hero14-analysis-2026-11-14.md` line 95).
- So the zeal table is wrong to buy Smite 9; the plain table is right to buy only 1 (Charge prereq) but wrong to bank the rest.
- A human spends early points on Might (damage percent, free), Sacrifice 1 only as the Zeal prerequisite, then Zeal at 12 (EXP). Smite 1 only as the Charge prerequisite (SRC prereq table).

## Skill order and prerequisites (SRC prereq table in `docs/sme/class-paladin.md`, EXP for priorities)
- Chain for BH: Holy Bolt 1 (clvl 6) then BH (18). Holy Shield (24) needs Charge (12, needs Smite) plus BH. Concentration (18) needs Blessed Aim (12, needs Might). Vigor needs Cleansing plus Defiance (so Prayer). Tables already satisfy these.
- Zeal: 4 points reach the 5-hit cap (SRC class-paladin.md line 61); the 5th adds only a little damage and attack rating. Acceptable.
- Concentration matters from the first BH point: the aura keeps BH from being interrupted (repo S claim, class-paladin.md line 137; EXP agrees) and adds damage. Buying it at clvl 50 means clvl 18-49 hammers get interrupted by the first hit.
- Holy Shield at 36 leaves clvl 24-35 (Act 2 to Act 3 Normal) with no block or defense buff. Mana 35 per cast is steep for a ~60-90 mana hero, so it needs mana potions (belt change below).

## Mana, stance, positioning
- Hammerdin is mana-heavy: BH costs 4.75+0.25*lvl (about 10 at L20, SRC), Holy Shield 35, and BH does magic damage, so no mana leech (EXP). `potions-belt.json` `mana_heavy_builds` lists `paladin_hammerdin_endgame90` but NOT the new `paladin_hammerdin_zeal_endgame90`, so the zeal hammerdin gets one mana column (SRC). Verify live: belt layout log line after the clvl 13 shop trip shows hp,mp,mp,hp.
- Zeal profile band 18 lists `Blessed Hammer, Smite, Attack`: when mana runs dry the fallback is Smite (no damage) instead of Zeal (SRC profiles.json). The generic paladin profile has Zeal there.
- `hammerdin_zeal` has no `stance` block (only `hammerdin` has insidePack standOff 2, leaveSurround 5, leaveLifePct 50). Numbers are sane for Normal: drink at 55 percent, leave at 50 (EXP). Surround 5 within 3 subtiles is low once Holy Shield is up (Hell packs routinely put 6-8 around a hammerdin) but fine before it (EXP, UNSURE for the bot's block rate).
- Hammer aim: BH leaves the caster toward the click and spirals outward; best result is aiming at the nearest pack member with the hero inside the pack, not at the centroid far side (EXP). The profile has no `aims` row for BH, so the cast aim is the target; H1 (does the click matter) is a live look still open (SRC stance.go header). No data change until H1 is measured: log hits per cast for aim=target vs aim=centroid.
- Zeal press row: the generic paladin profile has a 600 ms slice; `hammerdin_zeal` has none, so the profile-wide `melee.holdMs` 600 applies. Same number, no change.

## Sorceress rules
- Teleport move row (range 18, minHop 4, escape at 4 in contact and life under 65, SRC profiles.json): range V live, triggers U. A sorceress with about 150-250 life at clvl 18-30 and 4 melee in contact has already eaten two or three rounds. A human hops at 2 to 3 attackers or about 75 percent life, then fires (EXP). Teleport costs 24 mana at L1 (EXP, UNSURE the exact L1 figure): with a pool of about 80 (35 + 2 per clvl, no Energy, EXP) a hop is a quarter of the pool, so gate it on mana >= hop plus one cast (the driver already refuses with a reason, SRC move.go header).
- Static Field and Frost Nova per build (data): Normal and Nightmare have no cold-immune regular types (SRC class-sorceress.md line 8), so the Normal30 table rightly never buys Static Field (the f1 key stays empty) and buys Frost Nova 1 only as the Blizzard prerequisite (L1 does 2-4 cold, 9 mana: never cast it). Static Field matters from Hell, vs bosses (cannot go below 33 percent NM / 50 percent Hell, SRC class-sorceress.md line 216). The endgame table buys it at clvl 6; harmless but a wasted early point. No data change.
- Glacial Spike: bands put Ice Blast before Glacial Spike, and the Normal table buys GS 6 at clvl 24-29 then Blizzard 1 at 30 (SRC). While Ice Blast is castable GS is never chosen, so those 6 points do nothing until Blizzard is deep. Ice Blast L15 is far better than GS L6 for clvl 24-30 (EXP).
- Charged Bolt 7 points clvl 3-10: right for a no-aim, multi-bolt early game; no synergy later (accepted, EXP).
- Levelplan: sorceress `active: false` (SRC levelplan-normal.json), so the Sorceress uses progression-normal rows until the caster survival model is traced. Slices (Blood Moor 1-4, Cold Plains 5-8, Stony Field 8-12, Tamoe 10-14, Catacombs 13-18, Forgotten Cellars 15-24) are sane for a caster (EXP). Keep false until a live run.
- Merc: Rogue for Act 1 (cheap), Act 2 Desert merc in Normal. The soak had the Paladin merc dead from the start with 66 of 83 route guards blocking on it (SRC hero14-analysis); `merc_revive` is priority 8 in `priority.json`, below mana and gold_carry. Act 2 aura names per difficulty in `mercs.json` (Defiance/Prayer Normal, Might Nightmare only) are UNSURE here, they match the repo's hireling.tsv claim.
- Resist thresholds: no per-difficulty resist floor exists in `docs/plan/data` or `internal/gear` data (grep, SRC). Gap for research. Proposal (EXP): penalty is 0 / -40 / -100 on Normal / Nightmare / Hell; displayed floor entering Nightmare Act 1 0 (raw 40), Nightmare Act 3+ 30, Hell Act 1-3 50, Hell Act 4-5 75 (cap). Lightning is the Sorceress killer, fire and cold the Paladin's.

## Ranked data changes (at most 10)
1. `internal/plandata/build-paladin_hammerdin_zeal_endgame90.json` rows 4-11: Smite +1 each (7 rows plus the Den quest point) -> Smite stays at 1 (clvl 3, Charge prereq); put the 8 points into Might (to about 7), Sacrifice 1 at clvl 11, Zeal 1 at 12, Charge 1 moved from 12 to about 19. Why: Smite adds no damage; Zeal comes online one level earlier with Might multiplying it. Confidence medium-high (EXP, soak SRC).
2. Same file: Blessed Aim 1 at clvl 19, Concentration 1 at clvl 20 (currently Concentration first appears about clvl 50). Why: uninterrupted BH and the damage aura; cheap two points. Confidence high.
3. `internal/combat/classdata/profiles.json` paladin/hammerdin_zeal band minLevel 18 attack: `Blessed Hammer, Smite, Attack` -> `Blessed Hammer, Zeal, Attack`; band minLevel 12: `Zeal, Smite, Attack` -> `Zeal, Attack`. Why: low-mana fallback should deal damage. Confidence high.
4. `docs/plan/data/potions-belt.json` (and the plandata copy) `mana_heavy_builds.builds`: add `paladin_hammerdin_zeal_endgame90`, and `mana_heavy: true` in that build file. Why: two mana columns for BH and Holy Shield. Confidence high.
5. `build-paladin_hammerdin_zeal_endgame90.json`: Holy Shield from clvl 36 to the Izual quest point at 26 (row 26 `Blessed Hammer` Q +2 -> Holy Shield +1). Why: block and defense for clvl 26-35 packs, needs Charge (change 1) and BH. Confidence medium.
6. `build-sorceress_normal30.json` rows 24-29: Glacial Spike +1 each -> Glacial Spike +1 at 24 only, then Ice Blast +1 for 25-29 (Ice Blast 15), Blizzard 1 stays at 30. Why: GS is never cast while Ice Blast is castable (cold band order). Confidence medium-high.
7. `profiles.json` sorceress cold/blizzard/lightning/fire `moves[Teleport]`: `escapeSurround` 4 -> 3, `escapeLifePct` 65 -> 75. Why: a 200-life caster should hop before a fourth attacker closes. Confidence medium (UNSURE on mana cost vs hop; the driver already gates mana).
8. `profiles.json` paladin/hammerdin_zeal: add the `stance` insidePack block of `paladin/hammerdin` (standOff 2, leaveSurround 5, leaveLifePct 50). Why: parity with the plain hammerdin; knob 2 still waits for the survival model. Confidence medium-low.
9. `internal/plandata/build-sorceress_normal30.json`: add `mana_heavy: true` and the table name to `mana_heavy_builds` (Normal cold sorc pool about 80 at clvl 24, 10-25 mana per cast). Confidence medium.
10. `internal/supply/priority.json`: for mana-heavy builds move `mana to min` from priority 5 (below repair and ammo) to 3 (above repair); needs the profile hook to read the mana_heavy flag (code, not data only). `merc_revive` 8 -> 6 for a Normal hero without a merc. Confidence low-medium.

## Verify live
- Change 2: `combat aura up skill=Concentration` after clvl 20; BH casts without an interrupt line.
- Change 3: cast log with mana under BH cost shows Zeal, not Smite.
- Change 4: belt layout line after the first shop trip.
- Change 6/7: log counts of Glacial Spike casts (expect 0 today) and `move escape` lines vs deaths in Catacombs.
- Soak records keep `writes after start: 0` (AGENTS.md crutch rule).

## Gaps in docs/sme
- No resist-floor table per difficulty and act; no merc aura-by-difficulty table verified against the game; FCR/FHR breakpoints for Paladin (0/9/18/30/48/75/125 FCR, EXP) and Sorceress (0/9/20/37/63/105/200) are not in a data file the gear scorer reads (grep SRC: only score.go mentions resist). Queue research.
