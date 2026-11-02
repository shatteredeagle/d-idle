# Tasks

```yaml
change: nmlevel
status: archived
agent: nmlevel
branch: agent/nmlevel
worktree: ~/code/d2lod-wt/nmlevel
skip_specs: true
owns:
  - docs/research/pr-nmlevel-draft.md
  - tools/emu/nm_pop.py
  - openspec/changes/nmlevel/**
exclusive:
  - fork-branch:local/nmlevel
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "docs and tools only; merged by the triage queue"
live: "not verified live; offline emulator differential only"
merge: af24e7c
```

- [x] locate where a spawned monster's level is chosen (rules::monster_stats) and the unit makers
- [x] differential: boss_level.py gen-area and the real-population comparison, before and after
- [x] fix, unit test, suites, pr_guard, presubmit --quick
- [x] det scenarios base against head
- [x] PR draft
