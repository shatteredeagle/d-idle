# Tasks

```yaml
change: casttrace
status: archived
agent: casttrace
branch: agent/casttrace
worktree: ~/code/d2lod-wt/casttrace
owns:
  - docs/research/cast-trace-gameexe.md
exclusive:
  - upstream-pr:cast
gates:
  - "game.exe 1.14d addresses re-checked with the upstream pr_guard rules"
  - "nothing is posted, pushed or commented on GitHub"
merge: ca1916e
result: "cast trace in game.exe, 3 PR branches drafted, nothing posted (docs only)"
```

- [x] Read upstream AGENTS.md, CONTRIBUTING.md, PR template, pr_guard, presubmit at 507e3c06
- [x] Trace the cast rate, the end test and the sequence path in game.exe
- [x] Run the unicorn oracle (cast_rates.py) against d2d's functions
- [x] Split into fix/cast-ticks, fix/lightning-sequence, fix/skills-doc-cast
- [x] PR_DESCRIPTION.md and the gh pr create commands per branch (not run)
- [x] Inventory of open upstream PRs that touch the same files
