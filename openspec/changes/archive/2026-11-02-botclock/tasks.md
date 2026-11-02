# Tasks

```yaml
change: botclock
status: archived
agent: botclock
branch: agent/botclock
worktree: d2lod-wt/botclock
skip_specs: true
owns:
  - internal/gclock/**
  - docs/timescale.md
  - openspec/changes/botclock/**
exclusive: []
gates:
  - "gofmt, go vet and go test of the touched packages; changecheck 0 warnings; leakcheck 0"
result: "go tests per package pass; not verified live"
merge: dbbc0f7
```

- [x] inventory and classification (docs/timescale.md)
- [x] internal/gclock with tests and the guard test
- [x] move game-clock call sites
- [x] docs/timescale.md
- [ ] live k=2 and k=4 re-measure (not part of this card)
