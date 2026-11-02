# Tasks

```yaml
change: crfixbundle2
status: in-progress
agent: crfixbundle2
branch: agent/crfixbundle2
worktree: d2lod-wt/crfixbundle2
skip_specs: true
owns:
  - docs/research/crfix-bundle2-20261102.md
  - docs/research/crfix-bundle2-20261102/**
  - openspec/changes/crfixbundle2/**
exclusive:
  - fork-branch:local/bundle-4
  - fork-branch:stage/bundle-4
gates:
  - "both comments judged with evidence: fixed and unfixed script paths run on constructed inputs and a real 400-trial run"
  - "bundle 4 commits: author and committer the numeric noreply identity, at most 3 body lines, last line Agent: Claude Code"
result: "docs and harness; the two comments are valid and fixed on bundle 4; verified on the worker host (emulator runs, 8 suites, pr_guard, presubmit --quick); no live game; nothing upstream or posted"
credit:
  component: Review fixes for bundle 3 (bundle 4)
  author: owner
  agents: [Claude Code]
  verified_by: []
  consent: none
```

- [x] 1 comments read and judged
- [x] 2 both proven on constructed inputs and a real 400-trial run
- [x] 3 local/bundle-4 and stage/bundle-4 built from bundle 3, suites and guards re-run, pushed to the private remote
- [x] 4 reply texts
- [x] 5 changecheck, leakcheck, ci check
