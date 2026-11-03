## Why

changecheck failed the `checks` job on community PRs because of GitHub's synthetic `Merge <sha> into <sha>` commit.

## What Changes

- cmd/changecheck skips commits whose subject matches `Merge <40hex> into <40hex>`; table test added.

## Impact

Tooling only, public repo.
