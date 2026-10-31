# d2d bot: "no exit to Blood Moor" (agent D2DEXIT, 2026-10-30)

Symptom: `d2bot --backend d2d --managed ... --goals 'farm:Blood Moor'`: nav says "no exit to Blood Moor" from the Rogue Encampment, even
with `--d2d-linkwait 90s`, while `cmd/d2dwalk` crosses fine. Tags: V verified, S secondary, U unverified.

## Actual live root cause (V live, 2026-10-30)

The first live repro (fixed Timeout, lockstep) STILL said "no exit" 1.3 s after arrival, and the new arrival log showed
`no layout verb: no outdoor borders (pending 0)`: the d2d binary `~/d2d-lab/fork/src/build/apps/d2d/d2d` (rebuilt 09:00 for the
vendor section) does not contain the P3 `debug layout/nearby` patch (`debug layout` answers the verb list as an error). With no
layout the world has no Adjacent, nothing is pending and nothing waits, so no LinkWait could help. Fix on the lab side: rebuild with
P3 (branch p3-layout-verb in ~/d2d-lab/fork/src); `bin/d2d.pre-vendor` has it. Code side: `Backend.Start` now logs a WARN when
`debug layout` fails. Live results, bot `--managed --no-save --class 1 --seed 5 farm:Blood Moor`, binary d2d.pre-vendor:
lockstep: gate x=279, Rogue Encampment -> Blood Moor in about 32 s wall, back to the camp later (arrival logs: 3 adjacent entries,
0 pending); real time (no --lockstep): same, Blood Moor at +21 s. The current (no P3) binary: the WARN and "no exit".

## Second cause found offline (V by failing test)

`d2dworld.World.Update` built ONE `context.WithTimeout(Options.Timeout)` (default 10 s) and the arrival link wait
(`waitLinked`, `Options.LinkWait`) ran inside it and ended on `ctx.Err()`. d2dwalk passes `Timeout: linkWait+30 s`, so its wait
was real; `d2denv.Build` did not set Timeout, so every link wait was capped at 10 s of wall time, and under lockstep on a loaded
The worker host 10 s is only a few step round trips, less than d2d's worker needs to link the neighbours. Worse, `d2denv` left LinkWait 0
(d2dworld's default is "no wait") unless `--d2d-linkwait` was given, and a ctx that expires mid-wait also fails the later
grid/borders verbs, so the world stayed with the arrival grid and no Adjacent entries: nav finds no gate.

Differences ruled out: the clock (Pacer vs wall): `LateClock.Poll` steps the game under lockstep and sleeps real time otherwise, like
d2dwalk's Sleep; the neighbour list refetch hook is the same code; the area id mapping is the same `ExitTo(2)` path.

## Fix

- `d2dworld`: `Update`/`Refresh` use `Timeout + LinkWait` as the bound; Timeout stays the per-verb budget. Test
  `TestLinkWaitOutlivesTimeout` (neighbour appears after 8 polls, 160 ms, with Timeout 50 ms: fails before, passes after).
- `d2denv`: `DefaultLinkWait = 90 s` when `Config.LinkWait` is 0.
- `d2dsrc`: the ~40 "unknown field" info lines at boot are now ONE line per snapshot that saw new fields ("d2dsrc: N new unknown fields:
  a, b, ..."), reasons stay in `Report.Unknown`. Test `TestLogOnce`.

## Stuck-but-pinging d2d: `d2denv.Beat`

`ping` is answered by the channel pump in the main loop, so a d2d whose world stopped but whose loop runs still pings. The old
`d2dHealth` heartbeat counted answered pings and could never look hung. `Backend.Beat.Next()` is a counter that grows only when a
progress marker moves: `clock` `game_ms` on a free-running d2d (must grow in real time), the `info` frame counter under `--lockstep`
(game_ms is frozen between steps by design) or when `clock` is missing. Wire it as `launch.Probes.Heartbeat` in
`cmd/d2bot/d2dbackend.go` (`func(int) (uint64, error) { return be.Beat.Next() }`, `HangAfter` default 45 s). Not wired here: that file
is owned by D2DCOMBAT/D2DTOWN. U: that frames keep counting while lockstep is idle, and that `clock` reports a live game_ms when not
lockstepped; offline tests only (`TestBeat*`). The failure it cannot see: a world that runs but a bot that makes no progress (that is
the goal manager's stall watchdog).
