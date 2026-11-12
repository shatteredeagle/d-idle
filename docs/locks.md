# Lock order and rules

> status: current (lock audit, 2026-11-12, after the sandbox freeze of 2026-11-12: nav.roam called done() under covMu while the observer held obMu and called NotePosition; fixed in #618, TASK-358). Tags: V = read in code and pinned by a test, S = read in code only, U = suspected, not proven.

The freeze class: a lock held across a callback, a call into another component or a blocking call. Every handler that
reads the runner (`LastSnapshot`, `Health`, `DeathReport`, `Progress`) takes `obMu`, so anything that holds `obMu`
for long, or waits on a lock whose holder waits for `obMu`, stalls the status panels, watch handlers, repair and the
readiness check at once (62 goroutines in the incident dump).

## Rules
1. No callback under a lock: copy the state under the lock, release, then call (`done`, `stopWhen`, `onCover`, `OnSnapshot`, `Pinned`, Desk and Hub hooks, `Health.Step`).
2. No blocking call under a lock: no channel send or receive, network or control-socket verb, `time.Sleep`, and no file read on a slow path. Exceptions are listed below and need a bound.
3. Never hold `obMu` across a call into another component. Handlers (watch, status, display, repair) read through `Runner.LastSnapshot` and friends and must not keep a Runner lock themselves.
4. A lock that is a leaf (its holder calls nothing that locks) may be held briefly under any other lock.
5. Guard: `internal/bot/lockguard_test.go` fails when `r.obMu` or `n.covMu` is held (straight-line Lock to Unlock, or to the end after `defer Unlock`) across `OnSnapshot`, `onCover`, `done`, `stopWhen` or `Health.Step`. Regression tests: `internal/bot/lockorder_test.go` (2 s deadlock detector), nav `TestRoamNeverCallsDoneUnderCovMu`.

## Named locks and allowed nesting
| Lock | Protects | May be held while taking |
|---|---|---|
| `Runner.obMu` | observer state (`ob`, last snapshot, bans, trips) | `bus.mu` (Publish never blocks), `Runner.mu` (briefly, in `minuteLine`), `killMu`, `hmu`. Never `covMu`, never `healthMu`, never a callback. V |
| `Runner.mu` | control state (goal, paused, state) | leaf (`Status`, `Pause`, `Stop` take only it). V |
| `Runner.healthMu` | serialises `Health.Step` (it may ping the d2d) | nothing; taken with NO other Runner lock held. V (new) |
| `Navigator.covMu` | roam coverage | nothing that takes `obMu`; `done()`/`onCover` run after release. V (#618) |
| `Navigator.planMu` | the shared potential field | leaf, `tune()` is a pure getter. S |
| `goals.Manager.mu` | goal list, mode | `Runner.Status/Pause/Stop/Progress` (so `Runner.mu`, and `obMu` through `Progress`/`Chain`). The runner never calls the manager under its locks: `emit` is a bus publish or a non-blocking channel push. V |
| `pact.Controller.mu` | pact state | `goals.Manager.Pause` through `cfg.Pause`. Nothing takes `pact.mu` under `Manager.mu`. S |
| `decide.Store.mu` | thoughts, rules, log | subscribers: `watch.Hub.mu`, `watch.Session.mu` (event callbacks run under it, by design). Subscribers must not call the Store. Callers of the Store must not hold `Hub.mu` or `Session.mu` (Hub copies with `snap()`, `Session` reads the log before its own lock). S |
| `watch.Hub.mu` | cached hero, thought records | leaf. `heroNow()` is taken by `QuestHints` and the order queue under their own locks: order `QuestHints.mu` / `OrderQueue.mu` -> `Hub.mu`. S |
| `OrderQueue.mu` | unlock order queue | `cfg.Halt` -> `pactHalt` -> `pact.Controller.mu` (read only). S |
| `invdesk.Desk.mu` | pins, queue, recent | copies `Config` hooks under the lock and calls `Snap`/`Inventory`/`SellBlock` after release. V (read) |
| `status.Hub.mu`, `event.Bus.mu`, `campaign.Controller.mu`, `loot.Memory.mu`, repair `trip.mu` | their own state | leaves in the code read. S |
| `d2dctl.Client.mu` | one control connection | holds across the exchange on purpose (one client per socket); callers must not hold a Runner lock. S |

## Known exceptions (bounded, not fixed)
- `d2denv.QuestSource.progress` holds its `s.mu` across a control-socket read with a 5 s context (single-flight cache). It is reached through `bot.pollQuests` and other `Quests.Read` callers that run under `obMu` (`pollQuests` documents "called with obMu held"). On the d2d backend a busy socket can therefore hold `obMu` for up to 5 s once per cache age. U (not seen live). Fix when it bites: have `pollQuests` return a job and read after the unlock, like `OnSnapshot` now.
- `decide.Store.commit` saves its log file under `Store.mu`; small local write. S
- `launch`/`d2denv` process start-up code holds `Proc.mu` across `exec`. Not reachable from a Runner lock. S

## Fixed in this audit
- `Runner.pollWith` called `Deps.OnSnapshot` (display tracker) under `obMu`; a callback that read the runner hung the poll. Now called after the unlock. V (test `TestPollNeverHoldsObMuAcrossOnSnapshot`).
- `Runner.pollWith` stepped the launch monitor under `obMu`; on d2d `Health.Step` pings the control socket (seconds when a long step holds the client). Now `healthProbe` runs it under `healthMu` only. V (`TestPollNeverHoldsObMuAcrossHealthStep`).
- `Runner.expectGameExit(false)` did the same drop-step under `obMu`. Found by the guard test. V (`TestExpectGameExitOffKeepsObMuFreeDuringHealthStep`).
