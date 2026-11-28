# Fable review 9: how a change travels, and why the dark ledger grows

> status: current (reviewer run fr9, 2026-11-27 evening, docs only, nothing changed). Independent second opinion: written before reading any other document of this review folder (section 8 says what was and was not compared).

Question from the user: "the dark ledger keeps growing (33 rows, 5 added in one afternoon, only 2 closed) and we keep falling behind; how are we pulling in changes, and how do we speed it up?"

Tags: **V** read or computed in this run (source named), **S** inferred, **U** taken from the brief, not checked. "Today" is 2026-11-27, 00:00 to 22:31 EST. Ops-repo files are cited as `ops:tools/<file>:<line>`.

Sources (all V): `git log origin/main --since='2026-11-27 00:00'` (904 commits), the queue records `~/.cache/d2-mergeq/queue.jsonl` (items 1424 to 1579) and `worker.log` (last 577 lines = today), `data/darklaunch.json` at 415b11f25, `ops:tools/mergeq.py`, `merge.sh`, `gate_scope.py`, `mergeq-union.txt`, `autodeploy.py`, the user timers on the head box, `docs/process/dark-launch.md`, `docs/process/generated-files.md`, `python3 -I tools/backlog/drift.py --summary`.

## 1. Short answer

The ledger grows because **opening a row is a free side effect of a merge and closing one is five manual hops through three systems, and nothing in the pipeline counts a change as unfinished after it merges.** The merge queue is busy but it is not the main cause; it is the tax on every hop. Today: 150 branches merged, 60 cards created, 5 knobs added dark, 8 rows moved to `enabled`, 2 knobs actually removed. The cap that merged at 22:25 stops the count from rising; it does not make anything close.

The 5 root causes, in order of weight:

