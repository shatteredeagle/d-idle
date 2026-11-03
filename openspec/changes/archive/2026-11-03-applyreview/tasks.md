# Tasks

```yaml
change: applyreview
status: archived
agent: applyreview
branch: agent/applyreview
worktree: d2lod-wt/applyreview
skip_specs: true
owns:
  - docs/adr/0015-goals-acceptance-on-game-exe.md
  - docs/plan/plan-20261103.md
  - docs/research/card-triage-20261103.md
  - openspec/changes/applyreview/**
exclusive: []
gates:
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh; openspec validate --all --strict"
result: "docs only"
live: "not verified"
credit:
  component: APPLYREVIEW
  author: owner
  agents: [Claude Code]
merge: ca2489d
```

- [x] decisions.md lines and ADR 0015
- [x] PLAN.md status block, roadmap mapping, GLOSSARY terms
- [x] STATUS.md hand part (60 lines or fewer)
- [x] plan-20261103.md and card-triage-20261103.md
- [x] archive merged and finished cards
- [x] checks
