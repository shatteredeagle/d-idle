# Tasks

```yaml
change: reviewprs
status: archived
agent: reviewprs
branch: agent/reviewprs
worktree: d2lod-wt/reviewprs
skip_specs: true
owns:
  - openspec/changes/reviewprs/**
  - docs/research/review-community-prs-20261102.md
exclusive: []
gates:
  - "changecheck 0 warnings for this card; leakcheck -mode fail 0"
result: "docs and tools only; merged by the triage queue"
merge: 5ffb902
```

## 1. Work

- [x] 1.1 Fetch PR heads, read briefs, issue claims and contribution rules
- [x] 1.2 Spot-check claims (emulator, disassembly, Skills/MonStats data, d2d source at the cited commit)
- [x] 1.3 Compare with our merged traces
- [x] 1.4 Hygiene: leakcheck, gamecode guard, changecheck on each PR head
- [x] 1.5 Verdicts and draft comments in docs/research/review-community-prs-20261102.md
