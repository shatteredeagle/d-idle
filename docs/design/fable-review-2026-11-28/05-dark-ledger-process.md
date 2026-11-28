# Fable review 5 of 8: the dark ledger, the flip tooling and the merge pipeline

> status: review, 2026-11-27 evening (branch agent/fable-fr5, base 415b11f25). Read-only review; nothing here is a decision. Consumer: the coordinator's next process card.

Tags: **V** = read in the repo or observed on the worker host this session (how is stated), **S** = inference from V facts, **U** = not checked. Line references are to the base commit. `ops:` paths are in the private ops repo. The sandboxes are called "the canary sandbox" (Doofer) and "the quest sandbox" (Questy).

## 0. Summary

The sandbox-first policy is the right direction, and the tooling around it is careful about the one thing it was burned by (the tuning file write). Everything else is thin:

1. **The watcher reverts on tool faults and cannot see most harm.** Tonight's two bundles (9 distinct knobs, both sandboxes) were reverted ten minutes after the flip by a JSON parse error, not by the bot (V, 1.1). In the other direction, a hero that is parked, paused, waiting for Start or dead trips none of the six hard counters, and a window closes on the clock alone (V, 1.3).
2. **The ledger is not the truth and is written in three places.** Four rows on main say `trial` for windows that ended 8 and 16 hours ago; the watcher's own checkout holds six uncommitted revert rows; the only ledger branch failed preflight (V, 1.5).
3. **"Clean close" is not evidence of anything a player cares about**, and graduation asks for nothing more (section 2).
4. **Merge churn has one dominant cause: task cards are edited on main and on the branch at once** (11 of about 20 conflict failures today), then `data/darklaunch.json` (V, 3.1). Both are cheap to fix structurally.
5. **The cap does not shrink the ledger**: 26 counted rows under a cap of 27 lowered by one a week, no existing row has a `flip_plan`, every open row is due on 2026-12-09 or later (V, 2.4).

## 1. Failure modes of the policy

### 1.1 BLOCKER: the installed watcher runs old code and false-reverts every window

- V (`systemctl --user status d2flip-watch`): the daemon started 16:04 EST. The fix "read only the first JSON document of `ops status --why`" (78a959a85) reached main at 21:50 EST. A Python daemon does not reload; nothing restarts it on merge.
- V (the two `.last` files in the flip state dir, and the uncommitted diff of `data/darklaunch.json` in the main checkout): both bundles flipped at 02:50 and 02:51 UTC were reverted at 03:01 UTC with `revert_reason: "status unreadable for 10 polls: Extra data: line 819 column 1"`, `revert_breach: ""`. The flip command (a fresh process, new code) verified fine; the watcher (old process) could not parse the same output. `HARD_MISS_LIMIT` (tools/flip/flip.py:41, 512) turned a parser bug into a revert plus a second container restart on both of the user's heroes.
- S: until the unit is restarted every flip is reverted after ten polls. Tonight's "sandbox-first" flips produced zero evidence.
- Fix (small): restart the unit now; make the unit `ExecStart` a tiny wrapper that re-execs when `flip.py` mtime changes (or have the watcher compare its own file hash each poll and exit, `Restart=always` brings the new code); the merge step restarts it when `tools/flip/` changes. Split the miss rule: a *parse* error is a tool fault (alarm, keep the window, never revert); only *transport* silence counts toward a revert.

### 1.2 HIGH: the watcher is blind in its own log

- V: `journalctl --user -u d2flip-watch` has no line after the start, although two REVERT lines and (per docs/process/dark-launch.md:53) the `mergeq.sh add` line "written to its log" were due. `Flipper.out` is `print` (flip.py:334) and the unit (flip.py:256-268) sets neither `-u` nor `PYTHONUNBUFFERED`; under systemd stdout is a pipe and block-buffered.
- V: after the two reverts no `agent/ledger-*` branch for the revert exists (one ledger branch in total, from the flip). Either `commit_ledger` failed (`LEDGER COMMIT FAILED`, flip.py:444) or never ran; the buffered log hides which.
- V: the installed unit has no `WorkingDirectory=` line although the template has one (flip.py:261): the unit on disk predates the template and nothing regenerates it.
- Fix: `print(..., flush=True)` in one `out` helper; a push notification on REVERT, REVERT FAILED, LEDGER COMMIT FAILED and CLOSE (the ops tree already has a push helper used by the merge watchdog); `flip.py watch --install-unit --check` in the CI hygiene step compares the installed unit with the template.

