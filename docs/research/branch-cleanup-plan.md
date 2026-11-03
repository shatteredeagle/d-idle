# Branch cleanup plan 2026-11-03

> status: plan for the lead's approval; nothing was merged, tagged or deleted by this card. Owner request 2026-11-03: "add a backlog item to clean up and merge the branches". Tags: V = read from git in this worktree (`git for-each-ref refs/heads/agent --format='%(refname:short) %(ahead-behind:origin/main)'`, `git branch --merged origin/main`), U = not checked. Inventory background: [housekeep-20261103.md](housekeep-20261103.md) (its list is older: several branches it names are gone, a few new ones exist).

## 1. Counts (V)

41 local `agent/*` branches, none on the remote. 5 are merged into origin/main (ahead 0): b2bundle, sorcdiff, sorcdiff2, stagea, and this card's own branchclean. The other 36 are unmerged and are classified below: **LAND 20, FOLD 6, TAG-AND-DELETE 5, KEEP 5**.

U for the whole table: I classified from the last commit subject, ahead/behind and the housekeep notes; I did not diff each branch. Step 1 of the plan (a dry run per LAND branch) turns the "blocker" column from a guess into a fact. Many branches are 300 to 1000 commits behind main, so a merge is a rebase of docs and card files, not a code merge.

## 2. The six dropped from the merge queue today

| branch | class | why |
|---|---|---|
| covmonskills2 | TAG-AND-DELETE | coverage ledger rows conflict with what main has; low value, the pass-1 branch covmonskills carries the useful rows |
| covtreasure2 | TAG-AND-DELETE | same ledger row conflicts; low value |
| verifydmg | TAG-AND-DELETE | superseded by the later verifier rounds and the oracleport PR bodies |
| dmgroll | TAG-AND-DELETE | superseded the same way; conflicts in Go files because it is based on an older tree (12 ahead, 242 behind) |
| barbdiff | FOLD | the Barbarian premise was wrong: the gate character is a lightning Sorceress. Keep only `docs/research/barbdiff.md` section 7: Howl D3/D4 already fixed upstream, War Cry covered by PR 158, D7 unreachable, only Whirlwind D1 and Howl D5 remain |

The request counted six dropped branches but named five; the sixth is not identified (U). Candidate: lockstepp4run ("not for merge"), listed under TAG-AND-DELETE below. The lead should confirm.

## 3. Classification of the 36 unmerged branches

Ahead is commits not in origin/main (V). Every command block is in section 4.

### LAND (20): finished docs/tools, valuable. Blocker = what stops a plain merge today (U until the dry run)

| branch | ahead | why | blocker |
|---|---|---|---|
| codereview2 | 2 | OWNERS agent column, STATUS block | OWNERS conflict (U) |
| covarmor | 1 | armor and itemtypes first-pass trace (data only) | missing ledger row merge, card (U) |
| covmonai | 2 | monster AI third pass, 54,000 cases, ledger rows | ledger conflict (U) |
| covmonskills | 3 | monster and common skill rows, pass 1 | ledger conflict (U) |
| covstates | 2 | states 2 and 21 trace and ledger rows | ledger conflict (U) |
| crfixbundle2 | 4 | CodeRabbit fix bundle | conflict with main (last commit is a merge), needs Go tests (U) |
| gate2analysis | 1 | Gate 2 failure analysis | none known; check leak |
| kittrip | 4 | kit trip work | conflict, may touch Go (U) |
| levelspawn | 6 | level spawn coverage | 525 behind, ledger conflict (U) |
| prbuildground | 2 | ground and burst PR draft with counted differentials | 670 behind; leak check (U) |
| reverifymissile | 3 | third check of missiles; first part already merged | 1057 behind: take only the later commits |
| verifycovmissiles | 1 | independent check of missiles coverage | ledger conflict (U) |
| verifycovstates | 1 | independent check of covstates | ledger conflict (U) |
| verifyitemweapons | 1 | recount of weapons pass | ledger conflict (U) |
| verifylevelsareas | 1 | level areas check | ledger conflict (U) |
| verifymonsterpace | 4 | monster pace check | last commit is a merge, conflict (U) |
| verifyphys | 5 | physical damage check | last commit "merge origin/main into none": check the card id |
| verifyskassassin | 1 | assassin skills check, 0 differences on ids 251..280 | none known |
| verifyskbarb | 1 | barbarian skills check | staged files seen in its tree earlier (housekeep): commit or drop them first |
| verifyskdruid | 2 | druid skills check, ledger note | none known |

Risk for all LAND: a conflict in a shared file (ledger, OWNERS, STATUS) is resolved by hand and can drop another branch's row; leak hits in old text (machine names, paths). Mitigation: rules in section 5.

### FOLD (6): copy one or two files to main as a digest, then retire (tag first)

| branch | ahead | what to copy | why |
|---|---|---|---|
| barbdiff | 2 | `docs/research/barbdiff.md` section 7 only | see section 2 |
| gate1run-20261101-1625 | 1 | the run record (verdict NOT PASSED: stability, deaths, belt, town portal, fault recovery) | records are evidence; digest into `docs/g1-log.md` |
| gate1run-20261102 | 1 | run record (NOT PASSED, Belt) | housekeep: another card text on main says PASSED, check which is current before digesting |
| gate1run-20261103 | 5 | run record plus any analysis file | most commits are merges from main |
| gate2run-20261102 | 2 | run record and the relative-worktree card fix | record only |
| zerohitrun | 3 | its run record or note | run record, 485 behind |

