# Whole-project cleanup plan (read-only inventory, 2026-11-05)

> status: proposal. Nothing in this pass deleted, renamed or pushed anything. Full lists: `docs/process/cleanup-inventory-20261105.tsv` (kind, name, status, ahead, last-date, proposed action). Claims are tagged V (verified with the named git/ls/grep command on origin/main 28cfbb0c and the local worktrees on 2026-11-05) or U (unverified, needs the owner).

Goal (user request 2026-11-05): before the ladder coding starts, tidy the repo, drop what is not needed, and give a nice starting point. Constraints that bind every batch: docs are never deleted (archive with `git mv` and a one-line stub, or a `> status:` banner); agents never push; remote branch/tag deletion needs the user's go; tag `openspec-final` stays; the user's own caches and game libraries on the worker host are out of scope; the Gate 2 runs on the worker host are live (do not touch the lab directory); other sessions have staged files in the private ops repo.

## 1. Summary counts (V)

| What | Count | Note |
|---|---|---|
| Local branches (public repo) | 60 | 54 `agent/*`, 1 `grill/*`, 1 `mergeq/work`, 3 `pr-*`, `main`. (`git for-each-ref refs/heads`) |
| Merged into origin/main | 11 | MERGED-SAFE; 4 of them have a dirty worktree |
| RECORD (gate1run-*, gate2run-*) | 25 | one record commit each (gate1run-20261103 has 5, merge commits) |
| UNMERGED-UNIQUE | 19 | docs/research, tools/emu, coverage ledger rows, one code branch (kittrip), one gclock patch |
| UNMERGED-ARCHIVE | 5 | only a retired card/STATUS differs, or pr-19/22/23 trace docs |
| Remote heads | 2 | `main`, `gh-pages`. No remote agent branches exist, so batch E is nearly empty |
| Tags | 2 | `openspec-final` (keep), `pages-history-pre-code` |
| Worktrees (public repo) | 60 | plus 4 detached repro checkouts in the merge-queue cache, 1 merge worktree |
| Dirty worktrees | 5 + 2 ops | b2bundle, ggcards, noareameasure, sorcdiff (all merged branches), skdefshieldd2d (384 staged entries, unmerged); two ops-repo worktrees have untracked `tools/emu/` |
| Merge-queue repro dirs | 4 | 205 MB total with queue files; two hold 411 changed paths |
| Stale session scratch | 1.1 GB | in the per-user temp dir (briefs and scratch; session-owned, only size listed) |
| Tracked files | docs 1518, internal 1178, tools 824, cmd 272, backlog 49, assets 50, web 21, deploy 19 | `git ls-files <dir>` |
| `.git` size | 122 MB | no gc needed (no garbage) |
| Dead-code candidates | 2 packages, 1 cmd, 3 scripts | `internal/memfd`, `internal/merc/sim`, `cmd/lockstepscan`; scripts below |
| Misplaced binaries | 4 | `internal/{combat,input,nav,town}/cmd/*` are `main` packages |
| docs not in `docs/INDEX.md` | 75 of 270 top-level `.md`; 379 of 911 `.md` overall | method: file name not found as text in INDEX |
| Broken relative markdown links | 9 of 1310 links checked | listed in the TSV (`broken-link`) |
| Docs with a `> status:` banner | 603; 36 say superseded | `git grep -il '^> status:' -- docs` |
| backlog tasks | 18 | 9 In Progress, 7 Deferred, 2 Proposed |
| backlog/merge-lines | 30 files, never pruned | |

## 2. Branches (V: `git branch --merged origin/main`, `git merge-tree --write-tree` against origin/main to test whether a branch still adds anything)

A raw "ahead" count over-states unique work: many agents merged `origin/main` into their branch and the OpenSpec cards they added were later removed from main. After ignoring `openspec/`, `docs/plan/STATUS.md`, merge-lines and `docs/OWNERS`, the residual per branch is:

