# Third verifier: monster damage roll r3 (DMGROLL, `local/batch2-monsterdmg-r3` 2df7d1eb)

> status: independent verifier report (job pa-verifydmg3, 2026-11-03, Opus, one agent, offline). Subject: the private fork commit 2df7d1eb (one commit on upstream main 3e073888), the round 3 block of `docs/research/dmgroll-draft.md` (branch agent/dmgroll, 8c035b41) and this verifier's earlier reports `verify-dmgroll.md` and `verify-dmgroll2.md`. Nothing pushed or posted. The fork was read through a private `git clone --no-checkout` in this job's scratch dir; the author's tree (`wt-mdmg-r3`) and build (`build-mdr3`) were only read and executed, never written.

Tags: **V read** = read in game.exe 1.14d by this verifier (upstream's disasm.py through `tools/emu/w3_dis.py`, addresses given); **V run** = run by this verifier on the worker host; **S** = someone else's, not re-run; **U** = not known.

## Verdict: NOT READY FOR PR (one small gap in the new absorb path; one fixed order that game.exe does not have)

| Claim | Mark |
|---|---|
| (1) Fixed records reach FUN_0057c1e0 (the author's correction of my section 4) | **VERIFIED** (V read); my r2 reading was **WRONG** on that point |
| (1) `exact` only for Corpse Explosion and Diablo's cold touch | **VERIFIED** (diff read; exact 9,000 frames 0 differ) |
| (2) Absorb first, then flat, then percent (Bone Armor) | **HOLDS WITH LIMITS**: right in `absorb_blows`, but `Fight::absorb` skips every state when the REDUCED hit is 0 (section 2) |
| (2) Energy Shield arithmetic on the physical field | **VERIFIED** (V read FUN_005ca6c0; 48,000 frames, life 0 differ; mana differs only through the guard) |
| (2) Energy Shield's place among the absorb states | **WRONG as a fixed order**: game.exe runs the state registered last first (section 3); r3 always runs the armors first |
| (2) Energy Shield on fire / lightning / cold | **NOT VERIFIED** (one whole-point field in d2d, per field in 256ths in game.exe; no element in my sets) |
| (3) roll / reduce / absorb / exact: r3 0 differences | **VERIFIED** (48,000 / 48,000 / 48,000 / 9,000 frames, 0 in life, pool, mana and seed) |
| (3) both / negpct / exactred: r3 0 differences | **WRONG** as stated in the draft: life 0, but the pool differs in 25,442 / 12,692 / 1,239 frames, all from the guard |
| (3) Never worse than base on any compared quantity | **VERIFIED** (every set, every quantity, counts and mean errors) |
| (4) No unit test for the merc death test and Corpse Explosion in 256ths | merc: **should be added** (cheap, behaviour change); Corpse Explosion's 256ths: **acceptable** as a ponytail (the game side is U) |
| (5) Hygiene, one purpose, merge with PR 178 | **VERIFIED** (section 5) |
| (5) CI parity g++ / clang++, the suites | see section 5 |
| (5) MSVC 14.51 | **NOT VERIFIED** here (the ops repo's Windows script is not readable in this job; S: the author's run, 0 `warning C`, 70 of 70) |

## 1. The fixed records: the author is right

V read `0057c6c0` (`w3_dis.py 57c6c0:140`): the function takes the unit at `[ebp+8]`, a flag at `[ebp+0xc]` and the record at `[ebp+0x10]` (`ret 0xc`); at `0057c793 cmp dword [ebp+0xc], 0; je 0x57c7a5` and `0057c7a0 call 0x57c1e0`. Both callers push 1 as that flag: `005c4dc0 push eax (record); 005c4dc1 push 1; 005c4dc3 push ebx; 005c4dc4 call 0x57c6c0` (Corpse Explosion, FUN_005c4d10) and `005ccf3e push eax; 005ccf3f push 1; 005ccf41 push ebx; 005ccf44 call 0x57c6c0` (Diablo's cold touch, FUN_005cce80). Neither calls FUN_0057b420. So a fixed record gets the absorb event and the reduction, but no roll, no 256ths added, no draw. My r2 text ("game.exe skips FUN_0057c1e0") was wrong; the r2 regression finding (the clamp and the draw on these records) stands and is fixed in r3.

r3 (diff read): `monster_blow(.., exact)`: `raw = low * 256`, no clamp, no draw, then `reduce_fine` and the absorbs as for any blow; `exact = true` only at ai.cpp 2444 (Corpse Explosion) and 2481 (cold touch). V run: `exact` (X..X, X = 0..5, no reduction) 9,000 frames, r3 0 differ in life and seed (r2: 8,980 / 9,000).

Limits (U): FUN_0057c1e0 decides whether the flat applies from the record's flags (`0057c32e test [ebx], 0x100`, `0x200`, `0x400`, V read); the flags these two records carry are built by the callers' callers (FUN_005c4d10 copies a prepared 0x70-byte record, `005c4d36 rep movsd`) and are not read. The 256ths of Corpse Explosion's record (whole points from the corpse's life in d2d) stay U.

## 2. Absorb, then flat, then percent: right in the rules, skipped by a guard in `Fight::absorb`

V read (as in r2): event 0xb at `0057c2fc`, the reduction loop from `0057c378`. r3's `absorb_blows` takes Bone Armor's pool from `StruckBlow::raw` (unreduced, 256ths) and `settle_blow` puts `reduce_fine` back on the rest: right.

But `Fight::absorb` (states.cpp, r3) starts with `fine = (foe.damage << 8) + foe.frac; if (fine <= 0) return 0;` and `foe.damage` / `foe.frac` are already REDUCED (`monster_blow` returns the reduced hit, `Foe::take` sums it). A hit whose flat and percent take it to 0 therefore reaches no state: game.exe still takes it from Bone Armor's pool and from Energy Shield's mana (both before the reduction), r3 does not. `hurt_side` calls `absorb` for every foe with a landed hit (tick.cpp 604), so the guard is on the live path.

V run, my differential (section 4): every frame where r3 differs from the game model is such a frame (diagnosis line `reduced-alone hit 0`: 25,442 of 25,442 in "both", 12,692 of 12,692 in negpct, 12,861 in eshield, 12,750 in esbone, 1,334 in exactred), and a build of r3 with only that guard removed (`w3_d2d_r3ng`, diagnosis only) is 0 / 0 / 0 / 0 in every set. Life is never affected (the hit is 0 either way); the pool and the mana are (mean 1.35 hp of pool in "both"). Base has the same skip, so this is not worse than base; it is a gap in r3's own claim ("both 0 differences"), which the author's harness misses because `mdmg2_d2d.cpp` calls `absorb_blows` directly, not `Fight::absorb`.

How often in play: needs flat damage reduction at least the size of a hit (a 1..2 Fallen against 2 or more flat), U in real runs.

Fix (small, in scope): guard on the unreduced total (any blow with `raw > 0`, or the elemental part) instead of the reduced one, or drop the guard and let the states see a 0 hit (both functions skip a field at 0). Test: a 1.x hit with 2 flat through a 3 hp pool leaves life unchanged and the pool at 3 * 256 - raw.

## 3. Energy Shield

Arithmetic (V read FUN_005ca6c0, `005ca7c0..005ca837`): per field of table 0x6e30e0 (physical first): value > 0 -> `part = FUN_00483360(value, calc1, 100)`, at most `FUN_00483360(mana, 16, calc2)` (`005ca7f8 cmp; jl`); `value -= part`; `mana -= FUN_00483360(part, calc2, 16)`, then `mana = max(mana, 0)` (`005ca819..005ca823`); stat 8 written once at the end (`005ca84e`); calc1 <= 0 returns at once; calc2 at least 1; no 95 % cap in this function (d2d's clamp 0..95 is pre-existing). At mana 0 the state is removed (`005ca8b0..005ca8f0`). r3's `shield_blows` on the physical field is the same in 256ths: V run "eshield" 48,000 frames, life 0 differ, mana differs only in the 12,861 guarded frames of section 2 (0 with the guard removed). Better than base (47,967 mana differences) and r2 (23,715).

Order among the absorb states (V read): FUN_005c0ad0 (the event registration, named by upstream's skills notes, S) allocates a node and puts it at the HEAD of the unit's list (`005c0b1e..005c0b40`: `node+0x1c = [unit+0x90]`, `[unit+0x90] = node`), and the dispatcher FUN_005c0c30 walks from `[unit+0x90]` along `+0x1c` (`005c0c4b`, `005c0c96`). So the state registered LAST absorbs FIRST. r3 always runs Bone / Cyclone Armor first and Energy Shield after. In my "esbone" set (both states, one hit) the other order changes the result in 47,569 of 48,000 frames, so the order matters whenever both are up; it needs two classes' states on one character (charges), U how often. Recommendation: keep the fixed order but correct the ponytail to "game.exe: the state cast last absorbs first (FUN_005c0ad0 pushes at the head); d2d: armors first", or order by cast time (`SelfState` order) if cheap.

Elements: game.exe runs the shield per field (fire, lightning, cold, and the conditioned +0x38 / +0x3c / +0x40) in 256ths on unresisted values; r3 uses one whole-point elemental field per blow, already resisted (ponytail present). NOT VERIFIED by a differential (my sets have no elements).

## 4. My differential (independent of the author's mdmg2.py and v3_run.py)

`tools/emu/w3_run.py N SEED BIN...` (V run): game.exe's FUN_0057b420 natively (EDX = 1, stats 21 / 22 hooked), then my game-order model of FUN_0057c1e0 from the reads above: Bone Armor, then Energy Shield (r3's order; the other order counted as `esfirst-differs`), then flat, then percent. The d2d side `tools/emu/w3_d2d.cpp` is compiled by `w3_build.sh` from the headers of my own `git archive` exports of 3e073888, b9ecb415 and 2df7d1eb (`w3_git.sh`) and runs each tree's `monster_blow`, `absorb_blows` (and r3's `shield_blows`) with the glue of each tree's `Fight::absorb`, guard included; `w3_d2d_r3ng` is r3 without the guard (diagnosis). Seed 314159 (new; r2's was 271828), N = 6,000 per class and set; exact = X..X for X = 0..5, exactred = X..X with X 0..40, a reduction, a pool (75 %) and Energy Shield (25 %).

Command: `tools/emu/run-on-worker.sh --ttl 3000 w3_run.py 6000 314159 BASE R2 R3NG R3` (binaries in the job's scratch). Full log: `docs/research/verify-dmgroll3-run.txt`. Frames that differ from game.exe in life / pool / mana / seed:

| set | frames | base | r2 | r3 | r3 without the guard |
|---|---|---|---|---|---|
| roll | 48,000 | 47,915 / 0 / 0 / 0 | 0 / 0 / 0 / 0 | **0 / 0 / 0 / 0** | 0 |
| reduce | 48,000 | 23,665 / 0 / 0 / 0 | 0 | **0** | 0 |
| absorb | 48,000 | 16,290 / 33,728 / 0 / 0 | 0 | **0** | 0 |
| both | 48,000 | 4,158 / 41,936 / 0 / 0 | 2,277 / 44,655 (pool worse than base) | **0 / 25,442 / 0 / 0** | 0 |
| negpct (-100..-1) | 48,000 | 17,469 / 19,853 | 3,093 / 21,426 (pool worse) | **0 / 12,692** | 0 |
| eshield | 48,000 | 33,235 / 0 / 47,967 | 10,011 / 0 / 23,715 | **0 / 0 / 12,861** | 0 |
| esbone | 48,000 | 9,854 / 37,861 / 14,764 | 1,761 / 22,338 / 7,291 | **0 / 12,750 / 3,666** | 0 |
| exact | 9,000 | 0 | 8,980 / 0 / 0 / 9,000 (worse) | **0** | 0 |
| exactred | 9,000 | 2,377 / 5,100 / 1,331 / 0 | 3,475 / 5,228 / 1,234 / 9,000 (worse) | **0 / 1,239 / 235 / 0** | 0 |

The runner prints `worse-than-base` per set on all 7 quantities (four counts, three mean errors): r3 `none` in all nine sets; r2 worse in both, negpct, exact and exactred. The base and r2 columns agree in direction with my r2 report and with the author's tables (S).

## 5. Hygiene, merge, CI, suites

V run (`w3_git.sh` on the private clone): author and committer the same numeric GitHub noreply identity; parent 3e073888, one commit, 9 files +333 -50; subject 57 characters; body 3 lines of 72, 71, 68; last line `Agent: Claude Code`; no co-author line. One purpose: yes (the roll and everything it needs downstream: the reduction on 256ths, the absorbs and Energy Shield on 256ths, the pets' and merc's carry, the life rule, the fixed-record exception, their tests). `git merge-tree --write-tree 2597875 2df7d1eb` (PR 178's head): rc 0, tree ae0afaff (equals the author's). The head is on the private remote (`local/batch2-monsterdmg-r3`).

V run (`tools/emu/run-on-worker.sh --ttl 3000 w3_ci.py`, read-only on the author's tree and build, no own build):
- The author's tree equals my `git archive` of 2df7d1eb (`diff -rq` of components/ and tests/: 0 lines).
- g++ and clang++ `-fsyntax-only` with the build's own compile command plus `-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror -Wshadow -Wrange-loop-construct -Wconversion` on ai.cpp, missiles.cpp, pets.cpp, states.cpp, tick.cpp, strike.cpp, test_game.cpp, test_combat.cpp: g++ 0 diagnostics; clang++ 133, all in untouched headers (first: cube.hpp:277, `-Wsign-conversion` from the build's own flags), **0 on a line the commit added**. VERIFIED.
- Suites of the author's Release build (asserts on), run by me in my scratch dir: test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_skills (the 8), plus test_sk_ground, test_monster_missile, test_missile_blow_poison: **rc 0 each**. HOLDS WITH LIMITS: built by the author; test_combat, test_game, test_game_rules, test_skills and the three extras are newer than the newest components/rules source, test_rules / test_monsters / test_monster_thinks / test_monster_ai are 736 s older (not rebuilt after the last rules edit, which the author describes as comments and a test name; ninja rebuilds what depends on a changed header, so either they do not include it or the last build skipped them: U).
- clang-tidy, the all-targets build and MSVC 14.51: not re-run (S: the author's Windows-worker runs, all rc 0, MSVC 0 `warning C`, 70 of 70).

## What must change before a PR

1. `Fight::absorb`: do not skip the states when the REDUCED hit is 0 (section 2); add the test. Then "both" / negpct / eshield / esbone / exactred are 0 in my harness (shown by the diagnosis build).
2. The Energy Shield order ponytail: replace "not read (U)" with the read (last cast first) and keep or change the fixed order; say it in the PR text.
3. A `test_game` block for the merc's death under one point (recommended; small).
4. Then re-run: `tools/emu/run-on-worker.sh w3_git.sh` (edit H), `w3_build.sh`, `run-on-worker.sh --ttl 3000 w3_run.py 6000 314159 BASE R2 R3NG R3`; expect r3 0 everywhere.

Leave out of the PR text: "both 0 differences" (not true through `Fight::absorb`), "game.exe's order among the absorb states is not read" (it is, section 3).

Ready for PR: **no** (item 1; small).
