# Tasks

```yaml
change: runab
status: archived
merge: 48cd739
agent: runab
branch: agent/runab
worktree: ~/code/d2lod-wt/runab
owns:
  - docs/d2d-run-ab.md
  - tools/runab/**
exclusive: []
gates:
  - livelock check passes clvl 3 with a handful of fresh-game lines at most
result: "live check: livelockfix alone FAILED at clvl 3 (fresh game every ~10 s: d2d restarts the process without a menu poll so newGames never moves); extra fix in freshGameToLevel (reset cleared after NewGame) PASSES (clvl 3 at 30 virtual min, 5 at 2 h, 1 fresh-game line); run-policy A/B 4 seeds x 2 h: off vs on: s/kill 22.1 vs 19.9 (p 0.44), kills/h 164 vs 182 (p 0.47), XP/h 4299 vs 4592 (p 0.69): not significant; gate v2 INCONCLUSIVE (XP/h 4523 vs 6600, deaths PASS 0 in 8 h); NEW stall: seed 42 walk-only arm freezes at virtual 90 s: combat loop spins on nav: no path: empty path without stepping lockstep (backlog); needs a G1/G2 check (Game.exe-affecting)"
```

- [x] livelock live check (2 virtual hours, lab6)
- [x] run A/B, 4 seeds x 2 h per arm
- [x] cleanup of own processes
