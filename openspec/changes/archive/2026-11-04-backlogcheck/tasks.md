# Tasks

```yaml
change: backlogcheck
status: archived
agent: backlogcheck
branch: agent/backlogcheck
worktree: ~/code/d2lod-wt/backlogcheck
skip_specs: true
owns:
  - docs/research/backlog-stale-20261104.md
  - openspec/changes/backlogcheck/**
exclusive: []
gates:
  - "python3 tools/merge_lines_check.py --changed passes"
  - "go run ./cmd/changecheck -cli: 0 warnings"
  - "go run ./cmd/leakcheck -mode fail: 0 hits"
merge: 5740a15
result: "docs only; no shared file edited; verdicts from git log and archive dirs"
```

- [x] scan docs/BACKLOG.md and docs/decomp-queue.md for status words
- [x] check 40 candidates against git log origin/main, archive and changes folders
- [x] write docs/research/backlog-stale-20261104.md with the table and the merger section
- [x] run the three gates (merge_lines_check, changecheck -cli 0 warnings, leakcheck 0 hits) and commit
