# Agency P1b: the planner drives (mode 2)

> status: partial (TASK-984, 2026-11-27). Offline pieces built; the review and the flip are the owner's. Tags: V = ran in a test, U = unverified.

## Built (offline, V `go test -p 1 ./internal/agency -run DriveOf`)
`agency.DriveOf(plan)` (`internal/agency/drive.go`) is the decision mode 2 hands the fighter: an attack or aura override only when the plan's pick is `supported`, differs from the combat profile's own pick and the archetype resolved; otherwise "keep profile" with a one-line reason. So mode 2 can differ from mode 0 only where the shadow line already shows a different pick. The viability line already ships in the view (mode 1).

## Not built, and why
- The knob `agency.planner` still has Max 1: raising it before the fighter reads `DriveOf` would accept a value that does nothing. Wiring (U): in `cmd/d2bot/agencywire.go` expose the last `Drive` and have the combat profile selection consult it once per fight start; log `planner drive:` with `Drive.Why`. The ledger row (`data/darklaunch.json`, `agency.planner`) already covers mode 2 ("owner go for mode 2").
- AC #1, review of the shadow lines: the offline replay is the agreement fixture (`internal/agency/testdata/agreement.json`, `UPDATE=1 go test ./internal/agency -run AgreementFixture`; 25 of 30 agree, each mismatch has a `why`). The real shadow lines come from a sandbox (U).

## Owner / live only
1. Read the `planner shadow:` lines of the sandbox heroes (a day or more) and record that no pick is wrong.
2. Owner go for mode 2.
3. Wire the fighter to `DriveOf`, raise the knob Max to 2, run the Paladin and Sorceress drill at clvl 18 with writes disabled, compare kills and quest progress mode 2 vs 0.
4. Flip on the sandboxes with a 24 h invariant-monitor watch and auto-revert.
