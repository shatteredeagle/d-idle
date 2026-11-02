# Tasks

```yaml
change: creditkit
status: archived
agent: creditkit
branch: agent/creditkit
worktree: d2lod-wt/creditkit
skip_specs: true
owns:
  - tools/credit/**
  - docs/process/credit.md
  - docs/community/credits-ledger.md
  - openspec/changes/creditkit/**
exclusive: []
gates:
  - "python3 -m unittest discover -s tools/credit -t tools/credit -p test_credit.py passes"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; tools/ci/check.sh ok"
result: "docs and tools; 12 unit tests in CI"
merge: e983945
credit:
  author: owner
  agents: [Claude Code]
  consent: none
```

- [x] credit block schema documented in openspec/README.md
- [x] tools/credit/credit.py and test_credit.py, wired into tools/ci/scripts.sh
- [x] ledger, seeded cards, licence sentence in chunker.md and agent-work.md
- [x] checks (changecheck, leakcheck, check.sh)
