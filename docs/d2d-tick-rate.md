# d2d tick rate versus the 25 fps original (agent PROBES-D2D, 2026-10-30)

Tags: V-d2d = measured on the private lockstep fork (`d2d.lab5`, seed 5, throwaway Sorceress, `--no-save`), S = read in code, U = not tested.
Probe: `cmd/d2dspawn ticks` via `docs/probes/run-d2d.sh` (own socket, lab lock held only for the run).

## Result

**One lockstep tick = 40 ms of game time = one original frame at 25 fps. 25 ticks = 1 game second. The conversion used everywhere is right; nothing to fix.**

| Check | Result |
|---|---|
| Fork constant (S) | `~/d2d-lab/fork/src/apps/d2d/lockstep.cpp`: `kTickMs = 40` ("the world's fixed step (25 Hz)"); `step` adds exactly one `kTickMs` to the virtual clock per stepped frame; the world itself runs fixed `kTickMs` ticks (`town.cpp`) |
| `step 250` (V-d2d) | `tick` 4 -> 254 (+250), `game_ms` 160 -> 10160 (+10000 = 10.00 game s, 25.0 ticks/s) |
| `step 1500` (V-d2d) | `tick` +1500, `game_ms` +60000 (60.00 game s) |
| A known per-frame effect (V-d2d) | max mana 250, mana 0: gain in 250 ticks 20.508, in 1500 ticks 123.047, exactly `floor(256*250/3000)/256` per tick (the original's rule: all mana back in 3000 frames = 120 s). Per game second 2.051 = the original's value at 25 fps. Same result in docs/d2d-fidelity.md check 1a (8 runs, other max values) |
| Wall speed (V-d2d) | 250 ticks in 11 ms, 1500 in 68 ms (idle town, one core) |

## What converts ticks to hours in our code (S, read)

- `internal/d2dctl/verbs.go` `Step`: n world ticks of 40 ms. `internal/d2dinput/pacer.go` `Sleep(d)` sends `ceil(d / 40 ms)` ticks.
- `cmd/d2dsoak/run.go`: `target := int(cfg.Hours * 3600 * 25)` and `-chunk 1500` (= one virtual minute): 1 virtual hour = 90,000 ticks, 8 h = 720,000, as docs/d2d-fidelity.md section on the replicated soak says.
- `Pacer`/`d2denv` clock: virtual time = `game_ms`, so every per-hour metric (xp/h, kills/h, deaths/h) is per 3600 game seconds at 25 frames per second.

## Caveats (U)

- The original runs 25 logic frames per second only while the machine keeps up; a loaded Game.exe can drop below it, so a Game.exe "hour" of wall time may hold fewer than 90,000 frames. d2d hours are exact game hours; Game.exe wall hours are an upper bound on frames. Compare per game time (frame counter, `+0x...` unknown here), not per wall time, when the original side fills the same columns.
- That the original's per-frame regen/cast/attack tables are frame based (not time based) is the premise of "tick = frame"; the 14-tick Sorceress bolt versus the table's 13 and the 14-versus-19 Lightning difference in docs/d2d-fidelity.md are d2d content differences, not a clock mismatch (the clock check above is independent of them).
