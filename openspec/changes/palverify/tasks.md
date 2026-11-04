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
result: "Offline, nothing live. CONFIRMED WITH CAVEATS (docs/research/palverify/verdict.md): head 9e9622eb 0 differ on 139,757 in-domain cases (seeds 20261104, 777, 4242; all 734 MonStats rows; both-class synthetic 0), base e5de6c10 99,698 differ, plain-without-Concentration 0 on base (unchanged class equal). Caveats: game.exe draws the magic roll twice on a hit (FUN_005a89a0 then FUN_005a8c70), the oracle matches the second; tests/game/test_hammer_hit.cpp:13 fails misc-include-cleaner (delete the line); bonus code magic-only in effect; PR body internal references. Sorceress differential 64,000 cases 0 on both; upstream suites rc 0 on both."
```

## 1. Work

- [x] 1.1 Own harness (game.exe oracle and d2d probe) in docs/research/palverify/
- [x] 1.2 Base and head built from clean on the worker host, counts recorded
- [x] 1.3 Break attempts (edges, zero, max)
- [x] 1.4 Author's evidence read last and compared
- [x] 1.5 Side checks: authorship and trailers, clang-tidy include, one PR or two, PR body tags
- [x] 1.6 Verdict doc
