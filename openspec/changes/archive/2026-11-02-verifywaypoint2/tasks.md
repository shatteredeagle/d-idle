# Tasks

```yaml
change: verifywaypoint2
status: archived
agent: verifywaypoint2
branch: agent/verifywaypoint2
worktree: ~/code/d2lod-wt/verifywaypoint2
skip_specs: true
owns:
  - docs/research/verifywaypoint2-trace.md
  - tools/emu/vw2_xref.py
  - tools/emu/vw2_tab.py
  - tools/emu/vw2_tab2.py
  - tools/emu/vw2_disp.py
  - tools/emu/vw2_hdis.py
  - tools/emu/vw2_op.py
  - tools/emu/vw2_game.py
  - tools/emu/vw2_d2d.sh
  - tools/emu/vw2_trace.py
  - openspec/changes/verifywaypoint2/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "docs only; first pass, not yet independently verified"
live: "not verified (emulator and d2d test binary only); drill recipe in the trace"
merge: caad9fd
```

- [x] writers and readers scan (own sweep incl. lea/add, pointer tables, call sites)
- [x] hostile path reachability (dispatcher, packet handlers)
- [x] own case set, full 32-bit clock, game.exe vs d2d binary
- [x] reproduce 59,309 / 132,964 with both programs
- [x] doc, card, merge lines
