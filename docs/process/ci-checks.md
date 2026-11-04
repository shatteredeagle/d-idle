# CI checks for the forge job (for the CI agent; no runner exists yet)

> status: current since OpenSpec removal stage 4. The old card checks (changecheck, the OpenSpec validator, `openspec/.strict.yaml`) are gone; the sections on `history_baseline` and branch-agnostic `changecheck -diff` below are history (tag `openspec-final`, `cmd/changecheck`).

Run from the repo root. The checks that exist:

```sh
python3 -I tools/contracts/check.py    # contract headers, requirements, numbers vs docs/knobs.md (docs/contracts/README.md)
python3 -I tools/backlog/check.py      # task files: goal label, owns, WIP cap, overlap
python3 tools/merge_lines_check.py --changed
tools/ci/check.sh                      # hygiene, merge lines, shims, contracts, gofmt, go vet
```

Do not run whole-repo tests on the head box (4 GB); on the worker host use `labrun worker -- ...`.

## leakcheck (change `pubsplit`)
```sh
go run ./cmd/leakcheck                       # warn mode (default): counts by class, exit 0; CI step "leakcheck (warn)" and the pre-push hook run this
go run ./cmd/leakcheck -mode fail            # exit 1 on any hit: switched on at go-live (edit the hook line and the CI step)
go run ./cmd/leakcheck -staged [-range A..B] # only changed files (index, or a pushed range); -list shows every hit, -json for a report
go test -p 1 -count=1 ./cmd/leakcheck
```
Rules, manifest and allow list: `docs/process/public-split.md`. Tags: V (run on the pubsplit branch tree).

## History-less repos and `history_baseline` (change `golivepre`)
The merge and diff checks read git history. They never fail for lack of it: on a shallow clone (`git rev-parse --is-shallow-repository`), a repo with a single root commit, or a failed `git log` when no baseline is set, `changecheck` prints `INFO no history before the baseline: merge/diff checks start at the next commit` and reports no finding for those groups (exit 0). When real history exists and the check fails (unresolvable baseline, bad merge subject), it stays a WARN and fatal under `openspec/.strict.yaml`.
`openspec/.strict.yaml` may carry `history_baseline: <commit>`: the merge check then only considers merges after that commit (it takes precedence over `openspec/.adopted`; `-since` takes precedence over both). A stale `.adopted` that names a commit missing from a fresh history is fatal as soon as the repo has a second commit, so the cut removes `.adopted` or sets `history_baseline`. Tests: `cmd/changecheck/history_test.go` (temp repos: one commit, merge history, baseline, shallow clone). Tags: V (go test run).
CI for the public repo is `.github/workflows/ci.yml` (GitHub-hosted runner, `GITHUB_TOKEN` only); `leakcheck` runs there with `-mode fail`.

## Branch-agnostic checks (change `golivepre2`)
GitHub `main` is the primary working repo (since the 2026-10-31 cut); the pre-cut private repo integrated on `master`; every check works on both. `changecheck -diff agent/X` compares against `defaultBranch()`: `origin/HEAD` when set, else the first of `main`, `master` that exists (local or `origin/`), else `main`. `tools/hooks/pre-push` runs for a push to `refs/heads/main` or `refs/heads/master`. `.github/workflows/ci.yml` triggers on push to `main`, tags `v*`, pull requests and manual runs, never on `gh-pages` (the status publisher's branch). Tests: `go test -p 1 ./cmd/changecheck` (`TestDefaultBranch*`) and `sh tools/hooks/test-pre-push.sh` (temp repo, local bare remote, fake `go`). Tags: V (run).
