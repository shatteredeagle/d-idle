# Tasks

```yaml
change: mercwire
status: archived
agent: mercwire
branch: agent/mercwire
worktree: ~/code/d2lod-wt/mercwire
owns:
  - cmd/d2bot/**
  - docs/knobs.md
  - docs/merc.md
  - docs/merc-wire.md
  - internal/d2dmerc/**
  - internal/env/**
exclusive: []
gates:
  - "merged with tests passing, see 840693d"
merge: 840693d
result: "wired on d2d behind a knob, default off (V offline tests); original game still has no merc reader, U"
```

## 1. Work

- [x] 1.1 Implemented and merged
