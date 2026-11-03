# Batch C: rows to add to the open batch A pull request

> status: PREPARED, not posted (card cport, 2026-11-03). Nothing is pushed to the public fork and no PR is open. Private branch `local/batch2-C-r3` on the fork's private remote, head `4ed8b651`: batch A (`local/batch2-A-r5`, head `41afdc6a`, 3 commits on upstream main `e1eea02a`) plus 3 commits. `local/batch2-C-r1` (`af91cec5`) and `-r2` (`dfd3948c`) are superseded and kept (no force-push): r1 left the batch A mana test reading the mana at the command, which commit 1 moves (found in the run below); r2 lacked an include that clang-tidy's include-cleaner wants in the test of commit 2 (found in the parity lane). Tags: V = run or read in this card (how stated), S = secondary, U = unverified.

Rule followed (V, [pr-upstream-rules-20261103.md](pr-upstream-rules-20261103.md)): the maintainer keeps one batch open and wants further work of the same subsystem added to it. Batch C is player skill casting like batch A (mana payment, Teleport): when a cast pays, how long its action frame takes, which self states a new one switches off. One claim per commit, each with its test and its oracle in `tools/emu`.

## Claims-table rows to append to batch A's table

Counts are V: the Windows build worker (Ubuntu under WSL, g++ Release, 1.14d data), the oracles running game.exe's own functions natively against the real d2d builds, one job at a time.

| Commit | Claim or change | Evidence | Re-check |
|---|---|---|---|
| `7620ce97` | A cast pays its mana on the action frame, not at the command; a cast cut off before it costs nothing | `0056f7f0`, `0056bfe0`, `00580460`, `005539b0` | `cast_mana.py` run below: 53,628 cases; before this commit every one of the 42,168 casts that began differs; after it and the next commit 0 differ |
| `eb1f87fa` | A cast's action frame comes on the animation scheduler's tick, not one tick early | `005539b0` | same run: with commit 1 alone 10,560 cases differ (the frame one tick early at 132 of 401 FCR values); with this commit 0 |
| `4ed8b651` | A self state switches off the other states of its States.txt group | `0056c740` | `armor_group.py` run below: 48 of 5,476 ordered skill pairs differ on main; after this commit 0 |

Commit 1 also changes the batch A test `tests/game/test_sk_mana.cpp`: the cast starts, then the owed payment is settled (`Fight::settle_cast`), so the test still compares what the cast costs and not when it is taken (see "How it sits on batch A").

## Evidence from game.exe 1.14d (rows to add to batch A's table)

| Address | What it shows |
|---|---|
| `005539b0` | the animation scheduler: walks the position in steps of the cast rate and files an event at tick `ceil(256 x frame / speed)` |
| `00580460` | the player mode event: value 1 or 2 runs the do unless the missile-made flag is set |
| `0056f7f0` | the do pays the mana (`0056bfe0`) for a skill with no start function |
| `0056c740` | a state's start clears each state of the same States.txt group the unit has |

## How it sits on batch A (conflicts, V: cherry-picks, then the runs below)

Batch A rewrote `Fight::pay`/`cost_of` and `rules/skills.hpp`; the private mana-timing branch was written on plain upstream main, so it conflicts in `fight.hpp`, `fight/tick.cpp`, `fight/missiles.cpp` and `tools/emu/README.md`, and the armor branch only in the README (adjacent rows, both kept).

