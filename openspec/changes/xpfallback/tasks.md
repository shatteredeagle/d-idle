# Tasks

```yaml
change: xpfallback
status: in-progress
agent: xpfallback
branch: agent/xpfallback
worktree: ~/code/d2lod-wt/xpfallback
skip_specs: true
owns:
  - openspec/changes/xpfallback/**
exclusive: []
gates:
  - "go test -p 1 -short ./internal/bot ./internal/plandata; changecheck 0 warnings"
result: "Offline, not verified live. XP fallback skips kit-gated steps (17, 18, 19) without a full kit, unknown monster level ranks last instead of the EnterLvl-3 proxy, ties are the later plan step. Seed 17 cause unconfirmed (no bot-0.log)."
```

## 1. Work

- [x] 1.1 R5 FallbackKit flag in plan data and Pick
- [x] 1.2 R6 unknown level ranks last, fallback table 7, 18, 19
- [x] 1.3 R7 tie rule and tests (two classes, Normal and Nightmare)
