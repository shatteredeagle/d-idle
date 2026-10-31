# Game lifecycle in Go (agent LIFECYCLE)

Code: `internal/launch` (`lifecycle.go` Life, `display.go` Display/HostDisplay/XvfbDisplay, `system.go` System seam),
`cmd/d2game`, `tools/game.sh` (thin wrapper), sentinel restarts in `internal/sentinel/act.go` (`lifeStep`).
Tags: the commands are a 1:1 port of the old game.sh (VERIFIED LIVE in docs/multi.md); the Go port is verified OFFLINE with a
fake System only (`go test ./internal/launch ./internal/sentinel`). UNVERIFIED live: one start/stop/helper/mkprefix cycle.

## API
- `launch.NewLife(out)`; `Life.Start/Stop/Status/EnsureHelper/MkPrefix(ctx, id)`. Output lines are game.sh's. `launch.UsageError` = exit 2.
  (The brief asked for `Instance{Start,...}`; `Instance` already is the plain description struct, so the methods live on `Life`.)
- `launch.System` (Procs, Marker, Signal, Run, Spawn, Exists, HelperUp, Sleep): all OS effects; `OSSystem` is real, tests fake it.
- `launch.Display`: `HostDisplay` (i0, the desktop; never created/stopped, session env passed explicitly) and `XvfbDisplay`
  (private `:10N`, 800x600x24, `-nolisten tcp`, created on demand, SIGTERMed by Stop). Extra instances get no WAYLAND_DISPLAY/XAUTHORITY.
- Marker: every child gets `D2BOT_INSTANCE=<id>` (+ Proton env, `-w -ns`), so `mem.FindGamesFor` separates games.
- `launch.LockInstance(id, wait)`: the same flock files as tools/live.sh. Life itself never locks (sentinel holds the lock).

## Differences from game.sh
- Games are started with `proton run "Diablo II.exe" -w -ns` directly (what d2-prefix.sh did), detached with setsid; no bash, no ssh.
- wineserver path comes from the resolved Proton runner (game.sh hardcoded GE-Proton11-11).
- `mkprefix i0` is refused (usage error); old script would have cloned into d2lod-inst/i0.
- Usage text still says `game.sh` for byte-identical output.

## Dev wrapper
`tools/game.sh <cmd> [id]` builds `cmd/d2game` when sources changed (cached in /tmp/d2game-build), ships it to
`~/d2lod-live/d2game/d2game` on the worker host and runs it there; on the worker host it builds into `~/.cache/d2game`. `D2GAME_BIN=` overrides.

## Sentinel
`restart-game`, `helper-gone` and `idle-reaper` now call `launch.Life` in-process. `Config.GameSh` and `-game-sh` are kept but ignored.
The sentinel process still needs setcap on the worker host; d2game does not (it never reads memory).

## Not done
Fleet/Supervisor (D8), in-process leases, admission checks, Restart(ctx,id,reason) on a Supervisor, memfd: later work packages.

## Helper (agent INFRA)
`d2game helper <id>` takes only `/tmp/d2lod-helper-<id>.lock` (30 s), never the game lock, so it works while a live run holds the game.
It is idempotent: a running, listening helper is left alone; a helper process that does not listen is killed after 5 s and replaced.
d2game marks inherited fds close-on-exec so the flock of a calling live run cannot leak into the spawned helper. `$D2INPUT_EXE` picks another
helper exe (test builds). The helper writes its exit/crash log to `C:\d2input-PORT.log` (see docs/input.md).
