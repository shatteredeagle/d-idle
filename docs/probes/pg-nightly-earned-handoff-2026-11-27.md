# Handoff: pgnightly and pgearned, what the ops side and the live check still need (2026-11-27)

> status: open list. Repo side is done (cards TASK-754 and TASK-756, branch agent/pgcards2); nothing here was run live. Tags: V verified offline by a named test, U unverified.

## Ops repo (private): scheduler and board

1. Timer on the second worker host, 23:00 EST daily (fixed UTC-5, no DST shift): calls `tools/provingground/nightly.sh --exec` from a synced worktree of `main` through the labrun long lane wrapper, `PG_EVENT=nightly` (already set by the script), `PG_QUARANTINED` filled from the stability state (`d2scenario ledger page-json` lists quarantined scenarios). The script refuses outside the window (exit 75); the timer must not set `PG_NIGHTLY_FORCE`.
2. After-push smoke: a merge-queue hook that, after each push to main, runs `d2scenario nightly plan -format cmds` stage T2 only, or simply `run.sh -tier ...` for the T2 ids, on the builder long lane with `PG_EVENT=smoke`, report-only: a RED never blocks. Count the consecutive clean days from the ledger and tell the user at 7 (Q2); do not wire blocking.
3. Night load guard: skip the night when the board host or builder is contended; one line in the board's night log with the plan notes (`d2scenario nightly plan` prints cap cuts).
4. Post the night's result: `d2scenario ledger check`, then `status_doc.sh` on the board host (docs/process/pg-ledger-ops.md); a RED stable scenario opens a hold-or-fix line (design 3.2).
5. Earned store: `PG_EARNED_DIR` on the builder (persistent disk, not the mirror), 14-day artifact rule does not apply to manifests; export hook after a GREEN EARNED T4 run: write `manifest.json` through `scenario.NewManifest` (a small `d2scenario earned export` command is the natural next piece; not written, it needs the run's save folder layout, U) with the save files copied next to it.
6. Release step: call `d2scenario ledger release-check -in <ledger> -commit <release commit>` before a release cut from the release after card pgearned (Q8); exit 3 stops the cut.
7. Do not stop, restart or touch the sandbox container; hidden instances only (the plan file asserts both).

## Live checks (need the game, lab slots, inside the night window)

- [ ] First night: `nightly.sh` dry run on the second worker, then one `--exec` with `PG_NIGHTLY_PARALLEL=1`; read T3 wall times to tune `cap_runs` and `max_wall_min` in `tools/provingground/nightly.json` (guesses today).
- [ ] Pick the first earned chain: run `necro.act1-earned` (tier T4, EARNED by the new tag rule, zero setup verbs) and confirm the ledger line says `EARNED` and `writes_after_start 0`.
- [ ] Save portability (AC2 of pgearned, the U of design 6.3): export that hero's save files after the run, load them on (a) the same d2d binary, (b) the next d2d build; read the hero back (level, skills, stash, quest records) and append a `loaded_on` row with `ok`, `refused` or `changed` to the manifest. Only `same-build` and `verified` saves may be used.
- [ ] Runner: copy the verified save into the instance in place of the setup step and remove the `Unsupported` entry in `BuildSetup` for `pre.from_save`; fill `from_save`, `save_run`, `save_portability` in the result (the fields exist in `Result` and the ledger line).
- [ ] First step scenario with `pre.from_save: clvl-12`, then its determinism pair.

## What the repo now enforces (V, `go test -p 1 ./internal/scenario ./cmd/d2scenario`)

Window and cap rules of the plan file; no CHEATED, IMPOSSIBLE, written, dirty or red run can export a save; a from_save scenario with any setup write or a field start is refused; a CHEATED scenario without the word CHEATED in its provenance is refused by `validate`; an EARNED ledger line with writes, or a from_save line without a verified save, is refused; the release check ignores CHEATED runs.
