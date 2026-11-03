# Gate 3 requirement gaps: card set, design and work packages (card gate3gaps)

> status: design, docs only. Source: section 2 of docs/research/gate3-prep-20261103.md (branch agent/gate3prep). Code facts below are V (read in this worktree: internal/d2dsoakstat, cmd/d2dsoak, internal/bot/observe.go); ops-repo facts (gate-sup.sh, gate-judge.py) are S, taken from the prep doc, because the ops repo is not readable here. Numbers marked U are proposals. The delta spec is `openspec/changes/gate3gaps/specs/soak-gates/spec.md`; no existing spec is edited.

## 1. Resource sampling (prep row 11, 12)

Gap (S): the d2d gate runner writes only the d2d `VmRSS` to `mem.csv`; the spec "Resource slopes" has no data on d2d. Also (V): `d2dsoakstat` and `d2dsoak` never read a `mem.csv` (grep: no reader), the judge for it lives in the ops repo.

### 1.1 What is sampled (gate-sup.sh, ops repo text)

Once per wall minute (period 60 s, aligned to the minute so 24 h gives 1440 rows), for the d2d pid and the d2bot pid of the slot (pids from the process registry, not `pgrep` by name):

| column | source |
|---|---|
| `t` | epoch seconds |
| `vmin` | last virtual minute from the newest `msg=minute` line of the bot log (empty if none) |
| `d2d_pid`, `bot_pid` | registry pids |
| `d2d_rss_kb`, `bot_rss_kb` | `VmRSS` line of `/proc/<pid>/status` |
| `d2d_thr`, `bot_thr` | `Threads` line of the same file |
| `d2d_fd`, `bot_fd` | count of entries of `/proc/<pid>/fd` (`ls | wc -l`) |
| `log_kb` | `du -sk` of the slot log directory |
| `disk_free_gb` | free space of the log filesystem |

Header line first; a missing process writes empty fields, never a skipped row. Goroutines are not readable from `/proc`: the bot's thread count is the proxy (U; a later item may read the bot's debug endpoint if one exists). The old `mem.csv` name and its first columns stay (`t`, `d2d_rss_kb`) so the current judge keeps working.

### 1.2 Judge (ops-repo text plus a repo-side library)

Proposal: put the maths in this repo so it is tested, and let the ops judge call it.
- New file `internal/d2dsoakstat/res.go`: `ParseMemCSV(io.Reader) ([]ResSample, error)`, `SlopePerHour(ts, vs []float64, warmupMin float64) (slope float64, n int)` (ordinary least squares after the first 10 minutes of each pid segment), `ResourceRows(samples []ResSample, lim ResLimits) []Row` with `DefaultResLimits()` = fd 5/h, threads 5/h, RSS 40 MB/h (spec numbers, V) and a minimum of 30 samples after warm-up (U), else the row is NO-DATA.
- New subcommand in `cmd/d2dsoak/main.go`: `d2dsoak resources -csv mem.csv [-json]` printing one row per series (d2d fd, d2d threads, d2d RSS, bot fd, bot threads, bot RSS) with slope, limit, PASS/FAIL/NO-DATA and the peak; exit code follows `verdictExit`. The ops `gate-judge.py` calls it and copies the rows into the record.
- Pid segments: a change of `d2d_pid` (injected fault or planned recycle) cuts the series; each segment is fitted separately after its own warm-up and the worst slope counts (spec scenario "Restart inside the series").
- RSS drops at a recycle must not be read as negative leak; the segment rule handles that.
- Heap runaway (8192 MB or +1024 MB in 5 min) can use the same file: `ResourceRows` also returns a `heap_runaway` row from `bot_rss_kb` (the spec requirement exists already, V).

Thresholds: spec numbers (V) for fd, threads, RSS. That they fit a d2d game process is U; the Gate 2 record peaked at 838 MB without trend (S). Proposal: the first Gate 3 run reports slopes as FAIL by the spec numbers, and the user recalibrates after the run.

