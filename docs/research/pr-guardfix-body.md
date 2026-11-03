# pr_guard: a cited tool is checked in the PR head, not in the base tree

> status: DRAFT body for the upstream guard-fix PR, not posted (card finala, 2026-11-03). Branch `local/guardfix-3` on the private fork remote (also `guardfix-3` on the public fork), head `93851bef`, 1 commit on upstream main `9785632c`. The lead opens it. Tags: V = run for this head (how stated), U = unverified.
>
> Title: `pr_guard: a cited tool is checked in the PR head, not the base tree`

## What changes

`tools/pr_guard.py` fails an added line that names a `tools/` script that isn't in the tree. The PR guard workflow runs the guard from the base checkout (so a PR can't weaken it), and the check was `os.path.exists`, so it looked in the base. A pull request that adds `tools/emu/x.py` and cites it was always told "isn't in the tree", which makes AGENTS.md's rule "the oracle goes in the same pull request" impossible to follow. Now the check asks git whether the PR head's tree holds the file (`git ls-tree --name-only <head> -- <path>`). The guard still runs from the base checkout, and a script that is in neither tree still fails.

## Evidence

This is tooling, not game.exe: no address applies.

| What | Where |
|---|---|
| The failure | A fork pull request that adds `tools/emu/sk_tele_spot.py` and `sk_mana.py` and cites them: the `guard` job failed with 21 FAIL lines, each "tools/emu/sk_tele_spot.py isn't in the tree" (or the same for `sk_mana.py`): https://github.com/shatteredeagle/d2decomped/actions/runs/41674514839/job/126051423888 (V, the run log; the workflow file on the fork is identical to this repository's) |
| Why | `.github/workflows/guard.yml` checks out `pull_request.base.sha` and fetches the PR only as `pr/head` for `git diff`; `os.path.exists` reads the checkout, which is the base (V, the workflow and `tools/pr_guard.py` at `e1eea02a` (unchanged since)) |
| The test | `tools/test_pr_guard.py`: `test_a_tool_the_pr_adds_and_cites_passes_from_the_base_checkout` (the working tree is left at the base, as CI does, and the head is passed by hash) and `test_a_tool_in_neither_tree_fails_from_the_base_checkout` |

## How to re-check

```sh
python3 -m unittest tools/test_pr_guard.py        # 12 tests, all ok on this branch

# the same tests against the old guard: the add-and-cite test must fail
git show upstream/main:tools/pr_guard.py > /tmp/old/pr_guard.py
cp tools/test_pr_guard.py /tmp/old/ && python3 -m unittest /tmp/old/test_pr_guard.py   # 1 failure
```

Counts (V): see "Results" below.

## Results

All V, on head `93851bef` on upstream main `9785632c` (the Windows build worker, Ubuntu under WSL, and a local run):

| Check | Result |
|---|---|
| `python3 -m unittest tools/test_pr_guard.py` | 12 tests, ok |
| the same test file against upstream's `pr_guard.py` | 1 failure: the add-and-cite test (the second new test, a script in neither tree, passes on both) |
| fork pull request 3's range (batch A, which adds and cites its oracles), run as CI does (working tree at the base, head by name) | upstream's guard: 21 FAIL, 0 WARN; this guard: 0 FAIL, 0 WARN |
| this branch with its own guard, and `presubmit.py --quick` | 0 FAIL, 0 WARN; presubmit 0 FAIL, 1 WARN (build skipped by `--quick`; the change has no C++), ruff ok |


## Not done

- The same flaw can't be reached for `check_addresses` or the order check: they read the added lines and `git show <head>:<path>`, already the head.
- A tool a PR *deletes* but still cites is not special-cased: the line would have to be an added line naming a file the head no longer has, which fails as it should.
- Nothing about the other workflows; this changes one line of the guard and its tests.

## Size

2 files, 25 added and 6 deleted lines (31), 1 commit.

## Checklist

- [x] A test in `tests/` (fix or port): `tools/test_pr_guard.py`, the guard's own test file
- [ ] The finding in `docs/research/re/`: not applicable, tooling
- [x] `python3 tools/pr_guard.py` passes

---

Agent: Claude Code
