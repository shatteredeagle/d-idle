# Tasks

```yaml
change: verifybundle
status: archived
agent: verifybundle
branch: agent/verifybundle
worktree: d2lod-wt/verifybundle
skip_specs: true
owns:
  - docs/research/pr-bundle-1-verifier.md
  - docs/research/pr-bundle-1-verifier/**
  - openspec/changes/verifybundle/**
exclusive: []
gates:
  - own harness run before the author's ground draft is compared
result: "docs only; changes applied in bundlefinal"
merge: b174a29
credit:
  component: Verifier: bundle 1
  author: owner
  agents: [Claude Code]
  consent: none
```

- [x] 1 Ground: own game.exe oracle and own d2d probe, base vs head counts
- [x] 2 Integration on the bundle head (shared files, suites, readers)
- [x] 3 CI parity (g++/clang++ warnings, MSVC by reading, include-cleaner, pr_guard, presubmit)
- [x] 4 Commit hygiene and credit
- [x] 5 PR body claims vs counts; size vs budget; merge with current upstream main
- [x] 6 Verdict; changecheck and leakcheck
