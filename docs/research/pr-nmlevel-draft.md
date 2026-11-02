# PR draft: a Nightmare or Hell monster starts at its area's MonLvl, not its own Level(N) / Level(H) (NMLEVEL)

> status: DRAFT held for the user's go (nothing pushed upstream, opened or posted). Candidate on the private fork remote only: branch `local/nmlevel-v5`, head `94562869`, on upstream main `3be426d3` (read-only fetch, 2026-11-02; it contains PRs #164 and #166): `19feb80d` (the rule), `94562869` (oracle and test mode; the tidy include for `monsters.hpp` folded in). The three includes of `tests/game/test_game_missiles.cpp` are already in upstream main (`54a28755`, `ab3bf0d3`), so no include commit is needed. Older branches kept as they were: `local/nmlevel` (`9672fc6c`, base `6935cf64`) and `local/nmlevel-v2` (`461afdcf`, the same plus the include commit); `local/nmlevel-v3` / `-v4` are intermediate (v4 has a broken rename). Upstream changes since `6935cf64` (61 files) that touch `ai.hpp` / `ai.cpp` / `test_monsters.cpp` (monster missile record, Imp Inferno end) do not overlap: both commits cherry-pick clean (one parameter rename in `ai.hpp` was needed, see Checks), no new `make_monster` / `spawn_monsters` caller, no change to `monster_stats` or MonStats level use. Independent verification: the Opus verifier's report, branch `agent/verifynmlevel` of the public repo, commit `96b50acf`, `docs/research/pr-nmlevel-verifier.md` (verdict READY WITH CHANGES; run on `9672fc6c`, base `6935cf64`; the changes are done here, the rule commit's code is unchanged by the rebase). Evidence: [levelspawn-trace.md](levelspawn-trace.md) D1 (branch agent/levelspawn) and [monster-stats-trace.md](monster-stats-trace.md) D1. Tags: V = run or read by us (how given), S = secondary, U = unverified.

Title (rule commit subject): `Nightmare and Hell monsters start at the area MonLvl, not their own Level(N) / Level(H)`

## What changes

game.exe's unit maker FUN_00573cb0 (V read, 0x573e15..0x573e70) takes a monster's level from the class MonStats `Level` / `Level(N)` / `Level(H)`. In an expansion game on Nightmare or Hell it takes the AREA's Levels.txt `MonLvl2Ex` / `MonLvl3Ex` instead (FUN_0061a1b0 gives the level id, FUN_0061dca0 the column), unless the class row is noRatio (MonStats flag byte +0xc & 4, mask 0x6ce270) or boss (& 0x40, mask 0x6ce280). Life, defence and experience then read the MonLvl row at that level (V read, FUN_006538a0 mask 7); to-hit and damage at stat 12 per hit (V read, FUN_005a4f50), which includes the mods' leveladd.

d2d read `Level(N)` / `Level(H)` for every monster (`rules::monster_stats`; the comment on `MonStats` said the area rule "waits for those difficulties"). The change adds `rules::start_level(type, difficulty, area_level)` (data driven: no_ratio and boss_column are columns of the row, no monster names) and an `area_level` parameter on `monster_stats` (-1: area unknown, the MonStats level as before). `make_monster` / `spawn_monsters` take the Levels.txt id (`Monster::level_id`), `make_boss` reads it, and every unit maker that knows its level passes it: the room population (`Fight::rooms_up`, the chest-risen spawn in `world/objects.cpp`), monster summons / clones inside `monster_update` (`level.id`), the worm of Impregnate, the Act 3 fetish pack, the Act 4 seal bosses and Diablo, Baal's stair copy. Unchanged on purpose: pets (`summons.cpp`, `pets.cpp`) and the co-op client's mirror of the host's monsters (`apps/d2d/town/net.cpp:443`), both pass no id. So a network client's copy still calls `make_monster` without an area and keeps `Level(N)` / `Level(H)` at creation (V read, the verifier); whether the host's replicated `stats.level` (`replication.hpp`) then overwrites it was read by the author (S for this PR) and is not re-checked.

## Proof

Differential (V run, the worker host): game.exe's own FUN_00573cb0, FUN_005a2120 (mods) and FUN_005a4f50 under the emulator against d2d's real `make_monster` / `make_boss` (`tests/game/test_boss_level.cpp --boss-level`), the existing `tools/emu/boss_level.py` extended: `gen-area N SEED` makes cases with a Levels.txt id and a new kind `p` (plain monster); in an area case FUN_0061a1b0 returns the case's id and FUN_0061dca0 runs for real (hooks otherwise as the file's docstring lists). Cases: every MonStats row alive in the difficulty x kind (plain, champion, unique, superunique, minion) with two ids drawn from the levels that have a MonLvl in that difficulty (plus ids 0 and count + 1, off the table: FUN_0061dca0 answers 1), then 100,000 random cases.

