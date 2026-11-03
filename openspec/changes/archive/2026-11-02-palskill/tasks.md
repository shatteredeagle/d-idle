# Tasks

```yaml
change: palskill
status: archived
agent: palskill
branch: agent/palskill
worktree: ~/code/d2lod-wt/palskill
skip_specs: true
owns:
  - docs/research/palskill-trace.md
  - tools/emu/pal2*
  - openspec/changes/palskill/**
exclusive: []
gates:
  - game.exe side run on the worker host (seeds 7 and 1) before the trace states a number
result: "docs only; first or second pass, not yet independently verified"
merge: 900ac2e
credit:
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifypalskill2, independent run)"]
  consent: none
```

- [x] 1 Read the five dispatch functions in the disassembly and the upstream notes
- [x] 2 Harness pal2 (game.exe calc machine, FUN_0057b420, FUN_0057d9b0; d2d real rules), seeds 7 and 1
- [x] 3 Candidate fix F measured before / after (copy of combat.hpp in scratch, upstream untouched)
- [x] 4 Write docs/research/palskill-trace.md; ledger line and coverage regenerated
- [x] 5 changecheck (0 warnings), leakcheck (0 hits), tools/ci/check.sh, openspec validate, commit
