# Fidelity fixes against game.exe 1.14d: skill mana check and payment, Teleport target rules, monster damage roll

> status: FINAL BODY on the NEW upstream template (card b2rebase2, 2026-11-03), not posted. Nothing is pushed to the public fork and no PR is open: the coordinator opens it after the user's go. Branch `local/batch2-bundle-r3` on the private fork remote, head `824da884`: 4 commits on upstream main `005c7cab` (PRs 179 and 180 merged; the maintainer's batch rules `549da358` read, see [pr-upstream-rules-20261103.md](pr-upstream-rules-20261103.md)). It replaces `local/batch2-bundle-r2` (head `6f036a40` on `552acc9a`, kept). The rules may ask for two batches instead of one: split bodies in [pr-batch2-body-split.md](pr-batch2-body-split.md). Tags: V = run for this head (how stated), S = secondary (an earlier run of ours or a verifier's report, carried because the patch is identical, not re-run for this head), U = unverified. Raw results: [pr-batch2-run.txt](pr-batch2-run.txt).

---

## Summary

Three fixes where d2d differs from game.exe 1.14d, four commits, one purpose per commit. Each has a differential: game.exe's own functions run natively under the emulator in `tools/emu`, the real d2d build runs the same generated cases, and the two are compared. Size: 17 files, +735 / -62 (`git diff --shortstat upstream/main...HEAD`).

| # | component | commit | files | +/- |
|---|---|---|---|---|
| 1 | Skill mana: the check has no minmana floor, the payment needs enough mana | `486e8ff1` | 8 | +190 -11 |
| 2 | Teleport lands only in the caster's room or its near list | `4fd9b356` | 2 | +148 -0 |
| 3 | Teleport lands on the nearest free spot of a blocked target | `9dda1012` | 3 | +53 -3 |
| 4 | A monster's hit is rolled in 256ths, absorbed, then reduced | `824da884` | 9 | +346 -50 |

Rebased from `552acc9a` onto `005c7cab` by `git cherry-pick` of the four commits in order: no conflict (V). `git range-diff 552acc9a..local/batch2-bundle-r2 upstream/main..HEAD` shows all four patches `=` (identical). Auto-merged by git: `gamedata.hpp` (PR 180 edited 4 lines of it elsewhere) and nothing else listed; PR 179 and 180 also touched `gamedata.cpp`, `gamedata_load.cpp`, `test_game_missiles.cpp`, `test_game_tables.cpp` and `test_static_field.cpp` (2 to 4 lines each, none in our hunks). Neither adds a caller of `monster_blow`, `Fight::pay`, `pet_hurt`, `mana_cost`, `Level::warp_spot` or the Teleport branch (V: file list of `git diff 552acc9a upstream/main` against our 17 files).

## Claims (batch)

One row per commit, as `.github/pull_request_template.md` asks. Commit 3 builds on commit 2 (same `Fight::spot` Teleport branch and test file; V: applying 3 alone on main conflicts); commits 1, 2 and 4 apply on main in any order (V: cherry-pick of 2, 1, 4 in that order is clean).

| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| `486e8ff1` | The skill mana check has no minmana floor; the payment takes `max(formula, minmana x 256)` and only when enough mana | `0056c160` (check), `0056bfe0` (payment) | `pb2_mana.sh 100000 7 BASE_BIN HEAD_BIN` (bins `test_sk_mana`): 105,978 compared, base 281 differ, head 0; `ctest -R "test_sk_mana|test_skills"` |
| `4fd9b356` | Teleport lands only in the caster's room or its near list | `00554ea0` (warp), `00463740` (room lookup), `005ca360` | `pb2_tele.sh 100000 1 BASE_BIN D1_BIN HEAD_BIN` (bins `test_tele_spot`): 40,974 not-near cases, base 15,854 equal, this commit 40,974; `ctest -R test_tele_spot` |
| `9dda1012` | Teleport lands on the nearest free spot of a blocked target (rings 1 to 49) | `00554ea0`, `0064e7b0`, `0064dea0` | same run: 1,461 blocked-target cases, 0 before, 1,461 after; the ring-49/50 asserts in `test_tele_spot` |
| `824da884` | A monster's hit is rolled in 256ths, absorbed on the unreduced physical, then reduced; fixed records keep min == max | `0057b420` (roll), `0057c1e0` (record order), `0057bf80` (reduce), `005c6370` (Bone Armor), `005ca6c0` (Energy Shield) | `v4_run.py 6000 20261103`, `mdmg2.py 20000 20261103`, `v3_run.py 10000 314159`; `ctest -R "test_combat|test_game"`; Energy Shield is NOT equal (section 4) |

## 1. Skill mana: no minmana floor in the check, payment only when enough

game.exe's check `FUN_0056c160` compares the mana with the formula alone, `(mana + lvlmana x (level - 1)) << manashift`; it has no minmana floor. The payment `FUN_0056bfe0` takes `max(formula, minmana x 256)`, and takes nothing when the mana is short of that or when the row has mana 0 and lvlmana 0; the cast has already happened by then. d2d refused a cast under the floor and paid the floor (Might: one point). `rules::mana_formula` is the check, `rules::mana_paid` the payment, `Fight::cost_of` is the check and `Fight::pay` takes `mana_paid`, on all four cast paths (`cast`, `cast_missile`, `cast_summon`, `start_swing`).

Counted (V, `tools/emu/pb2_mana.sh 100000 7`: every Skills row x levels 1..60 x thresholds around the formula and the floor, plus 100,000 random; game.exe's side is `sk_mana.py`, `FUN_0056c160` then `FUN_0056bfe0` natively; d2d's side is the real `Fight::cast`, `cast_missile`, `cast_summon` and `start_swing` in `tests/game/test_sk_mana.cpp`): 190,840 cases, of which 105,978 have a cast path in this build and are compared (a passive, an aura, a corpse skill or a weapon the test character lacks has none). Base (upstream main plus the test file): 281 differ; head: 0 differ. All 281 are "game.exe casts, d2d refuses": Teleport 74, Dopplezon 72, Guided Arrow 68, Bone Prison 67. The same run on the earlier branch heads gave the same 281 and 0 (S, `pr-batch2-draft.md`); an independent verifier's own model and probe on the real builds, seeds 3 and 11, found base 117 of 28,454 and 106 of 28,534 differ, head 0 (S, `verify-batch2.md`).

Limits: Might and Summon Resist have no cast path in this harness, so they are not claimed. `FUN_0056bfe0` also has a state 0x72 branch (`FUN_005d2b60`) and a srvdofunc 116 shortcut (Werewolf, Werebear) that this change does not model; both are unchanged from main (U; named in the `mana_paid` comment). Both game functions return at once for a unit that is not a player, so the change is for the player (V read by the verifier); whatever d2d does for a merc's or a pet's mana through `Fight::pay` is outside the evidence. Timing of the payment and interrupted casts are not part of this commit.

## 2. Teleport lands only in the caster's room or its near list

`FUN_00554ea0` finds the target's room with `FUN_00463740` (the caster's room, then its near list) and refuses when there is none; d2d landed anywhere. `Fight::spot` now checks `missile_rooms` for Teleport (srvdofunc 27) only; Dragon Flight and Leap share the branch and keep their old rules.