| Class | Branches | Evidence | Proposed action |
|---|---|---|---|
| MERGED-SAFE (11) | b2bundle, cleanplan (this one), floormeasure, ggcards, noareameasure, shimupdate, sorcdiff, specledger, grill/0005-automation-ladder, mergeq/work, main | `--merged` | Delete the 3 clean ones (floormeasure, shimupdate, specledger) in batch A. Keep cleanplan, grill, mergeq/work until closed. The 4 dirty worktrees: the owner first saves or discards (details in the TSV). Local `main` is behind origin/main: fast-forward it. |
| RECORD (25) | gate1run-* and gate2run-* (see TSV) | one generated "Gate N run ..." commit; 19 have residual 0 (record already on main under docs/soaks and docs/probes/soak-logs) | Bundle, then delete. 6 branches (gate1run-20261101-1625, -20261102, -20261103, -20261103-0628, s19-...-ms2, gate2run-20261102) carry a record file that is not on main: copy `docs/soaks/<run>.md` and the minutes file onto main first (batch B), or they stay in the bundle only. |
| UNMERGED-ARCHIVE (5) | groomland, reverifymissile, pr-19, pr-22, pr-23 | residual 0 (groomland, reverifymissile); pr-* hold 1 trace-doc edit each | Bundle, then delete. pr-* need an owner glance (U): later pal/durability traces on main probably supersede them. |
| UNMERGED-UNIQUE (19) | barbdiff, covmonai, covmonskills, covmonskills2, covstates, covtreasure2, levelspawn, skdefshieldd2d, verifyskbarb, zerohitrun (coverage traces + `tools/emu` helpers + ledger rows), dmgroll (46 files: leakcheck-clean wording after the merge, plus 12 commits), verifydmg, verifyphys, prb, prc, prpal, stagea (PR body and verifier drafts), kittrip (8 code files), lockstepp4run (gclock patch, "not for merge") | `git merge-tree` shows these files absent or different on main (V). `tools/emu` does not exist on origin/main at all, so the emulator helpers live only on these branches (V: `git ls-tree`). | DECISION: land the coverage/trace branches through the merge queue (docs and emu tools), or accept loss. Default: bundle everything, then ask. kittrip code: owner reviews against main (U). lockstepp4run: bundle only. |

Remote (V: `git ls-remote --heads origin`): only `main` and `gh-pages`; nothing to delete remotely. Tags: `openspec-final`, `pages-history-pre-code` (keep both unless the user says otherwise).

## 3. Worktrees and caches (V: `git worktree list`, `git status --porcelain`, `du -sh`)

| Item | State | Action |
|---|---|---|
| ~54 `~/code/d2lod-wt/<name>` worktrees | all branch-attached; about 50 clean | remove together with their branch (batch A for merged, after the bundle for the rest) |
| Dirty: b2bundle (`.scr/`), noareameasure (3 untracked `tools/emu` scripts), sorcdiff (`scratch-sd/`), ggcards (7 modified files), skdefshieldd2d (384 staged) | owners' unsaved work | keep until the owning session says; copy untracked files into the bundle directory first |
| merge-queue repro dirs (4, 52 MB each, 2026-11-03) | detached; two have 411 modified paths (a conflict repro) | `git worktree remove --force` when the queue is idle; they are ours |
| merge worktree and its queue/lock files | live | leave |
| Per-user temp scratch | 1.1 GB | list only; sessions clean their own |
| Private ops repo | 36 worktrees, 28 branches merged, 35 unmerged, 82 tool scripts, state dir 128 MB (94 MB api-agent, 34 MB plan-agent) | see batch F |

## 4. Repo root, code and tools (V unless tagged)

Root files: AGENTS.md, CLAUDE.md, README.md (120 lines), PLAN.md (historical, 43 references), CHANGELOG.md (80 KB), GLOSSARY.md, CONTRIBUTORS.md, THIRD_PARTY.md, LICENSE, `.coderabbit.yaml`, `local.env.example`, go.mod. Misplaced: **five python wheels** (`attrs`, `jsonschema`, `jsonschema_specifications`, `referencing`, `rpds_py`, 0.5 MB) and `.run28.sh` are tracked at the root; the wheels came in with the CodeRabbit setup commit and nothing in `tools/` references them (V: `git grep jsonschema -- tools` empty). Top-level `openspec/` is a one-file stub (README); it can stay until the removal plan is closed.

Go packages (42 under `cmd/`, 77 under `internal/`; V: grep of import paths, no build run):

