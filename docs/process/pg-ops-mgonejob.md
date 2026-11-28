# Merge gate as one locked builder job (card mgonejob, TASK-745)

> status: repo half built 2026-11-28 (card road5), ops half NOT done. Design: `docs/design/fable-review-2026-11-21/proving-ground-gate.md` sections 4.1, 4.2 and 4.7, card 11. The merge script and the queue tool are in the private ops repo; this page describes them by role. Tags: V run on the branch, S secondary, U unverified.

## What exists in this repo (V, `go test ./internal/gatemark ./cmd/gatemark`)

`cmd/gatemark` reads the `go test -json` stream of a gate run. It starts nothing and reads only its input.

| need (design) | what `gatemark run` does |
|---|---|
| positive ran-marker instead of the timing guess (4.2) | prints `gate ran: packages N of M (ok A, no test files B, failed C, cached D), tests T` only when at least one package reported and N equals `-want`; otherwise `gate NOT RUN: ...` and exit 3. A gate answered from the cache in a second is `ran`; an empty stream (wrong directory) is not |
| failing tests and tail stored (4.7) | prints `FAIL <package> <test>` per failing test (the failing subtest, not its parent; a package that failed without a test is named alone) and writes `-record FILE` with the failures, the last 60 output lines of the failing packages, `-commit`, `-job`, `-load` |
| flake ledger (4.7) | appends to `-ledger FILE` one run line (commit, green or not) and one line per failing test; a run that did not run writes no line |
| quarantine list (4.7) | `gatemark quarantine -ledger FILE` prints `tools/ci/flaky.txt` rows for tests that failed on two commits which also passed the gate unchanged; a failure on a commit that never went green is not counted |

Exit codes of `run`: 0 green, 1 a test or package failed, 2 usage or unreadable input (plain text instead of `-json` is an error), 3 did not run.

## What is left, in the ops repo (none of it done, U)

1. One locked builder job: the merge script calls the builder once per code merge (the builder's gate mode does vet, whole-repo test and race under one lock, S design 4.1) instead of twice.
2. In that job: `want=$(go list ./... | wc -l)`, then `go test -json -short -p 4 ./... | gatemark run -want "$want" -commit "$sha" -job "$job" -load "$load" -record "$out/gate-fail.json" -ledger "$ledger"` with `set -o pipefail` off for the pipe and the exit code read from `gatemark` (it is the verdict; `go test` exits 1 on a red gate by design).
3. Replace the minimum-seconds guess of the merge script by: the output contains a line that starts with `gate ran:`. Without it the item fails as `gate did not run`.
4. Store `gate-fail.json` with the queue item (today its result text is empty, S design 4.1) and show its `failures` in the queue status.
5. Keep the ledger on the head box outside git; run `gatemark quarantine` in the coordinator's tick and put its rows before the owner or coordinator (a row is a proposal: `tools/ci/flaky.txt` is edited by a card, never by the tool).
6. Race step: a second `gatemark run` over the `-race -json` stream of the touched packages, `-want` their count, the same ledger.

Check after wiring (ops repo test, then one real merge): a merge whose gate job is started in an empty directory fails with `gate NOT RUN`; a red merge leaves a record naming the test; two red-then-green merges of one test print its quarantine row.
