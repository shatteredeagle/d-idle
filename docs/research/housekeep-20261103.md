# Housekeeping inventory 2026-11-03

> status: read-only inventory for the lead's approval; nothing was deleted. Snapshot of origin/main a45b9af4. Tags: V = read from git/ls on the dev box, U = not checked.

## 1. Done in this branch

- Archived `soak-minutes-virtual` (agent specmin, merged 12c11b9) as `archive/2026-10-31-soak-minutes-virtual`, applied its soak-gates delta by hand (task 1.5 of that card; 1.4 live proof stays open), dropped its 10 OWNERS rows. It was the only in-progress card whose overlap warned (V: `changecheck -cli` on main shows 0 warnings; the stashfield warning was its overlap with this card).
- Other in-progress cards whose branches are merged were left open ON PURPOSE (each says "card kept open", awaiting live proof or a later step): actgate, armorhit, buildv3, drills6, fixbelt, fixcastgap, fixmanamodel, fixprogress, fixskilllvl, forknext, ledgerkey, mapitem, nexthit, skmeleewire, skmiss, skresistfix, wpcooldown, m1-liveproof. Merge commits found by `git log --grep "Merge <branch>"` (V). They produce no overlap warnings today; archive them when their proofs land.
- gate1run-20261101 / gate1run-20261102 cards: result commits only, no merge; coordinator decides.

## 2. Correction banner for docs/research/msvcc2026-trace.md (not applied)

The file exists only on the unmerged branch agent/msvcc2026 (V), so it could not be edited from this branch. Put this under the title as the `> status:` banner:

> status: Correction 2026-11-02: the conclusion "real latent upstream problem" is wrong. Upstream CI builds this TU with MSVC 14.51.36231 (VS 2026, windows-2025-vs2026) and it passes. The same source at 3e073888 compiled with MSVC 19.51.36260 on the Windows build worker (V) with no C2026. The failure is specific to MSVC 14.44 (VS 2022). The limit change between 14.44 and 14.51 is U. Supported-compiler floor: VS 2026.

## 3. Agent branches merged into origin/main (safe to delete; worktree and local branch)

No `agent/*` branch exists on the remote (V, `git branch -r`). Ahead 0 of origin/main:

| branch | last commit | worktree state | recommendation |
|---|---|---|---|
| agent/b2bundle | 2026-11-03 | untracked `.scr/` | delete after checking `.scr/` is scratch |
| agent/housekeep | 2026-11-03 | this job (fresh from main) | keep until merged |
| agent/mercstats | 2026-11-02 | untracked tools/emu/mst_kill.py, mst_kill_all.sh | commit or copy the two scripts first, then delete |
| agent/prbuildskmelee3b | 2026-11-02 | untracked `.scratch/` | delete |
| agent/proof3 | 2026-11-02 | untracked planproof3.txt | delete (read it first) |
| agent/verifycovarmor, verifycovmonai, verifycovmonskills, verifystateev | 2026-11-02 | clean | delete |

Also: `~/code/d2lod-wt/baselineruns-20261102` is a 48 MB directory that is not a registered worktree (V, `git worktree list`); delete after a look, or `git worktree prune` does not cover it.

## 4. Unmerged agent branches (all local; keep unless noted)

Date is the last commit; "ahead" is commits not in origin/main.

| branch | date | ahead | purpose | note |
|---|---|---|---|---|
| b2rebase2 | 11-03 | 3 | batch 2 bundle rebase | |
| covarmor | 11-02 | 1 | armor/itemtypes coverage trace | |
| covmonai | 11-02 | 2 | monster AI coverage pass 3 | |
| covmonskills | 11-02 | 3 | monster skills coverage pass 1 | |
| covmonskills2 | 11-02 | 5 | monster skills pass 2 | staged, uncommitted changes in tree |
| covpal | 11-02 | 1 | Paladin coverage pass | |
| covstates | 11-02 | 2 | states coverage trace | staged, uncommitted changes in tree |
| covtreasure2 | 11-02 | 1 | treasure rows, differentials | staged, uncommitted changes in tree |
| crfixbundle2 | 11-02 | 4 | CodeRabbit fix bundle | |
| dmgroll | 11-03 | 11 | damage roll rounds, harness | an earlier part was merged (8e99f3c6), later rounds not |
| gate1run-20261102 | 11-02 | 1 | Gate 1 run record (NOT PASSED, Belt) | a different card text says PASSED on main; check which is current |
| gate1run-20261103 | 11-03 | 5 | Gate 1 run | |
| gate1run-20261103-0338 | 11-03 | 2 | Gate 1 run 0338 | |
| gate2analysis | 11-02 | 1 | Gate 2 failure analysis | |
| gate2run-20261102 | 11-02 | 2 | Gate 2 run record | |
| kittrip | 11-02 | 4 | kit trip work | |
| levelspawn | 11-02 | 6 | level spawn coverage | |
| lockstepp4run | 11-02 | 1 | local harness patch, "not for merge" | candidate to delete after the lead agrees |
| msvcc2026 | 11-02 | 3 | MSVC C2026 trace (conclusion wrong, see section 2) | apply banner, then merge or archive |
| prbuildground | 11-02 | 2 | ground/burst PR draft | |
| reverifymissile | 11-02 | 3 | missile re-verification | first part merged (a43549d5), later commits not |
| skdefshieldd2d | 11-02 | 3 | SKDEFSHIELD d2d | staged files in tree look like a whole-repo snapshot (CHANGELOG, AGENTS.md, .coderabbit.yaml): inspect before touching |
| specdelta | 11-03 | 2 | gate-judge spec delta, report line | |
| verifyb2 | 11-02 | 5 | batch 2 verification report | |
| verifycovmissiles | 11-02 | 1 | verify missiles coverage | |
| verifycovstates | 11-02 | 1 | verify covstates | |
| verifydmg, verifydmg2, verifydmg4 | 11-02/03 | 3/4/4 | damage roll verification rounds | |
| verifyitemweapons | 11-02 | 1 | recount of weapons pass | |
| verifylevelsareas | 11-02 | 1 | level areas check | |
| verifymonsterpace | 11-02 | 4 | monster pace check | |
| verifyphys | 11-02 | 5 | physical damage check | last commit message "merge origin/main into none" |
| verifyskassassin, verifyskbarb, verifyskdruid | 11-02 | 1/1/2 | skill verifications | verifyskbarb has staged files in tree |
| zerohitrun | 11-02 | 3 | zero-hit run | |

U: whether any of these were squash-merged under another name (only `Merge <branch>` messages were searched).

## 5. Ops repo (d2lod-ops, branch master, 1 commit ahead of the ops remote master)

- Untracked `tools/<winworker dir>/b18-b2r2.cmd`, `b18-b2r2base.cmd`, `b18-bundle.cmd`, `b18-mdr4.cmd` (256 to 270 bytes each): Windows build-worker ctest job files, same shape as the tracked `b18-base.cmd` and `b-*.cmd` (V). Recommendation: commit them (they document how the batch 2 numbers were produced); `b18-tmai.cmd` is already tracked.
- One local commit (979b83d, gate-judge Gate 1 note) is not pushed to the ops remote: push it when the lead allows.

## Shared-file lines for the merger

See openspec/changes/housekeep/merge-lines.txt.
