# fid-stamina (matrix row ST08): walk and run speed, stamina drain and regeneration, game.exe vs d2d vs the bot

> status: finished 2026-11-11 by agent decomp10 (Sonnet), offline. Result: **d2d's stamina drain and regeneration equal game.exe's own code (2 x 20,000 drain cases and 2 x 20,000 regen cases, 0 mismatch); walk and run speed were already V run upstream; nothing new deviates.** The new part is the exact bookkeeping the bot can use to know whether the character runs. Consumer: TASK-329 (runstate). No PR candidate.

Tags as in fid-potions.md. Scripts: `tools/emu/d10_stam.py`, `d10_stam_test.cpp`, `d10_stam_run.sh` (`d10_build.sh` first; `tools/emu/run-on-worker.sh --ttl 1500 d10_stam_run.sh 20000 1 7`).

## Questions

**Q1. Speeds. (S, D)** CharStats `WalkVelocity` 6 and `RunVelocity` 9 for every class (D, charstats.txt, read here); running adds `run x 100 / walk - 100` = 50 to stat 67; path velocity `(walk << 8) x velocitypercent / 100` (FUN_00462a20): **walk 0.375 subtile per tick (1.875 cells/s), run 0.5625 (2.81 cells/s), ratio 1.5**, +velocitypercent from gear or chill scales both (S: re/char-select.md "Movement speed", V run by upstream `walk_rates.py` for the animation rate; docs/d2d-run.md measured 1.85 and about 2.8 cells/s live). Town: the same speeds, no stamina use.

**Q2. Drain per running tick outside town. (V read FUN_0057f240, V run)** `drain = RunDrain x 2`, times `(armor.txt speed of the worn body armor / 10 + 1)`, less `stat 154` percent of it, at least 1, all in 256ths. RunDrain (CharStats +0x42, V run from the loaded table): **20 for the six old classes, 15 for the Assassin**. armor.txt `speed` for body armor (D): 0 for Quilted, Leather, Hard Leather, Studded Leather, Breast Plate, Light Plate (18 rows), 5 for Ring Mail, Chain Mail, Splint Mail, Field Plate, Gothic Plate, Ancient Armor (18 rows), **10 for Scale Mail, Plate Mail, Full Plate Mail, Tigulated Mail, Templar Coat, Chaos Armor and three more (9 rows)**. Integer division: speed 5 adds nothing, speed 10 doubles the drain. So a non-Assassin runs at 40 per tick = 3.9 stamina per second in light armor and 7.8 per second in a heavy one; the Assassin 2.9 and 5.9. Time from full: a 142-stamina character (36,352 raw) runs 36 s in light armor, 18 s in heavy. In town nothing drains (FUN_0061ab00 true: return 1 at once).

**Q3. At zero. (V read, V run)** After the drain the function re-reads the stamina: if it is above 0 it returns 1 (keep running); otherwise it writes 0 and returns 0, and the caller (FUN_00580c20, run mode 3) switches the unit to walk mode 2 (FUN_0057f090). d2d clamps at 0 and its next frame has `run` false: same result, the same number of ticks (V run, drain differential, 40,000 cases incl. `cur` 0 to 40: 0 mismatch). A run request with stamina 0 does not run (upstream S).

**Q4. Regeneration per tick (FUN_00580500, V run).** By unit mode: standing (1) and town standing (5): `max >> 8`; walking (2) and town walking (6): `max >> 9`, **but in walk mode nothing is added while the stamina is below 1.0 (256 raw)**; running (3) and every other mode: nothing, unless `staminarecoverybonus` (stat 28) is 1000 or more (then `max >> 8`). With a bonus: `add += add x bonus / 100`; the result is capped at max. d2d's `stamina_regen` returns the same value in 40,000 cases (modes 0 to 16, `cur` from 0 to max + 3000, bonus -90 to 1200, max from 0 to 102,400). Standing refill from empty is 256 ticks = 10.2 s (matches the live 7.2 per second on 74); walking regenerates at half that, and **a character that reaches 0 while walking never regenerates until it stands still** (docs/d2d-run.md row "regen walking" says the same; now V run).

**Q5. Equilibrium for a long trip.** Running drains 40 raw per tick and gains nothing; walking regains `max >> 9` = 71 per tick at max 36,352. With the bot's policy (run off below 8 percent, on again from 25 percent, docs/d2d-run.md) the stamina oscillates between those lines; at the logged "2000 of 36352" (5.5 percent) the policy has run off, the character walks and regains about 6.9 stamina per second (71 raw per tick) until 25 percent, so 2000 to 9000 takes about 4 seconds of walking; the walk is not zero (2000 is above 256) so recovery does happen. The Assassin and light armor run longest; armor speed 10 halves the run time (Q2).

## Deviations d2d vs game.exe

None found (drain, zero stop and regeneration are equal by V run). Carried, not run: the wrapping `World::stamina_frame` was transcribed, not called (needs a whole World); the exact tick at which `walking`/`running` are set from the key (U).

## Bot findings (not changed)

- B1 `internal/d2dinput/run.go:100-140` (`RunInfo`, `manageRun`): the believed toggle cannot be read from the game (docs/d2d-run.md: pmode -1 both ways), yet two stamina samples show it exactly (V run): moving outside town, the stamina goes down by a multiple of 40 (light armor) or 80 (heavy) per tick while running, and up by `max >> 9` (71 at 36,352) while walking; standing up by `max >> 8`. A correction from the numbers would stop `runBelieved` flipping. Consumer TASK-329.
- B2 `internal/env` run thresholds (`D2D_RUN_ON`, `D2D_RUN_OFF`): by Q4 a walk that starts below 1.0 stamina is stuck; the 8 percent line keeps the character far above it. No change.
- B3 Town: no drain and no mode 3 benefit in town beyond speed; running in town is free (Q2), so the policy should hold run on in town regardless of stamina.

## Drill recipe (not run)

R1: lab d2d, throwaway Barbarian in Blood Moor, `step 100` after a run key: stamina drops 40 raw per tick (4,000 in 100 ticks); equip Scale Mail (speed 10): 80 per tick. R2: reach 0, release the run: `step 256` while walking: stays 0 (Q4). R3: stand: gains `max >> 8` per tick.

## Matrix proposal

ST08 stamina: "drain, zero stop, regen: V run, 0 mismatch (80,000); speeds S (6/9, 0.375/0.5625); armor speed table D". Bot: B1 open (TASK-329).
