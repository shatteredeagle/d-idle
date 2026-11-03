# Second verifier: monster damage roll r2 (DMGROLL, `local/batch2-monsterdmg-r2` b9ecb415)

> status: independent verifier report (job pa-verifydmg2, 2026-11-02, Opus, one agent, offline). Subject: the private fork commit b9ecb415 (one commit on upstream main 3e073888), the draft `docs/research/dmgroll-draft.md` and `docs/research/dmgroll-r2-differential.txt` (branch agent/dmgroll, f34d02f1), and the first verifier's report `docs/research/verify-dmgroll.md` (branch agent/verifydmg, 99fd9a6d). Nothing pushed or posted. The fork was read through a private `git clone --no-checkout` in this job's scratch dir; the author's tree (`wt-mdmg-r2`) and build (`build-mdr2`) were only read and executed, never written.

Tags: **V read** = read in game.exe 1.14d by this verifier (upstream's disasm.py, addresses given); **V run** = run by this verifier on the worker host; **S** = the author's, not re-run; **U** = not known.

## Verdict: NOT READY FOR PR (one new regression; one order gap that leaves one quantity worse than base)

| Claim | Mark |
|---|---|
| Finding 1 fixed: flat, then percent (FUN_0057bf80) | **VERIFIED** (80,000 reduce frames, 0 differ) |
| Finding 2 fixed: the absorb remainder keeps its 256ths (FUN_005c6370) | **VERIFIED** (80,000 absorb frames, 0 differ in life and pool) |
| The roll on the 8 Act 1 classes | **VERIFIED** (80,000 roll frames, 0 differ in value and seed after) |
| Energy Shield takes its share of the fine total | **HOLDS WITH LIMITS** (game: per field, section 3) |
| Life under one whole point is 0 (FUN_0057c6c0) | **VERIFIED** for the player (V read); merc **HOLDS WITH LIMITS** |
| The clamp applies to every non-missile `monster_blow` | **WRONG** for Corpse Explosion and Diablo's cold touch (section 4) |
| Reduce before absorb (the open item) | in scope; leaves the pool-left error **worse than base** in the "both" set (section 5) |
| Missile exception | **HOLDS WITH LIMITS** (section 6) |
| Pet and merc carry | **HOLDS WITH LIMITS** (section 6) |
| Commit hygiene, merge with PR 178 | **VERIFIED** (section 7) |
| CI parity (g++, clang++ on added lines) | **VERIFIED** (0 on added lines) |
| The 8 suites (plus 3) | **HOLDS WITH LIMITS** (rc 0, on the author's build of a verified-identical tree) |
| MSVC | **NOT VERIFIED** here (S: the author's Windows run) |

## 1. My harness (independent of the author's mdmg2.py)

`tools/emu/v2_run.py N SEED BIN...` (V run): game.exe's FUN_0057b420 run natively under upstream's emulator (EDX = 1, no weapon, stats 21 / 22 hooked; the fine value and the seed after), then this verifier's game-order model of FUN_0057c1e0 from the read: the absorb event first (0x57c2fc: Bone Armor FUN_005c6370, pool > phys -> pool -= phys, phys = 0, else phys -= pool, pool = 0), then the reduction loop (0x57c378, FUN_0057bf80: value <= 0 -> 0; value -= flat << 8; value > 0 and res != 0 -> value x (100 - min(res, 100)) / 100, truncated toward zero), the life takes the result when it is above 0 (0057c852). The d2d side `tools/emu/v2_d2d.cpp` is compiled by `v2_build.sh` from the headers of this verifier's own export of 3e073888 and b9ecb415 (`v2_trees.sh`, `git archive` from the clone) and calls each tree's `monster_blow` (sure hit, standing) and `absorb_blows`, with the glue of each tree's `Fight::absorb` Bone Armor branch (one blow a frame). Cases from Python's generator seeded 271828: per class 10,000 frames for each of roll, reduce (percent -50..50 or 0, flat 0..max + 2), absorb (pool 1..(2 max + 2) x 256, half with a fraction), both; plus "exact" (fixed records X..X, X = 0..5, 2,500 each).

Command: `tools/emu/run-on-worker.sh --ttl 2700 v2_run.py 10000 271828 $D2_SCRATCH/v2_d2d_base $D2_SCRATCH/v2_d2d_r2`. Full log: `docs/research/verify-dmgroll2-run.txt`.

| set | frames | base: life / pool / seed differ, mean abs error (hp) life, pool | r2: same |
|---|---|---|---|
| roll | 80,000 | 79,829 / 0 / 0, 1.1619, 0 | **0 / 0 / 0**, 0, 0 |
| reduce | 80,000 | 39,496 / 0 / 0, 0.6482, 0 | **0 / 0 / 0**, 0, 0 |
| absorb | 80,000 | 26,829 / 56,460 / 0, 0.3910, 0.8307 | **0 / 0 / 0**, 0, 0 |
| both | 80,000 | 6,815 / 70,096 / 0, 0.1124, 2.0852 | 3,731 / **74,599** / 0, 0.0183, **2.1527** |
| exact | 15,000 | 0 / 0 / 0, 0, 0 | **14,952** / 0 / **15,000**, **0.6654**, 0 |

Every class row is 0 / 0 / 0 for r2 in roll, reduce and absorb (per-class lines in the log). A d2d-order model (reduce, then absorb, on game.exe's own roll) equals r2 in every frame of every set (0 / 0): the "both" gap is the order and nothing else.

Afterwards compared with the author's log (S): same direction everywhere. Author's "both" (1 to 3 blows a frame): r2 life 8,310 vs base 8,425, pool 36,117 vs base 33,784 of 40,000; mine (one blow): life 3,731 vs 6,815, pool 74,599 vs 70,096 of 80,000. Both logs show the pool-left count worse in r2 than in base.

## 2. Findings 1 and 2: VERIFIED

`rules::reduce_fine` (V read of the diff): `fine <= 0 -> 0; fine -= flat x 256; fine > 0 and percent != 0 -> fine x (100 - clamp(percent, -100, 100)) / 100; max(fine, 0)`. Matches FUN_0057bf80 (V read 0057bf80..0057bff4: a field <= 0 is stored 0; with the call's flag 0 the flat is subtracted; res >= 100 is set to 100, a negative res has no floor in game.exe, d2d floors it at -100: equal for any curse in 1.14d data, Amplify Damage being -100; U for gear below -100). 80,000 frames 0 differ.

`absorb_blows` (V read): the physical pool in 256ths, `cut = min(pool, physical x 256 + frac, damage x 256 + frac)`, the rest keeps its fraction; Bone Armor's pool keeps its fraction from `start_state` (`absorb_frac`). 80,000 frames 0 differ in life and pool left.

## 3. Energy Shield (FUN_005ca6c0): HOLDS WITH LIMITS

V read 005ca6c0..005ca90b: calc1 (skill +0x138) must be > 0; mana = stat 8 (256ths); calc2 (+0x13c) at least 1. For the eight record fields of table 0x6e30e0 (V dump: +8 physical, +0x10, +0x1c, +0x24, +0x20 unconditioned; +0x38, +0x3c, +0x40 only when the record is not a type-1 record with FUN_0063ee90 = 0): value > 0 -> part = value x calc1 / 100, at most mana x 16 / calc2; value -= part; mana -= part x calc2 / 16, at least 0. All in 256ths, field by field, mana set once at the end.

r2 does the same on the summed fine hit (`part = fine x clamp(calc1, 0, 95) / 100`, cap `mana x 16 / per16`, mana `-= can x per16 / 16`). Right in units (256ths, mana in 256ths, the remainder keeps its fraction): an improvement over base (whole points). Limits: per-field truncation in game.exe vs one truncation on the sum (a few 256ths a hit), the 95 cap is d2d's (not in this function; pre-existing), and Energy Shield sits on the same event as Bone Armor before the reduction (0x57c2fc; the event entry for Energy Shield was not read, U), so it shares the order gap of section 5. Not measured by a differential.

## 4. Life under one point (FUN_0057c6c0): VERIFIED; the clamp on fixed records: WRONG

V read 0057c852..0057c874: when the record total +0x4c is above 0, life = stat 6 - total; a result under 0x100 (signed) is stored as 0. r2's `hurt_side` does `life -= fine; if (life < 0x100) life = 0` only when something landed: same rule. Exercised by the new test_game block (S, see section 7 for the run).

New regression (V read, V run): r2 clamps every non-missile `monster_blow` call (`low >= 1, high >= low + 1`). Two callers in `components/game/ai.cpp` pass fixed records, not a FUN_0057b420 roll:
- Corpse Explosion (line 2439, `a1_min = a1_max = dmg / 2`, models FUN_005c4d10);
- Diablo's cold touch (line 2477, `a1_min = a1_max = 0`, models FUN_005cce80).
Both game functions hand their record straight to FUN_0057c6c0 (V read: 005c4dc4 and 005ccf44 call 0x57c6c0; neither calls 0x57b420 nor 0x57c1e0). So game.exe adds no 256ths, no draw. Base matched that (rng.range(x, x) = x, no draw). r2: X..X becomes X + 0..255/256 and 0..0 becomes 1..2 hp of physical, with one extra draw: 14,952 of 15,000 exact frames differ in life (mean +0.67 hp; +1.50 hp for 0..0), 15,000 of 15,000 in the seed after. Worse than base on every frame. Fix: skip the clamp and the fine roll for these two callers (an `exact` flag, or roll `low x 256` when low == high and the caller is not FUN_0057b420's), and a test for each. (Both callers also apply `reduce_fine` where game.exe skips FUN_0057c1e0; that is pre-existing in base, not this PR's.)

## 5. The open item: reduce before absorb

game.exe (V read FUN_0057c1e0): event 0xb (absorbs) at 0x57c2fc, then the reduction loop at 0x57c378. d2d reduces in `monster_blow`, absorbs after in `Fight::absorb`. My run: the "both" gap is exactly this order (0 frames differ against a d2d-order model). In r2, life loss is better than base (3,731 vs 6,815 frames differ; 0.018 vs 0.112 hp mean error) but the pool left is worse (74,599 vs 70,096 frames; 2.153 vs 2.085 hp mean error, +3 %): base's pool loses its fraction at `start_state`, which by chance offsets part of the order error; r2 keeps the fraction and the offset goes.

In scope: yes. The PR already rewrites both ends (`monster_blow`'s reduction and `Fight::absorb`'s pool in 256ths), and with the r2 pieces the fix is small: `monster_blow` returns the unreduced fine physical (plus the reduction to apply), `Fight::absorb` runs Bone Armor and Energy Shield on it, then `reduce_fine`; callers without a pool (pets, merc) reduce at once. Expected result: "both" 0 differ (the order is the only gap, above). The alternative, to leave it out, means the PR is not "never worse": the pool left of a damage-reduced Bone Armor user drifts 3 % more than on main. Recommendation: fix it in this PR.

## 6. Missile exception and pet / merc carry

Missile (V read of the diff, line 894 `monster_blow(..., missile.src, true, rng, true)`): no clamp, but the missile now rolls `low x 256 + rng((high - low) x 256)` on the shared rng. game.exe FUN_005a89a0 (first verifier, V read): stat 22 <= 0 or stat 21 <= 0 -> physical 0 without a draw; min > max swapped; min + rand(max - min) on the MISSILE's seed; whether its stats are 256ths is U. On real data the missile rows are 0..0 (first verifier's census), where r2 gives 0 without a draw (rng(0) does not draw, V read `Rng::operator()`), as base did. HOLDS WITH LIMITS: same as base for live rows; the seed and the 256ths of a non-zero missile row stay U and are documented in the code.

Pets (V read): `pet_hurt(.., blow.frac)` -> the existing `hurt(.., fraction)`, whose `life_frac` is "already taken" (life = hit_points x 256 - life_frac, ai.hpp:178) and whose death test is 0057c86c's. Right. Merc (V read tick.cpp 510..516): `life_frac += frac; damage += life_frac >> 8; life_frac &= 255`, the same sign as `hurt`; but the death test stays `hit_points <= 0`, so a merc at 1 point with `life_frac > 0` (life under one point) lives where game.exe sets it to 0. That state is new with r2 (base's monster hits on the merc were whole). HOLDS WITH LIMITS; one-line fix: use `hurt`'s test (`hit_points < 1 || (hit_points == 1 && life_frac > 0)`).

## 7. Hygiene, merge, CI, suites

V run (`tools/emu/run-on-worker.sh v2_git.sh`, on the private clone):
- Author and committer: the same numeric GitHub noreply identity (the numeric-id form) for both. Parent 3e073888, one commit, 8 files +202 -45. Subject 70 characters; body 3 lines of 72, 72, 72; last line `Agent: Claude Code`; no co-author line. VERIFIED.
- `git merge-tree --write-tree 2597875 b9ecb415` (PR 178's head, present in the fork's objects): rc 0, tree dbbee164, no conflict. VERIFIED.
- The branch is on the fork's private remote (its remote-tracking `local/batch2-monsterdmg-r2` contains b9ecb415). VERIFIED.

V run (`tools/emu/run-on-worker.sh --ttl 3000 v2_ci.py`, read-only on the author's tree and build; no own build, see the limit below):
- The author's tree equals this verifier's `git archive` of b9ecb415 (`diff -rq` of components/ and tests/: 0 lines), and every suite binary is newer than the newest source in components/rules.
- g++ and clang++ `-fsyntax-only` with the build's own compile command plus `-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror -Wshadow -Wrange-loop-construct -Wconversion`, on pets.cpp, states.cpp, tick.cpp, strike.cpp, test_game.cpp, test_combat.cpp: g++ 0 diagnostics; clang++ 95 diagnostics, all in untouched headers (the first is cube.hpp:277, `-Wsign-conversion` from the build's flags), **0 on a line the commit added**. VERIFIED (added lines taken from the commit's diff).
- Suites (the author's Release build `build-mdr2`, run by this verifier in its scratch dir): test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_skills (the 8), plus test_sk_ground, test_monster_missile, test_missile_blow_poison: **rc 0 each**. The tests keep their asserts in this build (the author's first test_combat log shows an assertion firing). HOLDS WITH LIMITS: the binaries were built by the author, not by this verifier; the identity of the sources is checked, the build itself is not redone. Note: the author's own `build-mdr2.run-test_combat.log` on the worker host still shows the pre-fix assertion failure; the binary of 22:01 passes.

MSVC: not re-run (the Windows lane is the ops repo's; S: the author's log, 0 `warning C`, the base's C2026 in `test_monster_ai` only).

## What must change before a PR

1. Do not clamp (and do not fine-roll) the fixed-record callers: Corpse Explosion (ai.cpp 2439) and Diablo's cold touch (ai.cpp 2477); add a test (0..0 gives 0 and no draw; X..X gives X x 256). Required: today it is worse than base on every such hit.
2. Absorb before the reduction (section 5), so the "both" set reaches 0 and no compared quantity is worse than base. If the author leaves it out instead, the PR text must say that the pool left gets 3 % worse for a damage-reduced Bone Armor user.
3. Merc: `hurt`'s death test (section 6). Small; recommended.
4. Re-run the "both" and "exact" sets (`v2_run.py 10000 271828`) on the new head; expect 0 / 0 / 0.

Ready for PR: **no** (items 1 and 2).

## Ready-to-run checklist for a job with the next head

1. `tools/emu/run-on-worker.sh v2_git.sh` (edit H), `v2_trees.sh`, `v2_build.sh`, then `tools/emu/run-on-worker.sh --ttl 2700 v2_run.py 10000 271828 BASE_BIN NEW_BIN`.
2. `tools/emu/run-on-worker.sh --ttl 3000 v2_ci.py` against the new build dir (edit TREE / BUILD).
3. MSVC through the ops repo's Windows parity script (its safety gate decides).
