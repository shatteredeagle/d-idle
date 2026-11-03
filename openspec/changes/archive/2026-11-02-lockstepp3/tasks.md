# Tasks

```yaml
change: lockstepp3
status: archived
agent: lockstepp3
branch: agent/lockstepp3
worktree: d2lod-wt/lockstepp3
skip_specs: true
owns:
  - docs/research/lockstep-p3.md
  - openspec/changes/lockstepp3/**
exclusive:
  - instance:i4
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "docs only; single runs, no noise band"
live: "i4 Booti"
merge: e5d5a4e
```

- [x] hook periodic STAT line (private repo, local commit)
- [x] unhooked control, 10 min town and 10 min Blood Moor
- [x] k=1 hooked
- [x] k=2 hooked
- [x] k=4 hooked, plus 5 min town
- [x] doc
