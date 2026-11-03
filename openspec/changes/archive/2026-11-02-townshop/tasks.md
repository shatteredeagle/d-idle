# Tasks

```yaml
change: townshop
status: archived
agent: townshop
branch: agent/townshop
worktree: d2lod-wt/townshop
skip_specs: true
owns:
  - internal/town/townshop_test.go
  - internal/town/shop.go
  - docs/townshop.md
  - openspec/changes/townshop/**
exclusive: []
gates:
  - "go test -p 1 ./internal/town ./internal/bot on the worker host; changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "go tests town bot pass on the worker host; real cause (field tome stat 70 drop) still open"
live: "not verified; checklist in docs/townshop.md"
credit:
  author: owner
  agents: [Claude Code]
merge: a788442
```

- [x] root cause from the Gate 2 log
- [x] budget in the buy loops, log line
- [x] proof top-up after a portal-empty report
- [x] tests and docs
