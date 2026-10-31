# Tasks

```yaml
change: d2d-lan-oracle-demo
status: archived
agent: lanoracle
branch: agent/d2d-lan-oracle-demo
worktree: ~/code/d2lod-wt/d2d-lan-oracle-demo
owns:
  - docs/d2d-lan-oracle.md
exclusive:
  - instance:i1
  - port:4000
  - d2d-lab-lock
gates:
  - "d2d net-join into an original-hosted LAN game on a hidden instance: first N packets logged via d2dproxy, town layout compared"
merge: archived-by-dispatch
result: "d2d client joined an original Game.exe-hosted LAN game (port 4000; net-join and d2d.lab6 --join; 0 unknown packets; 187 S>C/10 C>S); Rogue Encampment 35/35 rooms match; Lut Gholein mismatch traced by lutgholein: comparison artifact (room bring-up order) + real d2d bug (room seeds two level-RNG steps behind game.exe); next: spawn/XP diff harness"
```

Effort estimate 3-5 agent-hours (WHISPERLOOK doc). Starts after origprobes finished; hidden instance only (i1 on its private Xvfb display).

## 1. Work

- [x] 1.1 Build the whisper side-load (ijl11.dll) and d2dproxy per the research doc
- [x] 1.2 Host an original game on the hidden instance, join with the headless d2d net-join client
- [x] 1.3 Log the first N packets through d2dproxy
- [x] 1.4 Compare the town layout seen by d2d with the original
- [x] 1.5 Write docs/d2d-lan-oracle.md (tag V/S/U)
