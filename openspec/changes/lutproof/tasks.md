# Tasks

```yaml
change: lutproof
status: in-progress
agent: lutproof
branch: agent/lutproof
worktree: ~/code/d2lod-wt/lutproof
owns:
  - docs/research/lutgholein-proof.md
  - docs/research/pr-lut-gholein-*.md
  - tools/emu/lut_*.py
exclusive:
  - instance:i1
gates:
  - "the two extra level-seed steps located in game.exe by address (emulator hook log), and the d2d code that skips them named"
  - "differential drlg.py vs drlg-dump, level 40, >= 20 seeds: mismatches before > 0, after 0; other towns and the act 1 Scan/Pops levels unchanged or improved"
  - "independent adversarial verifier report attached; PROOF complete or incomplete stated with the missing piece"
merge:
result:
```

## 1. Work

- [x] 1.1 Trace every level-seed write for level 40 in the emulator (lut_trace.py); name the two extra steps
- [x] 1.2 Find why d2d skips them; fix on a fresh branch off upstream/main in the clone; pr_guard, presubmit, tests
- [x] 1.3 Differential before/after with upstream's diff_drlg.py: level 40 seeds/tiles/units 1-20; levels 1, 2, 75, 103, 109 1-10
- [ ] 1.4 Independent adversarial verifier (Agent, general-purpose, read-only)
- [ ] 1.5 Docs: lutgholein-proof.md, pr-lut-gholein-draft.md, pr-lut-gholein-evidence-pack.md
- [ ] 1.6 Cleanup: no instance started, no pids left; report
