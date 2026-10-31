# Tasks

```yaml
change: publicfix
status: archived
agent: publicfix
branch: agent/publicfix
worktree: ~/code/d2lod-wt/publicfix
owns:
  - tools/publish/runs.py
  - tools/publish/tests/**
exclusive: []
gates:
  - "python3 -m unittest discover -s tools/publish/tests -t tools/publish passes"
merge: c9fa2c6
result: 'runs.py keeps newest log per source+character; caches last good fetch (900 s) in a private cache outside public/; 54 tests'
```

## 1. Work

- [x] 1.1 dedupe per source+character
- [x] 1.2 fetch cache with 15 min expiry and honest note
- [x] 1.3 tests (duplicates, failed fetch with cache, expiry, between runs)