### 1.3 HIGH: "no hard breach in 2 h" is weak evidence

What the six Hard rows are (V, internal/bot/invariants.go:34-48): `potion-not-drunk` (8 s), `life-zero-alive` (3 s alive at life 0), `refusal-storm` (10 refusals without progress), `xp-frozen` (1800 s in the field), `mode-no-exit` (1200 s), `corpse-unresolved` (900 s). They are liveness alarms. Consequences:

| Gap | Evidence | Scenario |
|---|---|---|
| A death is not a breach | V: `tick` returns early when `f.dead` and clears the stopwatches (invariants.go:257-262); no Hard row counts deaths. The policy text says a death reverts (dark-launch.md:39) | A flipped combat knob makes the canary hero die four times in two hours, each corpse recovered inside 900 s: clean close, row `enabled`. In Nightmare or Hell each death costs experience and gold |
| No exposure requirement | V: `poll` closes when `clock >= end` and no counter moved (flip.py:516); nothing checks field time, kills, state or that the flipped code path ran | The bot waits for the Start tap (V: the status output appends "waiting for Start" and "auto-start held back" lines, which is what broke the parser) or the user has it paused: two idle hours close the window |
| Parked and paused silence four of six rows | V: `xp-frozen` resets on `paused` or `parked` or in town (invariants.go:325), `mode-no-exit` on paused, parked or Idle (336), `refusal-storm` on parked or paused (380) | Tonight's bundles contained `quests.refusalPark`, whose effect is to park. A park that never ends is, by construction, a clean window |
| Town-side failure is invisible | V: `xp-frozen` and `potion-not-drunk` need `field`; `town-thrash`, `no-earn`, `economy-stall`, `field-entry-no-kit`, `resupply-in-field`, `portal-not-cast`, `stuck-in-pack` are soft | A town knob that sells the wrong thing, buys nothing, or makes the hero leave town without potions never reverts |
| Slow effects need more than 2 h | V: `economy-stall` is 7200 s and soft; nothing compares rates | XP per hour down 30 percent, gold drained by a merc or repair loop, drop pickup rate down: invisible |
| Irreversible effects | S | A sold or dropped item, a spent quest reward (imbue, socket, respec), a spent stat or skill point, a discarded rune. The revert restores `tuning.json`, not the hero. V: `flip` backs up only the tuning text (flip.py:390-407); no save file is copied |
| A single hit can be noise | V: any counter above baseline reverts (flip.py:506). The ops flake list already names `TestRefusedHandledOnceNoStrike` | One `refusal-storm` from an unrelated cause reverts six knobs and stamps `revert_reason` on six rows |
| Counter saturation | V: `limit` rate-limits by name (60 s default gap, invariants.go:392-417), so `count` is "minutes with a hit", fine for a 0/1 test, useless as a rate | Any future "tolerate N" rule must not use `count` |

Two hours of clean liveness answers "did the bot hang" and nothing else. That is worth having; it is not what the row status `enabled` and the word "graduate" imply.

### 1.4 HIGH: harm a six-knob bundle can do to a played hero