| Place | Resolution | Both behaviours kept because |
|---|---|---|
| `Fight::pay(skill, lvl)` (A) and the new `owe` / `settle_cast` (C) | the owed record is `{skill, lvl}` (it was `{skill, cost}`); `owe(skill, lvl)` either records it (srvstfunc none) or calls A's `pay` at once; `settle_cast()` calls A's `pay(skill, lvl)` | A's `pay` takes `mana_paid(skill, lvl, mana)`: the amount (formula or minmana, nothing when the mana is short, nothing for a row with no mana cost) is decided at the moment of payment. C only moves that moment, so A's rule applies unchanged at the action frame; the old C test "short of the mana by then" is `mana_paid` itself |
| the check `cost_of` (A: the formula alone) | untouched; `Fight::cast` / `cast_missile` still check with it at the command | game.exe checks at the command and pays on the frame; C splits exactly these two |
| `cast`, `cast_missile` call sites | `pay(*skill_row, lvl)` became `owe(*skill_row, lvl)`; `cast_summon` and `start_swing` keep `pay` (a start function or ammo pays at the command, `0056f640`) | A's hunks around them are unchanged |
| `fight.hpp` | `Owed` and `action_ms` fields added next to each other, A's `pay`/`owe`/`settle_cast` declarations | no overlap |
| `tools/emu/README.md` | both rows kept | sorted by area as before |
| `tests/game/test_sk_mana.cpp` (A's file) | `start()` calls the old body, then `settle_cast()` | without it A's counted differential reads the mana before the action frame and 26,867 of 105,978 cases differ (V, the first run on `af91cec5`): the payment moved on purpose, the test measured the old moment |

## Counts on the exact head `4ed8b651` (V)

Run of the stages build, mana, armor, amana, atele (a script of the private ops repo; the build worker's Linux lane, `-j4`, ccache, one job at a time under its lock; 11 minutes warm). Heads: A `41afdc6a`, C `4ed8b651`; "C~2" is `7620ce97` (commit 1 only).

| Differential (oracle against the real d2d build) | Cases | A head | C~2 (pay only) | C head |
|---|---|---|---|---|
| mana timing: Charged Bolt, Frost Nova, Fire Bolt, Frozen Armor at levels 1-20, six mana pools, FCR 0..400, get-hit cuts (`cast_mana.py gen 1`) | 53,628 | not run (the harness needs the payment moved) | **10,560 differ** | **0 differ** |
| of them: casts that began / refused for lack of mana | 42,168 / 11,460 | | | agree |
| of them: cut before the event, game.exe pays nothing | 21,084 | d2d pays (SORCDIFF2, harness-only build `cb07926f`: all 21,084; S) | | **0 pay, equal** |
| armor group, ordered pairs of the 74 state-giving skills (`armor_group.py game`) | 5,476 | 48 differ (SORCDIFF2, main; S) | | **0 differ** |
| A: mana check and payment (`sk_mana.py gen 100000 7`), compared 105,978 with a cast path | 190,840 | 0 differ | | **0 differ** |
| A: Teleport destination, seed 1 | 100,000 | 100,000 same | | **100,000 same** |
| A: Teleport destination, seed 9 | 100,000 | 100,000 same | | **100,000 same** |

The residual on C~2 is one case family: the action frame one tick early (the first line of the diff: FUN_005539b0 gives tick 7 at 0 FCR 9, d2d 6). `test_cast_mana` and `test_armor_group` without an argument pass on the C head (rc 0). The A differentials are identical on the A build and the C build, so A's claims are unchanged, once the A mana test settles the owed payment (first run on `af91cec5`: 26,867 of 105,978 "paid differs" because the test read the mana before the frame; fixed in commit 1, re-run above).

## Honest limits

- U, the cut-cast assumption: a mode replaced before the event (a get-hit) leaves the queued event unrun, so the cast costs nothing. The 21,084 cut-before-event cases agree with the oracle only because the oracle's scheduler ran the same assumption (it files the event and the harness drops it when the mode changes); the read of `FUN_005533d0` and `FUN_00553990` (the mode start's calls before scheduling, which may clear queued events) and of the server's order of events on the same tick is the independent Opus check's job. The case generator never puts the get-hit on the event's tick.
- Payment moves only for skills with no start function (`srvstfunc` none): Charged Bolt, Frost Nova, Fire Bolt, Frozen Armor are in the cases. Inferno, Telekinesis, Thunder Storm, Hydra and the ammo skills still pay at the command (U: not in the cases; read in `0056f640`).
- The Frozen Armor trigger (D5: d2d freezes on `damage > 0` of any element, game.exe on the physical record) is not covered: no game-side harness exists; not claimed.
- The armor-group claim holds for the 74 skills that give a self state; which names group 223, 228, 232, 233, 350, 258 and 267 was not checked. The bot casts only Frozen Armor, so a gate run does not change.
- Player only (as batch A); merc, pet and monster casts are not claimed. No live game, no soak, no determinism hash comparison.
- Presubmit WARNs on the three commits: siblings (unit kinds, hit paths) as in batch A; error-path at `cast_mana.py` (the AnimData parse is guarded: the record loop stops on a short read, the heuristic does not see it); build and GCC/tidy: see Checks.

## Size against the CodeRabbit budget

Batch A: 16 files, 1,013 lines (+1,001 / -12). Batch C alone: 16 files, +639 / -9. A and C together, a range against `e1eea02a`: 25 files (6 files are touched by both), +1,637 / -18, 1,655 lines, 6 commits. Far below the 100-file cap seen in [coderabbit-limits-20261102.md](coderabbit-limits-20261102.md) (V: `git diff --shortstat`); lines are above A's alone, and each push spends the hourly review allowance, so add the three commits in one push.

## Checks (V unless said)

- Commits (head `4ed8b651`, 3 on top of A): subjects 65 to 67 characters, body 3 lines of at most 72, last line `Agent: Claude Code`; author and committer the numeric noreply identity for all 6 commits of the range (`git log --format='%an <%ae> | %cn <%ce>' e1eea02a..HEAD`); no co-author or AI lines (presubmit attribution ok).
- `python3 tools/pr_guard.py 41afdc6a HEAD`: 0 FAIL, 0 WARN (and per commit); `python3 tools/pr_guard.py --sorted e1eea02a HEAD`: 0 FAIL. `python3 tools/presubmit.py --quick`: 0 FAIL, 6 WARN (siblings x2, error-path x2, build, gcc: the last two because `--quick` has no build dir; the build worker run below covers them). Ruff: 7 files, 0 findings; the tool tests pass (D2_MPQ_DIR unset: game-backed ones skip).
- Parity lane (g++ and clang++ with the CI flags, suites, extra warnings, clang-tidy on the lines the range adds), run on `local/batch2-C-r3` against upstream main: every stage rc 0 including tidy. The same lane on `dfd3948c` (r2) had tidy rc 1: misc-include-cleaner wanted `<combat.hpp>` in `tests/game/test_cast_mana.cpp` (`cast_action_ticks`); fixed in commit 2 (rebased to `eb1f87fa`), re-run clean. The suites stages of the r3 run took 0 s (ccache-warm build dir; the 8 suites ran on r2 at 100 s and 83 s, rc 0; U that the include-only change needs a re-run of them, tidy and builds did run).
- MSVC 14.51 (the lane CI uses): `/W4` build of the head, `ctest` 76 of 76 passed, 0 `warning C` lines, 224 s.
- Not run: GCC `-Wshadow -Wconversion -Werror` per changed TU through presubmit's own `check_gcc` (needs a presubmit build dir; the g++ lane above builds with the CI flags); a live game; a soak.

## Shared-file lines for the merger

See `openspec/changes/cport/merge-lines.txt`.
