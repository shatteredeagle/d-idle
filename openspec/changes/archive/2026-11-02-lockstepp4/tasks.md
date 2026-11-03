# Tasks

```yaml
change: lockstepp4
status: archived
agent: lockstepp4
branch: agent/lockstepp4
worktree: d2lod-wt/lockstepp4
skip_specs: true
owns:
  - docs/research/lockstep-p4.md
  - openspec/changes/lockstepp4/**
exclusive:
  - instance:i4
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "docs only; single runs, no noise band"
live: "yes: i4 Booti, hook k=2 and k=4, bot built from agent/botclock plus a local --timescale arg patch (not committed to any branch of this card)"
merge: 6af67a2
```

- [x] k=2 run with scaled bot clock
- [x] k=4 run with scaled bot clock
- [x] doc docs/research/lockstep-p4.md, merge lines