- **The restart itself.** V: `flip` restarts the whole container (flip.py:141-144, default `container`) with no check of what the hero or the user is doing; a revert restarts it again. S: if the user is playing that hero, or the bot is mid-fight, the session dies twice in ten minutes (it did tonight). The `bot` restart mode is "U: not drilled" (dark-launch.md:44).
- **Whole-file restore clobbers other edits.** V: the backup is the full tuning text and `_restore` writes it back (flip.py:447-459). V: dashboard edits live in the same file (internal/bot/tuner.go:198). Any knob the user changed in the dashboard during the window is silently undone by a revert, and a revert after a failed-then-retried restore can land hours later.
- **Interaction and attribution.** V: a bundle is reverted as a whole; `bisect` prints one re-flip line minus one knob (flip.py:532-539). With six knobs that is up to six more 2 h windows, each with a restart, each as weak as the first, and it cannot find a pair interaction (A and B both needed). V: the ledger row is keyed by knob name only (flip.py:318-330): `quests.refusalPark` was in both bundles on both sandboxes; the second revert overwrote the first one's `revert_bundle`; had one closed and one reverted, the row would read whichever wrote last.
- **Bundle composition is by name prefix, not by interaction.** V (agent/autoflip, tools/flip/autoflip.json): `bot.`, `combat.`, `town.`, `nav.` all go to the canary in bundles of `max_bundle` 6, grouped by the text before the dot. Two knobs that both change the leave-town decision land in one bundle by accident of naming.
- **Build drift inside the window.** S: main took about 370 first-parent commits today (V: `git log --since`), and `dev-update` deploys are frequent. A deploy inside a window restarts the child, the baseline resets (flip.py:504-505), and a later breach is charged to the bundle although the build changed. The window does not record the build id at start or refuse/annotate when it changes. `ops: tools/autodeploy.py` (canary auto-deploy with its own hard-invariant rollback) is not installed on this host today (V: no timer, no deploy log); if enabled it becomes a second controller on the same signal with no knowledge of flip windows, and it parses the status with a bare `json.loads` (ops: tools/autodeploy.py:100), the same bug as 1.1.

### 1.5 HIGH: the ledger is edited in three places and is already wrong

The same rows are written by (a) the checkout or worktree where `flip` ran (`trial`), (b) the checkout the watcher unit runs from (`enabled` or `dark`), (c) feature branches (new rows, graduation edits) and the coordinator. Observed today (V):

- **Four orphan `trial` rows on main**: `bot.tickLogSec`, `bot.arbiterShadow` (window 1200 s from 06:01 UTC), `town.plan`, `combat.deliveryUpkeep` (2700 s from 13:40 UTC). No open window exists in the flip state dir. Nothing will ever move them; they count against the cap; whether those knobs are on in the sandboxes is U from the repo alone.
- **Six revert rows are uncommitted in the main checkout** (` M data/darklaunch.json`). S: the next pull of main that touches the ledger (25 commits touched it today; the base commit of this review already does) will refuse, the checkout goes stale, and with it the watcher's code and ledger.
- **The one ledger branch failed preflight** (queue item reason: leakcheck, 3 hits in 1 file) and sits unmerged; its worktree landed in a nested `d2lod-wt/d2lod-wt/` directory because `commit_ledger` derives the sibling path from `ROOT` (flip.py:286), which is wrong when `flip` runs from a worktree.
- **The race**: `flip` in worktree A writes `trial` to A's file; the watcher writes `enabled`/`dark` to the main checkout's file, where the row is still `dark` with main-checkout-old text. `commit_ledger` then copies the *whole row* from that file onto a branch from origin/main (flip.py:292-293): any field origin/main changed since the main checkout last pulled (`why_off`, `review_by`, a new `flip_plan`) is silently rolled back. Two branches (the flip's `trial`, the watcher's `enabled`) then race through the queue in either order; if the `trial` one lands last the row is wrong for good.
- **`data/darklaunch.json` is not in the union rules** (ops: tools/mergeq-union.txt has three globs, none for it) and new rows are appended at the end of one array: every pair of branches that add a row conflicts (V: three failures today, 3.1).

Root cause: the ledger mixes **design intent** (name, why, criteria, owner: changes with code, belongs in git) with **runtime state** (trial, window, revert: changes on the worker host at 3 a.m., does not belong in git). Fix in section 3.3 (B).

### 1.6 MEDIUM: other auto-revert gaps

- V: a dead watcher is a WARNING at flip time (flip.py:387-389), not a refusal; `autoflip run` adds no check. A window opened without a watcher is never closed or reverted.
- V: the window store is a local directory (`D2_FLIP_STATE`). A flip run from another machine or user is invisible to the watcher; again only the warning.
- V: `revert` on a failed restore retries every poll forever (flip.py:497-498) with no alarm beyond a line in the unreadable log (1.2).
- V: `flip_plan.instance` must be `doofer`, `questy` or `both` (tools/darklaunch/ledger.py:13), while the tools address containers by their real names (the `.last` files, `trial_instance` on main) and autoflip treats the string as a container name (agent/autoflip, autoflip.py `row_instances`: `"both"` becomes an instance called `both`, reported as "sandbox both unreadable"). The first row written under the new rule cannot be planned.
- V: `watch.paletteFeed` was already 1 in both sandboxes (`"old": 1`) and was flipped and "reverted" anyway; the tool does not notice a no-op flip, so its row can reach `enabled` having tested nothing new.

