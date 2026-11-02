## Why

docs/research/lockstep-p3.md: the clock hook scales the game by k, but the bot's own waits are real time, so at k=2 it does half the work per game minute and at k=4 it is unusable. The cause is U (fixed real waits and real-time timeouts); this card adds one time-scale facility and moves the game-clock waits onto it.

## What Changes

- New package internal/gclock (D2BOT_TIMESCALE, default 1, bounds 0.25..16, parse error gives 1): Now, Since, Sleep, After, NewTimer, WithTimeout, WithTimeoutCause, D, and a Clock type. At k=1 every call is the plain time call.
- env.RealClock and input's wallClock delegate to it, so everything that already waits on env.Clock (nav, combat, loot, talk, bot pauses, the failsafe ladder windows) scales at once.
- Direct call sites moved: bot step and safety timeouts, town restore and sell caps, OOG input gaps and mouse settle, party flow bound and sleep, boss and farm budgets and the hotkey rebind bound in cmd/d2bot.
- Guard test internal/gclock lists the remaining direct time calls in the game-clock packages.
- docs/timescale.md: classification and how to run with the hook (live test not part of this card).

## Impact

Behaviour at k=1 unchanged. Offline tests only; not verified live.

Overlap (declared): the call-site edits touch files owned by other cards in docs/OWNERS (internal/town, internal/bot, internal/input, internal/party, internal/env, cmd/d2bot) by one line each.
