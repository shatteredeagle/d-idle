# Backlog grooming pass, 2026-11-06

> status: done on branch agent/backloggroom (docs and task files only). Tags: V verified (read the file, ran the tool, git log), S secondary, U unverified. Rules: [../backlog-howto.md](../backlog-howto.md).

The user asked whether the board had ever had a real pass, what is old, and what work is missing. It had not: statuses lagged merges, labels still named the retired goals, one title was a bare `>-`, and 14 pieces of open work had no task. This page is the audit; the fixes are applied except the removals, which need the user.

## Headline

- 33 tasks audited, 14 created (TASK-34 to TASK-47), 47 on the board. `python3 tools/backlog/check.py` clean, `cards_md.py --check` ok.
- Existing 33: 22 kept as they are (after the label mapping), 3 statuses fixed (3, 13, 32), 2 titles fixed (18, 24), 1 box ticked (23), 10 notes added, 2 proposed archives (15, 24), 2 proposed "reword, then Done" (12, 14), 1 proposed "finish or defer" (9), 1 optional archive (22).
- Done is set on no new task: every merged-but-not-Done task (32, 31, 23, 22, 9) has an open box or is a false positive (below).

## Causes (V)

- Stray "labels" TASK30, TASK31, TASK23: not reproducible. Every task file on main and in the history of tasks 23, 25 and 31 has clean labels (V: `git show` of each version), `backlog task list --json` and `task 23 --plain` show `Labels: G-F` only, and `new_task.py` passes only `-l` values. TASK-23 and TASK-25 carry those ids in `dependencies:` (and TASK-31 depends on TASK-23). The pinned web board shows dependencies next to the labels (S, not checked in a browser), so the likely source is a dependency chip read as a label. Fix: labels are now checked against a known set (an unknown label is a finding), and the howto says dependencies are never labels.
- Bare `>-` title (TASK-24): V. The CLI rewrites any title over about 90 characters as a folded `title: >-` block on its next edit (`--append-notes` re-folded it during this pass), and the checker read `>-` as the card id. Fixed three ways: `check.py` now fails a folded title, `new_task.py` unfolds on create (TASK-34 was folded by the CLI and caught), the howto says to keep titles under 80 characters.
- TASK-18 title `TASK-A mana-restock`: the card id is `taska`; renamed `taska: caster mana potion restock`.
- Status lag: no task existed for several jobs and merge never set statuses; rule added to the howto (task before launch, coordinator sets status at merge).
- Quote and format noise: only what `backlog task edit` rewrites anyway changed (single quotes, `updated_date`); no bulk reformat.

## Label scheme (applied)

Tiers T0..T5 for ladder work (ADR 0016), area labels `watch tester gate upstream bot-defect drills tooling decisions onboarding`, modifiers `soak parked`. G-A..G-G are gone from all tasks; the checker still accepts them. Config allows labels freely (V: the CLI and MCP do not validate), so the mapping was applied: G-A gate; G-B to T4 for the Act 1 cards and bot-defect for TASK-18; G-C bot-defect (10, 11) or drills (9); G-E upstream; G-F tester (23 to 33), tooling (12, 22); G-G T0 (19, 20, 21). Documented in docs/backlog-howto.md, Labels.

## Audit table (existing 33 tasks)

Acceptance is ticked/total after this pass. Last activity is the newest commit on origin/main touching the task's owns paths (V, `git log`), which can come from other tasks' work.

