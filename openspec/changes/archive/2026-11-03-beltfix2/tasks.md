# Tasks

```yaml
change: beltfix2
status: archived
agent: beltfix2
branch: agent/beltfix2
worktree: d2lod-wt/beltfix2
skip_specs: true
owns:
  - docs/research/beltfix2-analysis.md
  - openspec/changes/beltfix2/**
exclusive: []
gates:
  - "go run ./cmd/changecheck -cli has 0 warnings; go run ./cmd/leakcheck -mode fail finds 0"
result: "docs/tools only"
credit:
  component: BELTFIX2
  author: owner
  traced_by: []
  reported_by: []
  verified_by: []
  agents: [Claude Code]
  consent: none
merge: 0121d34
```

- [x] 1. Find every BELT window over 5 and the NOKILL windows (replay of the d2dsoak rule over the minute lines)
- [x] 2. Classify each window with evidence from bot-0.log
- [x] 3. Check that the BELTFIX code paths ran and behaved as designed
- [x] 4. Explain why 37 and not 12; d2d backend part
- [x] 5. Ranked fix plan with files, functions, tests and criteria
- [x] 6. Opinion on the belt rule
