# Tasks

```yaml
change: palverify
status: in-progress
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
result: "pending"
```

## 1. Work

- [ ] 1.1 Own harness (game.exe oracle and d2d probe) in docs/research/palverify/
- [ ] 1.2 Base and head built from clean on the worker host, counts recorded
- [ ] 1.3 Break attempts (edges, zero, max)
- [ ] 1.4 Author's evidence read last and compared
- [ ] 1.5 Side checks: authorship and trailers, clang-tidy include, one PR or two, PR body tags
- [ ] 1.6 Verdict doc
