# Verifier: upstream batch 2 (minmana floor; Teleport D1 room limit, D2 nearest free spot)

> status: verifier report (card verifyb2, 2026-11-02, Opus, one agent). Offline: no game, no d2d slot, nothing pushed or posted; the fork checkout was only read (an own clone in the lab dir for builds). Subject: [pr-batch2-draft.md](pr-batch2-draft.md) (card prbuildb2), fork branches `local/batch2-minmana` 82f8d3e4 and `local/batch2-teleport` d2572d6c (D1 74a17909), base upstream main 3e073888.

Tags: **V read** = disassembly or diff read by this verifier; **V run** = run by this verifier (harness `tools/emu/vb2_*`, commands in section 6); **S** = the author's or an earlier doc, not re-run here; **U** = not verified.

## Verdict

| branch | ready for PR | condition |
|---|---|---|
| local/batch2-minmana | **yes, with text changes** | F3, F4 (comment and test naming), F5 (wrap the commit message) |
| local/batch2-teleport | **yes, after a one-line fix** | F1 (`ring <= 50` must be `ring < 50`), then re-run test_tele_spot; text changes F2, F5 |

Order of work: read game.exe first (section 1), own model and harness (section 2 and 3), real d2d builds base vs head (Release, `D2D_WERROR=ON`, built from an own clone), and only then the author's numbers (section 4). No disagreement with the author on any count; one deviation the author's oracle could not see (F1), and the "U" tie order is now read in the code (it holds).

## Claims