| # | Root cause | Today's number (V unless tagged) |
|---|---|---|
| RC1 | "Done" means merged. The card closes at merge (auto-Done), live-check boxes move to a list, and no step owns deploy, flip, ledger commit or graduation. | 67 auto-Done pushes; drift: 112 Done cards with open boxes; 10 rows `enabled` with the code default still 0; 29 rows open, median age 7 days, oldest 28 |
| RC2 | Exit criteria are prose the pipeline cannot run, and most need the scarcest resource. | about 13 of 25 dark rows ask for a live drill or check, 6 for an owner go, `criteria_met` is "U" on the new rows; lab budget 3 runs a night; no row has a scheduled run |
| RC3 | Intake is pushed from boards whose status is wrong, with no limit tied to what downstream can verify. | 60 cards created, 150 merges, 904 commits on main; drift 150 findings (33 merged but not Done); 3 dropped queue items; 3 rebuilds and about 15 percent wasted launches (U, brief) |
| RC4 | Shared state lives in hand-edited files that every branch touches at the same spot, and ledger state changes travel as branches. | 18 conflict failures in 150 merges (12 on task cards, 3 on `data/darklaunch.json`); 205 "merge main into branch" commits; 14 branches touched the ledger; one ledger branch (#1575) failed twice and is still not on main |
| RC5 | The queue charges a fixed 6 to 9 minutes per run whatever the risk, on one serial worker, and flakes roll back whole batches. | busy 283 of 375 afternoon minutes; a ledger row forces the whole-repo gate (30 merges); preflight median 100 s, gates 234 s, 0 percent test cache; 3 of 3 batch rollbacks were flakes; 58 min held on "CI of main is pending" |

## 2. The path as it ran today (facts)

### 2.1 Intake and briefs
- 60 task cards were added to `backlog/tasks/` today (V: `git log --diff-filter=A`), 1070 cards in total. 256 non-merge commits touched a card.
- The board is not a reliable picture of what is merged: `drift.py --summary` reports 150 findings: 112 Done with open boxes, 33 merged but not Done, 4 all boxes ticked but open (V). An agent launched from that board can be handed finished work. The brief names three cases today (proptest, pgstab, dlvcast; U). V traces: #1490 agent/dlvcast and #1498 agent/dlvwave3 "dropped by hand", `a8b2f6196 Merge branch 'agent/dlvcast' into agent/dlvwave3`.
- The agent cap (6, `docs/process/coordinator-efficiency.md` item 6) exists to protect the head box's memory. It is not a flow limit: nothing compares "started" with "verified".
- Three card-sweeper branches ran in one afternoon (closecards, closecards2, closecards3, cardfix, cardfix2: 5 queue items) to repair card state by hand. Each one edits many cards and so collides with every feature branch that edits its own card (2.2).

### 2.2 Branch lifecycle
- 904 commits reached main for 150 merged branches: 150 merge commits, 150 "shared files (generated)" commits, 205 "Merge main into branch" commits, 67 auto-Done status commits, the rest real work. That is 6 commits per change, more than half of them bookkeeping (V).
- Conflicts that stopped a merge: 18 (V, `worker.log` lines `failed: non-additive conflict`):
  - 12 name a `backlog/tasks/task-*.md` file (14:39 #1490 and #1492, 14:53 #1496, 14:56 #1503, 15:02 #1499, 15:19 #1503, 16:54 #1530, 17:52 #1540, 18:08 #1554). Cause (S, from the mechanism): the auto-Done follow-up and the sweeper branches rewrite a card's front matter on main (`status`, `updated_date`, quote style: AGENTS.md says `backlog task edit` rewrites those) while the owning branch edits `result:` and boxes in the same hunk.
  - 3 name `data/darklaunch.json` (15:19 #1499, 15:23 and 15:42 #1507). Every new row is appended before the same closing bracket (file line 571), so two knob branches always conflict. The brief counts 5+ including the ones agents resolved by hand before queueing; V for those: `2ae395801`, `f688ba652`, `689b57e34` "darklaunch.json: both rows kept".
  - 3 others in real code or data (classdata.go, fake.go, funwire.go, potions-belt.json, vault.json, gear.md).
- **Bug (V): the conflict list is split on whitespace.** `ops:tools/mergeq.py:578` is `u = unmerged.split()`. Task files have spaces in their names, so `backlog/tasks/task-983 - bsmdef-....md` becomes three "files" (`backlog/tasks/task-983`, `-`, `bsmdef-...md`; log 17:52:33 "conflict in 4 file(s)" for 2 files). No rule in `ops:tools/mergeq-union.txt` can ever match a task card until this is `splitlines()` (or `-z`).
- `ops:tools/mergeq-union.txt` has three rules (verdicts, scenarios, invariants). Neither the ledger nor the cards are in it. The existing `json-list` kind keys rows by `"id"`; ledger rows use `"name"`.
- `docs/knobs.md` is meant to be generated at merge (`docs/process/generated-files.md` rule 2), yet 11 non-generated commits changed it today (V): agents still regenerate it in the branch. The worker resolved one such conflict by regeneration (log `prepare: docs/knobs.md resolved by regeneration`), so this one is contained.

### 2.3 Merge queue (numbers in section 5)
- One worker, one merge worktree. Code batches up to 4 with disjoint files (`MERGEQ_BATCH_CODE=1`, `ops:tools/mergeq.py:114-115`); a batch gate failure rolls all back and each item runs alone (`mergeq.py:890`).
- The worker was down from 10:14 to 13:01 and restarted three times at 13:01 to 13:15 (log `worker started`); two items then failed on a missing merge worktree. The afternoon wave started against an empty-handed queue.
- Gate scoping exists (`MERGE_SCOPED=1`) but 69 of 120 code merges still ran whole-repo: 30 because "`data/darklaunch.json` has no Go package above it" (`ops:tools/gate_scope.py:67`), 26 for a scenario or Proving Ground JSON, 13 periodic. When it does scope, the reverse-dependency set is about 90 of 225 packages, so a scoped run is 6.1 min against 7.3 (V, medians).
- Tests are never cached: `gflags=-count=1` (`ops:tools/merge.sh:551`), "cache hit median 0%" (`ops:tools/gate_phases.py` over 717 merges). The switch `MERGE_GATE_CACHED=1` exists and is off "until the 20-merge measurement is read".

### 2.4 Deploy, flip, ledger, graduation
- Deploy to the two sandboxes is by hand (three deploy rounds today: 88, 94 and one more; V from branch names). The local-mode bug was fixed today (TASK-1086, merged 21:15).
- **An auto-deployer exists and is not switched on (V).** `ops:tools/autodeploy.py` deploys the newest green main to the canary sandbox every 10 minutes with a 15 minute watch and rollback. No timer for it is installed (user timers present: merge watchdog, nightly race, gate measure) and its log `deploys.log` does not exist. The deployer is itself a dark launch.
- Flip is by hand (`tools/flip/flip.py`), one open window per sandbox, 2 h each; the watcher service is active (V). A branch `agent/autoflip` (plan, run, graduate-list) is item #1578, queued at 22:25.
- Every status change of a row is a commit that must pass the queue: `deploy88-ledger`, `deploy94-ledger`, `ledgertrial`, `ledgerenabled`, and the watcher's own branch #1575, which failed preflight twice (22:03 and 22:17, "leakcheck FAILED ... secret 2 hits 1 files") and is not merged. So main's ledger is behind the sandboxes right now, and the report that decides what to flip next reads main.
- Graduation has no defined end state in practice: `bot.farmReset` was graduated today (040ade134 removed the knob) and its row still says `"status": "enabled"` (file line 318), not `default-on` or `retired`. Ten rows are `enabled`; the cap does not count them.

### 2.5 Feedback
- Post-deploy audits found real bugs in each round (U, brief; V trace: branches postdeploy2, xplostfalse, holereach, pgfix merged after 21:00). That is the pipeline working, but it is the only place a merged feature is looked at, and it is a person-launched agent, not a step.
- Live evidence: about 13 of 25 dark rows wait for a live drill, probe or check (my read of `enable_criteria`: minionRead, fullGrid, warpRead, layoutRead, mercReviveUnverified, cubeJobs, identifyTome, invdesk.verbs, managementFast, input.planner, hotkeyMap, diffReadyV2, vault). None names the command that would produce the evidence or the night it is booked for.

## 3. Root causes, argued

**RC1. The unit of completion is the merge.** The merge step sets the card Done (`auto-done: ... set Done, pushed`, 67 times today), and AGENTS.md lets live-check boxes stay unticked ("they move to docs/live-checks.md at merge and the card still closes"). From that moment the feature has no owner card, no agent and no queue position. Everything after (deploy, flip, window, ledger commit, default flip, knob removal) is coordinator memory. A system that pays out at merge produces merges: 150 of them, against 2 knobs removed. This is why adding tools (flip.py, the watcher, the cap) has not bent the curve: none of them changes what counts as finished.

**RC2. Rows cannot close without a resource nobody schedules.** The row's exit test is a sentence. Nothing can execute "Warp drill across 5 areas with the read on". The lab allows 3 heavy runs a night and the rows do not compete for those slots in any list. The sandbox-first policy written today helps the roughly 5 rows a sandbox can observe; it does nothing for the 13 that need a drill and the 6 that need the owner.

**RC3. Push intake from stale status.** Work is cut from boards and audits whose state is wrong in 150 places, so some launches rebuild merged work, and sweeper branches are launched to fix the board, which then collide with feature branches (RC4). There is no rule of the form "no new knob branch while N rows wait for evidence".

**RC4. Hand-edited shared state.** The ledger is one JSON list appended at one spot; a card is edited by its branch, by auto-Done and by sweepers; ledger status is a git commit that needs the full queue. Each is a guaranteed collision that a person or agent then resolves, re-queues and waits for again: #1540 failed at 17:52 and merged at 18:28; #1499 took 67 minutes and #1507 73 minutes from add to merge, both on ledger and card conflicts.

**RC5. Flat per-run cost on one serial worker.** A one-row ledger change costs the same as a combat rewrite (whole-repo vet, test and race). The worker was 75 percent busy all afternoon with a mean of 4.9 items waiting, so every retry from RC4 and every flake rollback pushes everything behind it.

## 4. Adversarial notes on what merged tonight (darkthrottle)

1. **The cap (27) equals today's count**, so it blocks growth and forces nothing to close. Worse, it gives a cheap way out: rows with owner `owner`, status `by_design` or `enabled` do not count (`docs/process/dark-launch.md`, Cap). Expect new rows to arrive as `owner` rows and the pile to move to `enabled` (already 10). Count `enabled` in the cap, or cap `enabled` separately at 5.
2. **Time bomb on 2026-12-09.** 23 open rows carry `review_by` 2026-12-09 (V). `tools/darklaunch/check.py:84-93` fails when a dark or trial row is past due, and `tools/ci/check.sh` runs it. On that morning hygiene fails for every branch until 23 rows are decided. Stagger the dates now, or make "past due" block only branches that add a row.
3. **`review_by` at most 3 days after `dark_since`** for new rows, with a queue where a ledger edit takes 13 to 90 minutes and the lab takes 3 runs a night, will be met by extending dates. A date nobody can keep trains everyone to edit dates.
4. A "net <= 0" rule per branch makes feature agents do a second, unrelated job (close someone else's row) inside a feature branch. That increases ledger-file collisions (RC4). Closing rows should be a separate, automated lane.

## 5. Speed

### 5a. Measured baseline, today (V, `queue.jsonl` and `worker.log`)

| Measure | Value |
|---|---|
| Items added / merged / failed for good / dropped / still queued | 154 / 148 (150 by log) / 2 / 3 / 1 |
| Merges per hour | average 6.7 over 22.5 h; afternoon (13:15 to 19:30) 14.4; peak hour 19 (16:00 to 17:00) |
| Add to merged (all day) | median 13.2 min, p90 43.2, max 95.5 |
| Add to merged (afternoon) | median 18.4 min, p90 55.0, max 95.5 |
| Wait before start | median 5.9 min, p90 37.0, max 91.2 |
| Run time per item | median 7.4 min, p90 10.3 |
| Worker runs | 92 runs for 148 items: 62 single, 13 of two, 8 of three, 9 of four |
| Worker busy | 578 min all day; afternoon 283 of 375 min (75 percent); 92 min idle while a mean of 4.9 (max 11) items waited |
| Failed attempts | 25 `failed` lines + 7 rollback lines over about 180 attempts (18 percent): 18 conflicts (12 cards, 3 ledger, 3 other), 3 batch rollbacks (7 items, all flakes), 2 leakcheck (#1575 twice), 4 worker-restart, 1 moved tip |
| Phase seconds (55 timed code runs) | preflight median 100 (max 147); gates median 234 (max 471); merge 5; push 3 |
| Inside gates (39 runs) | whole-repo test median 214 s, race median 262 s (run in parallel), cache hits 0 |
| Run time by kind (median) | docs alone 1.2 min; docs batch 3.1; scoped code 6.1; whole-repo code 7.3; code batch 8.6 to 9.0 |
| Held on "CI of main is pending" | 14 holds, 58 min in total, median 227 s, max 534 s |

"Add" is when the branch entered the queue, not when the agent finished; agent-done to add is not recorded anywhere (U). A re-queued item keeps its id, so the add-to-merged times above include the failed rounds. The brief's "preflight about 35 s" does not match: measured median is 100 s.

The three batch rollbacks (01:13 #1431+#1432 `TestStreamSendsEachFrameWhenPublished`; 14:11 #1486+#1487; 17:04 #1532+#1536+#1540 `TestRingBounded` at load 19) all merged unchanged on the retry: 100 percent flakes. The 17:04 one cost 10.5 min of gate time and its last member reached main at 18:28.

### 5b. Ten speedups, ranked by (minutes saved per merge x merges per day) / effort

Savings are estimates (S) from the table above at about 150 merges a day.

| # | Change (exact) | Saves | Effort |
|---|---|---|---|
| 1 | **Fix the path split and add card and ledger resolvers.** `ops:tools/mergeq.py:578` `unmerged.split()` to `git diff -z` + split on NUL. New union kind `card` for `backlog/tasks/*.md`: front matter merged per key (`status`, `updated_date` from main; `result:` and boxes from the branch), body by `lines`. `json-list` takes a key field: `json-list data/darklaunch.json items name`. | 15 of 18 conflict failures; each costs a re-merge round of 30 to 70 min of latency and an agent or coordinator turn. About 8 h of summed latency a day. | 1 to 2 h |
| 2 | **Ledger and cards are not Go.** `ops:tools/gate_scope.py:67`: a diff whose non-Go files are all under `data/darklaunch*.json`, `backlog/`, `docs/` is docs-class: run `tools/darklaunch/check.py` and hygiene only. A knob branch then scopes by its Go files. | 30 whole-repo runs become scoped; pure ledger commits drop from about 7 min to 1.2. About 40 to 60 worker minutes a day. | 30 min |
| 3 | **Retry failed tests once before rolling a batch back**, any test, not only the listed ones (`ops:tools/mergeq-flakes.json` holds 8 names; neither of today's two is in it): rerun the failing packages with `-count=1`; a pass continues the batch and appends the test to the flake ledger with a count. | 3 rollbacks a day, each 10 min of gate plus 2 to 3 serial reruns (about 25 worker minutes each, and 30 to 80 min latency for the members). | 1 h |
| 4 | **Turn the test cache on**: `MERGE_GATE_CACHED=1` in the worker env (`ops:tools/merge.sh:551`), `GOFLAGS=-trimpath` so worktree paths share cache keys. The nightly race stays uncached. | whole-repo test 214 s to an expected 60 to 90 s (U until measured); race then becomes the long pole, see 5. | 10 min + one day of watching |
| 5 | **Race only the changed packages at merge**, whole-repo race nightly (the timer exists). Today race is 262 s median over the reverse-dependency set. | with 4: gates 234 s to about 90 s, 2 to 2.5 min x 100 code merges. | 1 h |
| 6 | **Stop counting bookkeeping heads against the in-flight cap.** `MERGEQ_NOWAIT_MAX` 3 to 6 (`ops:tools/mergeq.py:118`) and do not count auto-Done heads (cards only) as in flight (`mergeq.py:1291`). | most of the 58 min of holds. | 15 min |
| 7 | **Fold auto-Done into the merge's own generated commit** (status Done written by sharedgen before the push) instead of a second push. | 67 pushes and CI runs a day, the cap pressure of 6, and the main-side card rewrite that causes the card conflicts of 1. | half a day |
| 8 | **Preflight at `mergeq add`**, in the background, keyed by tree (the `pf-tree-ok` cache already exists); the worker's turn then starts at the gate. Also prefetch the next batch's preflight while the current gate runs on the build host. | 100 s x 92 runs when the queue is non-empty: up to 2.5 h of worker time a day. | half a day |
| 9 | **Priority lane for docs, cards and ledger items** (`priority` exists on the item): classify at add; these run in 1.2 min and should not wait behind 9-minute code batches. | their add-to-merged from about 18 min to about 3; this is what unblocks ledger commits and sweepers. | 30 min |
| 10 | **Merge train**: build batch N+1 on batch N's candidate in a second merge worktree while N's gate runs; on N's failure drop N+1's result. Bisect a failed batch in halves instead of all-alone. | up to 2x at saturation; only worth it after 1 to 5, since the build host then carries two gates. | 2 days |

Not on the list on purpose: a bigger batch size. 9 of 92 runs reached four items; batches are limited by file overlap, and the overlap is the cards, the ledger and the generated files. Fix those (1, 7) and batches grow without touching the cap.

Expected result of 1 to 6 (S): run time per code item from 7.4 to about 4 min, failed attempts from 18 to about 5 percent, afternoon add-to-merged median from 18 to about 8 min, p90 from 55 to about 20.

### 5c. Ledger side: from "row merged" to "row graduated"

Today (V): no row has gone from merged to knob removed in under a day by tooling; open rows have a median age of 7 days, the oldest 28; closed today: 2 knobs removed, both by a hand-launched agent.

| Hop | Today | Change | Target |
|---|---|---|---|
| merged to on the canary sandbox | hand deploy in rounds (3 today) | install the timer for `ops:tools/autodeploy.py` (10 min cycle, 15 min watch, rollback built in); second sandbox follows after one clean watch | 30 min |
| on sandbox to flipped (`trial`) | hand, when remembered | `autoflip` (#1578) on a 30 min timer: picks rows with a `flip_plan` whose named scenario is green in the gate of the merge that added them, one bundle per free sandbox | 1 h |
| `trial` to `enabled` | watcher, 2 h | keep | 2 h |
| ledger state to main | a branch through the queue; #1575 failed twice | either lane 9 plus resolver 1, or better: runtime status (`trial`, `enabled`, timestamps) moves to a state file the watcher owns outside git, and `report.py` reads both; git keeps only the row's definition | 0 to 5 min |
| `enabled` to graduated | hand-launched agent, no fixed end state (`bot.farmReset` row still `enabled`) | graduation bot: 24 h after `enabled` with no revert, a templated card and one Sonnet agent per row (default flipped, knob and old path deleted, row `retired`), at most 2 in flight; `check.py` fails a row that is `enabled` for more than 3 days | 24 to 36 h |

Target elapsed time per sandbox-observable row: **under 30 hours** from merge to knob removed, against a median open age of 7 days. Capacity check: two sandboxes, 2 h windows, bundles allowed: 12 windows a day each is the ceiling; 5 new rows a day fits with room.

For rows that need the lab (about 13): each row gets an `evidence_cmd` (the drill or probe command) and a slot in one ordered list that the night window (and now daytime) drains, 3 a night: the current backlog clears in about 5 nights if nothing is added. For the 6 owner-go rows: one page, one decision each, asked once as a batch; a row the owner does not want becomes `by_design` or is retired.

### 5d. Tomorrow morning, under 2 hours in total

1. **(40 min) Path split + ledger union.** `mergeq.py:578` to NUL-separated; `json-list` key field and the line `json-list data/darklaunch.json items name` in `mergeq-union.txt`; test in `ops:tools/tests/test_mergeq_conflict.py` with a task file whose name has spaces. Removes the ledger conflicts at once and makes the card resolver possible.
2. **(30 min) Ledger, cards and docs are docs-class in `gate_scope.py`** and get queue priority (5b rows 2 and 9). Every ledger status commit then merges in about a minute.
3. **(30 min) Switch on what is already built:** install the `autodeploy` timer for the canary sandbox; set `MERGE_GATE_CACHED=1` and `MERGEQ_NOWAIT_MAX=6` in the worker env; add `TestRingBounded` and `TestStreamSendsEachFrameWhenPublished` to the flake ledger. Read `result.timings` the next evening.

Then, same day if time allows: stagger the 23 `review_by` dates (section 4 item 2), and set `bot.farmReset` to its real end state.

## 6. The redesign (same people, same tools), ordered by benefit over effort

1. **Tomorrow:** 5d steps 1 to 3.
2. **This week, small:** card resolver and auto-Done folded into the merge commit (5b rows 1 and 7); flake retry (row 3); race scoped (row 5); count `enabled` in the cap.
3. **A change is finished when its knob is gone.** Keep the card open through a `Verify` state: merged sets `Merged`, not Done; the card closes when its row is `retired` or `default-on` (or, for knob-less changes, at merge as today). The STATUS block shows "merged, not verified" as its own number. This is the one change that addresses RC1; without it the rest only makes merging faster.
4. **One file per row:** `data/darklaunch.d/<knob>.json`, definition only; runtime status in the watcher's state file. No branch then touches another's row; the union rule of step 1 becomes unnecessary.
5. **Pull, not push, for knob work:** the coordinator launches a new knob-adding agent only when (rows in `dark` + `trial` + `enabled` for sandbox-observable features) is below 8; otherwise the next slot goes to evidence or graduation. This is the flow limit the 6-agent cap is not.
6. **Intake from git, not from status fields:** before a launch, `drift.py` must be clean for the card and its family (a 5-second check in the launch script); the board is generated from merge commits, never swept by hand. Retire the closecards-style sweeper branches once auto-Done is in the merge commit.
7. **Executable exit criteria:** `evidence_cmd` and `evidence_slot` per row; a row without them cannot be added.
8. **Later:** merge train (5b row 10); preflight at add (row 8).

Not recommended: a second serial worker on the same worktree model before steps 1 to 5 of 5b; it doubles the collisions on the same files.

## 7. Open points for the coordinator

- Why did #1575 (the watcher's ledger branch) fail leakcheck with "secret" hits? Not investigated here (the branch is outside this run's scope). If the watcher writes a host or container name into a row, every automatic ledger commit will fail the same way.
- The worker outage 10:14 to 13:01 and the three restarts: the watchdog log shows "restart limit (3 per hour) reached" alarms the night before. Worth one look before more automation leans on the worker.
- Agent-done time is not recorded; add `done_at` to `mergeq add` (from the status board) so the real "done to merged" can be measured.

## 8. Comparison with the other reviewers

Not done in this run: when this was written the review folder held no other document in this worktree or in the main checkout (V, `ls`), so `05-dark-ledger-process.md` was not read and there are no disagreements to record. Points where I expect a difference of opinion: this review ranks the definition of Done (RC1) above queue speed, treats tonight's cap as necessary but insufficient (section 4), and argues against more merge workers before the shared files are fixed.
