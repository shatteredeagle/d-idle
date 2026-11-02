# Tasks

```yaml
change: decomprepair
status: archived
agent: decomprepair
branch: agent/decomprepair
worktree: ~/code/d2lod-wt/decomprepair
skip_specs: true
owns:
  - docs/research/price-trace.md
  - docs/research/pr-price-proof.md
  - tools/emu/price_oracle.py
  - tools/emu/price_model.py
  - tools/emu/price_terms.py
exclusive:
  - "fork-branch:trace/price"
gates:
  - "price_model.py equals game.exe on 100,000 oracle cases; changecheck -cli 0 warnings; leakcheck -mode fail 0; openspec validate --all --strict"
merge: 90101b4
result: "Formula V (emulator, model 100000/100000); d2d deviations D1-D7 proven by differential; bot findings B1-B6; PR candidate not built (time box). Not verified live on the original game; drill D3.r (d2d) price agrees."
live: ""
credit:
  component: Vendor repair gate (trace)
  author: owner
  agents: [Claude Code]
  verified_by: ["Claude Code (card verifyprices, independent run)"]
  consent: none
```

- [x] 1.1 trace FUN_0062efb0 (buy, sell, repair) and the gold caps
- [x] 1.2 emulator oracle and formula model, 100,000 cases
- [x] 1.3 differential against d2d, deviations by term
- [x] 1.4 bot findings with file:line, docs and index lines
- [ ] 1.5 upstream PR candidate (fix commits, tests, verifier): not done