| # | claim (draft) | verdict | evidence |
|---|---|---|---|
| M1 | FUN_0056c160 checks mana against the formula alone, no minmana floor | **VERIFIED** | V read; V run: own model equals game.exe's native code in 45,517 + 45,605 cases (seeds 3, 11), 0 differ |
| M2 | FUN_0056bfe0 pays max(formula, minmana x 256), nothing when short, nothing for mana 0 and lvlmana 0 | **VERIFIED** | same runs (the paid amount is part of each answer) |
| M3 | d2d base refuses a cast game.exe makes; head equal on every compared case | **VERIFIED (counts)** | own probe on the real builds: seed 3 base 117 of 28,454 differ, head 0; seed 11 base 106 of 28,534, head 0. All base differences "game.exe casts and pays nothing, d2d refuses": Teleport, Dopplezon, Bone Prison, Guided Arrow (the author's four) |
| M4 | the four cast paths all take `mana_paid` | **HOLDS WITH LIMITS** | V read the diff (cast, cast_missile, cast_summon, start_swing). Own probe drives cast / cast_missile / cast_summon only; start_swing (melee) is covered by the author's harness alone (S) |
| T1 | D1: Teleport lands only in the caster's room or its near list | **HOLDS WITH LIMITS** | V read FUN_005ca360 -> FUN_00554ea0: FUN_00463740 on the target, refuse when null. Not re-run here (needs room structures); S: verify-teleport-room.md (HOLDS) and the author's 25,120 -> 0 |
| T2 | D2: a blocked target lands on the first ring with a free spot, least abs(dx) + abs(dy), tie order as stated | **VERIFIED for rings 1..49; WRONG at ring 50** | V read FUN_0064dea0 (below); V run: game.exe's FUN_0064dea0 natively vs the real `Level::warp_spot` of d2572d6c, 20,000 cases (seed 5), 18,990 moved spots, 0 differ on rings 1..49, 179 differ, all "game.exe refuses, d2d lands on ring 50" (F1) |
| T3 | tie order "an inference from runs (U for rings above 7)" | **VERIFIED (read and run)** | V read: the scan is rows top to bottom with the left then the right column (corners included), then the inner columns left to right, top then bottom row; strict `<` keeps the first. Equal to d2d's order for every distance that can tie (corners only tie with corners, in the same order). Run as T2 |
| T4 | the plus footprint with mask 0x1c09 less the units' bit | **HOLDS WITH LIMITS** | V read: FUN_00554ea0 passes 0x1c09 and the unit's pattern (FUN_00620510) to FUN_0064e7b0; the pattern being the plus is S (the author's oracle runs FUN_0064d9b0 natively; this harness hooks it, so the footprint is not re-proven here) |
| P1 | identity, one purpose per commit, three-line body, `Agent: Claude Code` | **HOLDS WITH LIMITS** | V run `git log`: author = committer = the numeric noreply identity on all three commits; body 2 lines; last line `Agent: Claude Code`; no co-author line. But subjects are 85 / 65 / 63 characters (82f8d3e4, d2572d6c, 74a17909) and the six body lines 113 to 154, unwrapped (F5) |
| P2 | CI flags g++ and clang++ clean; extra warnings 0 on added lines | **VERIFIED** | V run: syntax-only g++ and clang++ with -Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Wshadow -Wrange-loop-construct -Wconversion on every changed .cpp plus tick / strike / missiles: 0 warnings on added lines for both branches (clang prints 270 / 168 on old lines). Release builds with `D2D_WERROR=ON` (g++) pass |
| P3 | the suites rc 0 | **VERIFIED** | V run on each head: test_rules, test_game_rules, test_monsters, test_monster_thinks, test_monster_ai, test_combat, test_game, test_sk_ground (the 8) plus test_skills and the branch's own test: rc 0 each |
| P4 | merge with #178 (head 25978757) clean; suites pass on the merged trees | **VERIFIED** | V run, section 5 (the author did not re-run the differential on merged trees; this verifier did) |
| P5 | MSVC /W4, windows.h macros | **HOLDS (read)** | V read: no `min`/`max`/`near`/`far` identifiers added; `std::max` as before; casts explicit; gamedata.hpp already includes `<optional>`, `<utility>`, `<cstdlib>`. A local lambda named `free` shadows the C function (legal; a reviewer may ask for another name) |

## Findings (ranked)

| # | sev | finding | recommendation |
|---|---|---|---|
| F1 | medium | `Level::warp_spot` searches rings 1 to 50; game.exe's FUN_0064dea0 stops at 49: after ring r it goes on only while r + 1 < 50 (`cmp esi, [ebp+0x24]; jl` with 50 from FUN_0064e7b0). 179 of 20,000 cases (all with every spot within ring 49 blocked) land in d2d and are refused by game.exe. Rare in play, but it is a deviation the branch adds | `ring < 50` in `warp_spot`, the comment "(1 to 49)", one assert in test_tele_spot (a target whose first free ring is 50 is refused) |
| F2 | low | The D2 commit and the comment say the tie order is "read off 20,000 oracle runs" / "U for rings above 7". It is now read in FUN_0064dea0 | say "FUN_0064dea0's scan order" in the comment and the body; drop the U |
| F3 | low | test_sk_mana: `auto& paladin = *rigs[1];` is the Amazon rig, and `if (paladin.start(might, ...)) assert(...)` is vacuous when no path starts Might (the draft says Might has no cast path). The header comment says the test shows "Might pays nothing"; it does not | name it `amazon`, drop the Might lines or move the zero-row case to test_skills (it is already there: `mana_paid(cheap, 3, 4096) == 0`) |
| F4 | low | `mana_paid` is commented as FUN_0056bfe0, but FUN_0056bfe0 has two more branches the change does not model: a unit in state 0x72 pays through FUN_005d2b60, and a skill whose srvdofunc is 116 (Werewolf, Werebear) passes FUN_0056c160 at once when FUN_0063a400 says so. Both are unchanged from base (not a regression) | name them in the PR body's limits (U), not in this change |
| F5 | low | Commit subject over 72 (82f8d3e4: 85) and body lines 113 to 154 characters, unwrapped (the bundle-1 verifier's F1 asked for 72) | wrap; hashes change only, re-run pr_guard |
| F6 | info | game.exe's room rule for the found spot: FUN_0064dea0 looks each candidate up with FUN_00463740 from the last room it found (the reference moves), so a spot is valid only in rooms reachable that way; d2d's `warp_spot` has no room test. Equal on the oracle's 5 x 5 rooms; real irregular levels not compared (U, the author says so too) | keep the author's limit line |
| F7 | info | Both functions return at once for a non-player unit (`cmp dword [unit], 0`), so game.exe neither checks nor pays a hireling's or a pet's mana through them. Whatever d2d does for mercs and pets through `Fight::pay` is outside this change's evidence (presubmit's siblings WARN) | one line in the body: the change is for the player |

## 1. game.exe read (before any model)

- FUN_0056c160 (ECX unit): null or not a player -> 1. Skill entry (FUN_00620250); none -> 0. A charge (FUN_00643ad0 != -1) -> its count > 0. Else row (FUN_00644140), lvl = FUN_006442a0(unit, entry, 1) - 1 clamped at 0, formula = (lvlmana x lvl + mana) << manashift; srvdofunc 116 and FUN_0063a400(unit) -> 1; else stat 8 >= formula (signed). No minmana. V read.
- FUN_0056bfe0 (ECX game, EDX unit, skill id, level): not a player -> 1; the entry's charge path -> FUN_0056bec0; mana 0 and lvlmana 0 -> 0 (nothing paid); cost = max(formula, minmana << 8); state 0x72 -> FUN_005d2b60(unit, cost); stat 8 < cost -> 0 (nothing paid); else stat 8 -= cost. V read.
- FUN_005ca360: the level's Teleport byte 0 -> refuse; 2 -> FUN_006229f0 line test with 0x804, refuse if blocked; then FUN_00554ea0(game, unit, room 0, x, y, 0, 0). FUN_00554ea0: target room by FUN_00463740 (null room, then the caster's room); none -> refuse; then FUN_0064e7b0(room, &xy, pattern, 0x1c09, 0) -> FUN_0064dea0(room, &xy, 0, pattern, mask, mask, 0, 50, 1). V read.
- FUN_0064dea0: the target itself if free; else ring r = 1, 2, ...: part 1 rows y - r .. y + r, in each the columns x - r then x + r; part 2 columns x - r + 1 .. x + r - 1, in each the rows y - r then y + r; distance abs(dx) + abs(dy), replaced only on a strictly smaller one; stop at the first ring with a find; next ring only while r + 1 < 50. V read.

## 2. Mana (own harness)

- game side `vb2_mana.py`: FUN_0056c160 then, if it says yes, FUN_0056bfe0, natively; only lookups hooked (entry, row with the patched Skills.txt mana / lvlmana / manashift / minmana / srvdofunc at their 1.14d offsets, level, stat 8, stat add recorded, state 0x72 -> 0, FUN_0063a400 -> 0 and counted: 447 calls, the Werewolf / Werebear rows). Cases: every row with a mana field x 11 fixed + 4 random levels x 15 thresholds around formula and floor, plus 20,000 random.
- d2d side `vb2_mana_probe.cpp` (scratch, not for upstream): a character of the skill's own class, the dispatch order of world/commands.cpp (self cast -> `Fight::cast`, missile or spot -> `cast_missile`, summon -> `cast_summon`); a case whose skill no path starts at full mana is not compared (17,063 / 17,071: passives, auras, melee).

| seed | cases | compared | base differs | head differs | own model vs game.exe |
|---|---|---|---|---|---|
| 3 | 45,517 | 28,454 | 117 (Teleport 44, Dopplezon 31, Guided Arrow 21, Bone Prison 21) | 0 | 0 of 45,517 |
| 11 | 45,605 | 28,534 | 106 (Teleport 32, Bone Prison 30, Dopplezon 29, Guided Arrow 15) | 0 | 0 of 45,605 |

## 3. Teleport search (own harness)

- game side `vb2_tele.py`: FUN_0064dea0 natively with the warp's arguments; hooked FUN_00463740 (one room for a 200 x 200 grid, none outside) and FUN_0064d9b0 (plus footprint on the case's blocked subtiles). Cases: random obstacle blobs (radius 1 to 14, density 0.3 to 1.0, scattered walls), 15 percent with the target freed, 5 percent with a full square of radius 46 to 50 (rings 48 to 52 first free).
- d2d side `vb2_tele_probe.cpp`: the real `Level::warp_spot` of d2572d6c on a Blood Moor copy, the grid walled round.
- seed 5, 20,000 cases: game.exe lands 19,612 (18,990 moved), refuses 388; d2d differs in 179, all "lands on ring 50" (F1); 0 on any ring up to 49, ties included.
- Not run here: D1 (room limit) on rooms, the whole `Fight::spot` path; the base build has no `warp_spot`, so base for D2 is the read "refuses a blocked target" (and the author's 1,461 -> 0, S).

## 4. Comparison with the author (read after the counts)

- Mana: the author's base 278 of 76,939 (seed 1) / 281 of 105,978 (seed 7), head 0, the same four skills. This verifier's numbers are lower because its probe has no melee path and fewer threshold cases; direction and skills agree. The author's game side is native (sk_mana.py, S); the equality claims are run, not arithmetic re-derivations. Agree.
- Teleport D2: the author's 1,461 -> 0 and "ring and distance matched in 20,000 of 20,000" agree with rings up to 49; the author's oracle never built a case with ring 50 as the first free one, so it could not see F1. D1 25,120 -> 0: S, consistent with the read.

## 5. Merge with #178 and the merged trees

- `git merge-tree --write-tree` of pull/178 (head 25978757, fetched read-only into the own clone) with each branch head: clean, no conflict (V run). Files both touch: minmana fight.hpp, missiles.cpp, strike.cpp, skills.hpp; Teleport strike.cpp, gamedata.hpp (other hunks).
- Throwaway merge commits (own clone, never pushed), Release builds with `D2D_WERROR=ON` (g++): both build, including #178's test_panel_states (built, not run here).
- On the merged trees (V run): the 8 suites, test_skills and the branch's own test rc 0 each; the own differentials re-run there: mana seed 3 head-with-#178 0 of 28,454 differ; Teleport seed 5 the same 179 ring-50 cases and nothing else (F1 is not changed by #178).

**P4: VERIFIED** (test_panel_states not run).

## 6. Re-run

On the worker host, from this worktree (outputs in the scratch dir):

    tools/emu/run-on-worker.sh vb2_setup.sh                       # own clone, diffs, metadata
    tools/agentrun.sh pa-verifyb2 --ttl 7000 -- labrun <worker> -- bash tools/emu/vb2_build.sh base 3e073888   # and mm 82f8d3e4, tp d2572d6c (+ test targets)
    tools/emu/run-on-worker.sh vb2_mana_run.sh 3                  # game.exe side; then 11
    tools/emu/run-on-worker.sh vb2_mana_d2d.sh 3 base mm
    tools/emu/run-on-worker.sh --ttl 2400 vb2_tele_run.sh 5 20000
    tools/emu/run-on-worker.sh vb2_tele_d2d.sh 5 tp
    tools/emu/run-on-worker.sh --ttl 3000 vb2_ci.sh mm test_sk_mana   # and tp test_tele_spot
    tools/emu/run-on-worker.sh vb2_merge.sh 82f8d3e4              # throwaway merge commit in the own clone

## 7. Not proven, leave out of the PR

- Melee (start_swing) mana: the author's harness only (say so).
- D1 on real irregular levels, the moving room reference of FUN_0064dea0 (F6), the plus footprint as the player's pattern (T4, S).
- State 0x72 payment and the srvdofunc 116 shortcut (F4); mercs and pets (F7).
- Det scenario hashes base vs head: NOT RUN (needs a d2d slot). Live game: NOT RUN.
- clang-tidy: NOT RUN by this verifier (the author's 0 on added lines is S).
