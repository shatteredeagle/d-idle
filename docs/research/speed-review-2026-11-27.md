# Speed review: where the wall time goes, and the plan to cut it (2026-11-27)

> status: current. Card speedreview (TASK-1025). Consumer: the coordinator session and the owner. Reviewer run, read-only on code.
> Tags: V = measured or read in this run (the how is named), S = derived from V numbers or from a doc, U = not verified.
> Window: the 48 h ending 2026-11-26 17:02 EST. All clock times are EST. Hosts are named by role: head box (4 cores, 4 GB), builder (6 cpu cap, one job at a time), worker host (16 cores).

## Executive summary

The merge queue is not the bottleneck. The worker was busy 13.3 h of 48 h (28 %) and a clean change goes from queued to merged in a median of 16 min. The time goes elsewhere:

1. **Verification, not merging.** 35 features sit in the dark-launch ledger, 0 on trial, median age 7 days, oldest 27 days. The flip rule allows one 24 h window per sandbox and one knob per window: 2 flips a day, so the 29 coordinator rows need about 15 days. The flip tool has never run on a real sandbox.
2. **Rework.** 19 % of branches (20 of 105) needed more than one queue entry. A failure adds about 65 min (median) of waiting for someone to notice, fix and requeue. 8 of 10 merge conflicts were in generated or duplicated data files and are mechanically resolvable.
3. **One false red CI run held the queue 4 h 43 min** in the middle of the day (41 % of all queue wait in the window). The rerun was green.
4. **Deploy lag.** The deploy command takes 5 to 6 s, but merged code reached the quest sandbox after a median of 69 min (p90 5.3 h); the other sandbox shows one release in 72 h.
5. **Fixed gate cost.** A Go change pays 86 s preflight + 191 s whole-repo test + 188 s race of `internal/bot`, serially, on the builder. Flags that halve this are built but off.

Plan: flip in bundles on lab evidence, auto-resolve generated conflicts, rerun red CI once, auto-deploy to a canary, turn the gate flags on. Targets: merged to running on the canary in under 10 min, queued to merged median 6 min, first-pass merge rate 95 %, no dark row older than 14 days.

## 1. Measurements

### 1.1 Queue, 48 h (V: `queue.jsonl` and `worker.log` of the merge worker, script over `added`, `started`, `finished`, `result.timings`)

| Measure | Value |
|---|---|
| Entries added | 129: 102 merged, 22 failed, 5 dropped by hand |
| Branches | 105; 20 (19 %) needed more than one entry |
| Queued to merged, clean entry | median 976 s (16 min), p90 2445 s (41 min), max 11395 s (3 h 10 min) |
| Queue wait (merged entries) | median 340 s, p90 2103 s, max 11089 s; sum 1766 min |
| Run time (merged entries) | median 574 s, p90 772 s, max 1061 s; sum 936 min |
| Worker busy | 47926 s = 13.3 h of 48 h (28 %) |
| Peak throughput | 11 merges in one hour (11-26 13:00, with batches); serial solo ceiling is about 6 per hour (3600 / 575) |
| Queue depth when an entry was added | median 4, p90 7, max 13 |
| Entries that waited over 10 min / 30 min / 1 h | 37 / 15 / 5 of 102 |
| Batches | 16 batch passes, sizes 1 to 4, 273 to 941 s each; 3 batch gates failed and rolled back (8 entries rerun alone) |
| Last commit to queued (round trip agent to queue) | median 347 s, p90 1719 s; 5 entries over 1 h (V: commit time of the merged branch tip vs `added`) |
| One stream of serial fixes (`toolstrade55` to `90`) | 47 of 129 entries (36 %) |

### 1.2 Where one solo Go entry spends its 574 s (V: `result.timings`, medians, n = 69 solo entries)

| Phase | Seconds | Where it runs |
|---|---|---|
| preflight (full-tree leakcheck of about 11300 files, ownscheck) | 86 (p90 110, max 201) | head box |
| go vet + go test whole repo, `-count=1`, 220 packages | 191 (p90 223, max 555) | builder |
| go test -race of touched packages (58 of 69 entries ran it; it is `./internal/bot` almost every time) | 188 (p90 218, max 263) | builder |
| other gate work, merge, generated files, push, cleanup | about 110 | head box |
| CI wait inside the merge | 0 (pushed without waiting) | - |
| CI after the push (not blocking) | 401 median, 488 max (V: `gh run list`, 67 green runs) | GitHub |

