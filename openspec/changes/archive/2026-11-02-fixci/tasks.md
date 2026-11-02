# Tasks

```yaml
change: fixci
status: archived
agent: fixci
branch: agent/fixci
worktree: d2lod-wt/fixci
skip_specs: true
owns:
  - docs/community/work-archive.md
  - tools/contrib/chunker.py
  - openspec/changes/fixci/**
exclusive: []
gates:
  - "tools/ci/scripts.sh passes; changecheck -cli 0 warnings; leakcheck -mode fail 0"
result: "CI fix: scripts.sh exits 0 locally"
live: "no game"
merge: 54fff8e
```

- [x] fix four research paths in the archive
- [x] chunker flag without free-text excerpt
- [x] run the Scripts job locally
