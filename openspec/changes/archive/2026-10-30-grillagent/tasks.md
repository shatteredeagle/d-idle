# Tasks

```yaml
change: grillagent
status: archived
agent: grillagent
branch: agent/grillagent
worktree: ~/code/d2lod-wt/grillagent
owns:
  - .claude/agents/grill.md
  - tools/grill.sh
  - tools/grill-guard.sh
  - docs/process/grill-agent.md
  - docs/process/grill-workflow.md
  - GLOSSARY.md
exclusive: []
gates:
  - "first real session confirms the hook payload fields and that --agent grill starts (U until then)"
merge: 004800f
result: "grill agent + tools/grill.sh + path guard hook; `claude --agent grill`; end-to-end start untested: accept workspace trust on first start"
```

## 1. Work

- [x] 1.1 Verify the mechanism from the docs (grill-agent.md)
- [x] 1.2 Agent, script and guard hook; verified with `sh -n` and sample payloads
- [x] 1.3 Docs, glossary, OWNERS
- [ ] 1.4 First live grill by the user confirms the U items
