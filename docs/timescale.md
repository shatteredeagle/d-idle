# Bot time scale (D2BOT_TIMESCALE)

> status: offline only (card `botclock`); not verified live. Background: docs/research/lockstep-p3.md.

The clock hook runs the original game k times faster than real time. The bot must then wait k times less in real time and see game time pass k times faster. `internal/gclock` does that: set `D2BOT_TIMESCALE=k` for the bot process (same k as the hook's `k=` setting). Unset, empty, unparsable or outside 0.25..16 means 1, and at 1 every call is the plain `time` call (no behaviour change).

Run (live test needs the user's go and instance i4; checklist only):

1. Start the hook with k=2 as in lockstep-p3 (cfg next to the DLL), log in.
2. `D2BOT_TIMESCALE=2 d2bot --attach --goals 'farm:Blood Moor' --run-for 300s` (run-for is a host timer: real seconds).
3. Compare kills per GAME minute with the k=1 run (lockstep-p3 table); then k=4 the same way.

## API

`gclock.Now/Since/Sleep/After/NewTimer/WithTimeout/WithTimeoutCause/D` and `gclock.Clock` (satisfies `env.Clock` and `input.Clock`). `Now` is base + real elapsed * k; durations are divided by k.

## Classification (S: from reading the code, counted over non-test code outside cmd/*live and tools)

GAME-CLOCK (about 15 sites, all moved): everything on `env.Clock` (`env.RealClock` now delegates: nav, combat, loot, talk, bot pauses, failsafe ladder and stall windows via `Clock.Now`), `input.wallClock`, bot step timeout (`runStep`) and the 15 s pause-fight and safety bounds, town 20 s weapon restore and the sell cap, OOG click/key gaps and the 40 ms mouse settle, party flow bound and default sleep, boss explore/attack/clear budgets and the 2 min hotkey rebind in cmd/d2bot.

HOST-CLOCK (kept; about 100 sites): process and game lifecycle (launch, managed, oog wire, sentinel act, 5 min recycle, restart timeout), helper dial backoff and RPC deadlines, d2bot supervisor, watchdog, heap guard, display, run-for, soak and viewport timers, the runner heartbeat (`lastStep`, supervised by wall time), d2d backend code (virtual time of its own).

Guard: `internal/gclock` TestNoNewDirectTimeCalls fails when a game-clock package gains a direct `time.Sleep/After/...` or `context.WithTimeout`.

## U: might still need per-site tuning

- The d2input helper runs its own real-time `sleep` and key `hold=` inside a Seq; those strings are not scaled. Holds must stay at least one game frame (10 ms at k=4).
- `Now()` is also used for log stamps in places; they run fast at k>1.
- Farm duration (`bot/farm.go`) and goal ActiveSeconds stay real time.
- Frame stalls (area loads, menus) are real-time work; their game-time share grows with k.
- Per-click waits that are Tune ms values already go through `env.Clock` and scale; whether a particular wait is too short in real time at k=4 (network of the helper, 1 ms floors) is only visible live.
