# Tasks

```yaml
change: merclive
status: archived
agent: merclive
branch: agent/merclive
worktree: ~/code/d2lod-wt/merclive
skip_specs: true
owns:
  - docs/probes/merclive/**
  - docs/merclive.md
  - openspec/changes/merclive/**
exclusive: []
gates:
  - "changecheck 0 warnings; leakcheck 0 hits"
result: "d2d live: claim PASS, revive flow PASS, paid hire BLOCKED; not game.exe"
merge: 0dcf201
```

- [x] Capacity check OK
- [x] D8.c free Rogue claim live
- [x] D8.a revive with 3000 gold live
- [x] Paid hire: design only, BLOCKED
- [x] Evidence and docs/merclive.md
- [x] changecheck, leakcheck
