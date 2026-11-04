# PR body: Energy Shield absorbs the raw field before the resistance

> status: DRAFT body for an Energy Shield pull request to psi29a/d2decomped (jobs esprep, esverify, esfix, esverify2, esbody2; 2026-11-04). Fork branch `local/es-absorb-order-r3`, head `3b8bb7df`, 1 commit on upstream main `0646c5da` (private fork remote only). r3 is r2 (`229aeca7`, on `561814df`) cherry-picked onto the current upstream main without a conflict (only `components/game/ai.hpp` moved upstream, in a different hunk); the counts below were measured on `229aeca7` and the code is the same. NOT posted, nothing pushed to GitHub, no PR opened. Tags: V = run (how stated), S = read without a run, U = unverified. Independent verifications: `docs/research/esverify/verdict.md` (first head `12b78b60`) and `docs/research/esverify/verdict2.md` (r2 `229aeca7`, verdict CONFIRMED WITH CAVEATS, code unchanged).
>
> Title: `Fidelity fix against game.exe 1.14d: Energy Shield absorbs the unresisted field`

## What changes

`Fight::absorb` took the Energy Shield's share from the resisted total of the frame. game.exe raises the shield's absorb event (0xb, `FUN_005ca6c0`) on each damage record before the resistance pass (`FUN_0057c1e0`), so the shield works on the raw field and the resistance, the flat magic reduction and the physical reduction work on what is left: the flat reduction first, then the resistance, and a field left under 0 stays negative in the record total (`FUN_0057bf80`), which is clamped once at the end.

One commit, one claim: for every blow the shield takes `MulDiv(field, calc1, 100)` of each unresisted field (physical, fire, lightning, cold, magic, in the table order of `0x6e30e0`), at most `MulDiv(mana, 16, calc2)`, drains `MulDiv(taken, calc2, 16)` mana per field, record by record, and the rest goes through the resistance.

Why it matters: a percent-limited shield gives the same result in both orders up to rounding. They diverge when the mana limit binds (mana pays a fixed amount of raw damage in game.exe, of resisted damage in d2d) and at vulnerability or immunity (game.exe pays mana for damage that is then fully resisted; d2d paid nothing at 100 % resistance).

| Code | Change |
|---|---|
| `Taken` | keeps the rolled fire / lightning / cold / magic before the resistance and what each dealt (`element_raw`, `element_dealt`) |
| `StruckBlow` | carries them with the resistance and the flat reduction they went through (`parts`, `mdr`); `Foe::take` fills them |
| `rules::EnergyShield`, `rules::energy_shield_blows` | the shield's arithmetic on a span of blows; a field it took from is put back through the resistance (the flat reduction off first, then `resisted_fine`; physical through `reduce_fine_signed`); a field under 0 stays in the record total, clamped once |
| `Fight::absorb` | calls it where the old block was; what is left of the mana is stored |

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `005ca6c0` | wearer has the state; pct = calc1, ratio = calc2 (at least 1); loop over the table at `6e30e0`: per field above 0, `a = MulDiv(field, pct, 100)`, `a = min(a, MulDiv(mana, 16, ratio))`, `field -= a`, `mana -= MulDiv(a, ratio, 16)` floored at 0 (V read) |
| `6e30e0` | 8 rows {record offset, monster-only}: +8, +0x10, +0x1c, +0x24, +0x20 always; +0x38, +0x3c, +0x40 only from a monster. Poison is not in it (V read, bytes read from the image) |
| `0057c1e0` | raises event 0xb on each record BEFORE it reduces it (the loop at `0x57c378`), then runs the resistance and reduction (V read) |
| event 11 | fires once per record at queue time, before the resistance pass (V run: in the second verification every shield call saw the raw fields, `snapshot_not_raw 0` over 95,453 shield calls; first seen in an earlier trace with a hook on the dispatcher `FUN_005c0c30`) |

## How the claim was measured (V run, the worker host)

