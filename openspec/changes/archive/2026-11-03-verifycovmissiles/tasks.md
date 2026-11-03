# Tasks

```yaml
change: verifycovmissiles
status: archived
agent: verifycovmissiles
branch: agent/verifycovmissiles
worktree: ~/code/d2lod-wt/verifycovmissiles
skip_specs: true
owns:
  - docs/research/verifycovmissiles-trace.md
  - tools/emu/vcm_*
  - openspec/changes/verifycovmissiles/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; coverage.py --check; tools/ci/check.sh; openspec validate --all --strict"
result: "Docs and emulator scripts only. V run on the worker host: 48,624 flight cases (0 differ, real d2d rules and loader), 302 loaded rows x 14 columns (0 differ), 77,560 hit-roll cases of 8 skills (d2d before 21,650 differ, after 0), pierce creation 1,616 cases (new deviation, game side run, d2d side read). Verdicts: first pass verified with limits. Not verified live: nothing was run in a game."
live: "docs only, nothing run live"
merge: c75a7be
```

- [x] read the first pass and the model verification (verify-cursedef.md)
- [x] own disassembly reading (FUN_005adf10, the hit and damage tables, pierce creation)
- [x] own flight run, loader dump, hit-roll run, pierce run
- [x] compare with the author's numbers, verdict per claim
- [x] trace doc, merge lines, checks
