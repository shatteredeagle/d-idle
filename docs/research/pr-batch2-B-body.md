# A monster's hit is rolled in 256ths, absorbed, then reduced

> status: FINAL BODY for batch B, not posted (card oracleport, 2026-11-03). Open it only after batch A (pr-batch2-A-body.md) has merged: both touch `fight.hpp` and `tick.cpp` in other hunks and `structs.tsv` in the same table, so rebase then. Branch `local/batch2-B-r2` on the private fork remote, head `0399e357`, 1 commit on upstream main `e1eea02a`. It replaces `local/batch2-B-r1` (head `2090dc15`, kept; r2 only makes the oracle's `compare` skip log lines). Tags: V = run for this head (how stated), S = secondary (an earlier run of ours with a harness that is not in this PR), U = unverified.

## What changes

d2d rolled a monster's hit in whole points, inclusive. game.exe rolls `(min << 8) + rand((max - min) << 8)` on the attacker's seed (min taken at least 1 and max at least min + 1, so `0..0` rolls 1..2 and draws once; the top is not rolled), raises the absorb event on the unreduced physical (Bone Armor), then reduces the record, the flat first and then the percent, and the life stat takes it in 256ths. Records built without the roll (Corpse Explosion, Diablo's cold touch: min equal to max) keep their value, draw nothing, and keep the reduction and the absorbs.

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `0057b420` | the roll in 256ths, min at least 1, max at least min + 1 |
| `0057bf80` | the reduction, flat then percent |
| `0057c1e0` | the record: absorb event on the unreduced physical, then the reduction; the life stat takes 256ths |
| `0057c852` | a record total that is not above 0 is ignored |
| `005c6370` | Bone Armor's absorb handler |

## Claims

| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| `0399e357` | Roll in 256ths, absorb on the unreduced physical, then reduce; records without a roll keep min and draw nothing | `0057b420`, `0057c1e0`, `0057bf80`, `005c6370` | the `monster_damage.py` run below: 560,000 frames in six sets, 0 differ in life, pool and seed after |

The oracle is `tools/emu/monster_damage.py`: game.exe's `FUN_0057b420` and `FUN_0057c1e0` natively on a defender unit, with Bone Armor's state through the real dispatcher (V: the run; the stat getters and state list are hooked, listed in its docstring). d2d's side is `test_combat --monster-damage` (`tests/test_combat.cpp`): the real `rules::monster_blow` and `rules::absorb_blows` built as `Foe::take` and `Fight::absorb` call them.

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure
cd tools/emu && uv sync
python3 -m unittest test_monster_damage.py        # the game-backed tests run when D2_MPQ_DIR is set
set -o pipefail
uv run python monster_damage.py gen 20000 20261103 > cases.tsv
uv run python monster_damage.py game cases.tsv > g.txt
../../build/tests/test_combat --monster-damage cases.tsv > d.txt
uv run python monster_damage.py compare cases.tsv g.txt d.txt   # each "# kind:" line ends "seed differ 0"
```

Counts of this head (V, the Windows build worker, 1.14d data; 8 Act 1 classes, game side about 4 minutes):

| set | frames | life / pool / seed differ |
|---|---|---|
| roll (no reduction, no pool) | 160,000 | 0 / 0 / 0 |
| reduce (percent 0..50, flat) | 160,000 | 0 / 0 / 0 |
| absorb (1 to 3 blows into a Bone Armor pool) | 160,000 | 0 / 0 / 0 |
| both (a reduction and a pool) | 40,000 | 0 / 0 / 0 |
| negative percent (Andariel) | 10,000 | 0 / 0 / 0 |
| exact (no roll, min equal to max) | 30,000 | 0 / 0 / 0 |

Before the change (S: the earlier harness `mdmg2.py` with the same game side and the base's header functions, seed 20261103, 20,000 per class): roll 159,654 of 160,000 differ in life, reduce 124,491, absorb 102,020 life and 68,173 pool, both 8,332 and 33,810. That harness is not in this PR; the base check here would need base's headers, which lack `Taken::raw` and `exact`, so it is not offered as a command.

## Not done

- Energy Shield is NOT equal to game.exe and this PR does not claim it: it runs after the armors on the reduced total here, while game.exe takes it per record field on the unreduced value and runs the absorb state cast last first (`FUN_005c0ad0`). The change carries the shield's mana and cut in 256ths; with the shield up the earlier out-of-tree harness (S) showed differences from game.exe both before and after the change, and in a shield-first order one pair of quantities (Bone Armor with a later Energy Shield) was slightly worse than before. The oracle in this PR does not cover the shield at all; the `ponytail:` note in `states.cpp` names it.
- Also not covered (U): Cyclone Armor (its fields are whole points), the flat's `+0x54 / 1024` scale and `FUN_0057c060`'s percent, elements (resisted before the absorb in d2d, after in game.exe), a missile's `FUN_005a89a0` roll (shares the roll, bounds as they stand), a merc or a pet as the defender (carried through `pet_hurt` and `life_frac`, unit tests only), and the real `Fight::absorb` (the oracle's d2d side calls the rules functions it calls, not the member itself; an earlier harness found them equal, S).
- The clamp (min at least 1, max at least min + 1) changes no live melee row in 1.14d's MonStats for Acts 1 to 3 (S: a census, not in this PR).
- Presubmit WARNs: siblings (unit kinds, player and co-op partner not named in the diff): a player and a co-op partner take the same `hurt_side` path (U, not exercised separately); siblings (hit paths): `monster_blow` serves melee, missile and skills alike; build: skipped by `--quick`, see Checks.
- A live game, a soak and a determinism hash comparison were not run (recorded hashes change after the first monster blow).

## Checks (V unless said)

- One commit, subject 57 characters, body three lines of at most 72, last line `Agent: Claude Code`; author and committer the numeric noreply identity; no co-author or AI lines.
- `python3 tools/pr_guard.py upstream/main HEAD`: 0 FAIL, 0 WARN. `python3 tools/presubmit.py --quick`: 0 FAIL, 4 WARN (above). On the build worker with a CMake build dir, GCC (-Wshadow -Wconversion -Werror) and clang-tidy: 156 of 157 TUs clean; the launcher TU needs Qt's `main.moc`, which the partial build does not make (environment, not this diff). `g++ -fsyntax-only` with `-Wall -Wextra -Wpedantic -Wshadow -Wconversion` on `tests/test_combat.cpp`: 0 warnings.
- `test_combat` without arguments: ok. Python: ruff with the presubmit's rule set, 0 findings; `test_monster_damage.py` 10 tests, all ok with game data.
- The full suites and MSVC were run for the earlier head of the same code (S, `pr-batch2-body.md`); not repeated for this head.

### CodeRabbit budget

| quantity | this PR | budget (target / ceiling) |
|---|---|---|
| files | 14 | 40 / 70 (cap 100) |
| added + deleted lines | 789 (738 + 51) | 1,000 / about 1,500 |
| commits | 1 | 15 / 40 |

## Checklist

- [x] A test in `tests/` (fix or port)
- [x] The oracle in `tools/emu`, its test, a README row, the finding in `docs/research/re/combat.md`, `structs.tsv` rows for the new fields
- [x] `python3 tools/pr_guard.py` passes

---

Agent: Claude Code
