## Why

User decision 2026-10-31: the repo goes public after a one-time fresh-history cut and GitHub becomes the primary working repo. The dry run of that cut found that changecheck -cli fails on a one-commit repository (git log exits 128, fatal under openspec/.strict.yaml), that only a forge-specific workflow exists, that the README claims a history the cut removes, and that .claude/settings.json is private overlay material.

## What Changes

- changecheck reports an INFO line and exits 0 for the merge/diff groups when there is no history before the baseline (shallow clone, single root commit, no baseline); real history stays fatal; openspec/.strict.yaml may name history_baseline
- .github/workflows/ci.yml for the public repo (checks, test, release on tags, GITHUB_TOKEN only)
- README history, Go version and contributing text corrected
- .claude/settings.json moves to the private ops overlay; tools/ops.sh link links it
