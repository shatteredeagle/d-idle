# Fidelity fixes against game.exe 1.14d: skill mana check and payment, Teleport target rules

> status: FINAL BODY for batch A, not posted (card oracleport, 2026-11-03). Nothing is pushed to the public fork and no PR is open: the coordinator opens it after the user's go. Branch `local/batch2-A-r5` on the private fork remote, head `41afdc6a`, 3 commits on upstream main `e1eea02a` (the merge of PR 181). It replaces `local/batch2-on181-r2` and `local/batch2-A-r4` (both kept: r4 has a `diff` command that miscounts when d2d's log lines are in its input; r5 fixes it). Batch B (monster damage roll) is a separate later PR. Tags: V = run for this head (how stated), S = secondary, U = unverified.

## What changes

d2d refused a cast when the mana was under the skill's `minmana` floor and paid the floor. game.exe checks the formula `(mana + lvlmana x (level - 1)) << manashift` alone and pays `max(formula, minmana x 256)` only when the mana covers it. d2d also let Teleport land anywhere and refused a blocked target; game.exe lands only in the caster's room or its near list, and puts a blocked target on the nearest free spot (square rings 1 to 49).

## Evidence from game.exe 1.14d

| Address | What it shows |
|---|---|
| `00463740` | the target's room in the caster's room or its near list |
| `00554ea0` | the Teleport warp: the target room, then the spot search |
| `0056bfe0` | the payment: `max(formula, minmana x 256)`, nothing when short or when mana and lvlmana are 0 |
| `0056c160` | the mana check: the formula only, no minmana floor |
| `005ca360` | srvdofunc 27, Teleport, which calls the warp |
| `0064d9b0` | the collision test with the player's footprint |
| `0064dea0`, `0064e7b0` | the spot search: the subtile itself if free, else the first free square ring, least abs(dx) + abs(dy) |

## Claims

Each commit carries its test and its oracle in `tools/emu` (the Teleport oracle is added by the second commit and extended by the third). Counts are V: the Windows build worker (Ubuntu under WSL, g++ Release, 1.14d data), `tools/emu/sk_mana.py` and `sk_tele_spot.py` running game.exe's functions natively against the real d2d builds `test_sk_mana` and `test_tele_spot`; "base" is upstream main plus this PR's test files.

| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| `14574699` | The mana check has no minmana floor; the payment needs enough mana | `0056c160`, `0056bfe0` | mana run below: 190,840 cases, 105,978 have a cast path and are compared; base 281 differ, head 0 |
| `5e0b4419` | Teleport lands only in the caster's room or its near list | `00554ea0`, `00463740` | Teleport run, seeds 1 and 9: base lands in 25,120 and 25,357 cases where game.exe refuses; after this commit 0 |
| `41afdc6a` | Teleport lands on the nearest free spot of a blocked target | `00554ea0`, `0064e7b0`, `0064dea0`, `0064d9b0` | same run: 1,461 and 1,555 blocked-target cases differ before this commit, 0 after; 100,000 of 100,000 equal at the head |

## How to re-check

```sh
python3 tools/pr_guard.py
cmake --build build && ctest --test-dir build --output-on-failure
cd tools/emu && uv sync
python3 -m unittest test_sk_mana.py test_sk_tele_spot.py test_structs.py   # the game-backed tests run when D2_MPQ_DIR is set

# mana (commit 1): head; for base, main plus tests/game/test_sk_mana.cpp from this PR
set -o pipefail
uv run python sk_mana.py gen 100000 7 > cases.tsv
uv run python sk_mana.py game cases.tsv > g.txt
../../build/tests/test_sk_mana --sk-mana cases.tsv | grep -E '^[0-9]+ ' > d.txt
uv run python sk_mana.py diff cases.tsv g.txt d.txt        # head: "different 0"; base: "different 281"

# Teleport (commits 2 and 3), seeds 1 and 9; build the commit under test (base: main plus the test file)
for seed in 1 9; do
  uv run python sk_tele_spot.py gen 100000 $seed --no-units > cases.tsv
  uv run python sk_tele_spot.py game cases.tsv > g.txt
  ../../build/tests/test_tele_spot --tele-spot cases.tsv | grep -E '^[0-9]+ ' > d.txt
  uv run python sk_tele_spot.py diff g.txt d.txt cases.tsv  # head: "100000  same"
done
uv run python sk_tele_spot.py probe                          # the landings round a blocked target, the refusals
```

