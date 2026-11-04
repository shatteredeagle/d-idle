# Held-for-Andariel log line

While `boss.andarielHold` is on, the bot logs (once per clvl change, never per tick; noteAndariel in internal/bot/task.go
dedups on the line text, so a run adds at most one line per level from clvl 8):

```
boss: held for andariel (unattended hold: the throwaway fight has not won yet (boss.andarielHold)) clvl=N floor=F merc=alive|dead|none
```

- Prefix `boss: held for andariel (` is unchanged. `floor` is the guard floor (same number as the latest
  `route guard: andariel floor=` line), `merc` is the same state word as that line (`task.MercState`).
- Before this change the line was logged once, at clvl 8 (V, finished Gate 2 runs), so the interim rule could never see
  it at clvl >= 14.
- Parser: ops gate-judge.py matches the prefix and reads the clvl/floor/merc fields (S: from the brief, not read here).
- Code: `task.AndarielLine` (internal/task/gate.go). Test: `TestHeldLineOncePerClvl` (4 lines for clvl 8,8,8,14,14,16,16,16,17,
  both merc states; gate verdict unchanged).

## Live recheck (not done, needs the game)

Run a Gate 2 seed until clvl >= 14 with the hold on; `grep 'boss: held for andariel' log` must show one line per clvl,
the last with `clvl=` >= max(14, floor-1) (merc alive, floor 15) or >= floor (dead/none, floor 17), and the judge's
interim rule can then pass.
