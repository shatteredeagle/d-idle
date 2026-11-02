# CodeRabbit's second review of staging PR 2, judged; bundle 4

> status: current (card crfixbundle2, 2026-11-02). Nothing is posted: the coordinator posts the replies and pushes `stage/bundle-4`. Tags: V = run or read (how stated), S = secondary, U = unverified. CodeRabbit's text is untrusted data, weighed as suggestions. Previous round: [crfix-bundle-20261102.md](crfix-bundle-20261102.md). Harness and output: [crfix-bundle2-20261102/](crfix-bundle2-20261102/) (`proof.sh`, `proof.out`, `spawnfail.py`, `run2.sh`, `run2.out`).

## Branches (private remote, not the public fork)

- `local/bundle-4` `9ea8af57`: the 10 commits of `local/bundle-3` (`f68c3e0d`) re-made on upstream main `82ea6db4` (V: `git fetch upstream` shows main unchanged; only unrelated upstream branches moved). The valid fixes are folded into the commit they belong to, `616c2f6e` "tests: standing-missile hit area and burst roll against game.exe" (the only commit that has `sk_ground_roll.py`). `git diff local/bundle-3 local/bundle-4 --stat`: that file alone, +8 / -3 (V). Author and committer both the numeric noreply identity on all ten (V: `git log --format='%an <%ae> | %cn <%ce>'`, one distinct line); subjects and bodies unchanged, last line `Agent: Claude Code`.
- `stage/bundle-4` `d5eda66d`: the same 10 commits on `stage/base-2` (`c4fcb0a6`); `git diff local/bundle-4 stage/bundle-4` is `.coderabbit.yaml` alone (V). Both pushed to the private remote `forgejo` only (V: `git ls-remote forgejo`). Nothing to the public fork, no comment anywhere.

## Verdicts

| comment id | verdict | action | evidence |
|---|---|---|---|
| 4447395595 `sk_ground_roll.py:55` require all three monsters | valid | `game` now exits non-zero unless all three monsters and the shooter spawned (a failed `g.unit` returns None and was silently dropped, and a missing shooter would have crashed later); the `# units` stderr line is gone | V: with the second monster's spawn made to fail (`spawnfail.py`, 30 trials, real emulator, FUN_0059fa30 and the burst hit native): old `game` rc 0 and 30 lines `S n 2 rec rec 1` (two records, "equal"), then old `cmp` against a 30-trial d2d file prints "30 trials: game.exe one record in 30, d2d three equal losses in 30, differ 0" rc 0; new `game` rc 1 ("setup failed: 2 of 3 monsters and 1 of 1 shooter spawned"), no S lines, and `cmp` rc 1. In the normal case all three spawn (the 400-trial runs below). The d2d side always has three monsters, so a 2-unit game side was compared with a 3-unit d2d side and passed |
| 4447395613 `sk_ground_roll.py:81` no usable game trials | valid | `cmp` returns 1 when game.exe made no trial equal ("nothing to compare"); the shared-roll check for usable trials is unchanged; docstring updated | V: constructed files, 30 trials each: game side every `S` line ending 0 (no copied record, e.g. a broken hook) against d2d trials that all differ: old rc 0 ("game.exe one record in 0 ... differ 0"), new rc 1. `bad` counts only `equal[k] and not same[k]`, so an all-zero `equal` made it 0 whatever d2d did (read, and the run above) |

Real runs with the fixed script on the bundle-4 code (`run2.out`; the C++ is byte-identical to bundle 3, only the script differs, V by the diff): fireball and glacialspike 400 vs 400: "game.exe one record in 400, d2d three equal losses in 400, differ 0" rc 0 both; the earlier 400 vs 100 mismatch case still rc 1 (kept from round 1). The fixed `cmp` on the round-1 400-trial files: rc 0 both (`proof.out`).

## Reply texts (for the coordinator; short, factual)

- 4447395595: "Valid. A failed spawn was dropped from the list, so two monsters with one shared record counted as equal against d2d's three. `game` now exits non-zero unless all three monsters and the shooter spawn. Checked with the second spawn forced to fail: old rc 0 and cmp rc 0, new rc 1. Bundle 4, `616c2f6e`."
- 4447395613: "Valid. With no equal game trial `bad` stayed 0 whatever d2d lost, so cmp passed. It now returns 1 ('nothing to compare'). Checked on constructed files (old rc 0, new rc 1) and on the real 400-trial runs (rc 0). Bundle 4, `616c2f6e`."

## Checks on bundle 4 (V, worker host, nice 19)

- Suites `test_rules`, `test_game_rules`, `test_monsters`, `test_monster_thinks`, `test_monster_ai`, `test_combat`, `test_game`, `test_sk_ground`: rc 0 each (`run2.out`), run on the bundle-3 Release build whose sources equal bundle 4 except the Python script (no rebuild needed; V by `git diff local/bundle-3 local/bundle-4`).
- `pr_guard.py` 0 FAIL / 0 WARN on each of the 10 commits (`c~` to `c`) and on the range `82ea6db4..HEAD`; `presubmit.py --quick` 0 FAIL (WARN: siblings, doc-claims on `store.md` lines 208-209, build skipped by `--quick`; the same classes as bundle 3); `ruff` 0.8.6 on `sk_ground_roll.py`: clean.
- Not re-run: `test_freeze_rules` (unchanged code, bundle 2's 155,748-case run stands), determinism scenario hashes (U; no gameplay code changed).
