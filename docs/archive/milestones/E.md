# E: party of 2-4 bots on one host

Placed after the good-bot gate as a feature, with its main risk retired early. Code exists offline: `internal/fleet`,
`internal/party`, `cmd/d2fleet` (V docs/fleet.md, fake runtime). 37 % today (rescored 2026-10-28; the spike is done).

## Early spike: DONE (V docs/net/multiplayer.md section 9, PARTYLIVE, i1 host + i2 join, 2026-10-28)
Two Wine prefixes join one TCP/IP game over 127.0.0.1; both characters are visible in both unit hash tables and rosters; host Save and Exit
returns to TCP/IP Options; there is no name/password/difficulty dialog; one host per machine (port 4000); the join IP box keeps old text, so the
flow sends backspaces; one join of about 8 sat on Loading for over 2 minutes (U cause). Joining does NOT party the players (partyid 65535).
Not measured: XP split on a kill. Remaining risk is party formation (invite flow, multiplayer.md 9 T6/T7), roles and CPU.

## Later work packages (after H, U)
| wp | what | effort |
|---|---|---|
| E-1 | Fleet live: add/remove slots at runtime through launch.Life, admission by CPU and RAM | 2 |
| E-2 | Leader/follower roles: follow, shared portals, town sync, follower death handling | 5 |
| E-3 | Per-instance CPU cap (cgroup cpu.max or frame limiter) and its effect on timing knobs | 2 |
| E-4 | 4 bots for 2 h, soak per instance plus host load | 1 + wall time |
| E-5 | Party as a game feature (recruiting a bot is an unlock) with L1 | 2 |

## Risks
- About one core per idle instance (V docs/multi.md): 4 bots plus the host's own load may not fit the worker host; the container target needs a frame limiter.
- Input arbitration across bots is per window already; party logic adds cross-instance timing that the single-bot failsafes do not cover.
