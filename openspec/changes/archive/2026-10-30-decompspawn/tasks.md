# Tasks

```yaml
change: decompspawn
status: archived
agent: decompspawn
branch: agent/decompspawn
worktree: ~/code/d2lod-wt/decompspawn
owns:
  - docs/research/spawn-density-trace.md
  - docs/research/upstream-pr-candidates-spawn.md
  - tools/emu/spawn_*.py
  - openspec/changes/decompspawn/**
exclusive: []
gates:
  - "every rule in the trace doc is tagged V (game.exe read or run under unicorn), D, S or U with an address"
merge: f081ec1
result: "monster placement rules traced (FUN_005be020 etc.); d2d placement equals game.exe; density gap = probe coverage (sweep 58-71 % vs original 80-92 %); champion/unique differences are room-activation artifacts; no PR. Merged f081ec1"
```

## 1. Work

- [x] 1.1 Read item3 (UPSTREAMGAPS) and the d2d sweep method; do not re-derive placement
- [x] 1.2 Room activation rule in game.exe (bounds the original census; S from upstream docs, d2d code read)
- [x] 1.3 Unique/champion/rare roll and counter traced in game.exe (addresses, V)
- [x] 1.4 Superunique rules (Bishibosh/Rakanishu/Treehead/Corpsefire; D tables, V counting)
- [x] 1.5 Sweep coverage: why d2d lab reaches 58-71 %, the comparable totals
- [x] 1.6 Analytic table vs measured
- [x] 1.7 spawn-density-trace.md, PR candidate doc, changecheck 0 warnings
