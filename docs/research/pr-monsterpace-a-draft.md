# PR A draft: the paces game.exe's Act 1 thinks set, Blood Raven's home rule and the Fallen Shaman's corpse reach (MONSTERPACE)

> status: DRAFT, ready for the PR-open step on the user's go (nothing pushed upstream, opened or posted). Candidate on the private fork remote only: `local/monsterpace-a`, head `260eb34f`, 13 commits on current upstream main `3be426d3` (read-only fetch 2026-11-02; it contains #164 and #166). Companion: [pr-monsterpace-b-draft.md](pr-monsterpace-b-draft.md) (base velocity 75). Replaces the single draft `pr-monsterpace-draft.md` after the independent verifier's report (`docs/research/pr-monsterpace-verifier.md` on branch agent/verifymonsterpace, READY WITH CHANGES). Evidence: [monsterai2-trace.md](monsterai2-trace.md), [monster-ai-trace.md](monster-ai-trace.md). Format follows [pr-missilerecord-draft.md](pr-missilerecord-draft.md). Tags: V run (run on the worker host), V read (disassembly read), S source, U not traced.

Title: `Monsters carry the pace game.exe's Act 1 thinks give them, Blood Raven goes home by FUN_005dc480's distances, and a unique Fallen Shaman's corpse reach is FUN_005f1380's`

## What changes (7 claims, nothing about the base velocity)

1. **Zombie** runs at pace 100 (`FUN_005de190(100, 0)` before its run, 0x5eff19). `47cc7433` is now `019bf4d6`.
2. **Brute** walks in at pace 100 - clamp(life %, 40, 100) (0x5efcbf). `6ab743d4`.
3. **Blood Hawk**: charge pace aip5 with steps = its distance, the aip2 wander -50, the back-off (also in melee) aip4. `8479d820` (the local `far`, a windows.h macro name, that an earlier fork commit used is not in this history: it never existed here; the build forces `cmake/no_near_far.hpp`).
4. **Corrupt Rogue** runs at pace aip4 (both runs) `0b91a2b8`; **Corrupt Lancer** runs at pace 100 (both runs) `1b515278`.
5. **Fallen** flees a dying kin at pace 50 (0x5f03a8). `87c6d88c`.
6. **Blood Raven**: pace 100 on the walk home, about the target, the 5 percent move about and the back-off, 50 on the circle (`1c8704ff`); her home rule: the flag, pace 100 and the walk home when the target is 50 or more from home, or she is more than 50 (game: `jbe`), both distances FUN_005dc480's (each axis less half the unit's size + 1, `point_apart`; `ai.cpp` feeds them for her only) (`a15ef268`).
7. **Fallen Shaman** (unique): corpse reach by the room-search callback FUN_005f1380: a dead class 19 / 58 monster within aip4 squared, not flagged champion or unique (d2d took any within 80 by `ai_distance`, champions included). `424727a4`.

Plus tests and evidence tools: `3c6b2fb3` (`mon_ais_act1.py`, `mon_ais_pace.py`), `91b2f659` (840 game.exe runs replayed through `mon_think`), `69304707` (unit asserts per think), `260eb34f` (docs/research/re/monster-ai.md, and an unused include out of the corpse harness). One author identity on all 13.

A pace is added on a separate stat list on top of the monster's velocity percent (FUN_005a6380 then FUN_00623f50; V read, verifier): these paces are deltas and are right under the base 100 d2d has today and under 75 (PR B). So A does not need B and B does not need A.

## Base, overlap with upstream moves (V read)

