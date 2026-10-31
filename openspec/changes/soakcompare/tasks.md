# Tasks

```yaml
change: soakcompare
status: in-progress
agent: soakcompare
branch: agent/soakcompare
worktree: ~/code/d2lod-wt/soakcompare
owns:
  - docs/probes/navsoak-compare.md
  - openspec/changes/soakcompare/**
exclusive: []
gates:
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0; tools/openspec.sh validate --all --strict passes"
merge:
result:
```

## 1. Work

- [x] 1.1 Analyse navsoak and navsoak2 logs read-only on the worker host
- [x] 1.2 Write docs/probes/navsoak-compare.md (<= 90 lines, VERIFIED/INFERRED tags)
- [x] 1.3 OWNERS line, changecheck status -write, gates
