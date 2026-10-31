# Tasks

```yaml
change: pubfix
status: archived
merge: 7fe28b6
agent: pubfix
branch: agent/pubfix
worktree: ~/code/d2lod-wt/pubfix
owns:
  - tools/publish/**
  - openspec/changes/pubfix/**
  - openspec/changes/stream-grid/specs/stream-scene/spec.md
exclusive: []
gates:
  - "python3 -m unittest discover -s tools/publish/tests -t tools/publish passes"
  - "tools/openspec.sh validate --all --strict passes"
result: 'offline only: 65 publisher tests pass; pages generated to a temp dir (not pushed); nothing live'
```

- [x] sanitizer rules and tests
- [x] runs cache honesty and hardening
- [x] live.py, publish.py fail-closed
- [x] leak-gate test over generated pages