Risk: copying a stale verdict over a newer one. Mitigation: date every digest line and cite the run id, not the branch.

### TAG-AND-DELETE (5): superseded or empty

covmonskills2, covtreasure2, dmgroll, verifydmg (all in section 2), and lockstepp4run (a local harness patch, "not for merge", 1 ahead).

### KEEP (5): in flight, today's runs, or private-fork related

| branch | why |
|---|---|
| b2on181 | batch 2 PR work (leakcheck rewording), last commit today; belongs to the upstream bundle, wait for its PR |
| gate1run-20261103-0338 | today's run, 6 ahead; the run agent decides |
| gate1run-20261103-0628 | today's run record |
| gate3gaps | design and delta spec for Gate 3, today; a live card |
| skdefshieldd2d | SKDEFSHIELD d2d is fork related; its tree held a whole-repo snapshot in staged files (housekeep): inspect before touching |

Also merged, plain delete (no tag needed, ahead 0): b2bundle, sorcdiff, sorcdiff2, stagea. A merged branch is already in main history. Check the worktree for untracked files first (b2bundle had a `.scr/` directory).

## 4. Exact commands

Run from the main checkout with `git -C`, never `cd` into another worktree. `X` = branch name without `agent/`.

Merged branches (delete only after `git -C ~/code/d2lod-wt/X status --short` is empty or the files are scratch):
```
git worktree remove ~/code/d2lod-wt/X
git branch -d agent/X
```
TAG-AND-DELETE (the tag keeps every commit reachable; `-D` is needed because the branch is unmerged):
```
git -C ~/code/d2lod-wt/X status --short        # must be empty, else commit or copy first
git tag archive/X agent/X
git rev-parse archive/X agent/X                # same hash, or stop
git worktree remove ~/code/d2lod-wt/X
git branch -D agent/X
```
Run it five times with X = covmonskills2, covtreasure2, dmgroll, verifydmg, lockstepp4run.

FOLD (X = branch, FILE = the one file; after the copy, tag-and-delete as above):
```
git show agent/X:FILE > FILE                   # in a worktree branched from main, e.g. agent/branchclean-fold
python3 tools/merge_lines_check.py --changed
go run ./cmd/leakcheck -mode fail
git commit ...                                 # message: "fold X: FILE (digest)"
```
For barbdiff, FILE = `docs/research/barbdiff.md` and cut everything but section 7 plus a two-line header that says why the Barbarian premise was dropped.

LAND (done by the merge queue, mergeq, one branch at a time on top of current main):
```
git merge-tree --write-tree origin/main agent/X   # dry run: conflicts listed?
git -C ~/code/d2lod-wt/X merge origin/main        # resolve in the branch's own worktree
go run ./cmd/leakcheck -mode fail
python3 tools/merge_lines_check.py --changed
go run ./cmd/changecheck -cli
git merge --no-ff agent/X -m "Merge agent/X: ..."   # coordinator, on main
git tag archive/X agent/X && git branch -d agent/X  # after the merge
```
Risk: low for docs-only; medium where a branch touches Go (crfixbundle2, kittrip): run `labrun`-hosted `go vet` and `go test -short` before the merge.

## 5. Execution plan

Order: (1) delete the 4 merged branches; (2) tag-and-delete the 5 superseded (reversible via the tag); (3) fold the 6; (4) LAND the 20, cheapest first: single-commit verifiers, then coverage passes (ledger rows, one at a time so each rebases on the previous), then the branches that touch Go; (5) revisit KEEP when their owners finish.

Rules:
- `go run ./cmd/leakcheck -mode fail` has 0 hits before any LAND or FOLD; old branches often carry machine names.
- `python3 tools/merge_lines_check.py --changed` clean; `merge-lines.txt` has only BACKLOG/INDEX/CHANGELOG lines.
- Tag before delete: `git tag archive/X agent/X`. Never delete a branch with unique commits that no tag or main preserves. Verify with `git rev-list --count origin/main..agent/X` (0 or a tagged hash).
- Never touch a branch another agent works on. In flight today (U, confirm with `tools/agentreap.sh --list` and the status board before each step): b2on181, gate1run-20261103-0338, gate1run-20261103-0628, gate3gaps, skdefshieldd2d, and this card's branchclean. A branch whose last commit is from today and whose worktree has uncommitted files is treated as in flight.
- Deleting needs the lead's approval of the list (section 3), once, not per branch. Nothing is pushed or posted to GitHub.

Effort and who:

| work | agent-hours | resource |
|---|---|---|
| delete 4 merged + tag-and-delete 5 + the mechanical tag checks | about 1 | a Haiku agent, commands in section 4 |
| fold 6 (read, copy one file, digest line, leakcheck) | about 3 | one Sonnet agent |
| LAND 20 (dry run, conflict fix, leakcheck, merge) | about 10 to 14 (0.5 to 1 each) | mergeq for the merge itself; a Sonnet agent for conflict fixes |
| approval of the list and of every deletion | about 0.5 | the lead |

## 6. Proposed branch hygiene rule (for the lead to put in AGENTS.md)

> Every card's last task is "land or tag and delete the branch": the agent either gets the branch merged (merge-lines.txt filled, leakcheck 0) or, if the work is superseded, tags it `archive/<name>` and says so in the report. The coordinator archives the card and removes the worktree in the same step. A branch that is neither merged nor tagged 3 days after its last commit is listed in the daily housekeeping inventory. Run records (gate runs) are digested into `docs/g1-log.md` and the branch tagged, not kept as a branch.

## Shared-file lines for the merger

See openspec/changes/branchclean/merge-lines.txt.
