# Fourth verifier: monster damage roll r4 (DMGROLL, `local/batch2-monsterdmg-r4` 9ea2afe9)

> status: independent verifier report (job pa-verifydmg4, 2026-11-03, Opus, one agent, offline, final round). Subject: the private fork commit 9ea2afe9 (one commit on upstream main 3e073888), the round 4 block of `docs/research/dmgroll-draft.md` (branch agent/dmgroll, 7ec54875) and this verifier's earlier reports `verify-dmgroll.md`, `verify-dmgroll2.md`, `verify-dmgroll3.md`. Nothing pushed or posted. The fork was read through a private `git clone --no-checkout` in this job's scratch dir; the author's tree (`wt-mdmg-r4`) and build (`build-mdr4`) were only read, linked against and executed, never written.

Tags: **V read** = read by this verifier (in game.exe 1.14d, or in the r4 diff); **V run** = run by this verifier on the worker host; **S** = someone else's, not re-run; **U** = not known.

## Verdict: READY FOR PR: yes (no code to cut; two text conditions, see the end)

| Claim | Mark |
|---|---|
| (1) The roll in 256ths with the monster clamp, seed after | **VERIFIED** (roll 48,000 frames: 0 / 0 / 0 / 0 in life, pool, mana, seed) |
| (1) `reduce_fine`: flat, then percent, in 256ths | **VERIFIED** (reduce 48,000 and negpct 48,000: 0) |
| (1) `exact` (Corpse Explosion, Diablo's cold touch): no roll, no draw, reduction and absorbs apply | **VERIFIED** (exact 9,000: 0; exactred 9,000: 0 in every frame without Energy Shield, 6,753 of 6,753) |
| (1) Bone Armor pool in 256ths off the unreduced field, reduction put back (`absorb_blows`, `settle_blow`) | **VERIFIED** (absorb 48,000 and both 48,000: 0 in life and pool) |
| (1) The absorb guard `fine <= 0 && blows.empty()` | **VERIFIED** (both / negpct: pool 0 differing, r3 had 25,442 / 12,692) |
| (1) The real compiled `Fight::absorb` equals the mirror | **VERIFIED** (354,000 frames: real `Fight::absorb` = my header glue = real `Fight::hurt_side`, 0 differing) |
| (1) Cyclone Armor pool in 256ths | **NOT VERIFIED** by a differential (no elements in my sets; whole points in d2d, ponytail present) |
| (1) Sub-point life rule (`life < 256` is dead), player / merc | **HOLDS WITH LIMITS**: V read of the diff, `test_game` asserts pass (V run); the game side is the earlier rounds' read of 0057c865 / 0057c86c (S); not in my differential (my life is large) |
| (1) Pets' and merc's carry of the 256ths | **HOLDS WITH LIMITS**: V read (`pet_hurt(.., fraction)`, the merc's `life_frac`), merc unit test passes; pets have no unit test (U) |
| (2) Never worse than base, Energy Shield sets included | **HOLDS WITH LIMITS**: true in all nine sets with the armor-first order; with Energy Shield absorbing first (game.exe's order when Energy Shield was cast after Bone Armor) the esbone set is worse than base on life count and pool mean (section 2) |
| (3) The `// ponytail:` note in states.cpp | **HOLDS WITH LIMITS**: right on the order and the per-field rule; one sentence misleads and two gaps (section 3) |
| (4) Hygiene, one purpose, merge with PR 178 | **VERIFIED** (section 4) |
| (4) g++ / clang++ parity, the suites | **VERIFIED** (section 4) |
| (4) MSVC 14.51 | **NOT VERIFIED** here (the ops repo's Windows script is not readable in this job; S: the author's run, 0 `warning C`, 70 of 70) |

## 1. My harness against the REAL `Fight::absorb`

The author's w4 glue mirrors `Fight::absorb`. To close that gap, `tools/emu/v4_probe.cpp` is linked against the author's r4 `libgame.a` with `test_game.cpp`'s own compile flags (`-O3`, asserts on, `-Wall -Wextra -Wpedantic -Werror`), sets up a world as `test_game` does (a new Barbarian, `World::enter`, 25 ticks) and per case runs r4's `monster_blow`, `Foe::take`, then the real `Fight::absorb` with Bone Armor (skill 68) and Energy Shield (skill 58, found by aurastate) put in `self_states` at the case's level, and again the same case through the real `Fight::hurt_side`. The Energy Shield calc1 / calc2 come from the real `Fight::calc` at levels 1..30 (calc1 0..85, never at the 95 cap) and go to the game model and the other sides.

Before linking (V run, `v4_build.sh`): `diff -rq` of the author's tree against my `git archive` of 9ea2afe9 (components, tests, cmake): 0 lines; no file of components/game or components/rules is newer than `libgame.a`.

Game side: game.exe's FUN_0057b420 run natively under upstream's emulator (EDX = 1, stats 21 / 22 hooked), then my game-order model of FUN_0057c1e0 from the reads in `verify-dmgroll3.md` (Bone Armor, then Energy Shield on the unreduced physical, then flat, then percent; `esfirst` = the other order). Base side: my header glue of 3e073888 (as in round 3). New seed 161803, N = 6,000 per class and set.

Command: `tools/emu/run-on-worker.sh --ttl 3000 v4_run.py 6000 161803` (after `v4_git.sh` and `v4_build.sh`). Log: `docs/research/verify-dmgroll4-run.txt`. Frames that differ from game.exe in life / pool / mana / seed, and mean |error| in hp:

| set | frames | base | r4 (real `Fight::absorb`) | r4 mean life / pool / mana | base mean life / pool / mana |
|---|---|---|---|---|---|
| roll | 48,000 | 47,908 / 0 / 0 / 0 | **0 / 0 / 0 / 0** | 0 | 1.1724 / 0 / 0 |
| reduce | 48,000 | 23,606 / 0 / 0 / 0 | **0** | 0 | 0.6399 / 0 / 0 |
| absorb | 48,000 | 16,083 / 33,876 / 0 / 0 | **0** | 0 | 0.3891 / 0.8333 / 0 |
| both | 48,000 | 4,223 / 41,984 / 0 / 0 | **0** | 0 | 0.1228 / 2.0857 / 0 |
| negpct | 48,000 | 17,720 / 19,699 / 0 / 0 | **0** | 0 | 0.8195 / 0.8759 / 0 |
| exact | 9,000 | 0 | **0** | 0 | 0 |
| eshield | 48,000 | 32,729 / 0 / 47,940 / 0 | 9,923 / 0 / 23,266 / 0 | 0.0850 / 0 / 1.4435 | 0.6677 / 0 / 2.4546 |
| esbone | 48,000 | 9,754 / 37,804 / 14,896 / 0 | 1,358 / 0 / 7,251 / 0 | 0.0093 / 0 / 0.2759 | 0.1825 / 1.4445 / 0.5344 |
| exactred | 9,000 | 2,305 / 5,210 / 1,310 / 0 | 626 / 0 / 1,055 / 0 | 0.0946 / 0 / 0.9041 | 0.6015 / 5.6649 / 0.9685 |

Real `Fight::absorb` against my r4 glue: 0 of 354,000 frames differ; real `Fight::absorb` against real `Fight::hurt_side` (life lost, pool, mana): 0. Every r4 difference is in a frame with Energy Shield up (the runner counts r4 differences among the frames without it: 0 in every set). The author's eshield 10,011 / 23,715 used my round 3 random calc1 / calc2; mine use the skill's own values, so the counts differ in size but agree in kind (S for his, V run for mine).

## 2. Never worse than base

`worse-than-base` over the 7 quantities (four counts, three mean errors), armor-first game order: **none** in all nine sets (V run).

Energy Shield first (V run, the same frames against the other order; game.exe runs the absorb state registered last first, V read in round 3):

| set | base life / pool / mana | r4 life / pool / mana | base means | r4 means | worse |
|---|---|---|---|---|---|
| esbone | 7,968 / 43,960 / 47,990 | 8,620 / 43,007 / 47,457 | 0.2097 / 1.5220 / 3.7350 | 0.0679 / 1.5550 / 3.5970 | **life count, pool mean** |
| exactred | 2,297 / 5,314 / 2,185 | 632 / 1,170 / 1,930 | 0.6104 / 5.2801 / 2.7371 | 0.1056 / 0.8136 / 2.6787 | none |

Why: with Energy Shield first, game.exe's shield takes its part of the unreduced field before the pool; r4's pool takes the whole unreduced field and the shield runs after on the reduced rest. Base took both from the smaller reduced hit, which by chance lands nearer. The life mean error still drops threefold (0.2097 to 0.0679 hp a frame); the pool mean error rises 2 % (1.5220 to 1.5550 hp). It needs Bone Armor and Energy Shield up together (charges on a second class), and the shield cast after the armor. So "never worse than base on every quantity" is true for every single-state case and for both states in the armor-first order. It is not true for both states in the shield-first order.

## 3. The ponytail note (states.cpp, Energy Shield block)

V read: "this is the shield as it was before the roll change, carried in 256ths and run after the armors on the REDUCED total, in landing order. game.exe takes it per record field on the unreduced value and runs the absorb state registered LAST first (FUN_005c0ad0 ...). Its physical arithmetic was matched for one state; the order with an armor up and fire / lightning / cold were not, so the shield is not changed further."

- Right: the reduced total, after the armors, 256ths, landing order; game.exe per field, unreduced, last registered first.
- Misleading: "Its physical arithmetic was matched for one state" reads as if r4's shield matches game.exe on the physical field alone. It does not: eshield (the shield alone) differs in 9,923 life / 23,266 mana frames, because r4 runs it on the reduced total. The match was r3's per-field form, which r4 removed. Suggested: "r3's per-field form matched game.exe for the physical field with the shield alone; this form does not (it runs on the reduced total)".
- Missing (small, not new in r4): the shield cuts only whole points (`can >> 8`) off the blows and leaves each blow's `frac`, so the blows `struck_big` reads after are off by under a point; d2d clamps calc1 to 0..95 where FUN_005ca6c0 has no cap (V read in round 3; no effect at levels 1..30, calc1 at most 85). Whether d2d drops the state at 0 mana, as game.exe does at 005ca8b0, is not checked (U).

## 4. Hygiene, merge, CI, suites

V run (`v4_git.sh`): author and committer the same numeric GitHub noreply identity; parent 3e073888, one commit, 9 files +344 -50; subject 57 characters; body 3 lines of 72, 71, 68; last line `Agent: Claude Code`; no co-author line. One purpose (the roll and what it needs downstream: the reduction and the absorbs in 256ths, the carry, the life rule, the fixed records, their tests). `git merge-tree --write-tree 2597875 9ea2afe9`: rc 0, tree 564ec146 (equals the author's). `2597875` is PR 178's head as the fork clone knows it; `pull/178/head` is not on the fork remote, so whether it is still the PR's head today is S. The head is on the private remote (`local/batch2-monsterdmg-r4`).

V run (`v4_ci.py`, read-only on the author's tree and build): g++ and clang++ `-fsyntax-only` with the build's own compile command plus `-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror -Wshadow -Wrange-loop-construct -Wconversion` on ai.cpp, pets.cpp, states.cpp, tick.cpp, missiles.cpp, strike.cpp, test_game.cpp, test_combat.cpp: g++ 0 diagnostics; clang++ 133, **0 on a line the commit added** (first: cube.hpp:277, `-Wsign-conversion`, untouched).

Suites (the author's Release build, asserts on, run in my scratch): test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_skills (the 8), plus test_sk_ground, test_monster_missile, test_missile_blow_poison: **rc 0 each**. Current: the five that link `libgame.a` are newer than every source; the six header-only ones (test_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_skills) were built after combat.hpp and test_combat.cpp (the newest source, states.cpp, is not in them; `build.ninja` read).

Not run: MSVC (above), clang-tidy and the all-targets build (S: the author's Windows-worker WSL lane, rc 0), a live game, a soak.

## What the PR needs

No code cut. Two text conditions:

1. The PR text must not say "never worse than base" without the limit: with Bone Armor and Energy Shield both up and the shield cast last, the pool is slightly worse (section 2). It must not say "0 differences" for anything with Energy Shield up.
2. Reword the ponytail's "Its physical arithmetic was matched for one state" (section 3); optionally add the whole-point blow cut and the 95 cap.

If the owner wants a PR with no "worse than base" case at all, the cut is: keep base's Energy Shield block unchanged (whole points on the reduced total). By my reasoning that does not remove the shield-first esbone case, because the pool's own change causes it (U, not run). The only full cut is leaving Bone Armor's pool out of the PR, which loses absorb / both / negpct (0 differences now). I do not recommend it: the regression is one rare state pair, and its life error still falls threefold.

Reproduce: `tools/emu/run-on-worker.sh v4_git.sh`, `tools/emu/run-on-worker.sh --ttl 1500 v4_build.sh`, `tools/emu/run-on-worker.sh --ttl 3000 v4_run.py 6000 161803`, `tools/emu/run-on-worker.sh --ttl 3000 v4_ci.py`.
