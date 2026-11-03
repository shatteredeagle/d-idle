# Tasks

```yaml
change: crsetup
status: archived
agent: crsetup
branch: agent/crsetup
worktree: d2lod-wt/crsetup
skip_specs: true
owns:
  - .coderabbit.yaml
  - docs/process/coderabbit-prescan.md
  - openspec/changes/crsetup/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh"
result: "docs and config; quota test in progress"
live: "docs only, nothing run live"
merge: 1063082
```

- [x] .coderabbit.yaml for d-idle, schema-validated
- [x] fork-coderabbit.yaml in the card dir
- [x] runbook with install, fork config, quota test, workflow, CLI
