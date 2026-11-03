# Branch triage, 2026-11-02

> status: read-only triage (card branchtriage). Nothing was merged, deleted or pushed. Base: origin/main 15506e05 after a fetch. Tags: V = checked with git in this run (merge-base --is-ancestor, git cherry, blob comparison against origin/main, git status of the worktree); U = judgement from the card text.

Method: every `refs/heads/agent/*` branch that is not an ancestor of origin/main (15 of them; all others, including the worktree-only names with no unique commits, are already contained). Per branch: `git log origin/main..branch`, `git cherry`, blob compare of every added or modified file against origin/main, the branch's own card (`tasks.md` status, `merge-lines.txt`) and `git status --porcelain` of its worktree.

Commands: merge = `tools/ops.sh run merge.sh agent/NAME --card NAME` (run from the public clone; add `--lines-file` is not needed, the card's merge-lines.txt is used). Delete = `git worktree remove ../d2lod-wt/NAME` then `git branch -D agent/NAME` from the public clone. None of them was run.

## Table

| branch | unique commits | verdict | reason | command |
|---|---|---|---|---|
| levelspawn | 5 (3 patch-unique, 2 merge commits) | MERGE (docs and tools only, still useful) | trace `levelspawn-trace.md` and `tools/emu/ls_*` absent on main; card in-progress with result lines; main has since changed `docs/coverage/ledger.tsv` and `rule-coverage.md`, so expect a merge-resolve step; worktree clean | `tools/ops.sh run merge.sh agent/levelspawn --card levelspawn` |
| lockstepp4run | 1 | DROP (never merge) | commit says "local harness patch, --timescale=K arg (not for merge)"; branched before botclock landed, so it would revert main's `internal/gclock`; no card; worktree clean | `git worktree remove ../d2lod-wt/lockstepp4run; git branch -D agent/lockstepp4run` |
| monsterpace | 9 | MERGE (docs and tools only, still useful) | PR A and PR B drafts (`pr-monsterpace-a-draft.md`, `pr-monsterpace-b-draft.md`) and `mp_*` / `mai2_*` scripts absent on main (the `propenmonsterpace*` cards on main only logged the posting); card in-progress; worktree clean | `tools/ops.sh run merge.sh agent/monsterpace --card monsterpace` |
| nmlevel | 3 | MERGE (docs only, still useful) | `pr-nmlevel-draft.md` and `nm_pop.py` absent on main; main has only the verifier record (`pr-nmlevel-verifier.md`) and the `propennmlevel` log card; worktree clean | `tools/ops.sh run merge.sh agent/nmlevel --card nmlevel` |
| playerroll | 4 | MERGE (docs only, still useful) | `pr-playerroll-draft.md` and the `hit_phys` det scenario absent on main (`det_scenarios_lab.py` differs only by that scenario); main has the `propenplayerroll` log card only; worktree clean | `tools/ops.sh run merge.sh agent/playerroll --card playerroll` |
| prbuildground | 1 | KEEP (live work) | `pr-ground-burst-draft.md` with numbers still pending, card in-progress, next step is an independent verifier; worktree has one modified file (generated `docs/plan/STATUS.md`, regenerable) | none yet; merge after the numbers are in: `tools/ops.sh run merge.sh agent/prbuildground --card prbuildground` |
| prbuildskmelee3b | 1 | MERGE (docs only, still useful) | card `prbuildskmelee3b` is already archived on main, but this later commit moves the D1 and D2 draft base lines to the rebase174 heads (`-r174` branches, base 48908176); main still shows the older `-final` lines; worktree has only an untracked `.scratch/` | `tools/ops.sh run merge.sh agent/prbuildskmelee3b --card none --summary "skmelee3 drafts retargeted to rebase174"` |
| reverifymissile | 3 (all WIP) | SUPERSEDED (delete) | by card `missilefix3` (commit e3fc8481 on main): main's `pr-missilerecord-reverify.md` has the third-check results filled in; the branch copy still holds the `RESULT_SUITES` placeholders (V: the whole diff to main is those 3 lines); worktree clean | `git worktree remove ../d2lod-wt/reverifymissile; git branch -D agent/reverifymissile` |
| reviewprs | 1 | MERGE (docs only, still useful) | `review-community-prs-20261102.md` absent on main; community PRs 19, 22, 23 are still open (V: `gh pr list`); the verdicts are provisional (60-minute review); worktree clean | `tools/ops.sh run merge.sh agent/reviewprs --card reviewprs` |
| skdefshieldd2d | 3 | KEEP (live work) | finished trace `skdefshieldd2d-trace.md` and `skdd_*` probes absent on main, but the worktree is in the middle of an unfinished merge of main: 225 entries, 3 unmerged (UU) paths, 183 staged adds; it needs its owner or the coordinator to finish or abort that merge before anything is merged | after the merge is resolved and committed: `tools/ops.sh run merge.sh agent/skdefshieldd2d --card skdefshieldd2d` |
| sorcrest2 | 1 | MERGE (docs and tools only, still useful) | `sorcrest2-trace.md` and `sr2_*` absent on main; card in-progress with result; `docs/coverage/ledger.tsv` differs from main (expect a merge-resolve step); worktree clean | `tools/ops.sh run merge.sh agent/sorcrest2 --card sorcrest2` |
| verifymonsterpace | 2 | MERGE (docs only, still useful) | card status done; verifier record `pr-monsterpace-verifier.md` plus probes absent on main; belongs with the monsterpace drafts, merge it after or with them | `tools/ops.sh run merge.sh agent/verifymonsterpace --card verifymonsterpace` |
| verifyphys | 3 (2 WIP) | MERGE (docs only, still useful; card needed) | `pr-physroll-verifier.md` and probes absent on main; the branch has no change card and no merge-lines (V), so add a card first or use `--card none`; verifier verdict READY WITH CHANGES for `local/skabrecheck-phys`; worktree clean | `tools/ops.sh run merge.sh agent/verifyphys --card none --summary "verifier record for the physical roll PR candidate"` |
| verifyplayerroll | 9 (8 patch-unique) | MERGE (docs only, still useful) | `pr-playerroll-verifier.md` and probes absent on main; card in-progress, result recorded (READY WITH CHANGES); carries a merge commit of main already; worktree clean | `tools/ops.sh run merge.sh agent/verifyplayerroll --card verifyplayerroll` |
| verifytele | 2 | MERGE (docs only, still useful) | not a duplicate: the card `verifyteleroom` on main says "first pass, not yet independently verified", and this branch is the independent re-check (`verify-teleport-room.md`, `vt_*`; verdict: the room rule holds, 20,000 cases), absent on main; worktree clean | `tools/ops.sh run merge.sh agent/verifytele --card verifytele` |

## Counts

MERGE 11, KEEP 2, SUPERSEDED 1, DROP 1, ALREADY ON MAIN 0.

## Findings

1. No unmerged branch is already on main by another route. The only content-level overlaps are `reverifymissile` (replaced by `missilefix3`, whose doc has the real numbers) and `prbuildskmelee3b` (its card is archived but its last commit is not on main).
2. Eleven branches are finished, docs-and-tools-only work whose output is missing from main: the PR drafts for monsterpace, nmlevel, playerroll and the skmelee3 retarget, and the verifier or trace documents for levelspawn, sorcrest2, verifymonsterpace, verifyphys, verifyplayerroll, verifytele and the community PR review. Two of them touch `docs/coverage/ledger.tsv` (levelspawn, sorcrest2), so merge them one at a time.
3. `skdefshieldd2d` has an unfinished merge sitting in its worktree (3 conflicted paths), so its branch tip is not what the worktree holds; `lockstepp4run` is a self-declared "not for merge" harness patch that would revert `internal/gclock` and must never be merged.

Worktrees whose branch is already contained in main (bundlefinal, stateevrun, zerohitrun, and others with no unique commits) are not listed: they hold nothing to merge and can be removed with the delete command above.