| id | title (short) | status | labels | acc. | last activity | verdict and evidence |
|---|---|---|---|---|---|---|
| 1 | m3-act1run | Deferred | T4 soak | 0/3 | none | keep: ADR 0016 item 24 keeps it as T4; docs/act1-run.md does not exist |
| 2 | m3-andariel | Deferred | T4 soak | 0/3 | 11-05 (boss pkg, other work) | keep: T4; only a research note exists |
| 3 | m3-merc | Deferred (was Proposed) | T4 soak | 0/4 | 11-05 (other work) | status fixed: Proposed next to two Deferred T4 siblings |
| 4 | gameexe-retire-soak | Deferred | gate | 0/2 | none | keep: Gate A card, not ranked until T4 |
| 5 | armorhit | Done | upstream | 6/6 | 11-01 | keep: upstream PRs merged (#177 batch) |
| 6 | nexthit | Done | upstream | 6/6 | 11-01 | keep: same |
| 7 | skmiss | Done | upstream | 5/5 | 11-06 | keep: same |
| 8 | skresistfix | Done | upstream | 6/6 | 11-01 | keep: same |
| 9 | m1-liveproof | In Progress | drills soak | 13/15 | 11-01 own work | finish or defer: stalled; open AC 14 (verdicts in docs/primitives.md) and 15; one docs job closes it (user: finish or defer) |
| 10 | fixbelt | Done | bot-defect | 6/6 | 11-06 (other work) | keep: result says not live, recipe in docs/fixbelt.md |
| 11 | fixprogress | Done | bot-defect | 4/4 | 11-06 (other work) | keep |
| 12 | ledgerkey | In Progress | tooling | 2/3 | 11-01 | reword then Done: code merged 11-01; AC 3 is a private ops item, so it cannot be ticked here. Proposal: remove AC 3, Done (task text is not mine to change) |
| 13 | d2d-container-soak | Deferred (was Proposed) | parked soak | 0/3 | none | status fixed; keep parked: step 9 needs the user's go, and the tester build now covers the container route |
| 14 | d2dmesh | Deferred | parked | 7/7 | 10-30 | propose Done: prototype merged, all boxes ticked, nothing deployed by choice; or leave parked. User decides |
| 15 | stream-grid | Deferred | parked | 4/7 | 10-30 | propose archive: the result line names tools/publish/grid.py and scene-grid.* but they are on neither main nor any branch (V, `git log --all`); claim cannot be checked |
| 16 | m2-roam | Deferred | parked soak | 0/4 | 11-01 | keep parked: Runner-tier (T2) work, relabel when T2 starts |
| 17 | m2-tripwp | Deferred | parked soak | 0/3 | 11-05 (other work) | keep parked: same; trip logic changed since (fixprogress), re-read before launch |
| 18 | taska (mana restock) | Done | bot-defect | 7/7 | 11-06 | title fixed |
| 19 | decision-point-store | In Progress | T0 | 3/5 | 11-05 | keep: AC 4 (rule file expresses today's defaults) and AC 5 (docs/decision-points.md, which does not exist) are open |
| 20 | watch-screen | Done | T0 watch | 5/5 | 11-06 | keep |
| 21 | t0-slice | In Progress | T0 | 3/5 | 11-05 | keep: AC 2 and 3 (live fight-time question, learned answer) open; new TASK-36 carries the engineered drill |
| 22 | decisions-mirror | Deferred | tooling decisions | 0/4 | none | keep Deferred (drift D is a false positive: the merge was the card itself; decisions_mirror.py does not exist). Optional archive: low value |
| 23 | testerwatch | In Progress | tester watch | 2/3 | 11-06 | AC 2 ticked (flag and tests on main, V); blocked on AC 3 (plan gaps G3, G4 result text) |
| 24 | testerimage (Wine) | Deferred | tester | 0/2 | 11-06 | title fixed; propose archive: superseded by TASK-31 (d2d first, 11-06) |
| 25 | testerlive | Proposed | tester | 0/3 | none | keep: waits for 31, 32, 33 |
| 26 | testerdocs | Proposed | tester | 0/2 | none | keep |
| 27 | testerreport | Proposed | tester | 0/1 | none | keep |
| 28 | testerlicence | Proposed | tester | 0/2 | none | keep |
| 29 | testerpack | Proposed | tester | 0/1 | none | keep |
| 30 | testertap | Done | tester watch | 2/2 | 11-06 | keep |
| 31 | testerd2dimage | In Progress | tester | 1/2 | 11-06 | keep: AC 1 needs the built image; note says the wizard hook and context.sh line are now due here |
| 32 | testerd2dwiz | In Progress (was Proposed) | tester | 1/2 | 11-06 | status fixed: merged (62d02892) but AC 1 open (1.14d hash list is empty, hook not applied) |
| 33 | testerrt | Proposed | tester | 0/2 | none | keep |

## Parked tasks 3, 13, 14, 15, 16, 17

3 and 13 now Deferred with the rest. 14: propose Done. 15: propose archive. 16 and 17: keep parked; they are T2 Runner work and fit ladder-t2 (TASK-38) when T2 starts. None of the six ranks work under ADR 0016.

## Duplicates and contradictions

- TASK-24 (Wine image) vs TASK-31 (d2d image): same job after the d2d-first decision; propose archive of 24.
- The Wine plan appendix in docs/research/tester-build-plan.md stays as history.
- docs/plan/STATUS.md hand-written part (last edited 11-03) still ranks by G-A..G-F and says no gate runs; new TASK-46. docs/cards.md prose and docs/backlog-howto.md "what is where" were corrected here.
- docs/BACKLOG.md still points at `openspec/changes`; it is a shared file, not touched (merger lines only).

## New tasks (TASK-34 to TASK-47)

| id | task | label | note |
|---|---|---|---|
| 34 | ordersux2: orders slice 2 | T0 watch | until-level, clear then home, queue, conflict card, Forgotten Tower Cellars |
| 47 | ordersux3: orders slice 3 | T0 watch | reorder, dismiss, cleared from nav, Acts 2 to 5, screen reader; depends on 34 |
| 35 | dpwire | T0 | wire the unwired decision points (count differs: 9 wired, 22 or 23 in the catalogue; first step reconciles) |
| 36 | t0fight | T0 | engineered live drill: fight-time question, learned answer (TASK-21 AC 2, 3) |
| 37 | ladder-t1 | T1 | cards and wiring for t1.identify, t1.keep.kind |
| 38 | ladder-t2 | T2 | urgency stage 2, t2.failstreak, t2.reset.game; altar stays parked |
| 39 | earlydeath13 | bot-defect | seed 13 clvl 4 death, deterministic in g2d, g2e, g2f |
| 40 | tpreserve-live | bot-defect | prove the scroll reserve with the tp field |
| 41 | g2f-record | gate | g2f result and diagnosis; ends "Needs the user" |
| 42 | fork-labverbs | tester | gate mutating debug verbs behind a fork flag; fork-branch exclusive |
| 43 | playsession | T0 tooling | one command for a LAN play session |
| 44 | onboarding-build | onboarding | placeholder until agent/onboardux lands |
| 45 | upstream-next | upstream | one batch at a time: off-weapon ED, Energy Shield leftovers C2/C3, monster-versus-monster |
| 46 | statusrefresh | tooling | STATUS.md hand-written part |

Checked and skipped because already tracked: TASK-19 AC 4 and 5 (rule file defaults gate, docs/decision-points.md); the 1.14d hash pin and the d2dsetup entrypoint hook with the context.sh build line (TASK-32 AC 1 and TASK-31 AC 1; both owns cover the files); Gate 2 re-run decisions (user items).

## Proposed removals and changes (need the user)

| id | proposal | why |
|---|---|---|
| 24 | archive | superseded by TASK-31 |
| 15 | archive | result claims files that are not in git |
| 14 | mark Done (or keep parked) | all boxes ticked, prototype merged, deployment parked |
| 12 | drop AC 3 (private item), then Done | code merged 11-01 |
| 9 | finish AC 14 as a docs job, or defer | stalled since 11-01 |
| 22 | optional archive | no ranked value, not started |

Agents never run `backlog archive`; the coordinator does after the user's yes.

## User-owned items (no task asks an agent to decide these)

- ADR 0003 amendment (the user's grill), GPL reviewer named or waived, and when the fork repo goes public (TASK-28 lists them).
- Gate 2: seeds 5 versus 3 for a real claim; whether Gate 3 runs (needs the user's go). TASK-41 collects the facts.
- Which upstream batch is next (TASK-45 prepares the table).

## Ops-private items (details live in the private repo; no public task)

- The live-run wrapper does not forward the run id to the bot.
- Some private tool tests fail on the ops repo head.
- The private ledger tools should key by instance (TASK-12 AC 3).
- The Homepage tile and proxy host entry for the backlog board live in the portal repo (docs/process/backlog-board-nas.md step 7).

## Not verified

The board in a browser (no browser here); live state of the g2f run (still running at the time of writing); whether any agent branch holds the stream-grid code (searched local and remote-tracking branches only).