`6935cf64..3be426d3` (61 commits) touched `ai.cpp` / `ai.hpp` (monster missile record and poison #164, get-hit needs mGH, swaps_ai, Imp Inferno, the ring-shot `-Wshadow` rename, cursor rule), `montypes.hpp` and `rules.hpp`; none touches a think pace, the velocity sum, Blood Raven, the shaman or the movement of a monster, and `components/rules/monsters.hpp` is unchanged upstream. All 13 commits cherry-picked onto `3be426d3` without a textual conflict and the suites pass (below). Semantic check: the new `ai.cpp` lines read only `type_info.size`, `target->size`, `monster.home_x/y`, `other.boss` and the unchanged `input` fields; none of the upstream changes renamed or retyped them (the build proves the names; V run).

## Proof (V run, worker host, seed 20261102, game.exe's own code run under unicorn, d2d's real `rules::mon_think` compiled and replayed)

100,000 cases per AI (Fallen: 73,240 without a dying kin, 26,760 with; leader command none or 1), `mp_run.sh ... none1` then `... less replay` (the Blood Raven home distance as game.exe takes it). "pace" = the pace field; "all" = every field. Counts regenerated on current main (they differ slightly from the first draft's: the generator's draws moved with upstream's `mon_ais.py`).

| AI | base `3be426d3`: all differ / pace differ | head A: all / pace |
|---|---|---|
| Zombie | 23,050 / 23,050 | 0 / 0 |
| Brute | 60,005 / 39,818 | 60,005 (act only) / 0 |
| Blood Hawk | 56,062 / 53,264 | 0 / 0 |
| Corrupt Rogue | 31,029 / 25,579 | 31,029 (n only) / 0 |
| Corrupt Lancer | 55,299 / 55,299 | 0 / 0 |
| Fallen with a dying kin (13,387 + 13,373) | 26,760 / 26,760 | 0 / 0 (the 73,240 others 0 before and after) |
| Blood Raven | 65,737 / 65,668 | 0 / 0 |

Replaying Blood Raven with the old feed (no `less`) on the head thinks leaves 1,174 differing (822 pace): the feed in `ai.cpp` is part of the claim; her replay hooks the unit size at 2, so it exercises `mon_think`, not `ai.cpp`'s feed (S; same limit as before).

Corpse reach: `tools/emu/mon_shaman_corpse.py` + harness, 100,000 candidates: game accepts 8,268; d2d before (`--old`) accepts 18,489 and differs on 10,221; head differs on 0.

The verifier's independent model (disassembly-derived, two seeds, 100,000 each) agreed: head 0, base tens of thousands; corpse head 0 of 200,000 (V run, verifier).

Committed regression: `tests/mon_ais_pace_cases.inc` (840 runs, 120 per AI) replayed by `test_monster_ai`; per-think asserts in `test_monster_thinks` (Blood Raven's 50 / 51 and target 49 / 50 boundaries included).

Suites (V run, Release build of the head, from `build/tests` with the MPQ folder set): `test_monster_thinks`, `test_monster_ai`, `test_monsters`, `test_rules`, `test_game_rules`, `test_combat`, `test_game` all exit 0 on base, A, B and A+B (a throwaway merge `b5f0028d`; 28 runs).

Det scenarios (V run, one hidden d2d slot, no game.exe, seed 3, lockstep; lab binaries = the lab patch set (8 commits, `local/monsterpace-lab-base`) cherry-picked onto `3be426d3` with no conflict, then A's 13 commits (`local/monsterpace-lab-a`); the 15 scenarios): `passive`, `hit_immune` and `merc` are **identical** (mon / state hashes equal); the other **11 differ** in both hashes, `fight` `1124443e0e4e` / `9f5bc7695b85` -> `7529f6a462f9` / `5c066d0edb3b`. Cause (V run, diff of the snapshot lines): in every one the first differing snapshot line (frame counters left out) is a monster position, or the pet's own position in the same snapshot as the monsters' (pet); the rest follows from walkers that move differently: a Zombie that now runs when it is hit or sees the player at pace 100 (a pace adds to the velocity, so most of these walkers move faster than before; the Blood Hawk wander -50 is the slower one), later missile / stun timing and a few hp values. Why `hit_immune` and `merc` are equal: their zombies never reach a pace branch (I, not traced). The base is deterministic (`base2` reruns of passive, fight and pet repeat the base hashes). No hash is claimed unchanged except those three.

## CI parity (V run, g++ 16.2 and clang++ 23.1, the TUs' own compile commands)

`-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror` plus `-Wshadow -Wrange-loop-construct -Wconversion -Wsign-conversion` (the extras as warnings) over `components/game/ai.cpp`, `tests/monsters/test_monster_ai.cpp`, `tests/monsters/test_monster_thinks.cpp`: rc 0 on both, with the same warning set as base (29 under g++, 33 under clang++, 0 new by message and file) and the corpse harness compiled under the same flags: rc 0. MSVC /W4 (read, not built): new locals `run` (lambda), `wide`, `dying_word`, `home_x`, `home_y`, `*_pace` hide no outer local, no narrowing (all int), `std::max` / `std::clamp` only in files that already use them; no windows.h macro name (`far`, `near`, `min`, `max`, `small`, `interface`, `IN`, `OUT`) as an identifier in the added lines (grep: only "near" in a comment).

clang-tidy (LLVM 23.1.4, `-p build`, the CI invocation, niced, one TU at a time): the selection is `tools/tidy-pr-select.py`'s rule (a changed file or a TU whose dependency list contains one; 3 TUs new on upstream had no dependency record in my build and got theirs from the compiler's `-MM`): **130 TUs, all run**. 14 unique findings, all `readability-identifier-length` (LLVM 23's check, which CI's older clang-tidy does not apply, as in earlier records), none on a line this branch added, no `misc-include-cleaner` finding. The first draft's "32 findings, all identifier-length" described base `6935cf64`, where `tests/game/test_game_missiles.cpp` fails three include-cleaner errors (fixed upstream by `ab3bf0d3`, in current main): that finding is gone with the rebase, and the first draft's sentence was wrong for that base.

## Limits

- Brute: game.exe's move is FUN_005ded40 (path type 0xd, flags 7), d2d returns `walk` 7: the act-only differences stay (60,005 of 100,000). Zombie / Lancer `run` are RN moves in d2d and FUN_005ded40 moves in game.exe: the animation mode of those runs is U. Corrupt Rogue: its run is FUN_005defb0 (RN, flags 3); `n` stays out (n-only differences 31,029).
- Blood Hawk's `steps` is set only when the caller gives a pointer (the tests); `ai.cpp` gives none; what steps do in game.exe is U.
- The pace "type" (7 / 0xd) that the thinks also pass is not modelled by d2d (pre-existing, V read, verifier).
- Corpse reach covers a unique shaman (room search, FUN_005a0180 8 true / 4 false). **U: whether a superunique carries bit 8**; if it carries only bit 2, game.exe takes a superunique's corpse and d2d still does not. A non-unique shaman (group search type 9) was not run.
- The replay hooks the unit size at 2 and always sets off a walk (same assumptions as `mon_ais.py`'s corpus). The thinks are not re-run natively by the verifier (his check is a disassembly-derived model).
- Not in this PR: the Fallen command word (no Act 1 sender), Foul Crow Nest, the base velocity (PR B).
- Not run: an MSVC build; full ctest.

## Reviewer commands

```
# thinks (tools/emu; per AI group)
PYTHONPATH=tools/emu uv run python tools/emu/mon_ais_pace.py Zombie,Brute,BloodHawk 100000 20261102 --dump out/   # MAI2_CMD=none1 for the Fallen
# corpse
uv run python tools/emu/mon_shaman_corpse.py gen 100000 20261102 > cases.tsv
c++ -std=c++26 -Icomponents/rules -Icomponents/d2s -Icomponents/txt -include cmake/no_near_far.hpp -o harness tools/emu/mon_shaman_corpse_harness.cpp
harness < cases.tsv > d2d.tsv; uv run python tools/emu/mon_shaman_corpse.py compare cases.tsv d2d.tsv
```

(The counting replay `mp_run.sh` / `mp_patch.py` that produced the table lives in the evidence repo, not in the PR.)

## Order and independence

A and B both branch from upstream main; a throwaway merge of the two is clean (`b5f0028d`: no conflict, the two doc edits sit on different lines of `monster-ai.md` by construction), the suites pass on it, and the speed oracle (B) reads 0 mismatches on it. Merge order: **A first** (the verifier's advice: cheap to accept, local to seven Act 1 AIs and one shaman, correct under either base), then B.