| build | seed | cases | Normal differ | Nightmare differ | Hell differ |
|---|---|---|---|---|---|
| base (upstream + test mode only) | 11 | 117,440 | 0 of 39,001 | 36,350 of 39,152 | 34,761 of 39,287 |
| base | 12 | 117,440 | 0 of 39,033 | 36,845 of 39,561 | 34,278 of 38,846 |
| head `7023cc73` (same code, on `6935cf64`) | 11 | 117,440 | 0 | 0 | 0 |
| head | 12 | 117,440 | 0 | 0 | 0 |

"Differ" = the level field, or a field that is not one of PR #157's named residuals. Level field mismatches: base 71,111 / 71,123, head 0 / 0. The named residuals (identical in kind and about the same count before and after; they are not this claim): life within 2 points (256ths against whole points), classes game.exe gives no life, S1 (stone skin, game.exe's defence twice d2d's) and H1 (the willowisp family's damage). Head seed 11: 29,091 of them (25,872 / 1,689 / 673 / 857), 0 outside the named classes; seed 12: 28,811, 0 outside.

Real populations, the author's sample (V run, 34 levels: this sample is named in every figure below): `tools/emu/ls_area.py` (card levelspawn: upstream's room population of each level under game.exe's code, a unit's class and level) over 34 levels of all acts the census covers, seeds 1 and 2, all three difficulties, against the same `test_boss_level` with the level the unit was made in (`tools/emu/nm_pop.py cases` / `compare`). Plain units (units of a unique's party, +3 level and x5 experience from the mods pass, and one level-0 class, are counted apart):

| difficulty | plain units | base: level differs | head: level differs | experience, mean per unit: game.exe / base / head |
|---|---|---|---|---|
| Normal | 7,731 | 0 | 0 | 130.6 / 130.6 / 130.6 |
| Nightmare | 7,360 | 2,166 (29.4 %) | 0 | 4,119.6 / 4,003.3 (-2.8 %) / 4,119.6 |
| Hell | 7,114 | 3,133 (44.0 %) | 0 | 37,437.5 / 35,065.9 (-6.3 %) / 37,437.5 |

Real populations, the verifier's sample (V run, `agent/verifynmlevel` commit `96b50acf`, its own harness, not our tools): every non-town level (2..136 without 1, 40, 75, 103, 109, minus the four crash levels below), two new seeds (0x1d2c3b4a, 0x00c0ffee), 88,951 monsters in all (start level of game.exe against d2d, per unit):

| seed | difficulty | units | start level differs on base | on head |
|---|---|---|---|---|
| 0x1d2c3b4a | Normal | 14,151 | 0 moved | 0 moved |
| 0x1d2c3b4a | Nightmare | 15,165 | 6,883 (45.4 %) | 0 |
| 0x1d2c3b4a | Hell | 15,864 | 7,681 (48.4 %) | 0 |
| 0x00c0ffee | Normal | 13,573 | 0 moved | 0 moved |
| 0x00c0ffee | Nightmare | 14,836 | 7,174 (48.4 %) | 0 |
| 0x00c0ffee | Hell | 15,362 | 7,832 (51.0 %) | 0 |

Head has 0 start-level mismatches for plain (every class), champion, unique, superunique and minion units; Normal moves nothing. The Normal rows show 17 and 20 mismatches on base and head alike: the level-0 clamp (below), not this change. Experience, plain units only (the verifier's probe compares like with like there), all levels: base is about -13.8 % / -14.2 % (Nightmare, seeds 1 / 2) and -3.6 % / -3.8 % (Hell) of game.exe's experience per unit, head 0 mismatches and 0.00 % in all six rows. The two samples agree in direction, not in size: our 34 levels give 29.4 % / 44.0 % of units at another level and -2.8 % / -6.3 % experience, all levels give 45 to 51 % and about -14 % / -3.7 %. Quote either only with its sample.

The classes most often at another level on base (class = MonStats row): Nightmare rows 20, 131, 3, 94, 74, 22; Hell rows 20, 19, 6, 0, 131, 94 (`nm_pop.py compare` prints the top six). Experience differs in 2,150 / 3,119 units on base and in 0 on head.

Unit test (`tests/monsters/test_monsters.cpp`, no game data): three hand-made rows (plain, noRatio, boss) x three difficulties x the leveladd of plain / champion / unique / superunique / minion taken from `boss_stats` (0 / 2 / 3 / 3 / 3): the level, the life / defence / experience row at the start level, to-hit and damage at the raised one, noRatio rows as written, Normal never reading the area, an unknown area (-1) changing nothing.

Single source (grep, V): every consumer of a monster's level reads `monster.stats.level` (XP penalty `kill_award`, drops `roll_drops`, poison / mana-burn / enchant rows in `make_boss`, `missile_blow`, auras, curses); the only other readers of `MonType::level` are `monster_ranges` (skill tooltips, the type's own level, unchanged) and the upgrade ladder `gamedata.cpp:377` (`level[0]`, Normal, unchanged). The co-op client takes the host's `stats.level` from the replicated unit.

Checks on the rebased candidate (V, run on the worker host, 2026-11-02; base `3be426d3`, head `94562869` = `local/nmlevel-v5`; builds `-j2` niced, Release, tests on):

- Suites `test_game`, `test_combat`, `test_rules`, `test_game_rules`, `test_monsters`, `test_monster_ai`, `test_monster_thinks`: exit 0 on base and on head (head re-run on `94562869`; the base run is `3be426d3`). `test_boss_level` skips without arguments (exit 77), as on base.
- `ctest` (69 tests): head 4 failed (`test_mpq` aborted, `smoke_d2d`, `quests_act1`, `quests_act4`) and 7 skipped; base fails the same 4 plus `pad_use_d2d` (a flake that passed on head); the 4 need game data this box does not have. Nothing new fails on head.
- `pr_guard` against `3be426d3`: 0 FAIL, 0 WARN. `presubmit --quick`: 0 FAIL, 4 WARN (answered below), ruff clean. (A run with `BUILD_DIR` exported fails `tools/test_tidy_pr_select.py`, 2 tests: a harness artefact, they pass without it.)
- Warnings: every changed .cpp (ai.cpp, tick.cpp, objects.cpp, act3/4/5.cpp, test_boss_level.cpp, test_monsters.cpp) with g++ and clang++ `-fsyntax-only` at the CI flags plus `-Wshadow -Wrange-loop-construct -Wconversion`: g++ 0 warnings on base and head; clang++ 234 on base and 234 on head, the same set (all `-Wsign-conversion` from `-Wconversion`, already in main). The first rebased head (`cb5265b9`) had 7 g++ `-Wshadow` warnings: the new `Monster::level_id` member against the parameter `level_id` of upstream's new `Monster::ally_not_interact`; the rule commit now renames that parameter to `in_level` (2 lines in `ai.hpp`) and the warning is gone.
- clang-tidy, the CI "Names and includes" selection (`tools/tidy-pr-select.py` over the head build: 88 translation units (the selection with the generated `header_check` TUs of my differently named build directory left out, which CI's `/build/` filter drops too), clang-tidy 23.1.4 (CI's is older): 0 findings in the 8 changed .cpp files, `ai.hpp` or `monsters.hpp`; 5 TUs clean, 83 TUs only carry 393 `readability-identifier-length` findings (structured binding names `x` / `y`) in files this PR does not touch (`outdoor.hpp` 328, `merc.hpp` 44, `pets.hpp`, `apps/d2d`, `world/setup.cpp`, `test_monster_ai.cpp`), upstream's and not flagged by CI's tidy. The first rebased head (`cb5265b9`) was not re-tidied separately; the old base run (`6935cf64`, verifier) found `test_game_missiles.cpp` missing three includes, now in main).
- Det state hashes, 14 scenarios (hidden slot, base = `3be426d3` + the lab verbs and a read-only `debug monstate`, head = base + the rule; each binary run twice): 13 scenarios identical in the monster hash and the state hash on all four runs (passive, fight, cold, static, cold2, hit_plain, hit_champion, hit_unique, hit_boss, hit_immune, hit_minion, merc, follow). `pet`: head and head2 equal base, but base2 (the same base binary, second run) differs from base in 7 of 9 snapshots, all in one line (a Spirit Wolf L30 at 44.1,21.1 against 44.3,21.0; level, life, damage, defence equal): a pre-existing position flap of the pet, not a head effect. Nothing legitimately changes: the scenarios spawn with `debug spawn` (no level id) or in Normal. The det binaries were built on `cd63f4f2`; the later parameter rename in `ai.hpp` (no behaviour) was not re-run through det, only through the suites above.
- The old-base runs on `6935cf64` (the author's and the verifier's) are not repeated; upstream's changes in between do not touch the monster level.
- Not run: an MSVC build, a live game, the CI Windows and macOS jobs.

## Not in this PR

- Monster regeneration (stat 0x4a = life256 x DamageRegen >> 12, added by FUN_00580610 / FUN_00580810; monster-stats trace D2): d2d has none. A missing feature with an unverified trigger and rate (U), no differential to fix it against: left out.
- Pets and Revive / summoned monsters from player skills: no level id passed (their stats come from the skill code); whether game.exe's pet maker applies the area rule to them is U.
- A unique's party members' leveladd (+3, x5) is PR #157's rule, untouched.
- Levels with MonLvl 0 and classes with MonStats level 0: d2d clamps the level to 1 (game.exe keeps 0); not reachable in the populations (counted apart above), unchanged.
- Classic (non-expansion) games: d2d runs expansion data only (the area column is the `Ex` one).

## Limits

- FUN_0061a1b0 (the unit's room to level id) is hooked in the oracle: the claim is the column and the exception test, not which level a room belongs to (d2d's level is the level it is placed in). U for monsters created by game.exe paths we did not run; V read: FUN_00573cb0 has one caller, the generic unit maker FUN_00574250, so summons and clones take the same rule (grep of the full disassembly).
- Drop quality and the XP penalty read `stats.level`, so they now follow the area level too; that game.exe reads stat 12 there was traced for the XP penalty only (S, `xp-award-trace.md`); drops: U.
- Presubmit WARNs, answered: *siblings, unit kinds*: pets (`summons.cpp`, `pets.cpp`) pass no level id and are unchanged (see above); mercs and co-op partners are players, not monster units; *data-gating*: `test_boss_level.cpp` already gates on `data->patched` (the hit is a comment line); *docs-sync*: no document describes the monster start level (grep of the .md files), no doc changes; *build*: answered by the ctest run above.
- Not covered by the rule (named by the verifier, V read): the network client's mirror of the host's monsters (`apps/d2d/town/net.cpp:443`, `make_monster` without a level id: it creates at `Level(N)` / `Level(H)`), and player pets / Revive (`fight/summons.cpp`, no area). In game.exe every monster unit goes through FUN_00573cb0 (one caller, FUN_00574250), so a revived Nightmare monster would start at the area level there (S; Revive itself unverified).
- Crash levels: levels 83 (Travincal), 110, 111 and 117 crash game.exe in the emulator (quest presets) on every difficulty, so the Council and those act 5 superuniques are untested against game.exe (U).
- Shared known difference (both trees, not this PR): game.exe stores a start level of 0 (town MonLvlEx, noRatio rows with Level 0) where d2d clamps to 1 (`std::max(..., 1)`); no monster spawns in a town. It is the 17 / 20 Normal-row mismatches above.
- Other FUN_0061dca0 users (outside this PR, U in d2d): trap-class monsters (FUN_005a3b50, rows 0x146..0x171) are set to the area level on every difficulty.
- Not run: an MSVC build, a live game, the CI Windows and macOS jobs.

## How a narrow verifier checks it

1. Read FUN_00573cb0 0x573e15..0x573e70 and FUN_0061dca0 with `tools/emu/disasm.py 573e15 40`, `disasm.py 61dca0`; confirm the masks 0x6ce270 / 0x6ce280 are 4 and 0x40 and that d2d's `no_ratio` / `boss_column` are those bits (`montypes.hpp`).
2. `uv run python tools/emu/boss_level.py gen-area 100000 11 > cases.tsv`, `... game cases.tsv > game.tsv`, `build/tests/test_boss_level --boss-level cases.tsv > d2d.tsv`, `... compare cases.tsv game.tsv d2d.tsv`: expect "level field mismatches 0" and "other fields not a named residual 0"; do the same on the base (the second commit's test file with `make_monster`'s last two arguments removed) and expect about 71,000 level mismatches.
3. `nm_pop.py` on `ls_area.py ... units` output as in `tools/emu/nm_pop.py`'s docstring, base and head.
4. Look for a unit maker that should pass a level id and does not (grep `make_monster(`), and for the claim that pets stay unchanged.

## Files

`components/rules/monsters.hpp`, `components/game/ai.hpp`, `ai.cpp`, `fight/tick.cpp`, `world/objects.cpp`, `world/quests/act3.cpp`, `act4.cpp`, `act5.cpp`; tests `tests/monsters/test_monsters.cpp`, `tests/game/test_boss_level.cpp`; tool `tools/emu/boss_level.py` (a tool change needs the maintainer's CODEOWNERS review).

## Shared-file lines for the merger

In `openspec/changes/nmlevel/merge-lines.txt`.
