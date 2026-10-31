# Tasks

```yaml
change: capfix
status: archived
merge: 4351294
agent: capfix
branch: agent/capfix
worktree: ~/code/d2lod-wt/capfix
owns:
  - tools/capacity.sh
  - tools/capacity.stub
  - docs/process/worker-capacity.md
exclusive: []
gates:
  - three real runs on the worker host finish within window + 10 s
result: 'capacity.sh: every child under timeout 5 with stdin from /dev/null, whole script one function, progress line, PSI io full avg60 >= 40 warn-only; caller timeout must be window+10 s; hang not reproduced (timing)'
```

- [x] harden script
- [x] PSI signal and doc
