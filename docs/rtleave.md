# rtleave: RT.leave wired (offline), live run pending

Not run live (nothing here is V live). Tags: V verified by code reading or offline test, S secondary, U unverified.

## What changed
- `cmd/d2drill/leave_lab.go` `leaveHook` is `Fighter.Leave` in `newLab` (live.go): `Cycler.LeaveGame` (Save and Exit through the d2d control path, `Proc.SaveExit`: "save", then "quit", the process goes and its socket file goes away) and then `Cycler.EnterGame` (a no-op when `Reenter` already restarted; V by code, proc.go). An exit error is returned (the drill reports it as ActErr = FAIL); a rejoin error likewise.
- `lab.left` is set by the hook between exit and rejoin. After the rejoin the snapshot is InGame again, so the old "state left InGame" check could never fire. `Retreat` now reports `LeftGame` with `TicksToEffect` 0 (the hook is synchronous) and skips the deferred `debug goto 1` (new process, no fight).
- PASS stays numeric: `LeftGame` within 200 ticks (`MaxTicks`), life floor above 0 (not `Dead`). Without the hook, RT.leave stays Blocked. Chicken threshold, class and skills come from data/tuning as before; no literals added.
- Tests: `internal/drill/retreat_leave_test.go` (verdicts: rejoined PASS, unwired Blocked, game not left FAIL), `cmd/d2drill/leave_lab_test.go` (hook order leave then enter, `done` only after a successful exit). Check that they bite (S): remove the `EnterGame` call or the `done()` call in `leaveHook` and `TestLeaveHookExitsThenRejoins` fails; return `LeftGame` false from the stub and the drill test fails. Not mutated by hand beyond this reasoning (U).

## Live run (later, d2-live, hidden slot, never the Gate slots)
Same recipe as docs/probes/retreat/2026-11-04-RT2-run.sh with the drill `RT.leave` and `-repeat 1` first (hidden slot number, a throwaway character, lab binary with cheats):
```
go build -o /tmp/rtleave-d2drill ./cmd/d2drill
D2D_AGENT=rtleave D2D_BIN=<lab d2d binary> tools/agentrun.sh rtleave --ttl 300 -- tools/d2dslot.sh run -n <hidden slot> -t 240 -- /tmp/rtleave-d2drill -group RT -only RT.leave -repeat 1 -label rtleave -out /tmp/rtleave-out -bin <lab d2d binary>
```
Expect: log "Save and Exit requested (chicken", verdict PASS, ticks to effect 0, life floor above 0 (U).

## Risks
- The d2d process quits and its socket goes away (V, proc.go comment). `lab.be.Client` must reconnect to the new process; `Proc.Start` makes a new client socket or reuses the path (U: check that `be.Client` stays valid, else the drill's later steps fail; `-repeat 2` would expose it).
- Under lockstep the new process starts unstepped; `EnterGame` waits for in-game (default 90 s timeout) (S).
- Attach mode (`Bin == ""`) cannot restart: `LeaveGame` only saves, `EnterGame` then waits on a dead game. Use a managed slot (V, proc.go).
- The lab cheat ledger and run belief refer to the old process; run RT.leave last or as its own run, and do not chain other drills after it in the same invocation without `Restart` (U).
- A second `repeat` run starts from the rejoined character, which may not be in the field (the setup `debug goto` handles it, U).
