# Tasks

```yaml
change: decompfreeze
status: archived
agent: decompfreeze
branch: agent/decompfreeze
worktree: ~/code/d2lod-wt/decompfreeze
skip_specs: true
owns:
  - docs/research/freeze-rules-trace.md
  - docs/research/pr-freeze-proof.md
  - docs/research/pr-freeze-draft.md
  - tools/emu/freeze_rules.py
  - tools/emu/freeze_rules_test.cpp
  - docs/decomp-queue.md
  - docs/research/pr-bundle-queue.md
  - docs/BACKLOG.md
  - openspec/changes/decompfreeze/**
exclusive:
  - fork-branch:fix/freeze-rules
gates:
  - "differential on real d2d builds: before > 0, after 0 per claim; pr_guard 0/0; presubmit --quick 0 FAIL; upstream tests rc 0"
  - "changecheck -cli 0 warnings; leakcheck -mode fail 0; openspec validate --all --strict"
merge: 2adbd6b
result: "Local upstream candidate fix/freeze-rules on #155 head ddc8281d (C5 9ec91600, C3 39ef104e, C4 61620d08, tooling bbf506c6): real d2d builds 225,748 cases, 185,746 -> 0 (per claim C5 48,946, C3 78,501, C4 58,299 -> 0); C3 wording corrected (champions carry flag 8); CI-class checks clean. Offline, nothing verified live, nothing pushed; verifier and det hashes open."
credit:
  component: Freeze rules (trace)
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifyfreeze, independent run)"]
  consent: none
```

## 1. Work

- [x] 1.1 Re-derive the freeze rules from the disassembly; decide C3 / C4 / C5
- [x] 1.2 Read d2d's freeze paths and list the deviations
- [x] 1.3 Differential oracle (game.exe natively vs real d2d builds), before > 0, after 0 per claim
- [x] 1.4 Claim commits and tooling commit on fix/freeze-rules (numeric-ID noreply identity)
- [x] 1.5 CI-class checks (clang-tidy names / includes, gcc sweep, ruff, pr_guard per commit, presubmit, four test binaries, file length, PR size)
- [x] 1.6 Docs: trace, proof, draft, queue rows, BACKLOG line

## 2. Open

- [ ] 2.1 Independent verifier (a second agent)
- [ ] 2.2 Det state-hash run (needs a d2d slot)
