# Tasks

```yaml
change: monsterpace
status: archived
agent: monsterpace
branch: agent/monsterpace
worktree: ~/code/d2lod-wt/monsterpace
skip_specs: true
owns:
  - docs/research/pr-monsterpace-a-draft.md
  - docs/research/pr-monsterpace-b-draft.md
  - tools/emu/mp_run.sh
  - tools/emu/mp_patch.py
  - openspec/changes/monsterpace/**
exclusive:
  - "fork-branch:local/monsterpace"
  - "fork-branch:local/monsterpace-clean"
  - "fork-branch:local/monsterpace-a"
  - "fork-branch:local/monsterpace-b-velocity"
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "docs and tools only; merged by the triage queue"
live: "no game; the lab d2d binary runs the det scenarios offline"
merge: 7fa2b9d
```

- [x] read upstream think / pace code and PR 156
- [x] real-build probes and fixes on local/monsterpace (commits counted in the draft)
- [x] PR draft
- [x] independent verification (Opus verifier: READY WITH CHANGES; changes applied: rebase, split, text corrections, CI parity, full tidy selection)
- [x] split into A and B, rebased onto 3be426d3, re-counted
- [ ] PR-open step (needs the user's go)
