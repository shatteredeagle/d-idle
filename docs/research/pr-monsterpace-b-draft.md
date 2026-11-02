# PR B draft: a monster starts at 75 percent velocity, not 100 (MONSTERPACE)

> status: DRAFT, ready for the PR-open step on the user's go (nothing pushed upstream, opened or posted). Candidate on the private fork remote only: `local/monsterpace-b-velocity`, head `8a77a4ab`, 2 commits on current upstream main `3be426d3` (read-only fetch 2026-11-02; it contains #164 and #166). Independent of [pr-monsterpace-a-draft.md](pr-monsterpace-a-draft.md) (the paces); merge order A first, then B (reason below). Replaces the single draft `pr-monsterpace-draft.md` after the independent verifier's report (`docs/research/pr-monsterpace-verifier.md` on branch agent/verifymonsterpace, READY WITH CHANGES, section 8 recommends this split). Evidence: [walk-speed-trace.md](walk-speed-trace.md), [monsterai2-trace.md](monsterai2-trace.md). Format follows [pr-missilerecord-draft.md](pr-missilerecord-draft.md). Tags: V run, V read (disassembly read), S source, U not traced.

Title: `A monster starts at 75 percent velocity, not 100 (FUN_00573cb0 SetStat(0x43, 0x4b))`

## What changes (one claim)

game.exe's monster stat initialiser FUN_00573cb0 ends its stat block with `SetStat(unit, 0x43, 0x4b)` (0x573f14..0x573f1b: `push 0; push 0x4b; push 0x43; push esi; call 0x627260`, unconditional once the unit-type / row guards at the top pass; V read, the verifier's and mine). The walk percent is that value plus chill, pace, aura and Spider Lay, at least 25 (FUN_00623f50, 0x62413e..0x6241c9; V read); velocity is FUN_00621360(row) x walk % / 100. d2d summed from 100, so a plain walk was 4/3 too fast and a pace of 100 gave 200 percent where game.exe gives 175. `ai.cpp` now sums from `kMonsterVelocityPct` (75, `ai.hpp`). Commits `f54c4f5c` (code, `tools/emu/walk_velocity.py`'s monster base taken from game.exe's own initialiser) and `8a77a4ab` (docs/research/re/monster-ai.md). One author identity.

Nothing in the think path resets the base: SetStat 0x43 writers in all of `.text` (V read, verifier): 100 in the client and player initialisers, 75 in the client's monster initialiser (0x4ae99c) and this one (0x573f18), the armour speed penalty, the chill list set (0x57b0a6), the pace (0x5a63cb, a separate stat list: it adds), the run (0x620ee1), AddStat for boss mods and the client. So a pace ADDS to the 75 (100 gives 175 percent).

## What gets stronger, what stays (the reach, said plainly)

- **Every awake monster that walks is 25 percent slower** (75 / 100); every det scenario with one differs for that reason (below).
- **Chill and slows get relatively stronger.** coldeffect, Spider Lay's -100, aura velocitypercent, boss_speed (fast +20 / fanatic) are all additive in game.exe and in d2d, so the same numbers weigh more against 75: a plain walker with coldeffect -50 goes from 50 percent to the **25 percent floor** (FUN_00623f50's minimum), one third of its unchilled speed instead of one half. That is game.exe's arithmetic (V read); it is the most visible gameplay effect.
- **Diablo's run and the whirl step stay at base 100** in d2d (`ai.cpp` lines 1939 and 1993 still use `100 + speed_pct + boss_speed`). FUN_00623f50 is the only path-speed setter read and reads the summed stat 0x43, so if those moves go through it they are 75-based in game.exe too (U, not traced). After this PR the code holds two monster bases; the PR text says so.
- **Mercs and pets are unchanged in d2d** (100: `pets.cpp`, `summons.cpp`, `merc.cpp`). That is "d2d unchanged", not "d2d right": hirelings and summons are type-1 units created through FUN_00555230 -> FUN_00574250 -> FUN_00573cb0 and the only later SetStat(0x43, 100) is in player initialisers, so in game.exe a merc and a pet probably also start at 75 (I, not traced; not this PR's change).
- Players: `World::walk_percent` (100) untouched, a different stat block. `apps/d2d/netgame.cpp` takes 0x43 from host packets: unaffected. XP, difficulty, damage, drops, AI decisions: no reader changed (diff read).

## Proof (V run, worker host, seed 20261102)

`tools/emu/walk_velocity.py` (PR #156's oracle, extended): the game side runs FUN_00573cb0 (hooked: FUN_005739d0, the random draws FUN_0045c390, the level block FUN_00573930, FUN_0063ee90, FUN_00626d40; SetStat FUN_00627260 recorded) for 26 MonStats rows x 3 difficulties and takes stat 0x43 from it (prints `monster base stat 0x43 (FUN_00573cb0): 75`), instead of the 100 the oracle assumed; d2d is the real build (`test_walk_velocity --walk-velocity`: `monster_update` one frame, `World::walk_percent`, `Fight::merc_turn`, `Fight::pet_walking`). The verifier ran FUN_00573cb0's stat block natively (`base43.py`): stat 0x43 set exactly once, to 75, in 6 of 6 runs; the row cannot move it.

| build | cases | compared | mismatches (monster rows) | merc / pet / player |
|---|---|---|---|---|
| upstream main `3be426d3` | 122,646 | 122,646 | 50,626 of 61,654 (RN 25,321 of 30,868; WL 25,305 of 30,786) | 0 / 0 / 0 |
| head B | 122,646 | 122,646 | 0 | 0 / 0 / 0 |
| head A+B (throwaway merge `b5f0028d`) | 122,646 | 122,646 | 0 | 0 / 0 / 0 |
| head A alone (paces only) | 122,646 | 122,646 | 50,626 | 0 / 0 / 0 |

(The A row shows the speed oracle is blind to the paces and the two PRs are independent. The base run's own tally says 46,914 of the 50,626 are the velocity x pct / 100 truncation note (game.exe truncates, d2d's float step does not); head compares at 0 either way.)

Suites (V run, Release build of each head from `build/tests` with the MPQ folder set): `test_monster_thinks`, `test_monster_ai`, `test_monsters`, `test_rules`, `test_game_rules`, `test_combat`, `test_game` all exit 0 on base, A, B and A+B.

Det scenarios (V run, one hidden d2d slot, no game.exe, seed 3, lockstep; lab binaries = the lab patch set (8 commits, `local/monsterpace-lab-base`) cherry-picked onto `3be426d3` with no conflict, then B (`local/monsterpace-lab-b`); 15 scenarios): **`passive` is identical** (mon `b6db27442898`, state `94d8c03bf4d3`: no monster walks); **the other 14 differ**, e.g. `fight` `1124443e0e4e` / `9f5bc7695b85` -> `5d0c86ece7ba` / `a63bd2386695`, `hit_plain` `e674e11e705e` / `92986553415f` -> `8e91f448fe3b` / `f9c76c0d16a8`. Why (V run, snapshot-line diff): in 13 the first differing line (frame counters left out) is a monster position, in `pet` the wolf's own position and the monsters' in the same snapshot; the chasers' trajectories are not a plain 75 percent scaling (the chase re-paths and the frame counter moves), so a hash is explained by "the walkers moved differently" and the rest follows: chill / stun timing, missile targets, a few hp values from blows landing at other moments; `merc` also differs (positions, the merc's own sums unchanged). Only `passive` is claimed unchanged. The base is deterministic (`base2` reruns of passive, fight and pet repeat the base hashes).

