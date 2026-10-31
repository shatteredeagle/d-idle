# Tasks

```yaml
change: slotscale
status: archived
agent: slotscale
branch: agent/slotscale
worktree: ~/code/d2lod-wt/slotscale
owns:
  - tools/d2dslot.sh
  - tools/d2dslot-test.sh
  - internal/d2dreplay/launch_pool.go
  - internal/d2dreplay/launch_pool_test.go
  - docs/process/worker-compute.md
  - openspec/changes/slotscale/
exclusive: []
gates:
  - "scaling table measured on the worker host up to the first tripped limit; default pool = knee - 1, never above 12"
  - "port map has no collision with the occupied-ports list (tools/d2dslot-test.sh ports)"
merge: 6465e3e
result: 'slot pool measured: knee 8 concurrent d2d (speed collapses at 10); default pool 7; admission control for slots >= 4; port map bot 9100+n view 9200+n; load-per-core rule added'
```

## 1. Work

- [x] 1.1 d2dslot.sh: D2D_SLOTS, admission control (slots >= 4), port block, pool.json
- [x] 1.2 d2dslot-test.sh ports check, d2dreplay pool reader with tests
- [x] 1.3 measure the knee, set the default, document