## 2. Graduation: what it should require, and what the tooling lacks

### 2.1 What graduation is

Flipping a code default and deleting the knob removes the only cheap undo and ships to every hero and every build, not two sandbox heroes. It deserves more evidence than the sandbox flip, not the same. It also should not need a wait: all of the below except (c) is offline and takes minutes.

### 2.2 Proposed evidence record (one JSON per knob, written by tooling, checked by CI)

a. **Exercise proof**: a counter that the gated code path ran N times in the window (a `knobhit` counter per knob in the explain view, incremented where the knob is read as non-zero and changes a decision). Zero hits means the window is void, not clean.
b. **Exposure**: field minutes, kills and areas in the window above a floor (for example 60 field minutes, not paused, not parked beyond the knob's own design). Read from the same status view.
c. **Outcome deltas against the hero's own previous windows**: deaths, XP per field hour, gold delta, potions bought, town trips, soft-invariant minutes (all rows, not only Hard). Thresholds are per knob in `enable_criteria`, which today is free text with `criteria_met: U` on 26 of 29 open rows (V).
d. **Offline A/B**: the accelerated lab soak or Proving Ground scenario set with knob 0 and knob 1 on the same seeds, for two builds (the AGENTS.md two-build rule), with the invariant totals compared. This is the evidence the policy text already names ("the accelerated lab soak, scenarios and replays are the evidence", dark-launch.md:12) and no tool requires or records it.
e. **Item and economy ledger diff**: inventory, stash and gold before and after the window (a save snapshot diff), reviewed for unexpected losses.
f. **Both sandboxes or a stated reason for one**, and the build id that was live.
g. **Old-path deletion in the same card** with the scenario still green, so the graduation is a deletion, not a default flip that leaves dead branches.

### 2.3 What `autoflip.py` (agent/autoflip, not yet on main) lacks

- V: `graduates()` requires `not r.get("revert_reason")`, and `close` never clears that field. Every row false-reverted tonight is barred from the graduate list forever, and so is any row that was legitimately reverted, fixed and later closed clean.
- V: the only evidence it reads is `status == enabled` and `enabled_at`. No exposure, no exercise proof, no offline run, no second sandbox.
- V: `cmd_run` flips without `--commit-ledger`/`on_ledger`, so `trial` rows stay uncommitted in whichever checkout ran it (1.5), and it does not refuse when the watcher heartbeat is stale.
- V: `healthy()` refuses on any nonzero hard counter since process start, so one old hit blocks all flips on that sandbox until an unrelated restart.
- V: `skip_reason` tests `row.get("by_design")` (a field no row has); it works only because the status test below it catches the row.
- V: no ordering by risk or interaction, no "one knob family per decision point" rule; bundles are alphabetical slices.
- V: the graduate "brief" is a grep of the knob name; it does not check that a scenario covers the knob, nor emit the ledger edit.
- Missing commands: `reconcile` (read live `knobs` from both sandboxes, print rows whose status disagrees), `evidence <knob>` (assemble 2.2), `graduate --check` (CI gate: no default flip of a `[dark-launch]` knob without an evidence record).

### 2.4 The cap does not clear the ledger

V (data/darklaunch.json at base): 43 rows; 22 `dark` + 4 `trial` coordinator rows = 26 counted, cap 27, lowered by one per week (docs/process/dark-launch.md:27): about half a year to zero. Ten rows are `enabled` and uncounted, so the cheapest way to obey the cap is to flip and stop, which moves the pile from `dark` to `enabled` with the knob still in the code. No existing row has a `flip_plan`; the 3-day `review_by` applies only to rows from 2026-11-28; all open rows are due 2026-12-09 or later. Recommendation: count `enabled` rows too, with their own short clock (for example 3 days from `enabled_at` to a graduation card or a written reason), and ratchet to the real count after every graduation, not by one a week.

## 3. Merge pipeline

### 3.1 What failed today (V: worker log of the merge queue, items added 2026-11-27)

About 210 items were queued today. Conflict failures at `prepare` by file: task cards under `backlog/tasks/` in 11 items (distinct cards 424, 454, 716, 726, 766, 880, 970, 982, 983, 1016, 1056); `data/darklaunch.json` in 2 items (3 attempts); plan data JSON (`docs/plan/data/admission.json`, `potions-belt.json`) 2; real code conflicts 5 (`internal/bot/explain.go`, `funwire.go` twice, `fake.go`, a test); one doc each for `docs/gear.md`, `docs/questboard.md`; 2 leakcheck preflights; 3 `go-whole` and 1 `go-race` gate failures. `docs/knobs.md` did not fail today (V: 35 commits touched it; the regenerate-on-conflict rule in ops: tools/mergeq.py:599-603 works). The generated shared files did not fail either. So the churn is cards first, ledger second, and the rest is honest.

### 3.2 Root causes

1. **Cards have two writers by design.** The branch ticks boxes and fills `result:`; main gets status edits from auto-Done, from the coordinator's cleanup passes (V: `cardclose4: backlog status sync (7 Done, 6 In Progress live-agent)` at 21:10, the fourth today) and from `backlog task edit`, which rewrites quote style and `updated_date` on every touch (AGENTS.md says so). 434 commits touched `backlog/tasks/` today. A cleanup pass that sets "In Progress" on the card of a *live* agent is a guaranteed conflict with that agent's branch. The conflicting cards are mostly old numbers (424 to 983): long-lived cards re-touched by sweeps.
2. **No resolver for cards.** ops: tools/mergeq.py `prepare` has fixed rules for coverage, OWNERS, STATUS, research, knobs, plan data, then the union list; a card falls to "no union rule" (the log even prints `last on origin/main: ?` because the path has spaces or was renamed).
3. **`data/darklaunch.json`**: one array, appended at the tail, plus runtime fields (1.5).
4. **Plan data JSON** has a copy under `docs/plan/data/` and the rule only fires when the same file exists in `internal/plandata` (mergeq.py:604); the two failures were real two-sided edits to one table.
5. **Throughput holds.** V: batching of Go-diff items is off by default (`MERGEQ_BATCH_CODE=0`, mergeq.py:114), so every code item pays the 3 to 4 minute `go-whole` gate alone; at about 200 items a day that is the queue's floor. `NOWAIT_MAX` 3 pushed heads may have unresolved CI; a red main holds everything (the log shows runs of "holding: CI of main is red/pending").

### 3.3 Structural fixes, ranked by benefit over effort

| # | Fix | Effort | Benefit |
|---|---|---|---|
| A | **Card frontmatter resolver** in `prepare`: parse both sides' YAML header; take `status`, `assignee`, labels from main; take `result:`, ticked boxes and body from the branch (a box ticked on either side stays ticked); `updated_date` = max. Falls back to a failure only when the body text conflicts | half a day, ops only | removes the largest class (11 of about 20 today) |
| B | **Split the ledger**: one file per knob `data/darklaunch/<knob>.json` holding design fields only (git); runtime state (`trial`, window, `enabled_at`, reverts, evidence) in a state directory on the worker host, exported read-only into the generated STATUS block. `check.py` and `report.py` read both | one day (tools/darklaunch, tools/flip, tests) | ends add/add conflicts, the three-writer race, ledger branches, the dirty main checkout and the orphan `trial` rows in one move |
| C | **Stop sweeping cards of live agents**: cleanup passes skip any card whose `branch:` has an unmerged local branch or a queue item; never run `backlog task edit` on them | one hour | removes the cause of A rather than its symptom; do both |
| D | Interim for B (ten minutes): add `json-list data/darklaunch.json items` keyed by `name` to ops: tools/mergeq-union.txt | minutes | today's ledger conflicts gone until B lands |
| E | **Status out of the card file**: `status:` derived at merge from git (the drift tool already computes it) and written only by the merge step in the generated commit; agents and sweeps never write it | one to two days, touches the Backlog.md CLI habits | makes "stale status" and the cleanup passes unnecessary; larger change, do after A and C |
| F | **Batch Go items** (`MERGEQ_BATCH_CODE=1`, max 4) (U: whether `merge_batch.sh` bisects a red batch; it must before this is turned on) | a flag plus a day of watching | roughly halves gate time per item at this volume |
| G | Per-card `result` files (`backlog/results/<card>.md`) | half a day | small on its own: A already resolves `result:`; skip unless E is rejected |
| H | `merge=union` in `.gitattributes` | minutes | **do not**: line union corrupts JSON and YAML silently; the keyed resolvers in ops are the right form |
| I | Stop committing generated files (`docs/knobs.md`, INDEX, STATUS, OWNERS) | medium; every doc link to them breaks (S) | low today: these no longer cause failures (V 3.1). Not now |

### 3.4 Smaller items

- The preflight leakcheck failure of the ledger branch should name the hit; the queue reason carries only counts, and a tool-written row (free-text `--why`, container names) is a standing leak risk. B removes it.
- `Remaining (live)` convention and live-check boxes: auto-Done already moves live-only boxes to `docs/live-checks.md`. The dark ledger and the live-check list are the same backlog of "unverified on a real hero" under two names; a row should link the live-check lines it closes, and a graduation should close them.
- The hygiene preflight (35 s) and `go-whole` (3 to 4 min) are fine individually; the cost is serial count, see F.

## 4. Ten improvements, in order

1. **Tonight**: restart `d2flip-watch`; commit or discard the six revert rows in the main checkout; clear `revert_reason` on rows reverted by the parser fault; reconcile the four orphan `trial` rows against the live knobs of both sandboxes. (1.1, 1.5)
2. **Watcher fault vs bot fault**: parse errors alarm and never revert; watcher reloads on code change; flushed log; push on REVERT / REVERT FAILED / CLOSE; `flip` and `autoflip run` refuse without a fresh heartbeat. (1.1, 1.2, 1.6)
3. **Exposure and exercise proof before CLOSE**: a window with too little field time, a paused or parked hero, or zero hits of the gated path is `void` and extends, never `enabled`. Add a per-knob hit counter to the explain view. (1.3, 2.2 a, b)
4. **Count deaths**: a Hard `death` row (or a watcher rule on the death counter with a per-window allowance from the hero's own baseline), and make the policy text match the code. (1.3)
5. **Save snapshot at flip**: copy the hero's save and stash files next to the tuning backup, keep them a few days; diff inventory and gold at close. The only undo for item and quest harm. (1.3, 1.4)
6. **Gentle, scoped writes**: flip only when the hero is in town and the user is not driving; restore only the bundle's keys, not the whole file; drill and default the `bot` restart mode; record the build id and annotate a window whose build changed. (1.4)
7. **Split the ledger** into per-knob design files in git and runtime state outside git (fix B), with fix D tonight as the bridge. (1.5, 3.3)
8. **Card resolver plus no sweeps on live cards** (fixes A and C). (3.2)
9. **Graduation gate**: an evidence record per knob (2.2), `autoflip graduate` assembles it, CI refuses a default flip of a `[dark-launch]` knob without it; fix the `revert_reason` bar and the `both`/container-name mismatch before autoflip merges. (2.3, 1.6)
10. **Make the cap bite**: count `enabled` rows with a 3-day clock to graduation, give the 26 grandfathered rows a `flip_plan` and a date this week, ratchet to the real count after each graduation; bundle by decision point (one family per window on the hero that exercises it), not by name prefix. (2.4, 1.4)

## 5. Not checked (U)

- The live knob values in both sandboxes (no sandbox was touched for this review), so the true state of the four orphan `trial` rows.
- Why the watcher's ledger commit after the reverts produced no branch (log buffered, 1.2).
- The exact leakcheck hits on the ledger branch.
- `tools/dev-update.sh` local mode and the deploy-window script beyond their headers; `ops: tools/merge.sh` and `merge_batch.sh` internals; `test_flip.py` coverage.
- Whether any of tonight's two forced restarts cost the heroes anything.
