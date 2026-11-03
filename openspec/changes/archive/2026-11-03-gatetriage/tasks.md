# Tasks

```yaml
change: gatetriage
status: archived
agent: gatetriage
branch: agent/gatetriage
worktree: d2lod-wt/gatetriage
skip_specs: true
owns:
  - tools/baseline/gate_triage.py
  - tools/baseline/tests/test_gate_triage.py
  - docs/research/gate-triage-examples.md
  - openspec/changes/gatetriage/**
exclusive: []
gates: python3 -I tools/baseline/tests/test_gate_triage.py; changecheck 0 new warnings; leakcheck 0 hits
result: "python tests pass; docs and tooling only"
credit:
  author: owner
  agents: [Claude Code]
  consent: none
merge: 5833ec2
```

- [x] 1. gate_triage.py: verdicts per current rules, windows, XP by band and area, log counts and context, cause hints, exit code
- [x] 2. Fixture tests (poverty clvl 2, funded 332-366, stash 1418 with pack 6, Cold Plains farming)
- [x] 3. Outputs for the five Gate 1 runs in docs/research/gate-triage-examples.md
- [x] 4. gate-run.sh one-line wiring documented (ops repo change is a separate task)