## 2. Restarts, primitives invariant, logs, inventory

### 2.1 Planned versus unplanned d2d restarts (prep rows 1, 13, 19)

The code (V): `ScanSupervisor` counts `launch attempt` lines into `ExitInfo.Launches`; `ScanInvariants` turns launches beyond the first into `BotRestarts` and violates RESTARTS on any bot restart (`violations`, invariants.go). It has no notion of a planned restart. The judge (S) counts `d2d process exited` minus injected faults as unplanned.

Proposal: the supervisor writes `planned recycle: reason=<r>` before a deliberate d2d stop (injected fault gets `planned recycle: reason=fault`). Judge: unplanned = exits without the preceding line. Whether the product should recycle d2d periodically at all is a user decision (no recycle exists today, S); until it does, the planned set is the injected faults and the restart interval is simply the recorded gap list. Repo side: `ExitInfo` gets `Recycles int`, `ScanSupervisor` counts the line, and RESTARTS ignores bot restarts equal to `Recycles` (small change in `violations`). The d2d-only restart interval is a record row (INFO) in the judge.

### 2.2 Primitives invariant (prep row 8)

Answer from the code (V): `d2dsoak` does NOT compute it. `grep -ri primitive` over `internal/d2dsoakstat` and `cmd/d2dsoak` (non-test) finds nothing; the invariants of `Invariants` are HANG, STALL, WEDGE, PROGRESS, BELT, NOKILL, DEATHS, RESTARTS, EXIT. So "`INVARIANTS: PASS` including the primitives invariant" in the Gate 2 and 3 spec text cannot be met. The delta spec drops the words from the requirement text (the primitives live in Gate 0, `cmd/d2drill`, where they are judged). Alternative if the user wants it in the soak: a PRIMITIVES invariant counting failed primitive log lines per kind; that needs a log vocabulary that does not exist (U), so it is not proposed.

### 2.3 Log size (prep row 14)

Facts: no mid-run guard, no rotation (S); size at 24 h is unmeasured (U). Policy proposal (U numbers): all slot logs go to one directory on the run disk `<run>/logs/` (bot log, d2d log, supervisor.log, mem.csv); `bot-*.log` rotated by the supervisor at 1 GB per file and the closed files gzipped (the parsers read `.gz` through the chain loader; check `loadChained` in cmd/d2dsoak/main.go, which takes a path list); the 24 h total must stay under 20 GB; mem.csv records `log_kb` and `disk_free_gb` each minute; the runner stops with an alert under 5 GB free (spec delta). Before Gate 3: measure `du` at the end of Gate 2 and multiply by 3 (the prep rule of "keep 3x free"); if that is over 20 GB, rotate by the above.

### 2.4 Inventory fill (prep row 15)

Fact (V): the bot emits `invPct` (internal/bot/observe.go, -1 unknown) but `d2dsoakstat.Minute` has no field for it, so no report reads it. Criterion proposal: `invPct` above 95 outside town for more than 60 consecutive minute lines (virtual minutes) FAILS (U threshold; docs/primitives-review.md saw pack-full at about 85, so 95 for an hour is a stuck sell cadence, not noise). Report an INFO row per 4 virtual-hour block: mean, max, share of minutes above 90. Repo work: add `InvPct int` and `HasInv bool` to `Minute` (minute.go `minuteFromKV`), a `FullBagMin` counter and `InvLimits.FullBagMin = 60` in invariants.go. It is a judged row, not an invariant that changes the exit code (spec: invariants never change exit).

## 3. Gate 0.5 smoke profile (45 wall minutes)

Why: the Gate 2 failures of 2026-11-02 (S, prep section 0) showed at virtual minute 7 (merc), 432 (BELT) and 864 (NOKILL). At 12x, 45 wall minutes is 9 virtual hours, which would have covered the BELT and NOKILL windows.