Counted (V, `tools/emu/pb2_tele.sh 100000 1`: `FUN_005ca360` and the warp `FUN_00554ea0` natively on 5 x 5 rooms of 8 x 8 tiles; d2d's side puts the case's collision words into a copy of a real level and calls the real `Fight::spot`, `tests/game/test_tele_spot.cpp`; the units' bit 0x1000 is taken out of the cases because d2d keeps units in the Crowd), 100,000 cases, equal to game.exe's answer (landed or refused, and the subtile):

| class | cases | base | after commit 2 | after commit 3 (head) |
|---|---|---|---|---|
| free target in a near room | 52,539 | 52,539 | 52,539 | 52,539 |
| Teleport 0 | 5,026 | 5,026 | 5,026 | 5,026 |
| target room not near | 40,974 | 15,854 | 40,974 | 40,974 |
| target blocked | 1,461 | 0 | 0 | 1,461 |
| all | 100,000 | 73,419 | 98,539 | 100,000 |

Commit 2 removes the 25,120 cases where game.exe refuses and d2d landed. A second set (seed 9, S: run on `6f036a40`, not re-run): base 73,088, after commit 2 98,445, head 100,000 of 100,000 (room not near 40,969, free target 52,397, Teleport 0 5,079, blocked 1,555; base landed 25,357 times where game.exe refuses).

Limits: the oracle's near list is the 3 x 3 touching rooms (a hook); d2d's near list gives the same set for 8 x 8 tile rooms, but real levels with irregular rooms are not compared (U). `FUN_0064dea0` looks each candidate up from the last room it found, so game.exe's valid rooms depend on that walk; d2d's `warp_spot` has no room test of its own (S, verifier note F6). Teleport value 2 (the clear-line test) is unchanged and not covered (the oracle hooks it to clear).

## 3. Teleport lands on the nearest free spot of a blocked target

`FUN_00554ea0` hands the target to `FUN_0064e7b0` and `FUN_0064dea0`: the subtile itself when it is free (the plus footprint, mask 0x1c09 less the units' bit), else the first square ring of subtiles, rings 1 to 49 (it goes on only while ring + 1 < 50), that holds a free spot, in it the least |dx| + |dy|; ties follow its scan order (left and right columns row by row, then top and bottom rows column by column, then corners). `Level::warp_spot` does that and `Fight::spot` uses it for Teleport only; d2d refused a blocked target.

The 1,461 blocked-target cases are in the table of section 2: base and commit 2 differ in all of them, the head in 0 (V). The ring limit and the tie order were read in `FUN_0064dea0` by the independent verifier and matched by his run of game.exe's function against the real `Level::warp_spot`, 20,000 cases, 18,990 moved spots, 0 differ on rings 1 to 49 (S, `verify-batch2.md`; his finding F1, a ring 50 that game.exe refuses, is fixed in this commit and pinned by an assert in `test_tele_spot`: a square blocked out to ring 49 is refused, one blocked out to ring 48 lands on ring 49's first corner).

Limits: the plus footprint as the player's pattern is read (the oracle runs `FUN_0064d9b0` natively) but was hooked in the verifier's harness (S). Ring 50 as the first free ring is rare in the generated sets; the unit test pins it.

## 4. A monster's hit is rolled in 256ths, absorbed, then reduced

`FUN_0057b420` rolls `(min << 8) + rand((max - min) << 8)` on the attacker's seed, with min taken at least 1 and max at least min + 1 (so `0..0` rolls 1..2 and draws once, and the top is never rolled); d2d rolled whole points, inclusive. `FUN_0057c1e0` then raises the absorb event on the unreduced physical (Bone Armor `FUN_005c6370`, Energy Shield `FUN_005ca6c0`), then reduces the record, flat first and then percent (`FUN_0057bf80`), and the life stat takes the result in 256ths; a result under one point is 0 (`0057c852`). d2d: `rules::monster_blow` rolls in 256ths and returns `Taken::damage`, `Taken::frac` (0..255) and `StruckBlow::frac`; `rules::reduce_fine` does the reduction in 256ths; `rules::absorb_blows` and `Fight::absorb` take the Bone Armor and Cyclone Armor pools in 256ths off the unreduced field and put the reduction back; `Fight::hurt_side` takes `(damage << 8) + frac` off the player's life and treats a life under 256 as dead; pets get the fraction through `pet_hurt(.., fraction)`, the merc through its `life_frac`. Fixed records (Corpse Explosion, Diablo's cold touch: min == max) reach `FUN_0057c1e0` without a roll, take no draw, and keep the reduction and the absorbs.

Counted on `6f036a40`, carried to this head (S: the patch is identical and not re-run on `824da884`; originally V on the worker host; game.exe's `FUN_0057b420` natively, then a game-order model of `FUN_0057c1e0` written from the disassembly; the d2d side is the real compiled `Fight::absorb`, linked from this head's `libgame.a`, and the real `Fight::hurt_side`). Frames where d2d differs from game.exe in life / pool, seed after in every row 0:

| set (8 Act 1 classes) | frames | base | head |
|---|---|---|---|
| roll (no reduction, no pool) | 48,000 | 47,896 / - | **0 / 0** |
| reduce (percent, flat) | 48,000 | 23,954 / - | **0 / 0** |
| absorb (Bone Armor pool, 1 to 3 blows) | 48,000 | 16,403 / 33,587 | **0 / 0** |
| both (reduction and a pool) | 48,000 | 4,121 / 41,931 | **0 / 0** |
| negative percent | 48,000 | 17,378 / 20,007 | **0 / 0** |
| fixed records (X..X) | 9,000 | 0 / 0 | **0 / 0** |

That is `v4_run.py 6000 20261103`: 354,000 frames in nine sets, the real `Fight::absorb` equal to the header mirror in 354,000 and to the real `Fight::hurt_side` in 354,000. Two earlier harnesses of ours on the same head, new seed (V; header functions `monster_blow`, `absorb_blows`; `mdmg2.py 20000 20261103` and `v3_run.py 10000 314159`): roll 160,000 frames base 159,654 differ, head 0; reduce 160,000 base 124,491, head 0; absorb 160,000 base 102,020 life / 68,173 pool, head 0 / 0; both 40,000 base 8,332 / 33,810, head 0 / 0; negative percent 10,000 base 9,999, head 0; second harness: roll 80,000 base 79,857, reduce 80,000 base 39,451, absorb 80,000 base 27,017 / 56,202, both 80,000 base 6,806 / 70,054, fixed 15,000 and fixed with reduction and pool 15,000 base 7,315 / 3,056, head 0 in all six. The head's mean error is 0.0000 hp in every one of these rows, the base's 0.12 to 1.16 hp for life and up to 2.09 hp for the pool.

**Where it is NOT equal: Energy Shield.** With Energy Shield up the head differs from game.exe and the text says so: shield alone (48,000 frames) base 32,669 life / 47,914 mana, head 9,881 / 23,323; shield with Bone Armor (armor first) base 9,670 life / 37,899 pool / 14,764 mana, head 1,359 / 0 / 7,167; fixed records with reduction, pool and shield, 9,000 frames, base 2,448 / 5,085 / 1,339, head 670 / 0 / 1,084. Every head difference is in a frame with Energy Shield up (V: 0 differing among the frames without it, in every set). The shield here runs after the armors on the reduced total, in landing order; game.exe takes it per record field on the unreduced value and runs the absorb state registered last first.

**"Never worse than base" holds with the armor-first order only.** In all nine sets, with the armor absorbing first, no quantity (life, pool and mana difference counts, and their mean errors) is worse on the head than on the base (V, `worse-than-base: none` in all nine). With Energy Shield absorbing first (game.exe's order when the shield was cast after Bone Armor) the shield-and-armor set is worse than base in two quantities: frames with a wrong life 8,561 against 7,838, and the pool's mean error 1.5670 against 1.5248 hp (about 3 percent); its life mean error still falls from 0.2066 to 0.0683 hp. The other shield-first set (fixed records with reduction, pool and shield) is not worse. The pair needs Bone Armor and Energy Shield up together, the shield cast last. We state this rather than claim a clean result.

Limits (all U unless stated): Energy Shield is whole points on the reduced total as before, now carried in 256ths; it cuts whole points only (`can >> 8`) off the blows, so the blows read after it are off by under a point; and `calc1` is clamped to 0..95 where `FUN_005ca6c0` has no cap (no effect at levels 1..30, calc1 at most 85 in the run). Whether d2d drops the state at 0 mana as game.exe does is not checked. Cyclone Armor's pool is in 256ths but its fields are whole points; no differential measures it. The sub-point life rule (`life < 256` is dead) and the pets' and merc's carry are covered by unit tests (`test_game`, a merc unit test), not by a differential; pets have no unit test. Fire, lightning and cold are already resisted in d2d's blows, game.exe resists after the absorb; `FUN_0057c1e0` scales the flat by the record's +0x54 / 1024 and runs `FUN_0057c060`'s percent, neither modelled. The clamp (min at least 1, max at least min + 1) changes no live melee row: in 1.14d's MonStats no enabled melee row of Acts 1 to 3 in any difficulty has min == max, min == 0 < max or min > max (S, a verifier's census). The missile path of `monster_blow` (`FUN_005a89a0`) shares the roll and takes its bounds as they stand (U). Consequence, reasoned and not measured in a soak: a Fallen 1..2 deals 1.00 to 1.99 instead of 1 or 2, the mean unchanged and the variance a third of before.

## The ponytail list (what the code comments name and leave)

- `states.cpp`, Energy Shield: the shield runs after the armors on the reduced total, in landing order; game.exe takes it per record field on the unreduced value and runs the state registered last first (`FUN_005c0ad0`). A per-record-field form matched game.exe for the physical field with the shield alone; this form does not, and with an armor up the order and fire / lightning / cold were never matched. It cuts whole points only (blows read after it are off by under a point) and clamps calc1 to 0..95.
- `combat.hpp`, `reduce_fine`: `FUN_0057c1e0` scales the flat by the record's +0x54 / 1024 when that is above 0 and the whole record by `FUN_0057c060`'s percent; neither is modelled.
- `combat.hpp`, `absorb_blows`: the physical reduction is put back after the absorb (raw); fire, lightning and cold are already resisted in d2d's blows, game.exe resists after.
- `combat.hpp`, Cyclone Armor: its fields are whole points here (no elemental fraction), so it takes whole points from the pool.
- `skills.hpp`, `mana_paid` (a comment): state 0x72 (`FUN_005d2b60`) and the srvdofunc 116 shortcut are not modelled; both unchanged from main.

## What is NOT in this PR

- Teleport: a monster at the target landing next to it (D3: units live in d2d's Crowd and the oracle injects raw unit bits only, so there is no honest differential); mana paid on a refused hop (D4, joins the pay-timing change); unit target, line mask and arrival bookkeeping (D5 to D7: read only).
- Mana: pets' and merc's tie order (shown at rule level only, never on a real build); payment timing and interrupted casts; Might and Summon Resist (no cast path in the harness).
- Damage roll: Energy Shield per record field and its order against the armors (above); an elemental fraction for Cyclone Armor; the flat's +0x54 scale and `FUN_0057c060`.
- A live game, a soak, a determinism hash comparison on a lab build (the roll draws the same number of times except for an empty range, which now draws once; recorded hashes change after the first monster blow, after a Teleport the room rule or the nearest spot refuses or moves, and after a cast the mana floor lets through).
- The harnesses (`sk_mana.py`, `sk_tele_spot.py` with `pb2_mana.sh` and `pb2_tele.sh`; `mdmg2.py`, `v3_run.py`, `v4_run.py` with their d2d-side programs) are not part of this PR (they live outside the repo tree and can be attached); only the three unit tests are.

## Credits

- Skill mana floor (commit 1): traced by Claude Code (the skill-mana trace of the ledger); written by the PR author; independently verified by Claude Code (card verifyb2, Opus, own model and probe on the real builds).
- Teleport room limit and nearest free spot (commits 2 and 3): traced by Claude Code; written by the PR author; independently verified by Claude Code (card verifyb2, own game.exe harness for the spot search; a separate room-rule check).
- Monster damage roll (commit 4): traced by Claude Code (the seed-distribution trace); written by the PR author; independently verified by Claude Code in four rounds (cards verifydmg, verifydmg2, verifydmg3 and verifydmg4, Opus, each with its own harness and seed; the fourth is the one that found the two text conditions applied in this head).

Only the git author and the Reported-by / Traced-by lines count for the gap map; verifiers are named here only.

## Checks

Head `824da884` on `005c7cab`. Worker host, Release build with `D2D_WERROR=ON` and the 1.14d game data (V unless said). Raw output: [pr-batch2-run.txt](pr-batch2-run.txt).

- Commits: 4; subject at most 72 characters (57, 65, 63, 68), body three lines of at most 72, last line `Agent: Claude Code`; author and committer the numeric noreply identity on all four, no co-author or AI line (V: `git log --format='%an <%ae> | %cn <%ce>'`; the `tools/presubmit.py --quick` attribution check is `ok`).
- Suites on the g++ build, rc 0 each: test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_sk_ground, test_sk_mana, test_tele_spot (V, `tools/parity.sh`; the clang++ build compiles and links the same suites). test_monster_missile and test_missile_blow_poison (in the earlier run) were not in this run's list.
- `tools/pr_guard.py`: 0 FAIL, 0 WARN on the range and on each of the 4 commits (V).
- `tools/presubmit.py --quick --build <g++ build dir>` (the form PR 180 added: GCC and clang-tidy on the changed files): 0 FAIL, 4 WARN (answered below). `gcc: 230 TUs, g++ -Wshadow -Wconversion -Wno-sign-conversion -Werror` ok; `clang-tidy: 230 TUs` ok (453 findings on lines this branch did not add, in `outdoor.hpp`, `merc.hpp`, `pets.hpp`: binding names `x` and `y`, present on main). Without a prior build the GCC and tidy part only warns.
- g++ and clang++ with the CI flags plus `-Wshadow -Wrange-loop-construct -Wconversion`, changed translation units, added lines: 0 warnings (V, parity `extra` stage). clang-tidy (parity `tidy` stage) rc 0.
- Every g++ target (parity `g-all`, informational): rc 1 on files this branch does not touch (`apps/d2d/ui.cpp`, `apps/d2d/town/walk.cpp`, `tests/test_protocol.cpp`: GCC 16 `-Werror` maybe-uninitialized and array-bounds); none of the files is in the diff (the base was not built for comparison, U).
- Counted differentials re-run on this head (V, `tools/emu/b2r3_diff.sh`): mana seed 7, 190,840 cases, 105,978 compared, base 281 differ, head 0; Teleport seed 1, 100,000 cases, equal 73,419 (base), 98,539 (after commit 2), 100,000 (head). Identical to the numbers on `6f036a40`. NOT re-run on this head: Teleport seed 9 (stopped when a long gate run started on the worker host) and the damage-roll sets (`v4_run.py`, `mdmg2.py`, `v3_run.py`). Their numbers in sections 2 and 4 are carried from the run on `6f036a40` (S): the patch is identical (`range-diff` `=`) and the files of commit 4 are not touched by PR 179 or 180 (V).
- NOT run for this head: the WSL lane and MSVC 14.51 on the build machine (its safety gate refuses weekday daytime); on `6f036a40` both passed (S: MSVC 0 `warning C`, ctest 73 of 73; WSL lane rc 0 in every stage). A live game, a determinism-hash comparison.
- Size against CodeRabbit's limits: see the budget table.

### CodeRabbit budget

Limits from `docs/research/coderabbit-limits-20261102.md` (the repo has no CodeRabbit configuration; the file cap is 100, the other numbers are the budget we set ourselves, U):

| quantity | one PR (bundle) | batch A (mana, Teleport) | batch B (damage roll) | budget (target / ceiling) |
|---|---|---|---|---|
| files | 17 | 10 | 9 | 40 / 70 (cap 100) |
| added + deleted lines | 797 (735 + 62) | 401 (389 + 12) | 396 (346 + 50) | 1,000 / about 1,500 |
| commits | 4 | 3 | 1 | 15 / 40 seen accepted |
| pushes to the PR branch | 1 | 1 | 1 | 1, then at most 1 an hour |

(V: `git diff --shortstat upstream/main...HEAD` on each branch.) No form is near a limit. A and B together list 19 files because `fight.hpp` and `tick.cpp` are in both.

## Merge

- The head is `005c7cab` plus 4 commits (V: `git rev-list --count upstream/main..HEAD`). If main moves again, `git rebase origin/main` (AGENTS.md section 6) and re-run the checks.
- One batch or two: [pr-upstream-rules-20261103.md](pr-upstream-rules-20261103.md). The split heads (`local/batch2-mana-teleport`, `local/batch2-monsterdmg-pr`) merge to this head's tree (V: `git merge-tree` of the two, empty diff against this head) and A equals this head's parent tree (V).

## Answers to the presubmit warnings

- siblings (player, merc, pet, monster; not a co-op partner): component 1 is for the player (both game functions return at once for a non-player); component 4 applies to every monster blow on a player, a pet (`pet_hurt` takes the fraction) and the merc (`life_frac`); a co-op partner is a player and takes the same `hurt_side` path (not exercised separately, U).
- hit paths (missile, skill, not melee): `monster_blow` serves the melee swing and the missile path alike (the missile's `FUN_005a89a0` bounds are taken as they stand, U); skills reach the same call.
- docs-sync: no `.md` changed; the notes of each rule are in the code comments, the ponytail list above names them.
- build: skipped by `--quick`; the build and the suites are listed under Checks.

## How to re-check

```sh
python3 tools/pr_guard.py
python3 tools/presubmit.py --quick        # GCC and clang-tidy on the changed files need a build first
cmake --build build && ctest --test-dir build --output-on-failure
```

Oracle (needs the maintainer's game.exe and `tools/emu`): mana `pb2_mana.sh N SEED BASE_BIN HEAD_BIN` (bins `test_sk_mana` of the base plus the test file and of the head); Teleport `pb2_tele.sh N SEED BASE_BIN D1_BIN HEAD_BIN` (`test_tele_spot`; the middle build is the head's second commit); damage roll `v4_run.py N SEED` after `v4_build` (the probe links the head's `libgame.a`), `mdmg2.py N SEED BIN...` and `v3_run.py N SEED BIN...` against the header programs built for the base and the head. The base for the first two is upstream main plus the head's test files.

Agent: Claude Code
