# Fidelity fixes against game.exe 1.14d: skill mana check and payment, Teleport target rules

> status: FINAL BODY for batch A (r9), not posted (card finala, 2026-11-03). Branch `local/batch2-A-r9` on the private fork remote (also `batch2-A-r9` on the public fork), head `57d7c39b`, 3 commits on upstream main `9785632c` (after PRs 182 and 183); it replaces r5 (`41afdc6a`, staged as fork PR 3) and r6, r7 and r8 (the same commits on older bases). Depends on the guard-fix PR (`docs/research/pr-guardfix-body.md`). Tags: V = run for this head (how stated), S = secondary, U = unverified.
>
> Title: `Fidelity fixes against game.exe 1.14d: skill mana check and payment, Teleport target rules`

## What changes

d2d refused a cast when the mana was under the skill's `minmana` floor and paid the floor. game.exe checks the formula `(mana + lvlmana x (level - 1)) << manashift` alone and pays `max(formula, minmana x 256)` only when the mana covers it. d2d also let Teleport land anywhere and refused a blocked target; game.exe lands only in the caster's room or its near list, and puts a blocked target on the nearest free spot (square rings 1 to 49).

## Depends on the guard-fix PR

The PR guard runs `tools/pr_guard.py` from the base checkout, so a pull request that adds an oracle under `tools/emu` and cites it fails "isn't in the tree" (AGENTS.md asks for the oracle in the same PR). The fix is its own one-claim PR (`guardfix-1`, `tools/pr_guard.py` and its test); it has to merge first because CI runs the base copy of the guard. Until it does, the `guard` job on this PR reads red with 21 FAIL lines, all of that one kind (V: the guard of upstream main run against this head, below); with the fixed guard run from the base checkout the same range gives 0 FAIL, 0 WARN.

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
| `3b335299` | The mana check has no minmana floor; the payment needs enough mana | `0056c160`, `0056bfe0` | mana run below: 190,840 cases, 105,978 have a cast path and are compared; base 281 differ, head 0 |
| `3fbb6803` | Teleport lands only in the caster's room or its near list | `00554ea0`, `00463740` | Teleport run, seeds 1 and 9: base lands in 25,120 and 25,357 cases where game.exe refuses; after this commit 0 |
| `57d7c39b` | Teleport lands on the nearest free spot of a blocked target | `00554ea0`, `0064e7b0`, `0064dea0`, `0064d9b0` | same run: 1,461 and 1,555 blocked-target cases differ before this commit, 0 after; 100,000 of 100,000 equal at the head |

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

## CodeRabbit's review on the fork (earlier head)

This batch was staged on a fork pull request first (head `41afdc6a`, the r5 form of the same three commits). CodeRabbit gave 10 items; this head is the result.

| # | Finding | What was done |
|---|---|---|
| 1 | the `guard` job fails because the oracle is not in the base tree | its own PR, above |
| 2 | `test_tele_spot --tele-spot` with no file read a null `argv[2]`; a file that did not open returned success | fixed: both print FAIL and return 1 (V, both run on this head) |
| 3 | `sk_mana.py diff` skipped case ids missing from the d2d file and could print "different 0" with nothing compared | fixed: ValueError when the d2d file does not cover the cases or no case has a cast path; unit test |
| 4 | `sk_tele_spot.py diff` zipped two files and stopped at the shorter | fixed: ValueError on empty or unequal files, `strict=True`; unit test |
| 5 | unescaped `|` in two README cells split the table | fixed, escaped like the neighbouring rows |
| 6 | the `gen` usage line lacked `--no-units` | fixed |
| 7 | one- and two-letter names (`id`, `x`, `y`) in `Rig::start` | not taken: the project's own names check (presubmit `short-names`, CI "Names and includes") passes |
| 8 | the game-backed Python tests skip unless `D2_MPQ_DIR` is set, not when `d2d.cfg` has `data =` | not taken: a skipped run, never a wrong result; both test files use the one gate and the README says so |
| 9 | `warp_spot` repeats `nearest_spot`'s ring search | not taken: it would move a template above `Level` in a verified path for no behaviour change (100,000 of 100,000 equal); happy to do it if you want it |
| 10 | docstring coverage 36 percent (warning, not gating) | not taken: no docstring rule here |

