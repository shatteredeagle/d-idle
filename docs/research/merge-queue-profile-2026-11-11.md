# Merge queue profile and safe speedups (2026-11-11)

> status: research. Consumer: card `queuespeed` (TASK-312). Read-only analysis; nothing in the queue, its worker or its environment was changed.

Tags: V = verified (how), S = secondary (derived from another number), U = unverified.

## 1. Data and its limits

- V: window = the last 48 h of `~/.cache/d2-mergeq/queue.jsonl` (104 items added: 94 merged, 6 failed, 3 queued, 1 running), `worker.log` (357 lines in the window, local clock), `git log origin/main`, and the last 100 `gh run list` entries for main (about 24 h: 93 `ci`, 7 `desktop-electron`).
- V: `MERGE_TIMINGS` is NOT set in `~/.config/d2-mergeq/env` (the file holds only `MERGEQ_AUTO_DONE=1`). No `result.timings` exists in any item. So per-phase seconds are NOT measured; the phase split below is S/U (derived from code and CI step times). Totals, waits and gaps are V.
- Skipped: the `go test -json` per-package timing job (candidate h). A gate was running on the worker host (load 2.8 of 16 cores, but `go test -p 4` of item #569 live), and the brief says not to compete with an active gate. A ready-to-run checklist is in section 7.

## 2. Headline numbers (V, 48 h)

| measure | value |
|---|---|
| wall window first start to last finish | 46.5 h |
| worker busy (started to finished) | 7.0 h (15 %) |
| gaps between consecutive items while the next one was already queued | 56 gaps, median 290 s, p90 352 s, total 6.3 h (14 % of the window, as long as 45 % of all busy time) |
| gaps with an empty queue (demand limited, not a sink) | 43 gaps, 33.2 h |
| queue wait (added to started), 94 merged | median 486 s, p90 4996 s, max 14576 s |
| time per item (started to finished) | median 286 s, p90 401 s, max 432 s |
| docs-only items (20) | median 108 s, p90 313 s |
| code items (74) | median 289 s, p90 412 s |
| final `ci` value of the 94 merged items | green 90, red 2, green via a later head 1, n/a 1 |
| batched (docs fast path) | 6 of 94 |
| items with `--after` | 7 of 104 |
| auto-Done follow-up commits | 52 pushed ("status sync (auto)"), 38 skipped by rule (no task, boxes open, already Done), 14 not yet evaluated |
| main CI (`ci` workflow) duration, 74 green runs | median 279 s, p90 321 s |
| main CI cancelled runs | 22, of which 21 in one burst at 11:24 EST (a mass push, not the queue) |

Hold reasons logged by the worker (V, `holding #N ...` lines, duration = until that item started): `CI of main is red` 60 episodes, 7.8 h; `merge lock held` 1; `CI of main is pending` 0 in the window (9 earlier, 1.0 h). Only two of the 60 red holds were a real red main (72 min and 141 min); the other 58 average about 4.9 min each, which is one CI run (see sink 1).

Failures (V, log, 48 h): non-additive conflict 7 (3 of them `merge-lines.txt`, gone since sharedgen), gate go-race 2 (#469, #561), gate go-whole 1 (#488), private merge worktree reset lock 2 (#535, #536, infra), preflight 0 in the window. Retries: every item has attempts = 1 except requeues after a failure; 0 backoff retries.

## 3. Phases of one item (S/U, from code and CI step times)

Code item, median 289 s; docs item, median 108 s. `merge.sh` order: mergeq CI check and `prepare()` (merge main into the branch, status regen), `--preflight` in a throwaway worktree (merge, sharedgen, `leakcheck`; timeout 240 s, `MERGEQ_PREFLIGHT_TIMEOUT`), lock, real merge, sharedgen, gates, push, CI poll (skipped when no-wait), cleanup, then auto-Done after CI resolves.

| phase | est. median | est. p90 | basis |
|---|---|---|---|
| prepare (fetch, merge main, status regen) | 10-15 s | 30 s | S: one `go run` + git |
| preflight (worktree add, merge, sharedgen, leakcheck) | 40-60 s | 120 s | S: CI leakcheck step is 24 s; mergeq comment says 45 s on a loaded box |
| real merge + sharedgen | 5-10 s | 20 s | S |
| `go-whole` (labrun, `go vet ./... && go test -short -p 4 ./...`) | 100-160 s | 250 s | S: CI test job is 272 s total (vet 50, tests 166 at -p 2) |
| `go-race` on touched packages (labrun) | 15-60 s | 120 s | U, depends on the package |
| `go build ./...` (local, on the small box) | 10-30 s | 60 s | U |
| leakcheck again + hygiene | 25 + 30 s | 90 s | S: CI 24 s and 31 s |
| push | 2-5 s | 10 s | S |
| CI wait inside merge.sh | 0 (no-wait) | 0 | S: the `MERGE_CI_POLLS=0` path of docs/mergefast.md 2b; item results carry `CI for <head>: pending` |
| docs-only: leakcheck + hygiene only | 60-90 s | | consistent with the measured 108 s median |

Leakcheck runs three times for a code item (mergeq preflight, the gate, then CI); hygiene runs twice (gate, CI). The sum of the estimates is about 250-330 s, which matches the measured 289 s median, so the split is plausible but U until `MERGE_TIMINGS=1` is on.

## 4. Top 5 time sinks

1. **Stale red hold: 58 holds, about 4.7 h, about 290 s lost between consecutive items (V by code and log).** `red_hold()` (mergeq.py) returns every merged item whose recorded `result.ci` is `red`, and returns nothing only when the tip of main is green at that instant. Three items are recorded `red` forever (#344 from 11-07, #545 and #551 from 11-10). So after EVERY push, until its CI turns green (about 280 s), `ci` is forced to `red`, `pending_ok` is false and the next item waits for the full CI run. This silently disables `MERGEQ_NOWAIT_CODE`/`MERGEQ_DOCS_NOWAIT` since #551. The 56 gaps with a queued successor have median 290 s, equal to the median CI duration (279 s); that is the evidence.
2. **A different workflow holds the queue: `desktop-electron` smoke (V).** `ci_state()` asks `gh run list` for headSha, status and conclusion only, no workflow name, so a failure of `desktop-electron` (path-triggered extra workflow, step `npm run smoke` with `D2E_SMOKE_NO_SANDBOX=1`) counts as main red. Three failures (02:04Z, 02:47Z, 04:06Z on 11-11) caused two real holds of 72 and 141 min (3.6 h) and the two permanent red marks of sink 1. The code did not break; a smoke test did.
3. **The code item itself, 289 s median (S split).** Whole-repo gate on the worker host plus the same hygiene checks that CI repeats after the push. Queue capacity today is one item per about 580 s (run 289 + gap 290); with sink 1 gone it is one per about 300 s.
4. **Follow-up pushes (V).** 52 auto-Done commits in 48 h. Each is pushed after the item's CI resolved green, produces a new head that is not in `pushed_heads`, so `pending_ok` (`sha in heads`) is false and the next item waits for that head's CI too (about 280 s) if it arrives during that window. They also add 52 CI runs (cancel-in-progress limits the waste). The generated shared-file commit is no longer separate (since sharedgen: 5 `shared files (generated)` commits only in 48 h; it rides in the merge).
5. **Failures and reruns (V).** 14 failed attempts in 48 h (7 conflicts, 3 gates, 2 infra lock; plus the 2 CI-red marks, which were merged items), each costs a full item time (60-290 s) plus the hold; 7 of them were `merge-lines`/`task` file conflicts that sharedgen and the no-lines rule have since removed (S, not yet re-measured).

Not a sink: the 33 h of empty-queue gaps (demand limited). Docs-only batching works (6 items batched) but docs items are only 20 of 94.

## 5. Candidate speedups, evaluated

Per-item saving is against the 580 s current cycle (289 run + 290 gap) when the queue is non-empty.

| id | candidate | safety | payoff | verdict |
|---|---|---|---|---|
| 0 | fix the stale red hold (new; section 4 sink 1) | Safe: a genuinely red tip still holds; only the permanent marks of already-fixed reds go away | about -290 s per queued item (cycle 580 s to about 290 s, throughput doubles in bursts) | DO FIRST |
| 0b | count only the `ci` workflow in `ci_state()`/`resolve_pending` (request `name` in the `gh` JSON); `desktop-electron` reported separately | Safe if `desktop-electron` stays a required check for releases and a failure still pings the coordinator; it does not gate Go code | removes 3.6 h of holds per incident class (2 incidents in 48 h) | DO SECOND |
| f | raise `MERGEQ_NOWAIT_MAX` (3) / keep NOWAIT_CODE (1) | NOWAIT_CODE is already 1 and docs/mergefast.md section 2b approved it; the cap only matters after 0 is fixed. Cycle with 0 fixed is already gate-bound, 3 in flight is enough (CI takes 280 s, items take 290 s, so at most 1-2 are in flight). Raising the cap gains nothing, lowering to 1 would gate on every CI. Risk of more in flight: a red main now needs a revert of up to N merged items | none beyond 0 | keep 3 |
| g | auto-Done follow-up: (i) add its pushed head to the in-flight set, or (ii) fold pending Done flips into the NEXT merge's sharedgen step, so no extra push or CI run | (i) is trivial and safe. (ii) needs the flip to wait for "CI green", which the next merge cannot guarantee, so write it only for items whose ci is already green at that time; same rule as today | (i) about -280 s per item that arrives in the 5 min after an auto-Done (about half); (ii) -52 pushes and CI runs per 48 h | (i) THIRD, (ii) later |
| e | preflight: drop the leakcheck inside merge.sh `--preflight` when mergeq already ran it, or lower `MERGEQ_PREFLIGHT_TIMEOUT` | Hygiene gap check: leakcheck is run again in the real gates and in CI, so preflight is an early-fail aid, not a barrier. Shortening the timeout (240 s, never hit: max item run is 432 s) changes nothing in the median. Dropping it moves a leak failure from "fail alone before the lock" to "fail inside the lock, roll back" | about -25-45 s per code item (S) | after timings |
| a | gate only changed packages plus dependents; full run is CI's backstop | A wrong dependent set lets a break through to a red main, which then holds everything (a red main costs 70+ min as seen). Safe only with a conservative rule: any change outside `cmd/`, `internal/`, or in `internal/<core>` (mem, plandata, combat, input) keeps the whole-repo gate; docs/tools/tests-only edits leaf packages. `go list -deps`/`go list -test` computes dependents exactly | about -60-110 s on the 30-40 % of code items that touch one leaf package (U until the package mix is counted) | after timings, with a leaf-only rule |
| b | skip the repeated gate when the agent's own gate passed on the same base tip | Needs the agent's gate result bound to (merge-base tip, branch head, merge of main) hash; the gate that matters is the one on the MERGED tree. After 0, the main tip usually moved since the agent started, so the hash would rarely match; if it does match the merged tree is byte-identical to a tree already gated | small hit rate, U | skip; fold into (a) |
| c | two gates concurrently on the worker host | The host has 16 cores and 62 GB (24 GB used, load 2.8 at the sample) but is shared with agents and the live games; `-p 4` uses 4. Two merges also race on main: the second must re-merge on the first's tip, so the second gate is only valid for a leaf-disjoint item (needs (a)). Complexity and a wrong-tree risk. | at most -290 s per pair, only in bursts | no, unless (a) is in and bursts still hurt |
| d | batch code items per gate (like docs fast path) | One failing item rolls back the whole batch; fallback exists for docs. The gate is the expensive part, so payoff is real: N items for about 1.3 gate times. Race/ownership of the same packages makes failures hard to attribute | -(N-1)*150 s per N code items | medium risk; try after 0 and timings show gate-bound |
| h | slow/flaky tests in the gate | not measured here (section 7) | U | measure |

## 6. Recommended ordered plan (for `queuespeed`, TASK-312)

All three first changes are in the ops repo (`tools/mergeq.py`), tested by `python3 -I tools/tests/test_mergeq.py`.

1. **Clear and fix the stale red hold (expected -290 s per item, about 4.7 h per 48 h).** (a) Now: mark #344, #545, #551 as resolved (for example set `result.ci` to `red (fixed)` or `green (via ..)` in queue.jsonl while the worker is idle, or add a `ci_ack` field the code honours). (b) Code: `red_hold()` ignores a red head once a later head on main has been green (use `git merge-base --is-ancestor` plus the green check, the same test `resolve_pending` already has) and `resolve_pending` rewrites old reds to `red (fixed)` on that event. Risk: a red that is real but unfixed is cleared by an unrelated later green: only clear when the red head is an ancestor of the green tip (then the fix is in the tree). Rollback: revert the commit; the three marks come back by hand.
2. **Judge main by the `ci` workflow only (expected: avoids 3.6 h of holds per Electron incident).** Add `name` to the `gh run list` fields; `desktop-electron` (and any other workflow) failing logs a warning and posts to the board, but only `ci` is red/pending. Risk: a broken Electron app on main goes unnoticed until the coordinator reads the log; mitigation: keep the warning and a separate hold flag `MERGEQ_HOLD_ON_ELECTRON=0`. Rollback: env switch.
3. **Count the auto-Done head as in flight (expected -280 s on about half the items).** After `auto_done()` pushes, add the new head to `pushed_heads`/mark an in-flight record so `pending_ok` stays true. Risk: none beyond the NOWAIT policy already approved (cap 3).
4. **Switch `MERGE_TIMINGS=1` on in `~/.config/d2-mergeq/env`** (needs a worker restart; the owner decides when) and read `result.timings` after 24 h. This is the input for steps 5 to 7 and costs nothing in behaviour.
5. After timings: drop the duplicate leakcheck in preflight (e), then leaf-only package gating (a) with the conservative rule above.
6. Later: fold auto-Done flips into the next merge commit (g-ii); consider code batching (d) only if the gate is still more than 60 % of the cycle.

Expected result of 1 to 3: cycle from about 580 s to about 290 s per item in a burst (throughput 6 to about 12 items/h), median wait (486 s) and p90 (4996 s) roughly halve, with no gate weakened. Steps 5-6 might save another 60-120 s per code item.

## 7. Ready-to-run checklist for the missing measurements (no game, worker host)

1. `labrun WORKER -- top -bn2 -d1` shows idle plus nice at least 25 percent AND `tools/mergeq.sh status` shows no running gate.
2. `labrun WORKER -- nice -n 19 sh -c 'go test -short -count=1 -p 4 -json ./... > /tmp/qp.json'` from inside a clean worktree at origin/main; then sum `Elapsed` per package: `python3 -I -c` over the `pass`/`fail` lines with `Test` unset.
3. Repeat twice to separate flaky tests from load noise; list the 10 slowest packages and any test that fails once in three.
4. Enable `MERGE_TIMINGS=1` (step 4 above) and compare against section 3.
