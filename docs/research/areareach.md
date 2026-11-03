# AREAREACH: why Dark Wood and Black Marsh were unreachable in Gate 1 run 0628

> status: current. Offline analysis of the bot log of gate1-20261103-0628 (read-only, worker host) and the code.

Cause (V read, bot-0.log, 28 "leveling target not reachable" lines): `unreachable` (internal/bot/task.go) accepted a route of more than 3 hops only when the character stood IN TOWN or at an activated waypoint AND a waypoint lay on the route. Every refusal happened with the bot in Den of Evil (8), Blood Moor (2), Burial Grounds (17), Crypt (18), Cave Level 2 (13) or Mausoleum: no waypoint there, so Dark Wood (4 to 5 hops), Black Marsh (5 to 6) and Cave Level 2 (4) were refused even though Cold Plains and Stony Field were activated (V read: "wp: activated" for areas 3 and 4 at 06:33 and 06:43). The fallback pick then stayed in the same low areas, so the rule re-fired for 80 minutes.

Not the belt fixes (I): no town trip, walkableHome or trip.go code is on this path. The other runs reached Dark Wood because they happened to be in town or at Cold Plains when the pick was made (I, luck of the position, not shown).

Fix (U live): `longestWalk` counts the walk as the hops from the start to the first waypoint on the path, then the hops after the last one; a pure walk up to 5 hops (Rogue Encampment to Black Marsh) is allowed and travelTo activates every waypoint it passes. Danger and banned areas stay refused by routeAllowed; stalls still end in the 150 s nav-stuck skip.

Fallback: `PickOptions.XP` (Runner.stepXP, the areaXP table) ranks the eligible steps when the preferred one is excluded. Expected gain per xphshort.md: +1100 (ranking) and +1200 to +1700 XP/h (reach), one seed, I.

Gate 1 re-run check: `grep -c "no activated waypoint" bot-0.log` should be 0 for areas 5 and 6; `grep "level plan: clvl 1[2-9]"` should name Dark Wood, Black Marsh, Crypt or Cave Level 2 rather than Cold Plains (area 3) at clvl 12 and up; `grep "wp: activated"` should list area 5 and 6 early.
