# Independent verification: local/nmlevel (Nightmare / Hell monsters start at the area level)

Candidate: private fork branch local/nmlevel, head 9672fc6c (7023cc73 rule, 1682f73f oracle and test, 9672fc6c one
include), base upstream main 6935cf64. Upstream main is still 6935cf64 (read-only fetch, 2026-11-02), so the branch
sits directly on it: the merge is clean (V).

Harness: my own, in `docs/research/pr-nmlevel-verifier/` (the author's boss_level.py / nm_pop.py were not used).
game.exe side: `vnl_game.py` brings every non-town level (2..136 without 1, 40, 75, 103, 109) up for a seed with
upstream's population boot (monsters.new_game / act_of) and records, per monster unit FUN_00573cb0 makes, the MonStats
Level column (0x573e18), the start level it stores (eax at 0x573e6a), the final stat 0xc after the population (leveladd),
the row flags, the MonsterData type flags, and the first experience / life it writes. `vnl_grid.py`: the rule on every
MonStats row x 3 difficulties x level ids 0..140, with game.exe's own FUN_0061dca0. d2d side: `vnl_d2d.cpp`, the REAL
make_monster / make_boss built against base and head (my own clean builds of both trees, own worktrees). Seeds
0x1d2c3b4a and 0x00c0ffee (new). Tags: V = run or read here, S = inference, U = not checked.

## Verdict: READY WITH CHANGES

The rule is right and the change does what it says (V, counts below). Before the PR is opened:

1. **(high) Add the three direct includes to `tests/game/test_game_missiles.cpp`** (`<character_store.hpp>` for
   `new_character`, `<combat.hpp>` for `d2d::rules::Blow`, `<skills.hpp>` for `d2d::rules::kMonsterCurses`), or wait
   for upstream to fix it. The PR edits `components/game/ai.hpp` and `components/rules/monsters.hpp`; CI's "Names and
   includes" re-tidies every TU that includes a changed header (tools/tidy-pr-select.py), test_game_missiles.cpp
   includes both, and clang-tidy on it at head fails with exactly those three misc-include-cleaner errors (V, run).
   This is the same failure as the last PR.
2. **(medium) Say the sample in the PR text.** The author's 29.4 % / 44.0 % and -2.8 % / -6.3 % are for 34 levels; on
   every non-town level I count 45-48 % (NM) and 48-51 % (Hell) of units at another level on base, and the base XP gap
   on plain units is about -14 % (NM) and -3.7 % (Hell). Direction agrees, magnitudes depend on the level set (V).
3. **(low) Name the uncovered callers in the PR text**: `apps/d2d/town/net.cpp:443` (a network client's mirror of the
   host's monsters) calls make_monster without a level id, so it keeps Level(N)/(H); player pets / Revive
   (`fight/summons.cpp:152`) also take no area. In game.exe every monster unit goes through FUN_00573cb0 (its one
   caller is FUN_00574250, V), so a revived NM monster starts at the area level there (S; Revive is unverified).

## 1. The rule (V, disassembly and run)

- FUN_00573cb0 0x573e15..0x573e53: level = MonStats row word +0xaa + 2 x diff; if the game is expansion ([game+0x70]),
  diff > 0 and byte +0xc has neither [0x6ce270] nor [0x6ce280], level = FUN_0061dca0(FUN_0061a1b0(room), diff, 1).
  FUN_0061a1b0 is the room's level id; FUN_0061dca0 returns Levels +0x16 + 2 x diff (MonLvlEx), or 1 for diff >= 3,
  id <= 0 or id past the table. Masks read from the image: 0x6ce270 = 4, 0x6ce280 = 0x40 (V).
- Byte +0xc bit 2 / bit 6 equal MonStats.txt noRatio / boss for all 734 rows: 0 mismatches (V, vnl_grid.py), so d2d's
  `no_ratio` / `boss_column` are the right gates.
- Grid, 310,482 cases (734 rows x 3 diffs x ids 0..140): head 0 start-level differences; base 86,537 (NM) and 83,241
  (Hell) on ratio rows in real levels, plus the edges (id 0, past the table, towns). Normal, noRatio and boss rows: 0 on
  both. Only remaining gap on both trees: game.exe stores a level of 0 (towns' MonLvlEx, noRatio rows with Level 0)
  where d2d's monster_stats clamps to 1 (pre-existing, `std::max(..., 1)`; no monster spawns there).

## 2. Real populations, base vs head (V, run)

Units / units whose game start level differs from the MonStats column / start-level mismatches base / head:

| seed | diff | units | moved | base | head |
|---|---|---|---|---|---|
| 0x1d2c3b4a | Normal | 14,151 | 0 | 17 | 17 |
| 0x1d2c3b4a | NM | 15,165 | 6,883 | 6,883 | 0 |
| 0x1d2c3b4a | Hell | 15,864 | 7,681 | 7,681 | 0 |
| 0x00c0ffee | Normal | 13,573 | 0 | 20 | 20 |
| 0x00c0ffee | NM | 14,836 | 7,174 | 7,174 | 0 |
| 0x00c0ffee | Hell | 15,362 | 7,832 | 7,832 | 0 |

- Per class (both seeds, NM+Hell), head has 0 start mismatches for plain, champion (828 units), unique (1,544),
  superunique (140), minion (7,384); base misses exactly the moved ones. Boss-column rows (act bosses etc., 122 units)
  never move in game.exe and match on both trees. No noRatio row appears in a real population; that branch is covered
  by the grid only.
- The Normal 17 / 20 are the level-0 clamp above, identical on both trees.
- Final level (after leveladd): head leaves 24 / 45 (seed 1 NM / Hell) and 61 / 104 (seed 2) mismatches, all on units
  game.exe raises by +3 without a champion / unique / minion type flag (minions my probe makes as plain): a probe limit,
  not the PR; base has the same units on top of its start mismatches.
- Not covered: levels 83 (Travincal), 110, 111, 117 crash game.exe in the emulator (fatal error from 0x544784 /
  0x5444cf, quest presets) on every difficulty: the Council and those act 5 superuniques are untested (U).

## 3. Callers and readers (V, read)

- Every make_monster / spawn_monsters call in components passes a level id on head (ai.cpp 11 sites, fight/tick.cpp 2,
  world/objects.cpp, quests act3 / act4 / act5), and make_boss reads `monster.level_id` (also the warping shrine, which
  re-makes an existing unit). Not passed: apps/d2d/town/net.cpp:443 (finding 3), the lab-only `debug spawn` verb, player
  pets in fight/summons.cpp (level overwritten afterwards; life from raw columns, Revive excepted).
- Readers of the old level: the only other `type_info.level[...]` read is `monster_ranges` (skill tooltip summon lines,
  own level by design) and gamedata.cpp:377 (Normal column for class picking). Everything else reads `stats.level` /
  `stats.exp` (drops' ilvl in deaths.cpp, conversion in strike.cpp / states.cpp), so they follow the fix. Monster skill
  levels in game.exe use a difficulty bonus from FUN_00573930, not the start level (V, 0x5740f9); MonEquip
  (FUN_005d6b60) does take the start level (cosmetic, U in d2d).
- Other FUN_0061dca0 users: FUN_005a3b50 sets trap-class monsters (rows 0x146..0x171) to the area level on every
  difficulty plus stat 0x13 = level + 50 (max 90): outside this PR, worth its own check (U in d2d).

## 4. Experience (V, run; plain units, where my probe compares like with like)

| seed | diff | plain units | base exp mismatches | base mean vs game | head |
|---|---|---|---|---|---|
| 0x1d2c3b4a | Normal | 13,308 | 0 | 0.00 % | 0 / 0.00 % |
| 0x1d2c3b4a | NM | 13,185 | 5,937 | -13.80 % | 0 / 0.00 % |
| 0x1d2c3b4a | Hell | 12,782 | 6,158 | -3.56 % | 0 / 0.00 % |
| 0x00c0ffee | Normal | 12,729 | 0 | 0.00 % | 0 / 0.00 % |
| 0x00c0ffee | NM | 12,962 | 6,205 | -14.18 % | 0 / 0.00 % |
| 0x00c0ffee | Hell | 12,378 | 6,171 | -3.83 % | 0 / 0.00 % |

Boss kinds are left out: my hook takes game.exe's first stat 0xd write (before the champion / unique multiplier), d2d's
make_boss value is after it (x3 / x5), so they differ on both trees alike.

## 5. Not run (why)

- Det scenarios base vs head: NOT RUN. My lab builds were stopped at the coordinator's wrap-up call (d2d.base built,
  sha256 5c1d5a40...; d2d.head not finished). Runner ready: `vnl_det.sh`.
- g++ / clang++ warnings with the CI flags plus -Wshadow -Wrange-loop-construct -Wconversion: NOT RUN (`vnl_ci.sh
  warn` ready). By reading the diff: no narrowing (every size_t cast explicit), no new shadowing names in
  test_monsters.cpp's block (checked against main's scope), designated initializers as elsewhere in the file; no MSVC
  /W4 hazard seen (S).
- Full "Names and includes" over every TU including ai.hpp / monsters.hpp: NOT RUN (hundreds of TUs at one core).
  Run: test_game_missiles.cpp FAIL (finding 1), test_monsters.cpp OK, test_boss_level.cpp only the local clang-tidy 23
  binding-name noise in components/drlg/outdoor.hpp (x / y, also on base; CI's older clang-tidy does not flag it, S).
- Suites (test_game, test_combat, test_rules, test_game_rules, test_monsters, test_monster_ai, test_monster_thinks),
  ctest, pr_guard, presubmit --quick: NOT RUN (both trees built and ready; `vnl_ci.sh tests`).

## 6. The author's claims (brief summary; draft not otherwise used)

- Rule, masks, FUN_00573cb0 / FUN_0061dca0: agree (V).
- Normal unchanged: agree, 0 moved on 27,724 units (V).
- Head 0 level mismatches outside named residuals: agree for the start level on 88,951 units and the full grid (V).
- 29.4 % NM / 44.0 % Hell, XP -2.8 % / -6.3 %: not reproduced as numbers (different level set); my all-level figures are
  45-48 % / 48-51 % and -14 % / -3.7 % (plain units). Not an error, but the PR text must name its sample (finding 2).
- Det 13 of 14 identical: not checked (U).
- Left out pets / summons / regeneration: agree they are left out; regeneration (stat 0x4a) is computed from the life
  FUN_00573cb0 just set (V, 0x573ff6..0x574058), so it follows the level through life.