The two builder steps add up (gates median 435 s = 191 + 188 + rest): they run one after the other although `MERGE_GATE_PARALLEL=1` is on, because the race step only runs next to the whole-repo step with `MERGE_RACE_PARALLEL=1`, which is off (V: `merge.sh` lines 558 to 572).

`internal/bot` is one package of 244 files, 44352 lines, 154 test files, no `t.Parallel()` (V: `wc`, `grep`). Its race run takes about 200 s by itself (V: last-fail log of entry 1408: `FAIL d2lod-bot/internal/bot 201.959s`). 192 of the file changes in the window were in this package.

### 1.3 Failures and rework (V: `reason` and `log_tail` of the 22 failed entries plus `worker.log`)

| Cause class | Entries | Detail |
|---|---|---|
| Merge conflict | 10 | `docs/knobs.md` 4; duplicated plan data 3 (`admission.json`, `povertyfarm.json`, `earlyroutes.json`, each kept as two byte-identical copies under `docs/plan/data/` and `internal/plandata/`); `cmd/d2scenario/invariants.json` 1; real code conflicts 2 (`internal/bot/kitplan_test.go`, `internal/bot/explain.go`) |
| go-whole gate | 5 | `TestKnobsDocMatchesTables` 2 (the knob table in `docs/knobs.md` was stale: deterministic, not flaky); `TestAgencyWireShadowLineAndView` + `TestAvoidListIsPerProfile` 1 (real, fixed and merged later); 2 without a test name in the stored tail |
| go-race gate | 3 | `TestReactionTimeDelaysInterrupt`, `TestArbiterControlGoldFarmAndRecovery` (failed in 0.09 s of a 202 s run; the branch changes one file), one without a name; all in `internal/bot` |
| Worktree had uncommitted changes when its turn came | 3 | the agent kept editing after queueing |
| Preflight | 1 ownscheck, plus in the log: 3 leakcheck hits on one entry (home path in text), 1 preflight over its 300 s budget | |

Flaky tests by name, 14 days (V: same fields): `TestReactionTimeDelaysInterrupt` 1, `TestArbiterControlGoldFarmAndRecovery` 1, `TestHeroNeverWaitsForTheInventoryView` 2, `TestVaultPollCost` 1; package `internal/watch` failed 6 times, `internal/bot` 4. The stored tails often lack the `--- FAIL` line, so the list is a lower bound (U for the rest).

Cost of rework (V for the per-branch times, S for the sum): the 18 branches that failed or were dropped and then merged took 11 to 188 min from first queueing to merged, median 80 min, against 16 min for a clean entry. That is about 20 h of added latency in 48 h. Failed entries burned 94 min of gate time; the three failed batches about 27 min more.

### 1.4 Holds (V: `worker.log`)

| When | What | Cost |
|---|---|---|
| 11-25 10:15 to 14:58 | `CI RED for pushed head 0cbd4fe0f: worker stops`. The test job failed on attempt 1 and passed on attempt 2 with the same commit (V: `gh run view`, attempts 1 and 2). The gate had passed the same tree. Which test failed is U (the attempt 1 log is gone) | 4 h 43 min; entries 1301 to 1304 waited 10601 to 11089 s each = 721 min, 41 % of all queue wait in the window |
| 11-26 03:17 to 08:33 | no entries (night), then three worker restarts for the flag cutover | none measured |
| "CI of main is pending / unknown" | 7 short holds | minutes each |

### 1.5 Deploy (V: read-only `docker logs -t` of both sandbox containers, release lines; `docs/dev-update.md` for the tool time)

| Measure | Value |
|---|---|
| Deploy command | 5 to 6 s with a warm cache (V in `docs/dev-update.md`, not remeasured) |
| Quest sandbox | 10 releases between 11-25 13:21 and 11-26 08:35; gaps 34 min to 6 h 17 min |
| Merged to next release on the quest sandbox | median 4120 s (69 min), p90 19030 s (5.3 h), n = 41 merges |
| Farm sandbox (Paladin) | one release line in 72 h of logs (11-25 08:17). S: it runs code more than a day behind main. U: both containers were restarted 11-26 10:57 and print no release line after that, so the running version was not read |
| Deploy log | none: `tools/dev-update.sh` prints `UPDATED ...` and deletes its temp log (V: script lines 63 to 74) |

### 1.6 Verification backlog (V: `data/darklaunch.json`, `docs/live-checks.md`, `tools/flip/flip.py`)

