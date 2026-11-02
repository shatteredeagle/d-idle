# Tasks

```yaml
change: skias
status: archived
agent: skias
branch: agent/skias
worktree: ~/code/d2lod-wt/skias
skip_specs: true
owns:
  - docs/research/sk-ias-trace.md
  - tools/emu/skias_bp.py
  - openspec/changes/skias/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs only; see the doc for tags, holds and limits; board item updated separately"
live: "docs only, nothing run live"
merge: 1316229
```

- [x] read d2d attack_speed / attack_ticks, breakpoint script
- [x] trace doc with deviations D1-D4 (U) and bot finding B1
- [ ] game.exe attack branch of 00623f50 under the emulator (open)