Two harnesses, both with the same shape: a d2d side (the real `rules::monster_blow`, `Foe::take` and `Fight::absorb` linked from a build of each head, the blow made exact with `sure` / `exact`) and a game side under the emulator.

1. First draft (job esprep): the game side ran game.exe's real `FUN_0057c6c0` with a hook that did the shield's arithmetic as a formula (I). 60,000 single-element blows, seeds 1 to 3.
2. Verification (job esverify, V run): the game side runs game.exe's own `FUN_005ca6c0` natively. The dispatcher `FUN_005c0c30` is a hook that at event 11 for the wearer sends execution into the real function and back, so the shield works on the record where game.exe's state node would, and the resistance pass then runs on what it left. Nothing of the shield arithmetic is written on the game side; the skills row, calc1 / calc2 (d2d's own Skills.txt values), the state calls and the stat reads are hooked. Cases: an edge grid plus seeds 7, 8 and 9 x 20,000 frames of up to 3 blows, physical plus up to three elements a blow, resistance -100..100, physical reduction 0..50 %, flat reduction and magic damage reduction 0..15 (about 20 % of the cases each), levels 1..99, mana 0..600 points with fractions and edges. Every shield case has a control twin: the same frame without a shield. "Comparable" below means the control twin agrees exactly (256ths) on game.exe and both d2d heads, so a difference is the shield's own.

Findings of step 2 on the first head `12b78b60` (V run): mana equal in all 51,065 shield cases; life equal on every comparable case without flat reduction or magic damage reduction (15,621); 171 of 24,174 comparable shield cases differed in life, all with flat reduction or mdr (caveat C1, below). The r2 commit fixes C1.

### C1 and the fix (V run, native)

game.exe's resistance pass (`FUN_0057bf80`, per field of the table at `0x732980`, read in the disassembly) does, for a field above 0: take the flat reduction off (physical: stat 34, then the percent; fire, lightning, cold, magic: the magic damage reduction), THEN apply the resistance if the result is above 0, and store the result, which may be under 0. `FUN_0057c1e0` sums the stored fields into the record total (+0x4c) and the life update ignores a total that is not above 0 (`0057c852`). So a field the shield leaves below the reduction goes negative and takes damage off the other fields of the record. Native probes (`esv_probe8515.py` of the esverify harness, and a no-shield probe on the same game side): physical 215, lightning 39, cold 10, flat 15, mdr 6, a 95 % shield with 119,494/256 mana deals 925/256 (physical -1088, lightning 989, cold 1024); the first draft's head gave 2,013/256. Without a shield, 100 fire at 50 % with mdr 10 deals 45 points (reduction first), and 3 fire plus 40 lightning with mdr 6 deals 31 (the first field's -3 is summed).

The r2 commit does this for every field the shield took from: reduction first, then the resistance, the negative kept in a local record total, clamped once at the end of each blow (`reduce_fine_signed`, `energy_shield_blows`). `test_energy_shield` holds the 925/256 case, a case with one negative field beside a positive one (43 points, not 44), the zero-flat control (55 points), and the flat-reduction-before-resistance case ((80 - 10) x 0.5 = 35; the first draft's test had 30, which assumed the opposite order).

### Counts before and after (V run: seeds 7, 8, 9 x 20,000 frames plus the 500-frame edge grid, the same cases on both heads and on game.exe)

| Comparable shield cases (control twin exact) | Cases | First head `12b78b60` life differs | r2 `229aeca7` life differs |
|---|---|---|---|
| seeds 7 + 8 + 9, whole points | 24,174 | 171 (209 in 256ths) | **0** (0 in 256ths) |
| seeds 7 + 8 + 9, mana (whole points and 256ths) | 24,174 | 0 | **0** |
| edge grid (seed 0), whole points | 5,500 | 25 (81 in 256ths) | 24 (80 in 256ths) |
| edge grid, mana | 5,500 | 0 | 0 |

The 24 edge-grid cases left are caveat C2 (all with a physical field above 4,096 points and a percent reduction; checked case by case). All 8,802 edge-grid shield cases: mana 0 differ on both heads. No-shield frames: the r2 and the first head give identical output on all 60,000 (the change is off the shield path); the base `561814df` equals them too (V run, both verifications).

Not comparable, so not claimed: the shield cases whose control twin already differs from game.exe (about 8,900 per seed). Those frames differ with no shield at all, so the difference is not the shield's: d2d resists each element in whole points where game.exe keeps 256ths and sums the record; and d2d clamps each part at 0 and takes the magic damage reduction after the resistance where game.exe takes it first and keeps a negative (the finding above). The same applies to d2d's code on frames without a shield; this change does not touch it.

Mana counts in the first draft (the 60,000 formula-hook cases) and its table are superseded by the native run; they said 0 as well.

### The magic-resistance rows (not part of this change)

The 2,262 shield rows with magic damage and a resistance other than 0 are left out of the zero above: they differ the same 1,951 on base and head, in life only (mana equal). d2d applies no percent magic resistance to a monster's magic blow (only the flat reduction); game.exe applies stat 0x25. The control rows without a shield show the same (577 cases, 560 differ). A separate rule, a separate change.

## Second verification and new seeds (V run, esverify2, head `229aeca7`)

New seeds 21, 22, 23 (20,000 frames each) and a new edge grid with flat / mdr, poison and two-field records, game side again `FUN_005ca6c0` natively:

- Comparable shield cases 23,931: head life differs 0 (whole points and 256ths), base 4,655 (6,768 in 256ths). Mana over all 50,977 shield cases: base differs 4,483, head 0.
- New edge grid: 11,006 comparable cases, 9 left (7 in whole points), all K1 below (a field after the mana ran out). Mana: head 0 of 24,655 shield cases (4,418 with poison, where game.exe takes no mana).
- No-shield frames: base and head output equal on all of them (86,588), the change is off that path.

## Tests (V run, the worker host, Release, g++, `-DD2D_WERROR=ON`, head `229aeca7`; rerun on `3b8bb7df` below)

`tests/game/test_energy_shield.cpp` has two parts. Without game data it checks `rules::energy_shield_blows` on typed blows: the seed 1 case 9 numbers (cap binds, vulnerable), calc1 0, mana 0, immune at 100 %, the flat magic reduction before the resistance, a negative field summed into the record, a physical field under the flat reduction, a field taken to nothing losing its element mark, mana running out across two records, and damage no field accounts for. With the data it drives the real `monster_blow`, `Foe::take` and `Fight::absorb` over a grid of 2,400 cases (all six damage types, resistances -100, -50, 0, 50, 99, 100, raw 1, 5, 50, 632, 1500, levels 0, 1, 18, 40, mana 0, 3.4, 100 and 600 points) against the arithmetic written out in the test, and the case above (860 points, 1/256). It skips with 77 when the 1.14d data is missing.

## Callers and what else moves

- `Fight::absorb` is the only caller; the other armors (Bone Armor, Cyclone Armor) run before the shield as before and are unchanged.
- Blows built without the new fields (Corpse Explosion, Diablo's cold touch, a Fire Golem's burn, a burn aura) and any damage no blow carries are taken from as they stand, which is what the shield did to everything before (S: source read; the test covers the first case).
- The shield state is not removed when the mana reaches 0. game.exe's `FUN_005ca6c0` removes it whenever the mana it leaves is 0, also when it absorbed nothing (mana 0 on entry), not when calc1 is 0 (V run, native: the state was removed in 9,232 to 9,338 of about 17,000 shield cases per seed). In play the shield ends on the first hit at empty mana; unchanged here, a separate change (limit C3).
- Siblings: only the player carries an Energy Shield; a merc, a pet or a co-op partner has none, so nothing else changes (S).
- `calc1` is still clamped to 0..95 where `FUN_005ca6c0` has no cap; no effect at levels 1 to 30.

## Checks (V, the worker host, head `229aeca7`; `pr_guard` and tests also on `3b8bb7df`)

- Author and committer are the numeric noreply identity; the body ends with `Agent: Claude Code`; no `-x` line; 3-line body.
- `python3 tools/pr_guard.py 561814df HEAD`: 0 FAIL, 0 WARN. On the rebased head `3b8bb7df` against upstream `0646c5da`: 0 FAIL, 0 WARN; Release build with `-DD2D_WERROR=ON` and with the 1.14d data, `test_energy_shield`, `test_combat` and `test_rules` return 0 (V run).
- Release build, `-DD2D_WERROR=ON`, g++: `test_energy_shield`, `test_combat`, `test_rules`, `test_game_rules`, `test_game`, `test_monsters`, `test_skills`, `test_monster_missile` return 0.
- clang-tidy with the repo's `.clang-tidy` on `fight/states.cpp` and `test_energy_shield.cpp`: no finding in the touched files.
- `python3 tools/presubmit.py --quick --base 561814df` (on r2): 0 FAIL, 3 WARN: unit kinds (answered above: only the player has the shield), build and gcc skipped by `--quick`.

## Not re-run

- MSVC, clang++ suites, the full parity lane, `test_game` neighbours beyond the list above, a live game, a soak, a determinism hash comparison.
- Cases with real shield and armor together (Bone Armor or Cyclone Armor with Energy Shield), and the monster-only fields (below).

## Limits

- U (S for the order): absorb states registered later run first in game.exe (`FUN_005c0ad0` puts the event node at the head of the unit's list); d2d keeps armor first, shield second. With an armor up the order is not matched, and a Cyclone Armor cut is not put back through the resistance (marked `ponytail`). Reaching it needs Bone Armor or Cyclone Armor together with Energy Shield (another class's skill, by item skills only).
- C2 (V run, pre-existing, left alone): a physical field above 4,096 points (0x100000 in 256ths) with a percent reduction: game.exe divides first (`(fine / 100) x (100 - pct)`), `reduce_fine` multiplies in 64 bits. Not reachable by real monster hits; it is the 24 edge-grid cases left above.
- C3 (V run, native, left alone): the state is removed by game.exe whenever the mana ends at 0, even with nothing absorbed; d2d keeps it. This change claims the single frame, not the frames after it.
- C4 (V read, untested): the three monster-only fields (+0x38, +0x3c, +0x40) are absorbed only when the attacker is a monster that is not a hireling; the harness attacker was a player unit with those fields 0. d2d has no such fields on a blow.
- K1 (V run): blows no shield touched keep d2d's per-part values (resistance, then mdr, clamped at 0), where game.exe takes mdr first and keeps a negative; frames without a shield differ the same way (above). Only the fields the shield took from follow game.exe's order. This includes fields after the mana runs out in the same record: one record can mix the two orders (hand case: physical 200 + fire 100, mdr 10, res 50, mana for part of the physical: game.exe 244.414, d2d 239.414).
- Follow-up, not in this PR (V run, game.exe natively, no shield): 100 fire at 50 % resistance with mdr 10 deals 45 on game.exe and 40 on d2d; negative fields are summed, 93 on game.exe and 94 on d2d (fire 5 + cold 100, mdr 6). 41 % of the shield-free frames with flat or mdr differ (593 of 1,449 and 414 of 1,097 in two runs). Fixing the no-shield order changes `rules::monster_blow` for every monster hit, so it is a separate PR.
- Two rows of one element type in one blow are resisted per row (as before) when no shield is up, and as one summed field when it is (U: no monster has two rows of one type that fire in the same mode; not checked against every row).
- Magic damage has no percent resistance on a monster blow in d2d (the 2,262 magic-with-resistance rows of the first run are reported, not counted); its field goes through the flat reduction only.
- `calc1` is still clamped to 0..95 where `FUN_005ca6c0` has no cap; no case reaches the clamp (V run, levels to 99).
- The differential harness is not part of this PR (it lives in the lab repo) and upstream cannot repeat its counts; only `test_energy_shield` is repeatable by upstream, so the counts above are cited, not proven, for the maintainer.

Agent: Claude Code
