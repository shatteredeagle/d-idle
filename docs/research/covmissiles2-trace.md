# Missiles coverage, second pass: the skill-sourced monster missiles of Acts 1 and 2 (card covmissiles2)

> status: card covmissiles2, 2026-11-02, offline (no game, no d2d slot). Follows docs/research/covmissiles-trace.md (first pass: table / flight of all 684 rows, element hit of 14 skills) and docs/research/monster-missile-trace.md + docs/research/missilediff.md (monster MissA rows). This pass takes the rows those left: the missile rows a monster fires through a SKILL (`srvmissile*` of a Skills.txt row: Fallen Shaman fire, Unraveler bolt, Andariel's spray and poison bolt, the vampire / Countess / Summoner fire walls, Sand Leaper, imp fire), which d2d makes in `andariel_missile` (ai.cpp), not on the MissA path. Nothing is changed in d2d or in the bot.

Tags: **V run** = game.exe's own code run under the emulator and the real d2d loader / rules / `ai.hpp` run by a probe, both sides compared line by line, counted cases; **V read** = read in source or disassembly; **D** = from the patched 1.14d tables (`mpq-table.sh`); **S** = an earlier doc of ours or upstream's; **I** = inferred; **U** = unverified.

## Summary

- **Run.** `cm2_run.sh` (this card): game.exe's `FUN_0059fa30` (-> `FUN_0059f900` -> `FUN_0064b860`) makes each skill-sourced row on a really spawned monster (Blood Moor game, difficulty byte 0 / 1 / 2, `FUN_005a4f50` mode overlay SC, owner damage rewritten with the `mm_diff.py` stat writer) at missile levels 1..40, 50, 60 and random; the record is read back with `FUN_00625480` (physical 0x15 / 0x16 in 256ths, elements 0x30..0x3b) plus the row's range as made (missile data +0x10) and the flag bytes (+0x18b ToHit, +0x186 CollideKill, +0x18a Size, +0x188 NextHit). d2d side: `cm2_probe.cpp`, built against the real d2d scratch build, calls the two things `andariel_missile` calls (`rules::row_damage`, `ai.hpp missile_blow`), and then the real `rules::monster_blow` on the produced stats. 27,000 cases (3,000 and 6,000 per difficulty, seeds 20261102 and 7), 17 rows, 5,074 of them on carried-skill rows (4,243) or Blood Raven's weapon row (831) that are NOT compared; **21,926 compared**.
- **Result (V run).** Element pair, CollideKill, Size and NextHit: **0 differ in 21,926** compared cases. Three deviations: **D1** physical level brackets dropped on Andariel's two physical missiles (1,608 of 1,640 cases differ), **D2** poison stored as a total and read as per-frame (every poison case, 1,640 of 1,640), **D3** the Countess fire wall's lifetime ignores LevRange (845 of 845).
- **PR candidate: none from this card.** D2 is the finding of the local branch `local/missilepoison-skillrows` (docs/research/pr-missilepoison-draft.md): this run re-proves "before" on the real d2d build for Andariel's rows (all cases differ), but I could not build the branch ("after"), so no before / after pair exists here. D1 and D3 have no fix branch.
- **Ledger.** Rows 22, 24, 77, 130, 346, 653 move READ-ONLY -> TESTED-0-DIFF (13,540 cases) and row 32 -> TESTED-DEVIATION (840 cases); rows 129, 131, 203, 277, 331 were run here too but already carry a first-pass table / flight claim and a row cannot be claimed twice, so they are named in the notes only (see Ledger).

## What ran

| run | command (worker host, `tools/emu`) | cases | seed | result |
|---|---|---:|---|---|
| 1 | `run-on-worker.sh cm2_run.sh 3000 20261102` | 9,000 (3,000 x 3 difficulties) | 20261102 | 7,310 compared: physical 551 differ, element 0, range 284, flags 0; poison units 564 of 564 |
| 2 | `run-on-worker.sh cm2_run.sh 6000 7` | 18,000 | 7 | 14,616 compared: physical 1,057 differ, element 0, range 561, flags 0; poison units 1,076 of 1,076 |

Not compared: 1,412 + 2,831 cases on rows that carry a Skills.txt skill (`Skill` column non blank: fireball 62, glacialspike 96, frostnova 119, vampiremeteorcenter 133; d2d takes their damage from the carried skill, `ai.cpp:371-374`), and 278 + 553 on raven1 135 (her record carries her weapon, monster-missile-trace D5, as `mm_diff.py` skips it). 4,243 + 831 = 5,074 cases together; 27,000 - 5,074 = 21,926 compared. The flag bytes of the carried-skill rows are printed in the run output (fireball, glacialspike, frostnova, vampiremeteorcenter: game and d2d equal by eye) but not counted.

Files (all `tools/emu/`): `cm2_run.sh` (build + chain), `cm2_game.py` (game side, reuses `mm_diff.py` / `mm_launch.py`), `cm2_probe.cpp` (d2d side), `cm2_cmp.py` (compare). The game side reads the tables from the worker host's `mmtab` directory (1.14d patch layer, as `mm_diff.py` does); the d2d side loads the MPQ stack at `D2_MPQ_DIR` (patched); both are 1.14d (D). The d2d tree is the worker host's `sorcrest-d2d` checkout (`ai.hpp:657 missile_blow`, `ai.cpp:371-380 andariel_missile`); its commit hash is not readable from this runner (U which upstream commit it is; the file lines are what I read).

Rows made (monster, skill -> row id; D = Skills.txt `srvmissile*` of the Act 1 / 2 classes of Levels.txt and the bosses): Fallen Shaman (ShamanFire) shafire1 22, the Gargoyle trap fire shafire3 24, Unraveler / Radament (UnHolyBolt) unholybolt1 77, Vampire (VampireFireball 129, VampireFirewall 130 / 131, VampireMissile firehead 277, VampireMeteor 133), Countess fire (CountessFirewall: corruptrogue3, maker 653, wall 331), Sand Leaper (Leap) leapknockback 346, Andariel (AndrialSpray 32, AndyPoisonBolt 203), Summoner (Fire Ball 62, Glacial Spike 96, Frost Nova 119, VampireFirewall), Blood Raven raven1 135. Rows not made because no Act 1 / 2 skill names them: willowisp and high priest bolts (320, 343, 666: Act 3 / 5 shooters, out of scope), trap rows 245 (a MissS1 row of a trap object, not a skill), hydra 247 (carried skill, not run here; docs/research/sk-ground2-trace.md has it READ-ONLY).

## Questions

| Q | Question | Answer | Where | Tag |
|---|---|---|---|---|
| Q1 | What does game.exe make of a skill-sourced monster row? | The same builder as a MissA row: `FUN_0059fa30` (flags 0x21, owner unit, row id, level at [0xc]) -> `FUN_0059f900` -> `FUN_0064b860` fills the 0x7c-byte record: physical min = (MinDamage + level bracket MinLevDam1..5) << HitShift (+ owner share x SrcDamage / 128 when the row has one), max alike; elements from EType / EMin / Emax with their level brackets; poison per frame (stat 0x39 / 0x3a, 256ths) and length (0x3b) | `0059fa30` / `0059f900` / `0064b860` | V run (17 rows, 27,000 cases) |
| Q2 | Physical of Andariel's spray and poison bolt | `andarielspray` (32): MinDamage 1792, MaxDamage 2560, HitShift blank = 0, MinLevDam1 / MaxLevDam1 1280: level 1 = 7-10 points, level 4 = 22-25 (5,632-6,400 in 256ths), level 8 = 42-45 (10,752-11,520), level 16 = 82-85, level 35 = 177-180: `(7 + 5 x (lvl - 1))`-`(10 + 5 x (lvl - 1))` points, exact 256ths multiples (0 of 7,310+ compared cases have a sub-point fraction). `andypoisonbolt` (203): 5-7 points at level 1, +5 a level the same way | game.tsv (`cm2_game.py`), table D | V run |
| Q3 | Element pair of the shaman, Unraveler, vampire, imp rows | `shafire1` fire 1-4, `shafire3` fire 2-7, `unholybolt1` magic 3-6 (HitShift 8) with their level brackets; `firehead`, `vampirefireball`: equal to d2d's `rules::row_damage` at every level 1..60 | d2d `ai.cpp:373` / `skills.hpp:582` | V run: 0 differ in all compared cases of these rows |
| Q4 | Poison of Andariel's rows | `andarielspray`: 32-64 per frame (256ths) over 400 frames at level 1 = 50-100 points total (level 4: 128-160 x 430, level 8: 256-288 x 470); `andypoisonbolt`: 32-64 x 800 at level 1 (100-200 points), level 8 298-330 x 870 | game.tsv stat 0x39..0x3b | V run |
| Q5 | Hit roll (ToHit) | ToHit (+0x18b) is 0 on every skill-sourced row made here: they always hit (like d2d's `to_hit = 1 << 20`, `ai.hpp:660`). Only raven1 (135, a MissA row) has ToHit 1; d2d has it too (flags probe: game 1, d2d 1) | record +0x18b, d2d `info.to_hit` | V run (0 flag differences) |
| Q6 | Collision: CollideKill, Size, NextHit | game record vs d2d `info`: CollideKill 1 on andarielspray, andypoisonbolt, fireball, firehead, glacialspike, raven1, shafire1, shafire3, unholybolt1, vampirefireball, 0 on the fire walls, frostnova, leapknockback; Size 1 (leapknockback 2, countessfirewall 1); NextHit 1 only on frostnova. 0 differences in 21,926 compared cases (carried-skill rows by eye only) | record +0x186 / +0x18a / +0x188 | V run |
| Q7 | Lifetime | game range as made = Range + LevRange x level (a short): equal to d2d's `max(range, 1)` on every row but `countessfirewall` 331 (Range 1000, LevRange 1: game 1001 at level 1, 1012 at level 12, 1030 at level 30) | `FUN_0059fa30` (first pass Q2) | V run |
| Q8 | Speed | Vel x 192 x 16 / 65536 per frame as in covmissiles-trace Q2 / Q3 (table-driven, 684 rows x 20 levels, V run there); the rows here have no VelLev (D: column blank in the printed rows) and none is on the Accel list of covmissiles-trace Q4 (D), so the formula is exact for them: shafire1 Vel 8 (0.375 subtiles a frame), shafire3 12, unholybolt1 12, andarielspray 15 | covmissiles-trace Q2 | V run (there), D (columns) |
| Q9 | Pierce column | d2d sets `pierce` on andarielspray, shafire1, shafire3 and not on the other rows; the game record's Pierce byte is not in the offsets of docs/research/re/missiles.md and was not read, so no game-side value is compared | `ai.hpp` info.pierce | U |

## Deviations d2d vs game.exe

| D | Rule in game.exe | Rule in d2d | Tag | Differential |
|---|---|---|---|---|
| D1 | physical = (MinDamage + (level - 1) x MinLevDam1 for level <= 8, then the later brackets) << HitShift: Andariel's spray 7-10 points at level 1, 42-45 at level 8, 82-85 at level 16 | `missile_blow` takes `info.min >> 8`, `info.max >> 8`: the row's raw MinDamage / MaxDamage, no bracket: 7-10 points at every level; the poison bolt 5-7 | V run: physical 1,608 differ of 1,640 cases on the two rows (the 32 that agree are level 1, I: the bracket is 0 there); at levels 1 / 4 / 8 (the levels d2d's `monster_skill_level` gives a skill level 1 monster, I): 1,384 compared cases of all rows, 69 differ, all on these two rows (counts below); same cause as missilediff D1 on the MissA path | yes, built and run (`cm2_run.sh`); the existing fix of missilediff does not touch `andariel_missile` |
| D2 | poison = per-frame 256ths (stat 0x39 / 0x3a) over frames (0x3b); total = per-frame x frames / 256: Andariel's spray 50-100 points at level 1, 128-160 per frame x 430 in Nightmare | `missile_blow` stores the TOTAL as a type-3 element (`ai.hpp:664`: `elo x frames >> 8`, length `elen`), which `monster_blow` reads in the MonStats El convention (per-frame roll x 10, 2 x duration ticks): level 1 spray 500-1000 per frame x 800 ticks = 1,562-3,125 points against 50-100 (x31); level 16 47,265-50,187 against 1,100-1,168 | V run: 1,640 of 1,640 poison cases differ (the real `monster_blow`); element pair equal (0 differ) so the fault is the unit, not the roll | yes, built and run; fix exists as `local/missilepoison-skillrows` (S docs/research/pr-missilepoison-draft.md); not rebuilt here |
| D3 | lifetime = Range + LevRange x level | `andariel_missile` uses `max(range, 1) x 40 ms`, no LevRange (`ai.cpp:378`); `missile_from` (`ai.hpp:672`) has the term, so it is the one call that forgot it | V run: 845 of 845 cases on countessfirewall (331), 0 on the other 16 rows | yes, built and run; effect small (+1 frame a level on a 1,000 frame wall: I) |
| D4 | `CollideFriend` / `CollideType` filter and `Pierce` chance | never read / own rule | S (first pass D3 / D4) | not run here; first pass recipe stands |

Level-1 / 4 / 8 count for D1: the run printed "503 cases, 27 differ" (seed 20261102) and "881 cases, 42 differ" (seed 7) for ALL compared rows at those levels, 172 + 281 of them at level 1 with 0 differ: the D1 deviation shows from level 4 on, so Normal Andariel (level 1, I: d2d's `monster_skill_level`, game side not run) agrees and Nightmare / Hell would not (I).

## Bot findings (described, not changed)

| B | Where | What | Tag |
|---|---|---|---|
| B1 | `internal/combat/threat.go:15-36` (`ThreatFromRow`, field `Dmg: m.MaxDamage`) | `Dmg` is the raw MaxDamage column; game.exe shifts it by HitShift (and adds the level brackets): Andariel's spray reads 2,560 where the real hit is 10 points at level 1 and 45 at level 8; the same column reads 1-4 for a HitShift 8 row that really means the same number of points. The unit differs by row | V run (game record), V read (bot); whether `Dmg` reaches a decision: U |
| B2 | `internal/combat/threat.go:15-36` speed | `Vel x 0.046875 x 25` is exact for all 17 rows of this pass (no Accel, no VelLev: D), so first-pass B2 does not bite here | D, V run (formula, first pass) |
| B3 | `internal/data/extra.go:361-366` `RangeAt` | `Range + LevRange x (lvl - 1)`: one LevRange short of game.exe's `Range + LevRange x lvl` (first-pass B1); on this pass's rows only countessfirewall (LevRange 1) is affected, by one frame of 1,000 | V run (game), V read (bot) |
| B4 | `internal/combat/threat.go` `Blockable: m.ToHit \|\| m.MaxDamage > 0` | for the rows here the block gate of game.exe (the missile's physical stat above 0) holds for andarielspray and andypoisonbolt (physical 7-10 / 5-7 points at level 1, V run) and fails for every shaman, Unraveler, vampire and imp row (no physical), which the bot also gets right (MaxDamage 0 there); consistent, no finding | V run, V read |

## PR candidates

None. D2 (poison units) is already a local branch's finding (`local/missilepoison-skillrows`, S) and this run adds a second "before" on the real d2d build; there is no "after" run here, so the before 1,640 / after 0 pair that a candidate needs is not proven by this card. D1 (brackets, skill path) and D3 (LevRange) would be new siblings of missilediff; they need a fix and an independent verifier run first.

## Dead ends and limits

- Rows with a carried Skills.txt skill (62, 96, 119, 133) are not compared: d2d's `carried` branch (`ai.cpp:371-374`) needs the monster's cast skill object, which my probe does not build. They stay as the existing lines have them (62, 96 hit rows tested, 119 and 133 read).
- The missile level a skill-sourced monster missile really gets (skill level + difficulty bonus) was not run: d2d's `skill_level` / `monster_skill_level` is what I used to say which levels matter (I). Needs the monster's skill do-function run, or a live `debug missiles` on Andariel.
- Pierce and CollideFriend / CollideType were not read (Q9, D4).
- My first compare used one rng seed of the real `monster_blow` for the poison and exempted rows whose min and max differ, so D2 first showed only Blood Raven's weapon rows; the probe now takes the range over 200 seeds (that is the run counted above).
- Whether the runner's `sorcrest-d2d` tree equals upstream main for `ai.hpp:657` was not checked by commit (no git access from here); its `missile_blow` equals what the missilediff verifier called wrong on upstream main (S, docs/research/pr-missilerecord-verifier.md).
- Not covered by this pass: every row not named in the Ledger section; the rows with only a table / flight claim (docs/coverage/ledger.tsv) still have no hit or collision test.

## Drill recipe for the fork verbs (lab only, one hidden slot, nothing here was run)

Verbs: docs/forkdbg.md (`debug spawn`, `debug mon set`, `debug missiles`, `debug points`, `dump stats`); binary and lockstep: docs/d2d-fork.md, docs/forkcheats.md.

- C1 (D1, D2 live): spawn Andariel (`debug spawn`), `debug goto` her level, stand in range, `step 1` until she sprays, `debug missiles` each frame: the row `andarielspray` with Vel 15, range 40; then read the life lost by a throwaway character with `dump stats` over the next 400 frames. game.exe: poison about 50-100 points in Normal (D2 predicts d2d loses 1,500-3,000), physical 7-10 per hit in Normal.
- C2 (D3): spawn a Corrupt Rogue and let her cast the Countess fire wall; `debug missiles` frame count of `countessfirewall`: 1,001 (game, level 1) against 1,000 (d2d).
- C3 (level): `debug mon set` cannot set a skill level; to learn which missile level Andariel's spray gets in Nightmare, compare the physical of a hit with the table: 22-25 points means level 4 (Q2).
- C4 (carried skills): summoner's Fire Ball / Glacial Spike / Frost Nova rows against `debug cast 47 unit <id>` of the player for the same skill level, to see whether the carried skill's damage equals the player's.

## Matrix-cell updates (proposed text; docs/research/monster-missile-trace.md Q4 table is not edited here)

- Row `shafire1` (22), `shafire3` (24), `unholybolt1` (77): add "skill path (andariel_missile): element pair 0 differ in 3,319 / 878 / 3,416 cases, flags and size equal (covmissiles2)".
- Row `andarielspray` (32), `andypoisonbolt` (203): add "skill path: physical brackets dropped (D1: 7-10 points at every level against 42-45 at level 8) and poison stored as a total read as per-frame (D2: x31 at level 1); counts in covmissiles2-trace".
- Row `countessfirewall` (331): "lifetime ignores LevRange (D3, 1 frame a level)".
- Rows `fireball` 62, `glacialspike` 96, `frostnova` 119, `vampiremeteorcenter` 133: "carried skill, not compared on the skill path (U)".

## Ledger

`docs/coverage/ledger.tsv`: two new lines, rows 22, 24, 77, 130, 346, 653 (TESTED-0-DIFF, 13,540 cases: shafire1 3,319, shafire3 878, unholybolt1 3,416, vampirefirewallmaker 2,541, leapknockback 2,570, countessfirewallmaker 816; the last three have no damage, so the aspect is range, flags and size) and row 32 (TESTED-DEVIATION, 840 cases, D1 D2); the READ-ONLY line of monster-missile-trace loses those rows. Rows 129, 131, 203, 277, 331 were run too (1,716 + 2,501 + 800 + 1,684 + 845 cases) but hold a TESTED-0-DIFF table / flight claim from the first pass, so they are not re-claimed; their deviations (D1 D2 on 203, D3 on 331) are listed here only.

## Shared-file lines for the merger

See `openspec/changes/covmissiles2/merge-lines.txt`. Ledger and generated table: `docs/coverage/ledger.tsv`, `docs/coverage/rule-coverage.md` (regenerated here).
