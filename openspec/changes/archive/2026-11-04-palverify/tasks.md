# Tasks

```yaml
change: palverify
status: archived
agent: palverify
branch: agent/palverify
worktree: ~/code/d2lod-wt/palverify
skip_specs: true
owns:
  - openspec/changes/palverify/**
  - docs/research/palverify/**
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
  - "own harness on BASE and HEAD built from clean; verdict with V/S/U tags"
result: "Opus verifier: CONFIRMED WITH CAVEATS; include fix and body edits needed before the PR"
merge: e079944
```

## 1. Work

- [x] 1.1 Own harness (game.exe oracle and d2d probe) in docs/research/palverify/
- [x] 1.2 Base and head built from clean on the worker host, counts recorded
- [x] 1.3 Break attempts (edges, zero, max)
- [x] 1.4 Author's evidence read last and compared
- [x] 1.5 Side checks: authorship and trailers, clang-tidy include, one PR or two, PR body tags
- [x] 1.6 Verdict doc