Counts of this head (V):

| Teleport seed 1 | base | after commit 2 | after commit 3 (head) |
|---|---|---|---|
| of 100,000 cases equal to game.exe | 73,419 | 98,539 | 100,000 |
| differ: target room not near | 25,120 | 0 | 0 |
| differ: target blocked | 1,461 | 1,461 | 0 |

Seed 9: equal 73,088, 98,445 and 100,000; room not near 25,357 before commit 2; blocked 1,555 before commit 3. Mana by skill on base: Teleport 74, Dopplezon 72, Guided Arrow 68, Bone Prison 67 (all "game.exe casts, d2d refuses").

## Not done

- Limits (U unless stated): the oracle's near list is the 3 x 3 touching rooms (a hook), so irregular levels' rooms are not compared; Teleport 2's clear-line test is hooked to "clear"; a monster at the target is left out of the cases (`--no-units`: d2d keeps units in its Crowd, not the grid); Might and Summon Resist have no cast path in the harness and are not claimed; `FUN_0056bfe0`'s state 0x72 branch and the srvdofunc 116 shortcut are not modelled and unchanged; the change is for the player (both functions return at once for a non-player; what a merc or pet pays through `Fight::pay` is outside the evidence); payment timing and interrupted casts are not part of it.
- Presubmit WARNs: siblings (unit kinds): the player only, as above; siblings (hit paths): skill casts only, not melee or missiles; build: skipped by `--quick`, the build is under Checks.
- A live game, a soak and a determinism hash comparison were not run.

## Checks (V unless said)

- Commits: 3, subject at most 72 characters, body at most three lines of at most 72, last line `Agent: Claude Code`; author and committer the numeric noreply identity; no co-author or AI lines (`git log --format='%an <%ae> | %cn <%ce>'`).
- `python3 tools/pr_guard.py upstream/main HEAD`: 0 FAIL, 0 WARN on the range and on each commit; `python3 tools/presubmit.py --quick`: 0 FAIL, 4 WARN (answered above). With a CMake build dir on the build worker, presubmit's GCC (-Wshadow -Wconversion -Werror) and clang-tidy: 158 of 159 TUs clean; the launcher TU cannot compile there because Qt's `main.moc` is not generated by the partial build (an environment gap, not this diff).
- Python: ruff with the presubmit's rule set, 0 findings; the three oracle test files and `test_structs` pass without game data and with it (21 tests, all ok).
- Parity lane (g++ and clang++ with the CI flags, extra warnings, suites, clang-tidy) and MSVC 14.51: see the result lines under "Parity" at the end.

### CodeRabbit budget

| quantity | this PR | budget (target / ceiling) |
|---|---|---|
| files | 16 | 40 / 70 (cap 100) |
| added + deleted lines | 1,013 (1,001 + 12) | 1,000 / about 1,500 |
| commits | 3 | 15 / 40 |

## Checklist

- [x] A test in `tests/` (fix or port)
- [x] The oracles in `tools/emu`, README rows, `structs.tsv` rows for the new fields
- [x] `python3 tools/pr_guard.py` passes

## Parity

All V, the Windows build worker, head `41afdc6a` on `e1eea02a` (2026-11-03):

- Linux lane (Ubuntu under WSL, 8 threads, ccache): guard rc 0; g++ configure and the suites rc 0 (100 s); g++ all targets rc 0; clang++ configure and suites rc 0; the run stage rc 0; extra warnings rc 0; clang-tidy rc 0; TOTAL 493 s.
- MSVC 14.51 (cl 19.51.36260, VS 2026, the compiler of upstream CI; RelWithDebInfo, /W4): 0 `warning C`, no error; ctest 74 of 74 passed. That lane has no game data, so `test_sk_mana` and `test_tele_spot` compile and link there and report Skipped; the data-backed runs are the ones above.

---

Agent: Claude Code
