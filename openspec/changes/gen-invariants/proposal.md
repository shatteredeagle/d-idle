## Why

The two overnight d2d soaks (docs/probes/navsoak-compare.md) ended in a permanent town wedge that the tooling did not flag: the supervisor saw a clean exit at the time limit and the report showed only a low XP/h, found by reading 200 MB of log. The generality study (docs/research/bot-generality.md section 3) separates invariants (the run is sound) from competence (the run is good). This card makes the first checkable per seed: counters and a verdict from data the bot already logs, plus a holdout seed convention so the 8 seed gate run says what transfers.

## What Changes

- internal/d2dsoakstat: `ScanInvariants` (one streaming pass over a bot log), seven invariants HANG, STALL, WEDGE, PROGRESS, DEATHS, RESTARTS, EXIT with thresholds, `BuildInvReport` (per seed, per group, coverage line, INVARIANTS verdict), `Run.Seed`, `Run.Inv`, `Run.Cover`.
- cmd/d2dsoak: run, extract and compare print the report and put it in `-json` and `verdict.txt`; `-holdout` (default 16-23); run.json carries the counters and the class and difficulty of the setup.
- Exit codes unchanged: the invariants verdict is a new field, never an exit code (the fidelity-gate verdict exit codes stay the only ones).
- Docs: docs/geninv.md, a section in docs/d2d-soak.md, raw validation under docs/probes/geninv.
- Delta: soak-gates ADDED `Per-seed invariants`, `Invariants verdict`, `Holdout seeds`.
