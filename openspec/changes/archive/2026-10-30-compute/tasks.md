# Tasks

```yaml
change: compute
status: archived
agent: compute
branch: agent/compute
worktree: ~/code/d2lod-wt/compute
owns:
  - docs/process/worker-compute.md
  - tools/d2dslot.sh
  - tools/d2dslot-test.sh
  - tools/capacity.sh
exclusive: []
gates:
  - "3 concurrent hidden d2d runs on slots 1-3 plus a legacy-lock run on slot 0: independent sockets, no cross-talk"
merge: aba4625
result: "d2d lab lock is a pool of 4 slots (tools/d2dslot.sh, slot 0 = legacy lock); capacity.sh counts d2d.lab5/d2bot.*; Tdarr node paused via its API (nodePaused) with resume command in docs/process/worker-compute.md; Tdarr not confirmed as I/O source. Merged aba4625"
```

## 1. Work

- [x] 1.1 Find and pause Tdarr on the worker host, measure iowait and PSI before and after
- [x] 1.2 tools/d2dslot.sh slot pool and tools/d2dslot-test.sh
- [x] 1.3 capacity.sh counts lab binaries (d2d.lab5, d2bot.*)
- [x] 1.4 docs/process/worker-compute.md and delta spec