## CI parity (V run)

g++ 16.2 and clang++ 23.1 with the TUs' own compile commands plus `-Wall -Wextra -Wpedantic -Wno-missing-field-initializers -Werror`, `-Wshadow -Wrange-loop-construct -Wconversion -Wsign-conversion` as warnings, over `components/game/ai.cpp` and the two monster tests (they include `ai.hpp`): rc 0, the same warning set as base (29 / 33), 0 new. MSVC /W4 (read): the diff is one `int` constant and one expression (`std::max` of ints, no narrowing, nothing shadowed); no windows.h macro name as an identifier.

clang-tidy (LLVM 23.1.4, `-p build`, niced, one TU at a time): `tools/tidy-pr-select.py`'s rule for `ai.cpp` and `ai.hpp` (3 TUs new on upstream had no dependency record and got theirs from `-MM`): **111 TUs, all run**; 13 unique findings, all `readability-identifier-length` (LLVM 23's check, not applied by CI's older clang-tidy), none on an added line, no `misc-include-cleaner` finding. The first draft's "32 findings, all identifier-length" was wrong for base `6935cf64`, where `tests/game/test_game_missiles.cpp` failed three include-cleaner errors that current main already fixes (`ab3bf0d3`); with the rebase that finding is gone.

## Overlap with upstream moves (V read)

`6935cf64..3be426d3` changed `ai.cpp` / `ai.hpp` (monster missile record and poison #164, get-hit, swaps_ai, Imp Inferno, the ring-shot rename); none touches `monster_update`'s walk sum, `pace` or `chill`. Both commits cherry-picked without a conflict.

## Limits

- Diablo's run and the whirl step: U (above). Mercs and pets in game.exe: I (above).
- The oracle takes the initialiser's value from game.exe's code with its helpers hooked; the MonStats row's fields it reads (resists, level) do not reach stat 0x43 (verifier, 6 of 6).
- The path-speed oracle over MonStats rows (196 of 196 formula check) was the author's, not re-run by the verifier; the formula is read (V read) and d2d's expression is the same with 75 for 100 (S).
- Not run: an MSVC build; full ctest.

## Order

B is independent of A (both branch from upstream main, the throwaway merge is clean, suites and the oracle pass on it). Merge A first: it is local to seven Act 1 AIs and one shaman and is right under either base; B changes the feel of the whole game and every det hash, and may be judged or reverted alone.

## Reviewer commands

```
uv run python tools/emu/walk_velocity.py gen 100000 20261102 > cases.tsv
uv run python tools/emu/walk_velocity.py game cases.tsv > game.tsv          # prints: monster base stat 0x43 (FUN_00573cb0): 75
build/tests/test_walk_velocity --walk-velocity cases.tsv | grep -E '^[0-9]+ ' > d2d.tsv
uv run python tools/emu/walk_velocity.py compare cases.tsv game.tsv d2d.tsv
```