## Not done

- Limits (U unless stated): the oracle's near list is the 3 x 3 touching rooms (a hook), so irregular levels' rooms are not compared; Teleport 2's clear-line test is hooked to "clear"; a monster at the target is left out of the cases (`--no-units`: d2d keeps units in its Crowd, not the grid); Might and Summon Resist have no cast path in the harness and are not claimed; `FUN_0056bfe0`'s state 0x72 branch and the srvdofunc 116 shortcut are not modelled and unchanged; the change is for the player (both functions return at once for a non-player; what a merc or pet pays through `Fight::pay` is outside the evidence); payment timing and interrupted casts are not part of it.
- Presubmit WARNs: siblings (unit kinds): the player only, as above; siblings (hit paths): skill casts only, not melee or missiles; build: skipped by `--quick`, the build is under Checks.
- A live game, a soak and a determinism hash comparison were not run.

## Checks (V unless said)

All on head `57d7c39b` on `9785632c`, the Windows build worker (Ubuntu under WSL with game data, and MSVC), 2026-11-03.

- Commits: 3, subject at most 72 characters, body at most three lines of at most 72, last line `Agent: Claude Code`; author and committer the numeric noreply identity; no co-author or AI lines (`git log --format='%an <%ae> | %cn <%ce>'`). Rebased from `e1eea02a` onto `9785632c` with no conflicts (`strike.cpp` merged automatically with PR 183's edits); the three differentials below were re-run on this base and give the same counts.
- Guard, run as CI does (working tree at the base, head by name): upstream's `tools/pr_guard.py` gives 21 FAIL, all "tools/emu/sk_mana.py (or sk_tele_spot.py) isn't in the tree"; the fixed guard from the guard-fix PR gives 0 FAIL, 0 WARN on the range and on each commit.
- `python3 tools/presubmit.py --quick`: 0 FAIL from the diff; ruff ok (pinned ruff, the presubmit's rule set), short-names ok, sorted ok, attribution ok, clang-tidy ok on 169 translation units (1 finding on a line this branch did not add), 3 WARN (answered under "Not done"). One FAIL is the `gcc` step on `apps/launcher/main.cpp` (Qt's `main.moc` is not generated by the partial build there: an environment gap, not this diff; the g++ lane below builds every other target with -Werror).
- Python: the oracle tests and `test_structs` pass with game data: 23 tests, all ok, the regression tests for findings 2 to 4 included.
- `test_skills`, `test_sk_mana`, `test_tele_spot` pass with game data.

### Size

16 files, 1,023 added and 12 deleted lines (1,035), 3 commits (`git diff --shortstat 9785632c..HEAD`). A little over the 1,000-line target (the oracles and their tests in `tools/emu` are 609 added lines, the C++ tests 321, the game code and the `structs.tsv` rows the rest).

## Checklist

- [x] A test in `tests/` (fix or port)
- [x] The oracles in `tools/emu`, README rows, `structs.tsv` rows for the new fields
- [x] `python3 tools/pr_guard.py` passes (with the guard-fix PR merged; see "Depends on")

## Parity

- Linux lane (g++ and clang++ with the CI flags `-Werror`, 8 suites, extra warnings on added lines, clang-tidy): every stage rc 0 (guard, g++ configure and suites, g++ all targets, clang++ configure and suites, run, extra, tidy).
- MSVC 14.51 (cl 19.51.36260, VS 2026, the compiler of upstream CI; RelWithDebInfo, /W4): 0 `warning C`, no error; ctest 84 of 84 passed (the game-data tests report Skipped there, as that lane has no game data; the data-backed runs are the ones above).

---

Agent: Claude Code