| Measure | Value |
|---|---|
| Dark-launch rows | 35, all `dark`; 0 `trial`, 0 `default-on` |
| Owner | coordinator 29, owner 6 |
| `criteria_met` | U 33, no 2, yes 0 |
| Age | median 7 days, 7 rows over 14 days, oldest 27 days |
| Flip rule | one open window per sandbox (flip.py line 249), one knob per flip, default window 24 h, two sandboxes: 2 flips a day. S: 29 rows = about 15 days if nothing reverts |
| Flip tool | never run on a real sandbox (V: result text of entry 1409 and step 7 of `docs/process/dark-launch.md`) |
| Open live-check rows | 104 |
| Accelerated lab | `wave.sh` SPEED 8: 10.2 virtual minutes in 33 s wall (V in `docs/process/second-worker.md`); wall limit per run `HOURS*60/SPEED*1.6+10` min, so 24 virtual hours fit in under 5 h wall and several arms run at once. `abrun.sh` arms are hard-coded to `bot.zoneModel` (V: line 13) |

### 1.7 The wall-time table: one change, from finished code to verified

| Stage | Typical now | Worst seen | Evidence |
|---|---|---|---|
| Agent work | U (branches arrive as one squashed commit; no start time recorded) | - | - |
| Last commit to queued | 6 min | over 1 h (5 times) | 1.1 |
| Queue wait | 6 min | 3 h (CI hold) | 1.1, 1.4 |
| Preflight + gates + push | 10 min | 18 min | 1.2 |
| Rework when it fails (19 % of branches) | +65 min | +170 min | 1.3 |
| CI after push | 7 min, not blocking | blocked everything 4 h 43 min once | 1.4 |
| Deploy to a sandbox | 69 min | 5.3 h; the farm sandbox over a day | 1.5 |
| Live verification of a dark feature | 7 days and counting | 27 days | 1.6 |

## 2. Root causes (structural)

| # | Cause | Evidence | Share |
|---|---|---|---|
| R1 | Verification is serial and wall-clock bound: one knob, one sandbox, 24 h, and no card owned most flips until 11-26 | 1.6 | days per change; the dominant term |
| R2 | A failure is handled by people: no automatic retry, regenerate or rerun, so each one costs a round trip | 1.3, 1.4 | 20 h latency + 4.7 h hold per 48 h |
| R3 | Generated and duplicated files are edited by hand in branches: the knob table, two copies of each plan data file, registry JSON without a union rule | 8 of 10 conflicts, 2 of 5 go-whole failures | 10 of 22 failures |
| R4 | `internal/bot` is one 44 k line package: every change pays its full race run, conflicts cluster there (`loop.go` in 19 of 102 merged branches, `kitplan_test.go` 14, `runner.go` 10), and scoped gates cannot skip much | 1.2 | 188 s per entry, 2 real conflicts |
| R5 | Gate steps run serially on a host limited to one job; preflight rescans the whole tree on the head box | 1.2 | about 270 s per entry that flags can remove |
| R6 | Deploy is a manual, batched step by a separate agent and leaves no log | 1.5 | 69 min median |
| R7 | The gate and CI run the same whole-repo tests; CI does not block, but one flaky CI result stops the queue | 1.2, 1.4 | 4.7 h once |
| R8 | One fix stream sends a third of all entries one by one; each pays the full gate and meets the same hot files | 1.1 | 47 entries |

Not a cause (V): queue capacity (28 % busy), post-push CI wait (0 s in the merge), the deploy tool itself (seconds), agents waiting for the queue (median wait 6 min).

## 3. Plan

### 3a. Immediate (today), ranked by speedup per effort

