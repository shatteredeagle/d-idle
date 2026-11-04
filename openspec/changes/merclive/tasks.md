# Tasks

```yaml
change: merclive
status: in-progress
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
result: "d2d live: D8.c claim PASS (merc RG in dump), D8.a revive PASS but price 0 (not the bot's assumption, U); paid hire BLOCKED (no drill); not game.exe"
```

- [x] Capacity check OK
- [x] D8.c free Rogue claim live
- [x] D8.a revive with 3000 gold live
- [x] Paid hire: design only, BLOCKED
- [x] Evidence and docs/merclive.md
- [x] changecheck, leakcheck
