# Tasks

```yaml
change: contribid
status: archived
agent: contribid
branch: agent/contribid
worktree: d2lod-wt/contribid
skip_specs: true
owns:
  - AGENTS.md
  - docs/contributing/agent-work.md
  - .github/pull_request_template.md
  - openspec/changes/contribid/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh"
result: "docs and template only"
live: "docs only, nothing run live"
merge: 41d992c
```

- [x] AGENTS.md identity rule split
- [x] reproducibility rule and checkboxes
