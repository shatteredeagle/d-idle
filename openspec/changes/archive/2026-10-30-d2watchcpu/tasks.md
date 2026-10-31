# Tasks

```yaml
change: d2watchcpu
status: archived
agent: d2watchcpu
branch: agent/d2watchcpu
worktree: ~/code/d2lod-wt/d2watchcpu
owns:
  - cmd/d2watch/**
  - docs/watch.md
  - internal/viewport/stream.go
exclusive: []
gates:
  - "live d2watch is redeployed by the main agent only"
merge: ade9eb1
result: "d2watch hot loop in serveStream when an instance disappears (closed stream returned instantly, cached frame, continue): fixed; idle CPU ~70% of a core -> ~0.7% (scratch measurement); opt-in -pprof flag"
```

- [x] reproduce the hot loop in a test and fix it
- [x] document evidence, cost and redeploy steps
