# Independent verifier: local/monsterpace-clean (monster base velocity 75, think paces, Blood Raven home rule, shaman corpse reach) (VERIFYMONSTERPACE, 2026-11-02)

> status: verifier record for the fork candidate `local/monsterpace-clean` (private fork remote, head eeebb4b6, 14 commits on upstream main 6935cf64; lab twin c396395f -> e3af9fea). Author's draft `docs/research/pr-monsterpace-draft.md` (branch agent/monsterpace) read only after my own counts. Offline: no game.exe instance, nothing pushed or posted; the author's `walk_velocity.py`, `mon_ais_act1.py`, `mon_ais_pace.py`, `mon_shaman_corpse*` and `mp_*` not used. Verdict in section 9.

Tags: **V run** (my harness or runs on the worker host), **V read** (my disassembly: the fork's `tools/emu/disasm.py --all`, grepped), **S** source read, **I** inferred, **U** not traced, **NOT RUN**.

Harness (mine; copies in `docs/research/pr-monsterpace-verifier/`): `paceprobe.cpp` (real `rules::mon_think` on random inputs, checked against a model I wrote from the disassembly), `base43.py` (FUN_00573cb0's stat block run natively), `corpseprobe.cpp` (real `shaman_corpse_reach` and base's expression vs my model of FUN_005f1380), `detcls.py` (which snapshot fields differ), `nsbuild.sh` / `werror.sh` / `tidy.sh` / `det.sh`.

Builds (V run): base = the clean Release build of 6935cf64 from my earlier verifier record (same commit, not the author's); head = my incremental build of a copy of that build dir with my own worktree of eeebb4b6 bind-mounted at the configured source path (private mount namespace; only the files the diff touches and their dependents recompiled, -j1, nice). Det used the author's lab binaries (self-reported `Git hash: c396395f` / `e3af9fea`, built 06:39Z / 06:49Z), run by me: a lab build of my own was not affordable in the CPU budget (section 5).

## 1. Diff, addresses (V read)

Diff 6935cf64..eeebb4b6: `ai.cpp` (+16 -4: the shaman call, Blood Raven's feed, `kMonsterVelocityPct`), `ai.hpp` (+3), `monsters.hpp` (+50: paces, `shaman_corpse_reach`, `target_home_dist`), tests, `mon_ais_pace_cases.inc` (841), four emu tools, `monster-ai.md`. The lab twin's content diff equals the clean one plus one `#include <cstdlib>` line.

| claim | address | my read |
|---|---|---|
| base 75 | 0x573f14..0x573f1b | `push 0; push 0x4b; push 0x43; push esi; call 0x627260` (SetStat), unconditional once the unit-type / row guards at the top pass; between SetStat 0x44 = 100 and 0x45 = 100 |
| path speed | FUN_00623f50 0x62413e..0x6241c9 | walk % = GetStat(0x43) (FUN_00625480, all lists summed) + FUN_00621740(4), floor 25 (0x62414e); velocity = FUN_00621360(row) x % / 100 truncated: the d2d formula with the base 75 |
| pace application | FUN_005a6380 | pct != 0: SetStat(0x43, pct) on a separate stat list (flags 0x4000, created if absent), then FUN_00623f50; so a pace ADDS to the base 75 (100 -> 175 %) |
| Zombie | 0x5eff19..0x5eff21 | got hit, or dist < aip2 and rand < aip1, or level 17 -> `FUN_005de190(100, 0)` then run; else wander 3; in melee a1/a2 without a pace |
| Brute | 0x5efca3..0x5efccd | pace = 100 - clamp(FUN_00621f20, 40, 100); FUN_00621f20 = (hp >> 8) x 100 / (maxhp >> 8) |
| Blood Hawk | 0x5f01d8 / 0x5f0241 / 0x5f025c / 0x5f0285 | charge pace aip5 (+0x6e) steps = dist, state 1; wander 4 at -50, wander 3 at 0; back-off (dist <= 3, or in melee past the aip3 roll) pace aip4 (+0x68); charged and in melee: a1, no pace |
| Corrupt Rogue | 0x5f0b5c, 0x5f0c70 | both runs pace aip4 (+0x68), path type 0xd |
| Corrupt Lancer | 0x5f5da8, 0x5f6013 | both runs pace 100 (the second only on the aip4 roll; else approach 3) |
| Fallen | 0x5f03b0 | pace 50 before the back-off 12; the only pace call in the think |
| Blood Raven | FUN_005e6320 | d_t = FUN_005dc480(target, home) (`jge 50`), d_self (`jbe 50`, so > 50), flag +0x1c; home at pace 100 (0x5e644a, 0x5e649b), about the target 100 (0x5e64fb, 0x5e6683), back-off 100 (0x5e67b8), circle 50 (0x5e6754). FUN_005dc480 = point_apart with less = (FUN_00620510 >> 1) + 1 |
| shaman | FUN_005f1380, 0x5f14c8..0x5f1502 | candidate class 0x13 / 0x3a, DD (mode 0xc), corpse bit, not raised; FUN_005b0bd0 = dx^2 + dy^2 <= aip4^2; FUN_005a0180(unit, 0xc) == 0 (flags +0x16 & (4 or 8)); the search runs only for a shaman with flag 8 and without 4 |

Every pace and boundary claim matches the disassembly. The pace "type" (edx 7 / 0xd) that the thinks also pass is not modelled by d2d (pre-existing, not claimed).

**Writers of stat 0x43** (all of `.text`, V read): SetStat (0x627260) at 0x460c55, 0x4aef19, 0x5342c6, 0x56b125, 0x5707ff = 100 (client and player/new-character initialisers), 0x4ae99c = 75 (the client's monster initialiser), 0x573f18 = 75 (this one), 0x557c18 / 0x62d3ad = -item field +0xd8 (armour speed penalty, items); list SetStat (0x6270b0) at 0x57b0a6 (chill), 0x5a63cb (pace), 0x620ee1 (run - 100); AddStat (0x6272b0) at 0x5a100b (+20, boss mod), 0x5a196b (+100), 0x4b0d67 (client). Nothing in the think path resets the base: paces, chill and auras sit on other lists and add. V.

## 2. Own counts

### 2a. Velocity base (V run, `base43.py`)
FUN_00573cb0's stat block run natively (0x573e8b..0x573f2c), zeroed and 0x7f-filled MonStats row, difficulty 0..2: stat 0x43 set exactly once, to 75, in all 6 runs (the 0x43 value is an immediate; the row cannot move it). The author's 78 row/difficulty pairs therefore follow from the instruction. I did not re-run the path-speed oracle over MonStats rows (the 196 of 196 formula check of the first pass); the formula is read above and d2d's line 2798 is the same expression with 75 in place of 100 (S).

### 2b. Paces (V run, `paceprobe.cpp`, 100,000 cases per think per seed, seeds 4711 and 90210)

| think | base 6935cf64 (4711 / 90210) | head eeebb4b6 (both) |
|---|---|---|
| Zombie | 40,739 / 41,027 | 0 |
| Brute | 59,348 / 59,273 | 0 |
| Blood Hawk (pace and steps) | 62,126 / 62,087 | 0 |
| Corrupt Rogue | 54,754 / 54,791 | 0 |
| Corrupt Lancer | 43,030 / 43,206 | 0 |
| Fallen | 50,125 / 50,287 | 0 |
| Blood Raven (pace + home rule) | 67,120 / 66,758 (13,296 / 13,211 of them the home rule) | 0 |

Break attempts inside the sets: aip at 0 and 100, Hawk aip5 -100..100, life % 40 / 100 / 0..100, dist 0..5, Blood Raven her distance at exactly 50 and the target's at 49 / 50, flag held with distance 5. My counts differ from the author's because my input distribution differs (mine hit each branch more often); the head side is 0 in both. Limitation: the model checks the pace given the act `mon_think` chose (and Blood Raven's home choice, the Brute and Zombie branch); it is not a native run of the thinks (the author's corpus is; I did not re-generate it).

### 2c. Corpse reach (V run, `corpseprobe.cpp`, 100,000 per seed, a third of them on the aip4 boundary)
seed 2718: head 0, base 15,692; seed 31415: head 0, base 15,619. Mapping assumed as in the PR: champion -> bit 4, unique and superunique -> bit 8. **U: whether a superunique carries bit 8**; if it carries only bit 2, game.exe takes a superunique's corpse and d2d does not (author says the same, Limits).

## 3. What else reads the velocity (S, grep at head)

- Changed: `monster_update` (ai.cpp:2798), the plain walk / run of every monster.
- **Left at 100**: ai.cpp:1935 (Diablo's run sequence) and ai.cpp:1989 (a whirl sequence's step) still use `100 + speed_pct + boss_speed`. FUN_00623f50 is the only path-speed setter read here and reads the summed stat 0x43, so if those moves go through it they are 75-based in game.exe too (U, not traced). After this PR the code holds two monster bases; the PR should say so (draft does, U).
- Pets (pets.cpp:218, 286), mercs (merc.cpp:287): unchanged, 100. **I**: hirelings and summons are type-1 units created through FUN_00555230 -> FUN_00574250 -> FUN_00573cb0, and the only later SetStat(0x43, 100) is in player initialisers (FUN_005706d0, FUN_0056af80), so in game.exe a merc and a pet probably start at 75 as well. Not traced; not this PR's change, but "mercs, pets unaffected" means "d2d unchanged", not "d2d right".
- Chill (coldeffect), Spider Lay's -100, aura velocitypercent (speed_pct), boss_speed (fast +20 / fanatic): all additive in both game.exe and d2d, so the base change makes each of them relatively stronger: a chilled plain walker with coldeffect -50 goes from 50 % to the 25 % floor (one third of its unchilled speed instead of one half). That is game.exe's arithmetic (V read), but it is the most visible gameplay effect and should be in the PR text.
- Players: `World::walk_percent` (100 base) untouched. `apps/d2d/netgame.cpp` takes 0x43 from host packets: unaffected.
- `mon_think` callers: only the new paces flow into `monster.move_pct` (existing path). XP, difficulty, damage, drops, AI decisions other than the paces: no reader changed (diff read, V). Brute's `life_pct` and Blood Raven's sizes come from existing ai.cpp feeds (S).

## 4. Det (V run, author's lab binaries, my run: one hidden d2d slot, seed 3, 15 scenarios)

passive identical (mon b6db27442898, state 94d8c03bf4d3, as the author). The other 14 differ (author ran 14 scenarios without hit_phys; my fight / hit_plain hashes equal his). `detcls.py` per scenario: in every one the first difference is a monster position (M fields 1..4, S `at`); the rest follows from the moved walkers: missile commands at a target position (CMD lines in hit_*), chill / stun timing (hit_*), the pet's and the player's path, goal and target (pet, follow), a few hp values (follow: 4 S lines and the player; hit_minion: 1) from blows landing at different moments. merc: positions only. No difference without a moved monster before it. Pet: the T differences start after the monsters moved (not at `start`), so here they are not the start-position flake seen in earlier records; I did not rerun pet.

## 5. CI parity

- g++ 16.2 and clang++ 23.1 (V run), each TU's own compile command plus `-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror -Wshadow -Wrange-loop-construct -Wconversion` on `ai.cpp`, `test_monster_ai.cpp`, `test_monster_thinks.cpp` (they include `ai.hpp` and `monsters.hpp`): rc 0. g++: 1 warning, `ai.cpp` `shot` shadow, present on base (line 2568). clang++: 33 `-Wsign-conversion`, none on an added line (ai.hpp:766, 825, 826 and untouched headers). The build force-includes `cmake/no_near_far.hpp`, so `near` / `far` identifiers fail on Linux too: the head build passing proves none is left (the e6a09533 `far` is gone in the clean branch).
- MSVC /W4 hazards (read): new locals `run` (lambda), `wide`, `dying_word`, `home_x`, `home_y`, `zombie_pace`, `lancer_pace`, `rogue_pace`, `brute_pace`, `hawk_pace`, `hawk_steps`, `fallen_pace`, `raven_pace`: no hiding of an outer local (g++ -Wshadow silent on them); no narrowing (all int); `std::max(`/`std::clamp(` calls are in files that already use them (NOMINMAX handling unchanged). No windows.h macro name in an identifier (grep of added lines for far/near/min/max/small/interface/IN/OUT: only `std::max`, and "near" in comments). `tools/emu/mon_shaman_corpse_harness.cpp` is not in the CMake build.
- **clang-tidy "Names and includes" step** (V run, `tools/tidy-pr-select.py` on my head build: **127 TUs** selected, because `monsters.hpp` and `ai.hpp` are widely included). Run: 49 of 127, ordered PR files first (ai.cpp, test_game_missiles.cpp, the three monster tests, header_check, components/, tests/), then stopped for time; 15 header_check TUs hit a missing-database error I caused mid-run and are counted as NOT RUN.
  - **`tests/game/test_game_missiles.cpp`: the same three misc-include-cleaner errors** (line 133 `new_character`, 223 `Blow`, 286 `kMonsterCurses`) that failed #164: this PR re-tidies that file, so **on base 6935cf64 the step fails**. Current upstream main (050294e5) already has the three includes (ab3bf0d3, "Include what's used"), so **rebasing onto current upstream main fixes it**; no file change needed in this PR.
  - PR files: 0 include-cleaner findings; `readability-identifier-length` only on untouched lines (test_monster_ai.cpp:312 `[x, y]`, outdoor.hpp, merc.hpp, pets.hpp): LLVM 23's check that CI's older clang-tidy does not apply (as in my earlier records).

## 6. Suites, merges (V run)

| suite | base | head |
|---|---|---|
| test_monster_thinks, test_monster_ai, test_monsters, test_rules, test_game_rules, test_combat, test_game (MPQ folder set) | rc 0 | rc 0 |
| pr_guard.py 6935cf64 eeebb4b6 | | 0 FAIL, 0 WARN |
| presubmit.py --quick --base 6935cf64 (ruff on PATH) | | 0 FAIL, 3 WARN (siblings: unit kinds; doc-claims monster-ai.md:216; build skipped) |
| ctest (full) | NOT RUN (my head build has only the suite targets) | NOT RUN |

Merges (`git merge-tree`, read-only fetch): upstream main is now 050294e5 (41 commits past 6935cf64, touching `ai.cpp` / `ai.hpp`): clean. With #164 (ee8a2093, touches ai.cpp/ai.hpp): clean. With #166 (7d4f6f0e): clean. upstream main + #164 + #166 + candidate in sequence: clean. Not built or tested after the rebase (NOT RUN).

## 7. The draft against my counts

Agree (V): every address and pace; the boundaries `> 50` own / `>= 50` target; FUN_005dc480's size term; corpse rule and mask; base 75 as the only base write; passive unchanged and the other det hashes moved by walking (fight / hit_plain hashes identical to mine); suites, pr_guard, presubmit.

Overclaims / errors:
1. **clang-tidy: "32 findings, all readability-identifier-length"** is wrong for the tree the draft describes: on 6935cf64 the step selects `test_game_missiles.cpp` and fails on its three include-cleaner errors. Rebase onto current upstream main and say so; the PR as drafted (base 6935cf64) would fail CI.
2. **Base 6935cf64 is stale**: upstream main moved 41 commits (ai.cpp / ai.hpp among them). The proof table and the det table should be re-run after the rebase (merge is clean, but hashes and line numbers move); the draft's line numbers 1931 / 1985 are 1935 / 1989 at eeebb4b6 already.
3. "nothing in the think path resets it" is true but undersells the reach: chill, auras and Spider Lay become relatively stronger at the same additive values (section 3). Say it.
4. "merc / pet / player 0 before and after" measures that d2d's merc and pet sums did not change, not that they match game.exe; by my read they probably also start at 75 (I). Phrase it as "unchanged, U".
5. The proof counts (122,646 velocity cases, 100,000 per think) are the author's runs; mine reproduce "head 0" on independent inputs, not those exact numbers. Fine, but cite the inputs.

## 8. Risk: one PR or split?

Recommendation: **split in two**. (A) "A monster starts at 75 percent velocity" (96bee429 plus its doc and walk_velocity.py lines): one-line behaviour change, game-wide, every awake monster 25 % slower and every slow relatively stronger; easy for the maintainer to judge on its own and to revert alone. (B) the think paces, Blood Raven's home rule and the shaman reach (the other commits plus the corpus): local to seven Act 1 AIs and one shaman. Reason: (B) is self-contained and cheap to accept; (A) alone changes every det hash and the feel of the game, and if the maintainer hesitates on it, (B) should not wait. Order: (B) first is safe (paces are additive deltas, correct under either base); (A) second. The shaman reach could ride with (B); splitting it out a third time buys little.

## 9. Verdict

**READY WITH CHANGES.** The code is right where claimed (V): every address and branch read; head 0 of 700,000 pace cases per seed (two seeds) and 0 of 200,000 corpse cases, base tens of thousands; the base 75 is the only base write of stat 0x43; suites pass on both; merges clean with upstream main, #164 and #166.

Required:
1. **Rebase onto current upstream main (050294e5 or later) before opening**: on 6935cf64 the CI clang-tidy step fails on `tests/game/test_game_missiles.cpp` (three include-cleaner errors, fixed upstream by ab3bf0d3). After the rebase re-run the suites, the det table and the tidy step's own files.
2. PR text: correct the clang-tidy sentence; state the chill / aura / Spider Lay relative effect; state that Diablo's run and the whirl step stay at base 100 and that mercs and pets are unchanged (and probably 75 in game.exe, U); update line numbers.
Recommended: split as section 8; one author identity across the commits.

Not run (U / NOT RUN): native re-run of the thinks (mine is a disassembly-derived model); the path-speed oracle over MonStats rows; my own lab build for det; full ctest; 78 of 127 tidy TUs; MSVC itself; superunique flag bits.
