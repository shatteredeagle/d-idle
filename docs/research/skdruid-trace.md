# Druid skills (Skills 221..250): calc cells, summon caps and rule reads; game.exe 1.14d vs d2d vs the bot (card skdruid, coverage gap skills.dru)

> status: card skdruid, 2026-11-02, offline (no game, no d2d slot, no Go change). FIRST PASS: only the table-driven calc aspect is run; the effects (shapeshift forms, Twister / Tornado flight, Hurricane / Armageddon ticks, vines, wolves) are NOT run and have no verdict. Method of docs/research/nec-skill-trace.md (same harness, id range changed).

Tags: **V run** game.exe's own code under the emulator against the real d2d build, counts given; **D** patched 1.14d `Skills.txt`; **I** inference; **U** not verified. Nothing here is V read: no disassembly was read in this pass.

## Summary

- **Every calc cell of the 30 Druid rows equals d2d (V run, 0 differ of 27,800 cases).** `calc1..4`, `petmax`, `sumsk1..5`, `delay`, aura range / length / period, `aurastatcalc1..6` and `passivecalc1..5` of Skills 221..250 by game.exe's calc machine (FUN_00646ca0) and by d2d `eval_calc`: levels 1..40 alone (clvl 30) and with every neighbour at 20 (clvl 99, masteries 50), plus 20,000 random cases. Same harness as NECSKILL, only the id range differs.
- **Summon caps (V run, D).** `petmax` by skill level 1, 2, 3, 5, 10, 20, 30: Raven 1, 2, 3, 5, 5, 5, 5 (`min(lvl,par2)`); Spirit Wolf the same (`min(lvl,par3)`, cap 5); Fenris 1, 2, 3, 3, 3, 3, 3 (cap 3); Grizzly 1 at every level; the plant / totem summons (Plague Poppy, Cycle of Life, Vines, Oak Sage, Heart of Wolverine, Spirit of Barbs) 1. Game and d2d agree on the number; how d2d enforces the cap (drop the oldest of the same `pettype`) was not compared with game.exe's spawn path: **U**.
- **Other values read off the run (V run, equal d2d):** Firestorm area rows have no calc; Armageddon / Hurricane aura period 150 ticks, range 8 / 9, length 250 ticks at every level (levels do not change them; the damage comes from the missile / state tables, not run); Cyclone Armor absorb 10,240 (256ths) at L1, 68,608 at L20; Feral Rage and Maul passive and aura numbers (Maul tohit 20 / 400, Feral Rage life-steal pair); Hunger calc1 -75 (leech) with 72 .. 176 on calc2 / calc3; Fury calc1 2 .. 5 hits, calc2 100 .. 423; Oak Sage / Heart of Wolverine / Spirit of Barbs aura values 100 and level-scaled passive.
- **Deviations d2d vs game.exe: none found in the calc aspect.** Not covered, so no verdict: shapeshift form stats, Twister / Tornado (srvdofunc 118) travel and hit, Hurricane / Armageddon (do 124) ticks, Fire Claws (do 2) damage, Volcano (do 123), Arctic Blast (do 19), Rabies / Hunger / Fury / Shock Wave / Eruption procs. Candidates for a second pass: D1 (U) Tornado / Twister missile hit roll, which SKMISS showed is top-inclusive in d2d if d2d uses `Rng::range` there (I, not checked); D2 (U) Cyclone Armor absorb accounting vs Energy Shield; D3 (U) the summon spawn path (base stats of wolves, bear, vines).
- **No PR candidate:** no deviation proven.

## 1. How it was run

| Piece | What | Tag |
|---|---|---|
| `tools/emu/skdruid_calc.py` | nec_calc.py with the Druid ids; game.exe FUN_00646ca0 on each calc pointer of record +0x138..0x144, +0xc0, +0xd0, +0x190, +0x64, +0x60, +0x128, +0x68, +0xa4 | V run |
| `tools/emu/skdruid_test.cpp` | d2d `eval_calc` on the loaded patched tables, built by hand against the scratch build | V run |
| `tools/emu/skdruid_run.sh N SEED` | builds the test, generates the cases, runs both, diffs | V run |

Runs through `tools/emu/run-on-worker.sh --ttl 2400 skdruid_run.sh 3000 1` = 5,400 cases, DIFF 0; `skdruid_run.sh 20000 7` = 22,400 cases, DIFF 0 (27,800 total). The game side printed non-trivial values (checked, tables above), not zeros. The emulator tree used is upstream main on the worker host.

## 2. Questions

**Q1. Do the Druid calc cells match? (V run)** Yes, 0 of 27,800.
**Q2. Summon caps? (V run)** Section Summary. Skills table names: 221 Raven, 222 Plague Poppy, 223 Wearwolf, 224 Shape Shifting, 225 Firestorm, 226 Oak Sage, 227 Summon Spirit Wolf, 228 Wearbear, 229 Molten Boulder, 230 Arctic Blast, 231 Cycle of Life, 232 Feral Rage, 233 Maul, 234 Eruption, 235 Cyclone Armor, 236 Heart of Wolverine, 237 Summon Fenris, 238 Rabies, 239 Fire Claws, 240 Twister, 241 Vines, 242 Hunger, 243 Shock Wave, 244 Volcano, 245 Tornado, 246 Spirit of Barbs, 247 Summon Grizzly, 248 Fury, 249 Armageddon, 250 Hurricane (D). The rows do not match the names in the brief ("Poison Creeper" is Vines / Plague Poppy in 1.14d).
**Q3. Effects (shapeshifting, twisters, storms)? (U)** Not run in this pass.

## 3. Bot findings

B1 (I): the bot has no Druid combat code (plan level); nothing to compare. Not checked beyond a search of the docs.

## 4. Drill recipe (U to confirm later, lab only; not run)

With the lab binary (docs/d2d-fork.md, docs/forkcheats.md): give a Druid skills 221, 227, 237, 247 at level 5 / 10, summon repeatedly and count live pets (expect Raven 5, Wolves 5, Fenris 3, Grizzly 1); cast Tornado / Twister at a stationary dummy and log the damage rolls against the missile table.

## 5. Matrix-cell update (proposed)

`skills.dru` rows 221..250: calc aspect TESTED-0-DIFF (27,800 cases); effects READ-ONLY/untested.

## 6. Dead ends

None. The first run failed only because the emulator tree path was wrong (an absent build directory); fixed in the script.

## Shared-file lines for the merger

See openspec/changes/skdruid/merge-lines.txt.