- Profile m24 on a hidden slot, wall limit 45 min (spec says wall limits stay wall, V), fault at 15 wall minutes (about 3 virtual hours, early enough that a bad recovery shows and late enough that the kit exists).
- Judged (subset): BELT, NOKILL, PROGRESS, WEDGE, HANG, DEATHS, the injected fault recovery (restart within 3 minutes, exactly one), the kit criterion, presence of resource samples. Kit criterion: at each town departure after the first, packHP plus belt potions at least 1 (from the minute fields already parsed: `PackHP`, `BeltBoxes`, `hasHealing`). That is a new small counter `KitMissing` (U) in `ScanInvariants`.
- Not judged: XP floors (1000 per hour), the 90 minute no-gain window, clvl 16, the Andariel attempt, the merc hire, resource slopes (too short for a 5 per hour slope: 9 virtual hours is under 1 wall hour of samples), the memory curve and all 8 h and 24 h effects.
- Cannot catch: leaks that appear over hours, the 15 virtual minute town resupply seen late, a bot that stalls only after clvl 14, restart of d2d after many hours, log growth, host-load slow-downs (the clock may reach only 3 to 6x: then NO-DATA, spec scenario), Act or area transitions beyond the first hours.
- Implementation: `gate-run.sh --profile smoke` in the ops repo (wall 45 min, `--fault-at 15`, subset flag); repo side `d2dsoak report -subset smoke` which prints only those rows and ends with `SMOKE: PASS|FAIL|NO-DATA` (U name).

## 4. Work packages (prioritised)

| # | package | where | tier | effort h | depends |
|---|---|---|---|---|---|
| 1 | gate-sup.sh sampling columns, 60 s period, pid registry, `log_kb`, `disk_free_gb` (1.1) | ops repo | Sonnet | 2 | none; do first so Gate 2's tail already records |
| 2 | `res.go`, `d2dsoak resources`, tests with a synthetic CSV (1.2) | repo | Sonnet (Go) | 4 | 1 for live data, tests do not need it |
| 3 | judge calls `d2dsoak resources` and prints rows; disk guard 5 GB (spec delta) | ops repo | Sonnet | 2 | 2 |
| 4 | `Minute.InvPct`, full-bag counter and per-block INFO rows (2.4) | repo | Sonnet (Go) | 3 | none |
| 5 | planned recycle line plus `Recycles` in ExitInfo, RESTARTS rule (2.1) | repo plus ops | Sonnet | 3 | user decision on recycling |
| 6 | spec text: remove "primitives invariant" wording from Gate 2/3 (2.2), merge the delta | coordinator | Opus review | 1 | none |
| 7 | Gate 0.5 smoke: `KitMissing`, `-subset smoke`, runner profile (3) | repo plus ops | Sonnet (Go) | 5 | 4 optional |
| 8 | log rotation and gzip in the supervisor, `du` measurement at Gate 2 end (2.3) | ops repo | Sonnet | 3 | Gate 2 data |
| 9 | per-4h-block XP/h and kill rate INFO rows (prep rows 3, 9) | repo | Sonnet | 3 | none |

Order: 1, 2, 3 (resource gap, the only "MUST" without data), then 6 and 7 (cheap early warning before the next Gate 2), 4, 5, 9, 8. Total about 26 h of agent work, 3 of coordinator work.

## 5. Ops-repo text (for the ops repo, not applied here)

- gate-sup.sh: append one row per 60 s to `mem.csv` with the columns of 1.1; write `planned recycle: reason=...` to supervisor.log before any deliberate d2d stop; stop the run with an alert when `disk_free_gb` is under 5.
- gate-judge.py: add rows from `d2dsoak resources -csv mem.csv -json`; unplanned d2d exit = exits minus `planned recycle` lines; keep the old INFO memory curve.
- gate-run.sh: `--profile smoke` as in section 3.
- docs/gate-run.md: describe the new columns and the smoke rung.

## 6. Limits

All thresholds marked U are proposals; the ops-repo script facts are S (from the prep doc, not re-read); the live host was not used; no code was written.

## Shared-file lines for the merger
See openspec/changes/gate3gaps/merge-lines.txt.
