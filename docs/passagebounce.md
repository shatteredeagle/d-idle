# Arrival bounce: Underground Passage, Crypt and every cave (TASK passagebounce)

Live bug 2026-11-11 (sandbox hero, clvl 13): "keeps going into the Underground Passage and leaving again"; later the same in the Crypt.

## Evidence (V, 3 h of the sandbox container log, 521 zone entries)
- 167 entries into Underground Passage 1 (area 10): 149 from Dark Wood (5), 17 from Stony Field (4). Median stay 0.8 s (from 5), 1.2 s (from 4). The Crypt (18): stays of 1.1 to 1.5 s, five in a row.
- The exit fires 0.15 to 0.22 s after the FIRST move order, whatever it was: step-away clicks 2 to 20 subtiles long, away from or toward the exit (arrival (265,275) with the return exit 11 away; arrival (380,25) 11 away; Crypt arrival (19,12)). Walking 11 subtiles takes about 1.8 s, so this is not the hero walking into the trigger: it is the d2d arrival bounce (docs/d2d-loop.md). The same click sometimes passes (2 of 17 from Stony Field, 0 of 149 from Dark Wood), so no click rule fixes it (U why).
- Not a planner flip-flop and not roam: the Crypt shows `roam: coverage points 71, exits 1`, so dropNearExits does not empty it. The planner wants Dark Wood (campaign "Inifuss tree") or, once there, Stony Field, and both routes cross the Passage.
- Why the breaker did not hold: it only counted `exploring crossed an area border` in the clearing state (replanCause). The travel walks end with `left level 10 for 5 but ended in 4` and never reached it; `area skipped ... unreachable` skips targets, not the bounced area. State is also lost on a bot restart (three in this window).

## Update: the real cause is the bot's priming click (arrivalsettle)
d2d's arrival spot and warp boxes are not at fault (decomp12, fid-warp-arrival.md). The first `cmd move` after a level change is preceded by `Kick`, a screen click at (400,300) = the player's own feet; on a fresh arrival the return warp unit stands there, so the kick enters it (S from code, matches: bounce about 0.2 s after the first order whatever its direction, a minority of arrivals pass). d2d's `cmd move` takes world cells, so no stale mapping exists. Fix: `Controller.primeToward` clicks `kickReach` (110) pixels from the player toward the move target (`Client.KickAt`), logged as `nav.trace kick`. The settle wait asked for is not added: bot decision latency after an arrival is already 0.7 to 1.2 s (V, log). Measure: the `arrival` line carries cumulative `arrivals` and `bounces` (rate = bounces/arrivals); expect it to fall from 88-97 % in the Passage. Not verified live. The #609 breaker stays as the safety net.

## Fix (breaker, passagebounce)
`noteArrival` (lifelog.go, called at each area change) logs `arrival area from x y` and, when a stay under 4 s ended by going straight back (neither side town), `arrival bounce` with the arrival and leave positions, and `noteBounce` (loop.go) counts it keyed by the bounced AREA. 3 within 2 min skip that area with the escalating border-loop skip (3, 6, 12, 24, 48 min); routes through a skipped area are banned for the planner (goal.go routeBan), so a target behind the Passage is dropped instead of retried. Class independent, no class data; test `TestArrivalBounceSkipsTheBouncedArea`.

## Not fixed / next
- Why the game fires the exit and why 12 % pass (U). A live look with `nav.trace` at 50 ms around the first order after a warp is still needed; the new `arrival` / `arrival bounce` lines give position data for it.
- A hero already inside Dark Wood with the Passage banned farms Dark Wood until the ban ends (no way back except a town portal). Not changed.
- Breaker state is in memory: a restart forgets the escalation.
