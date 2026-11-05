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

CI for the public repo is `.github/workflows/ci.yml` (GitHub-hosted runner, `GITHUB_TOKEN` only); `leakcheck` runs there with `-mode fail`.