| Candidate | Evidence | Action |
|---|---|---|
| `internal/memfd` (4 files, 2 tests) | no importer; mentioned in CHANGELOG, `cmd/oog/fdprobe.go` (comment) and a container README | remove after reading the CHANGELOG entry (U whether it is a planned library) |
| `internal/merc/sim` (2 files) | no importer; named in `docs/merc.md` and `internal/merc/types.go` | owner decides: it is a simulator (U) |
| `cmd/lockstepscan` | zero references in tools, deploy, CI, docs | ask the lockstep owner |
| `internal/{combat,input,nav,town}/cmd/*` | `main` packages inside `internal/`, referenced by docs (1 to 4 files each) | not dead; move under `cmd/` (batch D, small link rewrite) |
| `cmd/d2multi`, `npcprobe`, `d2dfight`, `d2dmap`, `d2dnode`, `d2saves` | 1 to 2 doc references each, last touched 10-30 | keep, list in README map as "probes" |

`tools/`: 75 entries; 22 files mention `ops.sh` (the stubs that forward to the private ops checkout, kept by design). Unreferenced scripts: `mute-d2-watch.sh`, `env-test.sh`, `d2dfarm-test.sh` (tests that nothing calls). Many `container-*.sh` have 2 to 5 references and relate to the parked container work.

## 5. Docs, backlog, cards (V)

| Area | Finding | Proposal |
|---|---|---|
| `docs/` | 1518 tracked files, 272 at top level; INDEX.md is 371 lines, lists 195 of the 270 top-level `.md` | rebuild INDEX by area (section 7) |
| `docs/research/` | 410 files; 164 were added on 2026-11-02 alone; 501 files reference the path `docs/research/` | do NOT move the directory (link impact too high). Add `docs/research/README.md` index by topic; move only finished PR-draft/verifier sets into `docs/research/archive-batch2/` with stubs (decision) |
| `docs/probes/` (450), `docs/soaks/` (37), `docs/process/` (47), `docs/d2d-probes` (38), `docs/sme` (32) | probes and soaks are records | keep; add a README per directory |
| Case collision | `docs/BACKLOG.md` (69 references) and `docs/backlog.md` (8) differ only by case: breaks checkout on case-insensitive systems | rename `docs/backlog.md` to `docs/backlog-howto.md` (8 references) |
| Dated names | `docs/plan/plan-20261103.md` (2 refs), `docs/process/*-20261104.md` (6), `docs/g1-log.md` (44), `docs/plan/plan-v2.md` (19) | keep names; add status banners; a date in a name is acceptable for a record |
| Banners | 603 files already carry `> status:` | audit only the 75 not in INDEX for a missing banner |
| Broken links | 9, first: `docs/INDEX.md` and `docs/README.md` point at a generated `needs-you.md` that is not tracked; four in `docs/research/` use paths relative to the wrong directory | fix in batch B |
| `backlog/tasks` | 18 tasks: 9 In Progress, 7 Deferred, 2 Proposed; task-5..8 are "fix for upstream" items likely overtaken by the upstream PR batches (U) | each owner confirms; close or retitle in a task, not here |
| `backlog/merge-lines` | 30 files; the merger applies them but never removes them | prune files whose lines are already in CHANGELOG/INDEX (script outline in batch B) |
| `docs/cards.md`, `docs/OWNERS` | cards.md is generated (66 lines); OWNERS 71 lines still lists OpenSpec-era rows | regenerate cards.md; trim OWNERS to live rows (decision: OWNERS is also in the guard set?) |

## 6. Private ops repo (generic summary)

82 scripts under `tools/`, one unreferenced (`mute-d2-watch.sh` twin); handoff notes from 2026-11-01 and 2026-11-03 (three files) are stale and should be archived into an `archive/` folder with a stub; `briefs/` holds 3 API permission-probe files that are one-off; `claude/agents` has a retired `spec-sme` stub (the public brief already says do not launch). 36 worktrees, many for merged agent branches. Large state directories (api-agent 94 MB, plan-agent 34 MB) hold run logs; rotate rather than delete. Other sessions have staged files there: no cleanup in this repo until they commit. No names or hosts go into public docs.

## 7. Proposed starting point

Order a newcomer or agent should read: (1) `README.md`, (2) `AGENTS.md`, (3) `docs/START-HERE.md`, (4) `docs/plan/STATUS.md`, (5) the area index. Proposal:

