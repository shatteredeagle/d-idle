# CI checks for the forge job (for the CI agent; no runner exists yet)

Run from the repo root on a checkout with full history (`fetch-depth: 0`, the merge check reads `git log --merges`). All offline except the first `npx` fetch.

```sh
go run ./cmd/changecheck -cli          # exits 1 once the dates in openspec/.strict.yaml have passed (cards 2026-11-03, merges 2026-11-06)
go run ./cmd/changecheck -strict -cli  # stricter: any warning fails, regardless of dates
sh tools/openspec.sh validate --all --strict   # needs node >= 20.19; changecheck -cli already runs it when node exists
go test -count=1 ./cmd/changecheck     # the checker's own tests
```

For a pull request or branch check add `-diff agent/<id>` (files changed outside the card's owns). `go run ./cmd/changecheck json` prints the same data as JSON for a report. Do not run whole-repo tests on the head box (4 GB); on the worker host use `labrun worker -- go test -short -count=1 ./...`. Tags: S (derived from the code, no CI run yet).

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
