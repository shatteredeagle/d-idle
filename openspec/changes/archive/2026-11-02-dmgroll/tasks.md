# Tasks

```yaml
change: dmgroll
status: archived
agent: dmgroll
branch: agent/dmgroll
worktree: d2lod-wt/dmgroll
skip_specs: true
owns:
  - openspec/changes/dmgroll/**
  - docs/research/dmgroll-draft.md
  - tools/emu/mdmg*
exclusive:
  - fork-branch:local/batch2-monsterdmg
gates:
  - "base mismatches > 0 and head 0 on >= 20,000 cases per class (8 Act 1 classes), real monster_blow against game.exe's FUN_0057b420"
  - "changecheck 0 warnings; leakcheck -mode fail 0"
result: "docs only; fork branches on the private remote"
credit:
  component: Monster physical damage roll
  author: owner
  traced_by: []
  agents: [Claude Code]
  verified_by: []
  consent: none
merge: 8e99f3c
```

## 1. Work

- [x] 1.1 Step 1: upstream main still open? (rolls whole points inclusive, no ponytail note, #177 covers only the player side, #178 does not touch it)
- [x] 1.2 Fix on local/batch2-monsterdmg, one commit
- [x] 1.3 Differential tools/emu/mdmg*: 8 classes x 20,000, base vs head
- [x] 1.4 Suites, CI parity, pr_guard, presubmit
- [x] 1.5 docs/research/dmgroll-draft.md, merge-lines.txt, OWNERS line