| # | Action | Exact step | Expected gain | Tag |
|---|---|---|---|---|
| I1 | Clear the six unmerged branches | `toolstrade89`: requeue unchanged (race flake, one-file diff). `admit2`: in its worktree `go run ./tools/knobdump`, commit, requeue. `admit3`: merge `origin/main`, resolve `internal/bot/explain.go`, queue after `admit2`. `toolstrade90`, `featuresoak` (no Go): queue. `toolstrade56` (one stray commit from 11-24): merge or delete | pileup gone in about 35 min of queue time | V for the causes, S for the time |
| I2 | Start the flip wave tonight instead of one knob per day | (1) `python3 tools/flip/flip.py <quest sandbox> quests.stepExec=1 --why "scenario green" --window 24h --dry-run`, then for real; same on the farm sandbox with `bot.arbiterControl`. (2) Triage all 35 rows in one pass with `python3 -I tools/darklaunch/report.py` into: retire, owner go (6 rows: one popup to the owner with all six), lab-provable, needs a live drill. (3) Ask the owner for decision D1 below | first two trials tonight; the tool gets its first real run | S |
| I3 | Turn on the built gate flags | in the worker env file add `MERGE_SCOPED=1`, `MERGE_SCOPE_FULL_EVERY=5`; and, with the gate host set to the worker host (`D2_LABRUN_HOST=<worker host>`, `D2_GATE_NICE=19`, `D2_GATE_P=12`), `MERGE_RACE_PARALLEL=1`; restart with `tools/mergeq.sh daemon stop; tools/mergeq.sh daemon start` between entries | gates 435 s to about 200 s (race next to the whole run saves about 190 s; scoped saves 60 to 90 s more on small diffs). The scratch test in the ops doc measured 122 s for a small item with all four flags | S; the worker-hours rule allows per-package runs at nice 19 at any hour; the every-fifth whole-repo run is not a race run. If that is not acceptable keep the builder and enable `MERGE_SCOPED=1` only |
| I4 | Red CI after a green gate: rerun at once | standing rule for the coordinator until S2 is built: on the line `CI RED for pushed head`, run `gh run rerun <run id> --failed` immediately; only a second red is a real red | removes holds like the 4 h 43 min one | V for the event |
| I5 | Deploy after every code batch to the quest sandbox (canary) | standing rule for the deploy agent: after each merged batch that changes `cmd/d2bot` or `internal/`, `tools/dev-update.sh --commit origin/main --to <quest sandbox>`; the farm sandbox follows after 30 min without a hard invariant; append the `UPDATED` line to a deploy log file | merged to running from 69 min to under 10 min | S |
| I6 | Regenerate instead of failing | until S1 exists: an agent whose branch changes a knob runs `go run ./tools/knobdump` as its last step before queueing, and edits plan data in `internal/plandata/` then copies the file to `docs/plan/data/` (put both lines in the worker brief) | removes 6 of the 22 failures | S |
| I7 | Fix the two named race flakes | one worker card: `TestReactionTimeDelaysInterrupt` and `TestArbiterControlGoldFarmAndRecovery` per `docs/process/flaky-tests.md` (reproduce with `-race -count=300 -cpu 1,4`) | 2 to 3 failed entries per 48 h, 9 min of gate each plus the round trip | V for the names |
| I8 | Fix streams queue in bundles | the `toolstrade` stream queues every third fix (or every 45 min), not every fix | about 30 fewer entries per 48 h, fewer self-conflicts on `docs/knobs.md` | S |

Decision the owner must make (not for an agent to assume):

- **D1, stacked flip windows.** Keep the 24 h watch and auto-revert, but stop treating it as a wait: allow up to 4 independent knobs per sandbox in one window (revert all on a breach, then find the culprit in the lab), and accept a lab A/B pass (24 virtual hours, two seeds, zero hard invariants) as the entry ticket. With two sandboxes that is 8 knobs a day instead of 2: the 29 coordinator rows clear in about 4 days. Needs a small change to `tools/flip/flip.py` (line 249 and the row update) and a line in `docs/process/dark-launch.md`, which records a user decision.

### 3b. Structural (so it does not recur), ranked

