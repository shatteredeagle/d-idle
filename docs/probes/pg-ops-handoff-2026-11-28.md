# Proving Ground: ops-side items after card fix6 (2026-11-28)

> status: handoff list, 2026-11-28. Nothing here is installed or run; card fix6 changed the repo only. Source: `docs/design/fable-review-2026-11-28/06-proving-ground.md` and `docs/process/proving-ground.md` section 14. Tags: V read or run on the branch, S secondary, U unverified.

Each item is for the ops repo or the lab hosts; none is done.

1. Merge worker: require the marker line `smoke: static N scenarios, prep 4, ledger 4, blocked 0` (from `tools/ci/check.sh`, or `go run ./cmd/d2scenario smoke` directly) when the diff touches `cmd/d2scenario/`, `internal/scenario/`, `tools/provingground/`, and the prep's dependencies `internal/d2dctl/`, `internal/d2dinput/`, `internal/leveler/`, `internal/plandata/`. The check script only scopes on the first three (V).
2. Scoped gate (`tools/gate_scope.py` in the ops repo, S from the review): it ignores `.sh` and `.py`, so `run.sh`, `pgenv.sh`, `nightly.sh` and `featuresoak.py` change untested there. `check.sh` now runs `nightcheck_test.sh` and `pgenv_test.sh` with the smoke (V).
3. Nightly timer: `tools/provingground/nightly.sh` has no timer or service on any host (S). Install it, and reopen criterion 1 of TASK-754 until one night has run. Not installed by this card on purpose.
4. After-push T2 canary (ADR 0027 item 2): the 7 T2 scenarios, first class and seed, on the builder long lane. Alarm on any row with page state `blocked` or `broken` and on exit code 4 of `d2scenario run` (`NOT RECORDED: N runs, M ledger lines`).
5. Exit code 4: `tools/provingground/nightly.sh` reports any non-zero job as `exit N` and goes on (V); the board and the push notice should name 4 apart from 3 (not green).
6. `ledger-refused.jsonl`: pull it with `ledger.jsonl` from each output directory; a non-empty file is an alarm (a run happened and the ledger does not have it).
7. Board page `/pg`: `page-json` cells carry two new fields, `state` (`blocked`, `broken`, or the stability word) and `blocked` (count). Show `state`; the `main` view now includes runs stamped `agent/*`, so show the `branch` of the latest run.
8. Ledger ingest on the board host: lines carry the new field `bot_sha256`, and BLOCKED lines may say `"setup":"UNBUILT"`, `"bot_commit":"missing"`, `"d2d_sha256":"missing"`. Check the ingest command accepts them (U; `d2scenario ledger check` does, V).
9. Reproducible bot build: `run.sh` now builds with `-trimpath -buildvcs=false`. Verify on the builder that two commits which do not touch `cmd/d2bot` and its imports give one `bot_sha256` (U), and that `D2LOD_BOT_COMMIT` is set on a tree without `.git` (the `tree-<hash>` stamp, V in the script).
10. Live check of this card (not run): one T2 row on the lab d2d, for example `tools/provingground/run.sh -id town.resupply -class Paladin -seed 11`, then `tools/provingground/run.sh -id town.plan.belt-refill-from-pack`. Expect `results N, ledger N`, no `ledger-refused.jsonl`, and the second no longer BLOCKED on `no room` (its 34 of 40 cells assume two starter scrolls, U per class).
11. Regenerate `docs/process/proving-ground-status.md` on the board host with `tools/provingground/status_doc.sh LEDGER.jsonl`; the checked-in page was regenerated from an empty ledger for the new columns (V).
12. Card TASK-1093 (pgfix) is merged with status Proposed and no result line (S from the review): close or correct it.
