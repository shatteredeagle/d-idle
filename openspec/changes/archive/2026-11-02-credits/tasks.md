# Tasks

```yaml
change: credits
status: archived
agent: credits
branch: agent/credits
worktree: d2lod-wt/credits
skip_specs: true
owns:
  - CONTRIBUTORS.md
  - docs/process/chunker.md
  - README.md
  - openspec/changes/credits/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh"
result: "docs only"
live: "docs only, nothing run live"
merge: 2f14b20
```

- [x] CONTRIBUTORS.md
- [x] Credit section, README and INDEX lines