1. `README.md`: keep the Status text, add a 10-line "Where to look" map (table below) at the end.
2. `docs/START-HERE.md` (new, one page, ~60 lines): the map, the reading order, the three commands every agent runs (`tools/ci/check.sh`, `python3 tools/contracts/check.py`, `python3 tools/backlog/check.py`), and where decisions live. It links to INDEX; it is not a second INDEX.
3. `docs/INDEX.md` regenerated by area with a script (`tools/cleanup/build_index.py`, new): reads each doc's first heading and `> status:` line, lists every `.md` once, and groups by the table below. Replaces the hand-kept 371-line list; the 75 missing docs get covered automatically.
4. `AGENTS.md` reading order: today it starts with `openspec/README.md` and change cards, which are retired (V: openspec dir is a removal stub). The "Read in this order" paragraph must be rewritten to: README, AGENTS, STATUS, your backlog task, GLOSSARY, `docs/d2-1.14d-memory.md`, your area doc. That is a guard-adjacent edit: the user or the coordinator makes it.

| Area (INDEX section) | Entry docs |
|---|---|
| Product direction: ladder | grill 0005 record, ADR 0015/0016, `docs/plan/` |
| Engine and fidelity (d2d, game.exe) | `docs/d2d-*.md`, `docs/coverage/`, `docs/research/` traces, `docs/d2-1.14d-memory.md` |
| Bot code | `docs/bot.md`, `docs/combat.md`, `docs/town*`, `internal/` map |
| Ops, merge, CI | `docs/process/`, `docs/ci.md`, `docs/backlog.md` (to be renamed), `docs/OWNERS` |
| Gates and soaks | `docs/contracts/`, `docs/soaks/`, `docs/probes/` |
| Community | `docs/community/`, `docs/contributing/`, `CONTRIBUTORS.md` |
| Archive | `docs/archive/` |

Proposed renames (old -> new, references found by `git grep -l`): `docs/backlog.md` -> `docs/backlog-howto.md` (8); `internal/combat/cmd/combatlive` -> `cmd/combatlive` (4); `internal/town/cmd/townlive` -> `cmd/townlive` (4); `internal/input/cmd/inputcal` -> `cmd/inputcal` (1); `internal/nav/cmd/navlive` -> `cmd/navlive` (1); root wheels and `.run28.sh` -> removed (0 references). No other rename is proposed: the large doc sets (research 501 references, g1-log 44, PLAN 43) are cheaper to bannerise than to move.

## 8. Execution batches (order A to F; each ends with a verification and a commit)

| Batch | What | Risk | Needs user |
|---|---|---|---|
| A | Local branch and worktree cleanup (no data loss): create the bundle, remove merged-clean worktrees and branches, remove the repro checkouts, fast-forward local main | low if the bundle is verified | go for the bundle step only |
| B | Docs: archive moves with stubs, record files from RECORD branches onto main, INDEX rebuild, merge-lines prune, link fixes, START-HERE, README map | low (docs only), medium for link churn | decision on research/ split |
| C | Dead code: remove the stray wheels, `.run28.sh`, unreferenced scripts, `internal/memfd`, `internal/merc/sim`, `cmd/lockstepscan` | medium: needs per-package vet and test on the worker host | owner OK for each package |
| D | Renames with a link rewriter and a link check | medium | confirm rename list |
| E | Remote: nothing to delete (only main and gh-pages exist); only a possible decision on `pages-history-pre-code` | none | yes before any tag drop |
| F | Private ops repo: archive stale handoffs, prune merged worktrees/branches, rotate state logs, drop duplicate scripts | medium (other sessions have staged files) | go, after the staged work is committed |

### Batch A commands (outline)

```sh
# 1. one bundle of every unmerged tip + the dirty extras, before deleting anything
mkdir -p ~/code/d2lod-archive
cd ~/code/d2lod-bot
git for-each-ref --format='%(refname)' refs/heads | grep -v -E 'refs/heads/(main|mergeq/work|grill/|agent/cleanplan)' > /tmp/refs.txt
git bundle create ~/code/d2lod-archive/unmerged-20261105.bundle $(cat /tmp/refs.txt)
git bundle verify ~/code/d2lod-archive/unmerged-20261105.bundle
git bundle list-heads ~/code/d2lod-archive/unmerged-20261105.bundle | wc -l   # equals the number of refs
# untracked files in dirty worktrees: tar them next to the bundle
tar czf ~/code/d2lod-archive/dirty-extras.tgz -C ~/code/d2lod-wt b2bundle/.scr noareameasure/tools/emu sorcdiff/scratch-sd
# 2. delete merged-clean worktrees and branches (never --force; git refuses a dirty one)
for b in floormeasure shimupdate specledger; do git worktree remove ~/code/d2lod-wt/$b && git branch -d agent/$b; done
# 3. after the bundle verified: remove RECORD and ARCHIVE worktrees/branches
for n in $(awk -F'\t' '$1=="branch" && ($3=="RECORD"||$3=="UNMERGED-ARCHIVE"){print $2}' docs/process/cleanup-inventory-20261105.tsv); do git worktree remove ~/code/d2lod-wt/${n#agent/}; git branch -D "$n"; done
# 4. repro dirs when the queue is idle
for d in repro10 repro10b repro-covmonskills2 repro-covtreasure2; do git worktree remove --force ~/.cache/d2-mergeq/$d; done
git worktree prune
```

