# PR body: batch B (a monster's hit rolled in 256ths, absorbed, then reduced)

> status: DRAFT body for the batch B pull request to psi29a/d2decomped (card prbprep, 2026-11-04). Local branch `local/batch2-B-r5`, head `7e1f8fc4`, 1 commit on upstream main `002ffab4` (after PR 187). NOT pushed to the fork remote and NOT posted: until the push, the hash is local only. Replaces `pr-batch2-B-body-r3.md` (head `7e1f8fc4`, base `8c9a92c7`). The rebase was a clean cherry-pick (no conflict; `fight.hpp`, `tick.cpp` and `tools/emu/README.md` merged without edits). Tags: V = run on this head (how stated), S = secondary or carried from the earlier head, U = unverified.

---

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
| `7e1f8fc4` | Roll in 256ths, absorb on the unreduced physical, then reduce; records without a roll keep min and draw nothing | `0057b420`, `0057c1e0`, `0057bf80`, `005c6370` | `monster_damage.py`, 560,000 frames in six sets, 0 differ in life, pool and seed after (table below) |

The oracle is `tools/emu/monster_damage.py`: game.exe's `FUN_0057b420` and `FUN_0057c1e0` natively on a defender unit, with Bone Armor's state through the real dispatcher (the stat getters and state list are hooked, listed in its docstring). d2d's side is `test_combat --monster-damage` (`tests/test_combat.cpp`): the real `rules::monster_blow` and `rules::absorb_blows` built as `Foe::take` and `Fight::absorb` call them.

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

Counts below were measured on the earlier head `17ec4a23` (same patch on base `8c9a92c7`; S for this head: NOT re-run after the rebase onto `002ffab4`) (the worker host, g++ Release build with the CI flags, oracle native, 1.14d data; 8 Act 1 classes; game side 4 min 32 s):

| set | frames | life / pool / seed differ |
|---|---|---|
| roll (no reduction, no pool) | 160,000 | 0 / 0 / 0 |
| reduce (percent 0..50, flat) | 160,000 | 0 / 0 / 0 |
| absorb (1 to 3 blows into a Bone Armor pool) | 160,000 | 0 / 0 / 0 |
| both (a reduction and a pool) | 40,000 | 0 / 0 / 0 |
| negative percent (Andariel) | 10,000 | 0 / 0 / 0 |
| exact (no roll, min equal to max) | 30,000 | 0 / 0 / 0 |

`test_combat` without an argument: rc 0 (`ok`). `python3 -m unittest test_monster_damage` with `D2_MPQ_DIR` set: 10 tests, OK, none skipped (so the 5 game.exe-backed tests ran).

Before the change (S: an earlier out-of-tree harness with the same game side and the base's header functions, seed 20261103, 20,000 per class, not re-run for this head): roll 159,654 of 160,000 differ in life, reduce 124,491, absorb 102,020 life and 68,173 pool, both 8,332 and 33,810. That harness is not in this PR; base's headers lack `Taken::raw` and `exact`, so the base check is not offered as a command.

## Not done

- Energy Shield is NOT equal to game.exe and this PR does not claim it: it runs after the armors on the reduced total here, while game.exe takes it per record field on the unreduced value and runs the absorb state cast last first (`FUN_005c0ad0`). The change carries the shield's mana and cut in 256ths; with the shield up an earlier out-of-tree harness (S) showed differences from game.exe both before and after, and in a shield-first order one pair (Bone Armor with a later Energy Shield) was slightly worse than before. The oracle does not cover the shield; the `ponytail:` note in `states.cpp` names it.
- Also not covered (U): Cyclone Armor (its fields are whole points), the flat's `+0x54 / 1024` scale and `FUN_0057c060`'s percent, elements (resisted before the absorb in d2d, after in game.exe), a missile's `FUN_005a89a0` roll (shares the roll, bounds as they stand), a merc or a pet as the defender (carried through `pet_hurt` and `life_frac`, unit tests only), and the real `Fight::absorb` (the oracle's d2d side calls the rules functions it calls, not the member itself; an earlier harness found them equal, S).
- The clamp (min at least 1, max at least min + 1) changes no live melee row in 1.14d's MonStats for Acts 1 to 3 (S: a census, not in this PR).
- Presubmit WARNs: siblings (unit kinds): a player and a co-op partner take the same `hurt_side` path (U, not exercised separately); siblings (hit paths): `monster_blow` serves melee, missile and skills alike; build and gcc: skipped by `--quick`, covered by the lane below.
- A live game, a soak and a determinism hash comparison were not run (recorded hashes change after the first monster blow). No MSVC run for this head.

## Checks (V unless said)

- One commit, subject 58 characters, body three lines of at most 72, last line `Agent: Claude Code`. Author and committer are the numeric noreply identity (V: `git log -1 --format=%an%n%ce`). The old committer open item is closed.
- Size: 14 files, +738 / -51 (V: `git diff --shortstat 002ffab4..HEAD`).
- `python3 tools/pr_guard.py 002ffab4 HEAD`: 0 FAIL, 0 WARN (V; one commit, so the per-commit run is the same).
- `python3 tools/presubmit.py --quick --base 002ffab4` with `D2_BUILD` pointing at a build that has `mpq-cat`: 0 FAIL, 4 WARN (V). The WARNs: siblings (unit kinds): a player and a co-op partner take the same `hurt_side` path (U, not exercised separately); siblings (hit paths): `monster_blow` serves melee, missile and skills alike; build and gcc: skipped by `--quick`. The earlier ruff finding (`B905` at `tools/emu/monster_damage.py:188`) does not appear with the ruff presubmit ran this time (ruff: 0 on added lines); the one-word `strict=True` fix is not applied (U: version difference not investigated).
- Release build (V: the worker host, Ninja, `-DCMAKE_BUILD_TYPE=Release`, targets `test_rules`, `test_game_rules`, `test_monsters`, `test_monster_thinks`, `test_monster_ai`, `test_combat`, `test_game`, `test_sk_ground`, `mpq-cat`): rc 0. Run: `test_rules`, `test_game_rules`, `test_monsters`, `test_monster_thinks`, `test_monster_ai`, `test_combat` rc 0. `test_game` and `test_sk_ground` returned 77 (skipped: no game data on that run), so the new `tests/game/test_game.cpp` cases did not run.
- Tooling tests (V): `python3 -m unittest test_monster_damage.py` OK with 5 of 10 skipped (no `D2_MPQ_DIR`, so the game.exe-backed tests did not run); `test_pr_guard.py` and the other tool tests under presubmit OK.

## Not re-run on this head

- The 560,000-frame game.exe oracle comparison (counts above, earlier head), the 5 game-backed unit tests, `test_game` and `test_sk_ground` with data.
- The parity lane (g++ and clang++ with the CI flags and `-Werror`, extra warnings, clang-tidy) and the `g-all` stage: carried from the earlier head (S). The rebase touched none of this patch's lines.
- No live game, soak or determinism hash comparison, no MSVC run.

## Checklist

- [x] A test in `tests/` (fix or port)
- [x] The oracle in `tools/emu`, its test, a README row, the finding in `docs/research/re/combat.md`, `structs.tsv` rows for the new fields
- [x] `python3 tools/pr_guard.py` passes

---

Agent: Claude Code
