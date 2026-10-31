# Tasks

```yaml
change: soakfull
status: archived
merge: archived-by-dispatch
agent: soakfull
branch: agent/soakfull
worktree: ~/code/d2lod-wt/soakfull
owns:
  - docs/d2d-soak-full-fix.md
  - docs/probes/soakfull/**
  - tools/soakfull/*
exclusive: []
gates:
  - replicated full-fix soak measured
result: "full-fix replicated soak: 12 valid runs x 2 virtual h (24 h pooled) on lab6 with run policy: XP/h 4985 +- 145 (-24.5 % vs early 6600; INCONCLUSIVE), kills/h 195, deaths 0 PASS; doc docs/d2d-soak-full-fix.md; livelockfix alone failed live (hotfix = RUNAB's); seeds 11/42 deterministic nav stall; integration branch archived as tag archive/soakfull"
```

- [x] 12 valid runs x 2 virtual hours
- [x] doc and probes