Verify nothing is lost: `git branch --no-merged origin/main` before and after (the list shrinks only by bundled names), `git bundle list-heads` contains every deleted name, and a test clone `git clone bundle /tmp/x` shows the tips. The UNMERGED-UNIQUE branches are not deleted in A: they wait for the user's land-or-bundle decision (then the same loop).

### Batch B (docs)

1. For RECORD branches with record files not on main: `git checkout <branch> -- docs/soaks/<run>.md docs/probes/soak-logs/<run>-minutes.jsonl.gz` in a docs branch.
2. `python3 tools/cleanup/build_index.py > docs/INDEX.md` (new script; sections per section 7; check that every tracked `.md` appears once).
3. Archive moves only for docs a banner already marks superseded: `git mv docs/X.md docs/archive/X.md`, stub `> status: moved to docs/archive/X.md` left behind (the stub keeps the 62 existing archive references valid).
4. Prune merge-lines: list files whose lines already occur in CHANGELOG.md/INDEX.md (`grep -F -x -f`), `git rm` those only.
5. Fix the 9 links; run the link checker (the same script as this inventory, to be committed as `tools/cleanup/linkcheck.py`).
Verify: link check reports 0, `python3 tools/contracts/check.py`, `tools/ci/check.sh --hygiene-only`, leakcheck 0.

### Batch C (dead code)

For each package: `git rm -r`, then on the worker host `go vet ./... && go test -short -count=1 -p 4 ./...` run from inside the worktree. Wheels and `.run28.sh`: `git rm` (one commit, no Go impact). Verify: whole-repo vet/test green and no doc still points at a removed path (`git grep -F internal/memfd`).

### Batch D (renames)

`git mv`, then a rewrite script (`git grep -l -F old | xargs sed -i 's#old#new#g'`) restricted to the files listed in the TSV, then the link checker and `go build ./cmd/...` on the worker host. Case-collision rename first; binary moves second.

### Batch E / F

E: none unless the user wants `pages-history-pre-code` gone (`git push origin :refs/tags/pages-history-pre-code`; the user runs it). F: bundle the ops repo's unmerged branches the same way, `git worktree remove` the 28 merged ones, archive the three stale handoffs with `git mv` into an `archive/` directory, rotate `state/` logs by gzip, remove duplicate scripts after a reference grep. Wait until the staged files of the other sessions are committed.

## 9. Decisions needed from the user

1. Create ONE bundle of all unmerged branch tips and then delete the local record and archive branches (default yes).
2. The 19 UNMERGED-UNIQUE branches: land the coverage/trace/emulator-helper branches through the merge queue first, or bundle-and-delete? (`tools/emu` exists nowhere on main.)
3. `docs/research/` (410 files): index only, or also move finished PR-batch sets to an archive subfolder with stubs?
4. Remove `internal/memfd`, `internal/merc/sim`, `cmd/lockstepscan` (owners unconfirmed), and the five root wheels and `.run28.sh`?
5. Rename `docs/backlog.md` -> `docs/backlog-howto.md` and move the four `internal/*/cmd` binaries under `cmd/`?
6. Rewrite the AGENTS.md reading order (still names the retired OpenSpec files)? Coordinator or user edit.
7. Drop the tag `pages-history-pre-code` (the user's go; the default is keep)?
8. Ops repo cleanup (batch F) after the other sessions commit their staged files.
9. The five dirty worktrees: who owns them (b2bundle, ggcards, noareameasure, sorcdiff, skdefshieldd2d)?

## Shared-file lines for the merger

See `backlog/merge-lines/agent-cleanplan.txt`.
