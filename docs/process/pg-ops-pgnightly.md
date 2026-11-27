# The nightly Proving Ground job (card pgnightly, TASK-754)

> status: repo side done 2026-11-27, scheduler side open (private ops repo). Design: `docs/design/fable-review-2026-11-21/proving-ground-gate.md` sections 3.2, 4.4, 4.6; ADR 0027 items 2 and 4 (owner answers Q2 and Q4). Handoff list for the ops side: `docs/probes/pg-nightly-earned-handoff-2026-11-27.md`.

## What the repo holds

| piece | where | what |
|---|---|---|
| the run set as data | `tools/provingground/nightly.json` (`didle.pg-nightly/1`) | window, job count, nice, smoke stage, three jobs with stages and caps |
| loader, validation, selection | `internal/scenario/nightly.go` | `LoadNightly` refuses a changed window, more than 3 jobs, no nice, a job longer than the window, a stage with no cap, a blocking smoke stage; `Expand` picks the runs of one night |
| the command | `d2scenario nightly check` and `nightly plan [-day D] [-format text|json|cmds] [-quarantined a,b]` | prints the night's runs; starts nothing |
| the job script | `tools/provingground/nightly.sh` | dry run by default; `--exec` refuses outside the window (exit 75), nices, cuts each call to the minutes left before 07:00 EST |

## The run set (the shipped file; change it there, not here)

- Window 23:00 to 07:00 EST (fixed UTC-5), at most 3 jobs a night, `nice -n 19`, lab slots only (hidden instances), the sandbox container untouched (ADR 0027 item 4, V `TestNightlyValidateRefuses`). U: item 4 says "at most 3 runs" without defining a run; the file reads it as 3 scheduler jobs, each of which may call `run.sh` several times. Say so if the user meant 3 scenario runs.
- Job `tiers`: T2 (first class, first seed), then T3 (all classes, first seed; the second seed joins on Sunday). Job `pairs`: 3 determinism pairs from T2 (`determinism-run -n 2`). Job `e2e`: one T4 item, only scenarios whose setup is not CHEATED (an EARNED act run feeds the release rule), rotating by day of year.
- Every stage has `cap_runs`. A stage over its cap is cut and the plan says so in a note; the cut window rotates by day so the same tail is not always dropped (V `TestNightlyExpandCapRotates`). Caps and wall limits are guesses (U) until the first nights are measured: T3 is about 570 runs against a cap of 120.
- Pending and quarantined scenarios are never planned (the quarantine list comes from the ledger stability state through `-quarantined` / `PG_QUARANTINED`).
- Smoke stage (acceptance criterion 2): T2, after each push, report-only, builder long lane, not bound by the window (it is not the second worker host); `clean_days_before_decision` is 7. After 7 clean days the user decides on blocking; nothing in the repo turns blocking on.

## Verified and not

- V (offline, no game): `go test -p 1 ./internal/scenario ./cmd/d2scenario` (window edges 23:00 and 07:00 EST, every refusal, expansion on fakes and on the shipped scenarios, quarantine, caps); `d2scenario nightly plan -day 2026-11-29` printed 134 runs in 3 jobs.
- U: no night has run; `nightly.sh --exec` was not run (it starts lab slots). The timer, the labrun call and the board posting are ops-side.
