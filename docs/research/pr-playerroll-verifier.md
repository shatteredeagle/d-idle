# Independent verifier: pr/player-roll (element roll top, physical roll in 256ths, -100 physical floor) (VERIFYPLAYERROLL, 2026-11-02)

> status: verifier record for the fork candidate `pr/player-roll` (private fork remote, head 28c0540a, four commits on upstream main 6935cf64) and its lab twin `local/playerroll-lab` 51e4eb1e. Author's draft: `docs/research/pr-playerroll-draft.md` on branch agent/playerroll, read only after my own counts. Offline: no game.exe, nothing pushed or posted, the author's harness files (`skab_phys*`, `sk_rest_hit.py`, `nec_missile_hit.py`, `tools/emu/missile_hit.py`) not used. Verdict in section 9.

Tags: **V run** (my harness or builds on the worker host), **V read** (my disassembly with the fork's `tools/emu/disasm.py`, or the source), **S** source read without a run, **I** inferred, **U** not traced.

Harness (mine; copies in `docs/research/pr-playerroll-verifier/`): `physroll_game.py` (from my physroll verifier, plus a `grid` mode), `elemroll_game.py` (new: game.exe's FUN_005a8910 natively), `test_rollprobe.cpp` (dropped into `tests/game/` of my scratch worktrees only: the REAL `missile_blow`, `missile_damage` and `hurt()`), `werror.sh`, `pipeline.sh`. Scratch worktrees on the worker host: base 6935cf64, head 28c0540a, labhead 51e4eb1e; every build mine, from clean, `-j2`, nice.

## 1. Diff and merge (V run)

- `git diff --stat 6935cf64 28c0540a`: `components/rules/combat.hpp` (+13 -4, the three hunks of `missile_blow`), `docs/research/re/missiles.md` (+9), `tests/test_combat.cpp` (+21), `tests/game/test_missile_hit.cpp` (new, 89), `tools/emu/missile_hit.py` (new, 155), `tools/emu/README.md` (+1). Four commits: 0e634c55 oracle, bb168658 element roll top, b543539d physical in 256ths, 28c0540a floor.
- Upstream main re-fetched read-only (`git fetch upstream main`): still 6935cf64, so the branch is a fast-forward of current upstream main; merge trivially clean.
- Lab twin: `git diff ee5e06d8 51e4eb1e` equals `git diff 6935cf64 28c0540a` except index lines and hunk offsets (same content).
- Authorship (low): 0e634c55 / bb168658 are authored by the maintainer's GitHub identity, b543539d / 28c0540a by `Claude <noreply@anthropic.com>`; all four committed by Claude. Make the PR's authors uniform before it is opened (whatever earlier upstream PRs used).

## 2. game.exe (my read)

- **FUN_005a8910** (V read 0x5a8910..0x5a8999): EAX the max stat, stack (unit, min stat, mastery stat or -1). max <= 0 -> 0; min <= 0 -> 0; min > max swaps; with a mastery stat its value p (if not 0) adds MulDiv(min, p, 100) and MulDiv(max, p, 100); returns `min + FUN_0045c3e0(seed at unit+0x20, max - min)`.
- **FUN_0045c3e0** (V read): n <= 0 returns 0 **without touching the seed**; else one LCG step (x 0x6AC690C5 + high), power-of-two n masks, else mod n. So the max is exclusive and `lo == hi` draws nothing. d2d's `Rng::operator()(n)` is the same (S, rules.hpp:167).
- **FUN_005a89a0** (V read): calls FUN_005a8910 for fire (0x31/0x30, mastery 0x149) -> +0x10, magic (0x35/0x34, 0x165) -> +0x20, lightning (0x33/0x32, 0x14a) -> +0x1c, cold (0x37/0x36, 0x14b), poison (0x3a/0x39, 0x14c), and a last pair (0x13d/0x13c, 0x149). The mastery is applied a second time here if the MISSILE unit carries the mastery stat (d2d applies it once in `elem_damage`); U whether a skill missile ever carries it, not this PR.
- Physical roll, floor at -100 for monster defenders (FUN_0057be00), `resisted_fine` = FUN_0057bf80 + MulDiv: as in my earlier record `pr-physroll-verifier.md` sections 2 (unchanged reading).

## 3. Own counts (V run, new seeds)

### 3a. Physical (real `missile_blow` + real `hurt()` vs game.exe FUN_0057c6c0 once per hit; life in 256ths after every hit)

| set | cases | upstream main 6935cf64 differ (any hit) | head 28c0540a differ |
|---|---|---|---|
| seed 4242 (edge grid, death grid at 1..3 points, random res -120..120, 1..60 hits) | 40,000 | 28,707 | **0** |
| seed 31337 (same shape) | 40,000 | 28,873 | **0** |
| floor grid seed 515: every res from -101 to -300 (200 values) x 14 edge rolls (1, 127, 128, 255, 256, 257, 511, 1000, 25600, 0xFFFFF, 0x100000, 0x100001, 0x3FFFFF, 0x7FFFFFFF), then random res -300..-101, 1..20 hits | 30,000 | 23,884 | **0** |

Head: **0 of 110,000**, every class: immune, res 0, 0 < res < 100, -100 <= res < 0, res < -100 (1,993 + 2,005 random cases plus the whole grid), rolls <= 0, rolls over 2^20 (MulDiv's other branch) and 0x7FFFFFFF (the wrap). Base differs in every non-immune class (e.g. res -150, roll 1: game.exe 25,599,998, base 25,600,000: the fraction dropped). The defender is a monster (Fallen) on both sides; every `missile_blow` caller has a monster target (section 4), so the floor is applied only where game.exe applies it.

Floor attribution: I did not build b543539d alone this time; my earlier record ran its content (5b13a2c4 = b543539d's hunk, checked by diff) and found every remaining difference at res < -100 (639 of 30,000, 1,456 of 60,000). Together with 0 at head on 30,000 grid cases, the floor commit is what closes res < -100 (V by combination).

### 3b. Element roll (my oracle: FUN_005a8910 natively, stat getter hooked, mastery 0, the case's seed at unit+0x20; d2d: real `missile_blow`, fire at res 0, Rng set to the same seed: roll = damage x 256 + frac, plus the seed after)

Inputs: the (elo, ehi) of the REAL `missile_damage` for every Sorceress and Necromancer skill with an element (25 Sorceress, 8 Necromancer: `--pairs sor nec`, levels 1..40, synergy levels 0 / lvl / 20, masteries 0 / 37 / 140), sampled with seed 8128, plus 23 edge pairs x 40 seeds.

| class | cases | base: roll differs | base: seed after differs | head: roll | head: seed |
|---|---|---|---|---|---|
| Sorceress, lo < hi | 29,540 | 29,516 | 0 | **0** | **0** |
| Necromancer, lo < hi | 29,540 | 29,491 | 0 | **0** | **0** |
| edges | 920 | 541 | 240 | **0** | **0** |

Edges per pair (base differs / 40; head 0 on all): lo == hi (256,256), (1,1), (3000,3000): **0 on base too**; (0,0): 0; (256,0), (256,-1): 0 (early return both); (0,256), (0,1), (-1,256), (-256,512): 40 (base rolls and draws, game.exe gives 0 and does not draw); swapped (512,256), (257,256): 40 (base returns lo, game.exe swaps and draws); spans 1, 2^n, 2^20, 2^20+1, near 2^31: 15..40.

What the counts mean (V run unless marked):
- At the roll level the old roll differs on nearly every seed (x mod (n+1) vs x mod n), not only at the top 256th. Because d2d draws from the shared fight Rng, not the missile's seed, the in-game effect is distributional: the max 256th is no longer rolled, a mean shift of 1/2 of a 256th (1/512 point) per element hit (I, arithmetic). Fidelity fix, gameplay-invisible.
- **The RNG draw count does not change for any skill-sourced element**: over all classes (`--pairs ama ass bar dru pal sor nec` and the class-less rows, 35,280 lines) no skill gives lo <= 0 < hi, and `missile_damage` clamps ehi >= elo; lo == hi (Poison Javelin at 45 level/synergy combinations) draws on neither side. The stream changes only for a `MissileDamage` built elsewhere with lo <= 0 < hi or lo > hi (`row_damage` rows without a Skill, `pet_aura`, shrine potions: U whether any does).

## 4. Caller audit (S, call sites read at 28c0540a; reach of the physical part as in my earlier record, unchanged on upstream main)

`missile_blow` callers: `Fight::strike` (missiles.cpp:686: player and merc skill missiles, `row_damage` rows, `missile.fixed`), `missiles.cpp:822`, `prg` release-area hits and Hurricane / Shiver Armor / Thunder Storm (states.cpp:166, 228, 255, 353), `spot()` (Psychic Hammer, Mind Blast, Telekinesis: strike.cpp:681, 687, 779), pets (pets.cpp:120 `pet_aura`, 161 Poison Creeper / vines, 948), shrine potions (tick.cpp:402). Every one lands on `monsters[i]` through `target_of` (a hostile monster); hostile monster missiles at the player use the other `missile_blow` overload (ai.hpp:657), unchanged. So no merc, pet or player is ever a defender of the changed code.

- Element roll (commit 1): every class's elemental skill missiles (Amazon arrows / javelins, Assassin traps and Fire Blast, Druid fire / cold, Necromancer Teeth / Bone Spear / Bone Spirit / Poison Nova, Paladin Holy Bolt / Blessed Hammer / Fist of the Heavens, Sorceress all), the act 1 and act 3 merc skill missiles (`by_merc` through `strike`), pet auras and vines (Poison Creeper), Hurricane, Shiver Armor, Thunder Storm, missile rows without a Skill, shrine potions.
- Physical roll and floor (commits 2, 3): Magic Arrow; Psychic Hammer, Mind Blast, Blade Sentinel, Blade Fury; War Cry; Twister, Tornado, Shock Wave, Volcano, Armageddon. No Sorceress, Necromancer or Paladin path.
- Side effects (S): sub-point physical hits now reach `hurt()` with frac > 0 (awareness and get-hit roll, get-hit size includes the fraction), as the draft says. `Fight::land` soak still uses whole-point `resisted(blow.phys, ..)` (latent, unchanged).

## 5. Det scenarios (V run)

Binaries: base = my lab build of ee5e06d8 (sha256 78126554e8d8, reused from my earlier record, same commit), head = my build of 51e4eb1e (b2b893eece37), physonly = my earlier build of ee5e06d8 + 5b13a2c4 (546835d0548d, = commit 2 alone). One d2d slot (`d2dslot.sh run`, under `agentrun.sh verifyplayerroll`), the dethash2 driver from main plus the author's `hit_phys` scenario (the scenario is the thing under test), seed 3, lockstep.

| scenario | head vs base | physonly vs base |
|---|---|---|
| passive, fight, cold, static, cold2, hit_immune, merc, follow | same | same |
| hit_plain, hit_champion, hit_unique, hit_boss, hit_minion | differ (mon and state) | same |
| hit_phys | differ: 5 of 27 snapshots, M lines only (Tornado's target 125 vs 126 of 147, then 95 vs 97) | differ, hashes identical to head |
| pet | differs at `start` (Spirit Wolf `at=`), rerun same as base | differs at `start` twice (both runs) |

Reruns (base2, playerroll2, physonly2): hit_phys reproduces both hashes; pet: base2 = base, playerroll2 = base, physonly2 = the other start position. The pet difference is in the `start` snapshot (T lines only, before any blow) and both values occur on binaries that cannot differ there, so it is the start-position flake, not the claim; it hit 3 of 6 non-base runs here, not "only the first run".

`hit_phys` does exercise real physical hits (V run, per-block hp in the logs): Twister 125 -> 120, Shock Wave 114 -> 101, Tornado 147 -> 126 -> 97 on base. Only Tornado moves a hash: the other two deal the same whole points on both binaries and their fraction never crosses a point inside the scenario. Note the targets are not the 147-life Drowned Carcass of the scenario comment for the first two skills (125 and 114 life).

So: 8 identical, pet identical after rerun (9 of 15), 5 cold `hit_*` moved by commit 1 alone, `hit_phys` moved by commit 2 alone; every hash change is explained by the claimed commit. Agrees with the draft.

## 6. Warnings-as-errors and the clang-tidy step

Compile (V run, `werror.sh`): with each TU's own compile command from `compile_commands.json`, plus `-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror -Wshadow -Wrange-loop-construct -Wconversion` (`-Wno-error` for the last two so they show), `-O2`, on `tests/test_combat.cpp`, `tests/game/test_missile_hit.cpp` and `components/game/fight/missiles.cpp` (a TU that instantiates `missile_blow`):
- g++ 16.2.1: 0 warnings, head and base, rc 0.
- clang++ 23.1.4: rc 0; 66 warnings on head (33 on base, which lacks test_missile_hit), all `-Wsign-conversion` in untouched headers (dt1.hpp, ds1.hpp, cube.hpp, mpq.hpp, ai.hpp); **0 in combat.hpp, test_combat.cpp or test_missile_hit.cpp**.

MSVC /W4 /WX (not runnable here; by reading the diff): the new locals `roll` and `fine` in the physical block are in the `if` scope and the outer `roll` / `fine` are declared after it, so no C4456; `std::max(target.res[0], -100)` is int/int; `low + rng(high - low)` int; test_combat's new names (`edge`, `edge_rng`, `phys_rng`, `phys`, `half`, `phys_hit`) did not trip g++ `-Wshadow` (which flags the same nested-scope hiding as C4456/C4457); test_missile_hit uses `std::getenv` (C4996 silenced by `_CRT_SECURE_NO_WARNINGS`, set in CMakeLists) and explicit casts on the seed. No /W4 hazard found (S).

### 6a. CI step "Names and includes (Linux)" (V run, coordinator's addition)

Run as the workflow does (`.github/workflows/ci.yml` 241-267): `changed.txt` = the PR's `.cpp` / `.hpp` (`components/rules/combat.hpp`, `tests/game/test_missile_hit.cpp`, `tests/test_combat.cpp`), `tools/tidy-pr-select.py` against my full head build (`ninja -t deps`), which re-tidies every TU that includes a changed file: **125 TUs** (124 without my probe), then `clang-tidy -p build --quiet --extra-arg=-Wno-unknown-warning-option` on each (`.clang-tidy`: `readability-identifier-length`, `misc-include-cleaner`, WarningsAsErrors `*`). Local clang-tidy is LLVM 23.1.4; CI installs Ubuntu 26.04's (older).

Findings:
- **`tests/game/test_game_missiles.cpp` (an upstream-main file the PR does not touch, re-tidied because it includes `combat.hpp`): 3 misc-include-cleaner errors**: line 133 `d2d::game::new_character` (character_store.hpp), line 223 `d2d::rules::Blow` (combat.hpp), line 286 `d2d::rules::kMonsterCurses` (skills.hpp). These are exactly the three that failed PR #164; the file is the same on upstream main 6935cf64, so **this PR will fail the step too** unless they are fixed first. Fix verified (V run, scratch copy, restored after): add `#include <character_store.hpp>`, `#include <combat.hpp>`, `#include <skills.hpp>` to test_game_missiles.cpp: 0 include-cleaner findings left in that TU.
- The PR's own files: **0 findings** in `combat.hpp`, `test_combat.cpp`, `test_missile_hit.cpp` (the check is live: deleting `#include <map>` from test_missile_hit.cpp in a scratch copy gives `no header providing "std::map"`).
- Everything else is `readability-identifier-length` "binding variable name 'x' / 'y' is too short" in untouched files (outdoor.hpp, merc.hpp, apps/d2d/*), identical on base TUs: a check of LLVM 23 that CI's clang-tidy does not apply (main passes the step), not this PR's.

## 7. Suites (V run, my Release builds; tests keep asserts, tests/CMakeLists.txt strips NDEBUG)

| suite | base 6935cf64 | head 28c0540a |
|---|---|---|
| test_combat | pass | pass |
| test_rules | pass | pass |
| test_game_rules | pass | pass |
| test_game | pass | pass |
| test_missile_hit (no args) | - | SKIP (77), oracle mode |
| pr_guard.py 6935cf64 28c0540a | | 0 FAIL, 0 WARN |
| presubmit.py --quick --base 6935cf64 (ruff on PATH) | | 0 FAIL, 2 WARN (doc-claims missiles.md:106 "never"; build skipped) |
| CI tooling tests (test_tidy_pr_select, test_gaps, test_pr_guard; tools/emu test_tooling) | | 17 OK; 7 OK (1 skipped) |

ctest (V run, `ctest -j1 -E rollprobe`, D2_MPQ_DIR and the patch installer set, my full builds): head 67 tests, 56 pass, 7 skip, 4 fail; base 66 tests (no test_missile_hit), 56 pass, 6 skip, 4 fail. **The same four fail on both**: `test_mpq` (abort on this MPQ folder), `smoke_d2d`, `quests_act1`, `quests_act4`. No failure is new at head; test_missile_hit skips without arguments (by design).

## 8. The draft against my counts (read after sections 3-5)

Agree (V): the diff and four commits; head 0 at every res for the physical (mine 0 of 110,000 on new seeds, incl. 30,000 at res -101..-300); the element roll 0 at head for Sorceress and Necromancer (mine 0 of 59,080 over all 33 skills, plus 920 edges); the det table (5 cold hit_* by commit 1, hit_phys by commit 2, pet flake); reach of the physical part; the side effect.

Overclaims and errors in the draft:
1. **"a `lo == hi` element no longer draws"** (What changes, item 1, and the verifier checklist): wrong. On upstream main `Rng::range(lo, lo)` already returns `lo` without a draw (rules.hpp:172); my edge cases lo == hi are equal to game.exe on base too, seed included. The real RNG-side change is the opposite case: a zero or negative low end no longer draws (and no longer deals damage), and swapped ends now draw. No skill produces either (section 3b), so the stream is unchanged for every skill missile. Fix the sentence.
2. **Reach, element roll**: "trap missiles with a Skill (`objects.cpp`)" is wrong: object traps put the damage into a hostile missile's `MonStats` elements (objects.cpp:702-708) and hit the player through the monster path, not `missile_blow`. "merc ... auras (`pets.cpp`)": merc skill missiles go through `Fight::strike` (`by_merc`), merc auras do not reach `missile_blow`; pet auras do (`pet_aura`). Missing from the list: Psychic Hammer / Mind Blast / Telekinesis element parts via `spot()`, shrine potions (tick.cpp:402), missile rows without a Skill.
3. Proof table row 2 ("0 of 30,000 and 0 of 60,000 ... every class including res under -100") cites my earlier seeds: my earlier record ran 5b13a2c4 (no floor) and a clamped-input run, not head 28c0540a. Replace with this record's numbers (head 28c0540a, 0 of 110,000; upstream main 6935cf64 28,707 / 28,873 / 23,884).
4. Element proof is narrow: "Sorceress (Fire Ball, Glacial Spike)" is two skills; this record covers all 25 Sorceress and 8 Necromancer element skills at the roll. The draft's Necromancer "10,509 differ before" is life after one hit (a different measure from my roll-level 29,491 of 29,540); not contradictory, but the PR text should say the gameplay effect is 1/512 point on average (section 3b), not imply 30 % of hits are wrong.
5. Det: "the pet differs only on the first run" (brief) / "first run" (draft): here the flake hit 3 of 6 non-base runs. Say "run-to-run flake of the pet's start position".
6. "Twister and Shock Wave did not move a hash here": true, but they did hit (5 and 13 points); say why (no fraction carry crossed a point), so a reader does not think they missed.
7. ctest: the draft's "68 tests" counts my old probe; the PR's tree has 67 (66 on base), and the same four fail on base (agree in substance). "Not run: the clang-tidy step": run here (section 6a), and it **fails** on an untouched file the PR pulls in.

## 9. Verdict

**READY WITH CHANGES.** The code is right (V run): head 28c0540a matches game.exe on 0 of 110,000 physical cases (every res class, 30,000 of them at res -101..-300 on a monster defender) and on 0 of 60,000 element rolls (all 33 Sorceress / Necromancer element skills plus 920 edge cases incl. the seed after the roll); upstream main differs on 81,464 and 59,580 of them. The diff is the four claimed commits, a fast-forward of current upstream main; det hashes move only where and by which commit the draft says; g++ / clang++ with the CI flags plus -Wshadow -Wrange-loop-construct -Wconversion: 0 warnings in the PR's files; suites and ctest: no new failure.

Changes, in order:
1. **Required (CI blocker): add `#include <character_store.hpp>`, `#include <combat.hpp>`, `#include <skills.hpp>` to `tests/game/test_game_missiles.cpp`** (as its own small commit in this PR, or wait until #164's fix of the same three lines is on upstream main and rebase). Touching `combat.hpp` makes CI re-tidy that file, and its three include-cleaner errors fail "Names and includes (Linux)" (section 6a; fix verified).
2. PR text: replace "a `lo == hi` element no longer draws" with the true RNG effect (a zero / negative low end no longer draws or deals damage, swapped ends now draw; no skill produces either, so the draw count is unchanged for every skill missile).
3. PR text, reach of the element roll: drop "trap missiles with a Skill (objects.cpp)" and merc auras; add the merc skill missiles (via `strike`), `spot()` skills, shrine potions, missile rows without a Skill.
4. PR text, proof table: use this record's numbers for head 28c0540a (0 of 110,000 physical, 0 of 60,000 element; upstream main 28,707 / 28,873 / 23,884 and 59,580); the earlier 30,000 / 60,000 runs were on the floor-less candidate. Say the element fix is a fidelity fix worth 1/512 point on average per hit.
5. PR text, det: "run-to-run flake of the pet's start position" (3 of 6 here), and that Twister and Shock Wave hit in `hit_phys` without moving a hash.
6. Low: one author identity across the four commits.

Not tested here (U): MSVC itself; CI's exact clang-tidy version; the mastery stat on the missile unit in FUN_005a8910 (section 2; d2d applies mastery once); the physical roll range (left out of the PR by design, still differs from game.exe).

## Reproduce (worker host; tools/emu of a lab tree as cwd for the oracles, PYTHONPATH there, D2_MPQ_DIR / D2_PATCH_INSTALLER set)

```
physroll_game.py gen 4242 40000 > phys_a.tsv; physroll_game.py grid 515 30000 > phys_grid.tsv
physroll_game.py game phys_a.tsv > game_phys_a.tsv
<build>/tests/test_rollprobe --physroll phys_a.tsv > head_phys_a.tsv
physroll_game.py compare phys_a.tsv game_phys_a.tsv base_phys_a.tsv head_phys_a.tsv
<build-head>/tests/test_rollprobe --pairs sor nec > pairs.tsv
elemroll_game.py gen 8128 60000 pairs.tsv > elem.tsv; elemroll_game.py game elem.tsv > game_elem.tsv
<build>/tests/test_rollprobe --elemroll elem.tsv > head_elem.tsv
elemroll_game.py compare elem.tsv game_elem.tsv base_elem.tsv head_elem.tsv
d2dslot.sh run -- det_scenarios_run.py run bin det <15 scenarios> base,playerroll,physonly
```
