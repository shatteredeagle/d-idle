# Swing model of the resource simulator (TASK-404)

Per-class plain-attack seconds in `internal/resource/swing.go` come from game.exe's closed form (V run, docs/research/swing1-trace.md Q1): swing ticks = `max(1, ceil(256 (F - s) / rate))`, 25 ticks per second, start frame `s` = 2 with a melee weapon for the Amazon and the Sorceress, 0 for the others. Sorceress staff: F 18, 16 ticks, 0.64 s (was 1.0, a U default); Paladin sword: 15 ticks, 0.6 s (unchanged). Add a class to `swingData` to give it a scenario; an unknown class is an error.

- Kill-time bucket (`internal/bot/killtime.go`): mode 12 (KK, a kick) now counts as casting; it was charged to waiting (swing1-trace.md B3).
- The private d2d fork is NOT changed: its Amazon and Sorceress swing is 1 to 3 ticks longer than game.exe (no start frame; fork patch, upstream draft row 1c). Kill-time inputs measured on d2d are therefore pessimistic for Sorceress melee by about 12 percent of swing rate (U beyond the trace's V run).
- Sorceress melee kill-time figures of the simulator before this change were 56 percent too slow against game.exe.