| # | Change | What exactly | Effect |
|---|---|---|---|
| S1 | Generated files are never merged by hand | (a) the knob table between the `knobdump` markers of `docs/knobs.md` is written by the merge step like the other shared files (conflict kind "regenerate", then `TestKnobsDocMatchesTables` still guards it); (b) one copy of each plan data file, the second generated at merge or embedded from one place; (c) union rules in the ops `tools/mergeq-union.txt` for `cmd/d2scenario/invariants.json`, `admission.json`, `povertyfarm.json`, `earlyroutes.json` (the current kinds take one list key, these files hold several lists: needs a recursive kind) | 8 of 10 conflicts and the knob-doc gate failures disappear |
| S2 | The queue heals itself | on a gate failure: rerun the failed package once when it is a race failure in a package the diff does not touch or the test is on the known-flaky list, and record the name in a flake ledger; on red CI after a green gate: one automatic `gh run rerun --failed` and an alarm within a minute; on a dirty worktree: merge the queued tip, not the worktree state; always store the `--- FAIL` lines in `log_tail` | failure round trips from 65 min to minutes; no silent holds |
| S3 | Verification lane | generalise `tools/survival/abrun.sh` arms from the fixed zone-model pair to `name@knob=value,...`; verdict from invariant counts; a nightly lab wave that takes the next lab-provable ledger rows; flip.py with stacked windows (D1); every card that ships a dark knob names the flip card in the same PR (extend `tools/darklaunch/check.py`: a `dark` row needs a `flip_card`) | dark age bounded: lab the first night, sandboxes the next day |
| S4 | Split `internal/bot` along the conflict lines | first cut by the hot files: loop and runner core, invariants and observe, routes and kit plan, explain and shadow; each a package with its own tests | race step runs only for the touched part (S: under 60 s for most diffs), scoped gates skip more, agents stop meeting in one directory |
| S5 | Preflight scans the diff, not the tree | leakcheck in range mode on changed files in preflight (the queue-time check already does); the full scan stays in CI and nightly | 86 s to about 15 s per entry, off the 4 GB box |
| S6 | Deploy is part of the pipeline | a worker hook after a pushed code batch: build and release to the canary, watch the hard invariants 30 min, promote to the second sandbox or roll back with `tools/dev-update.sh --rollback`; one JSON line per deploy | no deploy agent round trip, a measurable lag |
| S7 | Concurrency by package, not by head count | at most 2 running agents whose cards own files in the same package, 1 for a registry file; enforced where cards are admitted (owns globs already exist) | fewer real conflicts while S4 is pending |
| S8 | Ready lane | a card that needs no decision is startable by any free slot without the coordinator: `lane: now` plus an owns set that passes S7; the coordinator only handles failures, decisions and verdicts | removes the over 1 h tail of the commit-to-queue round trip |
| S9 | Whole-repo race moves to the nightly run | per merge: race of the touched packages only (already the case); nightly: everything, with the flake ledger of S2 as its output | keeps coverage without per-merge cost |

### 3c. What not to do

- Do not add a second merge worker or more parallel queues: the worker is idle 72 % of the time; ordering is not the constraint.
- Do not raise the code batch size above 4 before S1 and I7: a failed batch costs one full gate plus every entry alone (3 failed batches in the window).
- Do not set `MERGE_GATE_CACHED=1` or drop a gate: the whole-repo gate caught real breakage (entry 1391, two tests), and cached test results hide flakes. Fix the flaky race tests (I7) instead of removing the race step.
- Do not push docs straight to main past the queue: docs entries already batch up to 8 and take about 200 s; the public repo needs the leak scan before the push. Shrink the preflight (S5) instead.
- Do not build a remote build cache: both gate hosts already keep a warm Go cache; `-count=1` test time is the cost, and scoping removes it.
- Do not add agents to drain the dark ledger: the limit is windows and evidence, not hands.
- Do not lengthen or add live windows for evidence: use the lab for anything longer than the watch.
- Do not write more process documents for this; every item above is a flag, a rule line in a brief, or a small tool change.

## 4. Targets and the dashboard

| Metric | Now (48 h) | Target | Where to read it |
|---|---|---|---|
| Queued to merged, median / p90 | 16 / 41 min | 6 / 15 min | `finished - added` of merged rows in `queue.jsonl` |
| Solo Go entry run time | 574 s | 300 s | `python3 tools/gate_phases.py` in the ops repo |
| Throughput at peak | 11 per hour | 12 per hour solo, 24 batched | merged rows per hour |
| First-pass merge rate (branches with one entry) | 81 % | 95 % | branches with more than one row |
| Longest hold | 4 h 43 min | 15 min | `grep -E 'CI RED|holding' worker.log` |
| Merged to running on the canary, p90 | 5.3 h (median 69 min) | under 10 min | the deploy log of I5 against `finished` |
| Dark rows without a trial or a decision, older than 14 days | 7 | 0 | `python3 -I tools/darklaunch/report.py` |
| Rows flipped per day | 0 | 8 with D1, 2 without | `python3 tools/flip/flip.py status` and the ledger |

One command for the queue numbers (run in the worker's cache directory):

```
python3 -c 'import json,time;n=time.time();r=[json.loads(l) for l in open("queue.jsonl")];w=[x for x in r if x["added"]>n-172800];m=sorted(x["finished"]-x["added"] for x in w if x["status"]=="merged");b={x["branch"] for x in w};print("merged",len(m),"median",m[len(m)//2],"p90",m[int(.9*len(m))],"failed",sum(x["status"]=="failed" for x in w),"branches",len(b))'
```

## 5. Limits of this review

- Agent work time and coordinator turn counts were not measured: branches arrive as one commit and the session transcripts were out of scope (U).
- Deploy times come from container logs of one sandbox; deploys from a worktree instead of main cannot be told apart (U).
- Gains in section 3 are S unless marked: none of the flags was switched in this run and no test was executed.
- The names of failing tests are missing for 3 of the 8 gate failures.
