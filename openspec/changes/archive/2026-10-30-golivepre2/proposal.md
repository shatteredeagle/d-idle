## Why

The public repo is the existing GitHub repo `shatteredeagle/d-idle` (default branch `main`, Pages from `main`). The cut moves the status site to `gh-pages` and puts the code on `main`, so the working tree must behave on both `main` (public) and `master` (the private remote), and gh-pages pushes must not trigger CI.

## What Changes

- changecheck `-diff` uses the detected default branch instead of `master`; unit tests
- `tools/hooks/pre-push` acts for `refs/heads/main` and `refs/heads/master`; `tools/hooks/test-pre-push.sh` tests it in a temp repo with a local bare remote
- CI trigger note in `.github/workflows/ci.yml` (it already runs on push to `main`, tags and pull requests only); docs updated
- The publisher switch, the go-live runbook and `golive-pages.sh` live in the private ops repo, not here
