# Tasks

```yaml
change: gatedelta
status: archived
agent: gatedelta
branch: agent/gatedelta
worktree: ~/code/d2lod-wt/gatedelta
owns:
  - openspec/changes/gatedelta/**
  - internal/d2dsoakstat/**
  - cmd/d2dsoak/**
  - docs/d2d-soak.md
exclusive: []
gates:
  - "go test ./internal/d2dsoakstat ./cmd/d2dsoak pass, v1 verdicts unchanged"
  - "changecheck 0 warnings, openspec validate --all --strict"
merge:
result: "offline only, not verified live; life proxies need minute-line fields the bot does not emit yet"
merge: 27d2fa1
result: "v2 gate: deaths at most 1 per 6 h pooled Poisson, lower 95% bound <= 0.15/h, deaths per 1000 kills Poisson, 3 proxies at 25%, d2dsoak compare -gate v2; v1 default"
```

## 1. Work

- [x] 1.1 delta spec (deaths bound, proxies, per-1000-kills bound form)
- [x] 1.2 d2dsoakstat gate v2 and proxy metrics, tests with fixtures
- [x] 1.3 `d2dsoak compare -gate v2`
- [x] 1.4 docs/d2d-soak.md section 3b
