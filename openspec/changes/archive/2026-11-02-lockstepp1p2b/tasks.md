# Tasks

```yaml
change: lockstepp1p2b
status: archived
agent: lockstepp1p2b
branch: agent/lockstepp1p2b
worktree: d2lod-wt/lockstepp1p2b
skip_specs: true
owns:
  - docs/research/lockstep-p1p2.md
  - cmd/lockstepscan/**
  - openspec/changes/lockstepp1p2b/**
exclusive:
  - instance:i4
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "docs only; k=4, same-day unhooked control and periodic hook status line still open"
live: "i4 Booti: read-only scans; hook run marker lockstep-p2-k1 written"
merge: 633ef30
```

- [x] P1 live read, 25 fps model
- [x] P2 k=1 control
- [x] P2 k=2
