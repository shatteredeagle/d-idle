# Waypoints (agent WPUSE, 2026-10-30)

Tags: **D2D** = from the d2decomped notes (`docs/sme/d2decomped-findings-2.md` section 6, traced in game.exe 1.14d), **V** = verified in our own code or live (how given), **U** = unverified. Offline work only: nothing here ran against the game.

## Facts

- D2D: waypoint state is a 16-byte bitfield per difficulty; the Rogue Encampment (index 0) is always on. The client copy sits at RVA 0x3BF081 (our `d2.questmem` reader), the test helper `CheckWaypoint` at RVA 0x260E50.
- D2D: touching an unlit waypoint lights it and does NOT open the panel; a second click opens it. The bit is set on the first touch.
- D2D: the travel packet (C->S 0x49) is silently dropped for 10 000 ms after any level change, for an inactive waypoint, for the current level and for another act.
- D2D: the panel does not open while an item is on the cursor.
- D2D: on arrival the character stands 3 subtiles from the destination waypoint, in the safe room of the waypoint.
- V (live, i4 soak 2026-10-29, `internal/nav/waypoint.go` comments): a row click on a never touched waypoint does nothing (~20 s lost). Act 1 panel geometry and row clicks work. Act 2 rows 7 and 8 react to the icon column only.
- V (code): `saves.ReadProgress` gives the activated waypoints from the .d2s; `questmem` gives them live; `bot.activeWaypoints` merges save, live, the task layer's own touches and the runner's touches.
- V (code): Act 1 waypoint levels 1, 3, 4, 5, 6, 27, 29, 32, 35 (`nav.wpLevels`). Campaign steps activate Cold Plains, Dark Wood, Black Marsh; Stony Field and the four Monastery/Catacombs waypoints are only lit "in passing" (`touchWaypointHere`, within 70 subtiles, nothing hostile near), and the Monastery ones only when the route guard lets the bot there.

## What changed (this agent)

| rule | where | status |
|---|---|---|
| wait until 10 s (+1 s margin) after the level change before clicking a row; the wait comes after the walk to the waypoint and the panel opening, so walking counts | `nav.waitCooldown`, `noteLevel` (arrival = first observation in the level, so never earlier than the real one) | tested with the sim clock |
| a row click that did nothing is retried once after 2 s while the character arrived recently (the game's clock may be ahead of ours); a long-standing character fails fast (no retry) | `nav.TakeWaypoint`, `wpRetryWindow` | tested |
| second click for an unlit waypoint: `openWaypoint` already interacts up to 3 times; it now counts the clicks and logs `wp: activated` when more than one was needed | `nav.openWaypoint` | tested (`TestWaypointSecondClickActivates`) |
| cursor item: refuse to click, call the `Navigator.ClearCursor` hook first (`ErrCursorItem` when absent or failing) | `nav.clearCursor`, `town.Town.ClearCursor` | tested; WIRING NEEDED in `cmd/d2bot/behaviors.go`: `nv.ClearCursor = tw.ClearCursor` (not my directory) |
| log lines | nav: `wp: used <from> -> <to> (n clicks, waited ms)`, `wp: activated` (area, clicks); bot: `wp: activated <area>` (in passing), `wp: not usable` (from, to, err) | |
| corpse run: from town, when the last town portal was read somewhere other than the corpse area and an activated waypoint shortens the way, take waypoint plus walk (`travelTo`) instead of the portal | `bot.preferWaypointRun`, `gotoDeathArea`, `observer.portalFrom` (set when the character goes field -> town alive) | tested; log `corpse: waypoint and walk instead of the town portal` |
| `BackToField` and `waypointToTown` honour `nav.useWaypoints` | `town/trip.go` | |

Existing and unchanged: `travelTo` already takes the furthest known-active waypoint on the route (leaving town toward Cold Plains / Stony Field / Dark Wood / Black Marsh, and returning after a chore trip when there is no portal); `wpTripWorthTrying` skips known-inactive and recently failed waypoints (10 min ban).

## Route guard

`travelTo` calls `routeAllowed` on the whole hop list BEFORE any waypoint is considered, so a waypoint to Outer Cloister, Jail, Inner Cloister, Catacombs (areas 26..37, Tamoe 7) is refused exactly like the walk while the guard is closed. The corpse run goes through `travelTo` too. Tests: `TestWaypointDoesNotBypassRouteGuard`, `TestCorpseRunWaypointRespectsGuard`. Nothing here lowers the guard.

## Knobs

| knob | default | meaning |
|---|---|---|
| `nav.useWaypoints` | 1 (on) | master switch. ON because waypoint travel already ran in every live soak since NAVSTALL (V: Act 1 rows, Act 2 rows); the new rules only add waiting and guards. Set 0 to walk everything. |
| `nav.wpMinDistance` | 2 | a waypoint hop must skip at least this many level hops (2 = shortest that saves a level). Measured in levels, not cells. |
| `nav.wpCooldownSec` | 10 | game.exe's 10 s refusal window (D2D); no reason to lower. |

Why default ON: the simulation tests prove the policy never bypasses the guard, never clicks inside the cooldown, never clicks with a cursor item (given the hook), and that the knob and distance rules fall back to walking. What the simulation cannot prove is the game's side (below), which is why the live checklist exists; if the first live use misbehaves, `nav.useWaypoints 0` restores the old walking behaviour.

## LIVE CHECKLIST (first live use; nothing here ran live)

1. Start a character with Cold Plains active (or activate it) and stand in the Rogue Encampment more than 15 s after the last game join. Expect in the log: `wp: used 1 -> 3 (N clicks, waited M ms)`. `waited` should be near 0 when the walk to the waypoint already took more than 11 s; if the wait was needed the log shows it.
2. Take a waypoint within 5 s of arriving in a level (e.g. town portal back, then waypoint at once). Expect a wait of about 11 s and a single row click. If the row click is eaten anyway: look for the retry round (two `nav click` lines on the row) and report `waited`, the observed game time since arrival and whether the click was eaten: that decides whether the 10 s window starts at the loading screen end or earlier (U).
3. Visit Stony Field for the first time with its waypoint not lit. Expect `wp: activated 4` (in passing) or, when the first interaction did not open the panel, `wp: activated` with `clicks=2` from nav. Check the d2s afterwards (`cmd/probes wpdump`): the bit must be set.
4. Pick something up (item on the cursor) and call a waypoint trip: expect `ErrCursorItem` if the hook is not wired, the item put into the pack if it is. Look for the panel opening afterwards.
5. Kill the character in a far area (Black Marsh) after a town trip from Cold Plains: expect `corpse: waypoint and walk instead of the town portal`, then `wp: used 1 -> 6` or the furthest lit waypoint, then the walk to the body. The G1 failure `no way back to level 26 from 3` must not repeat.
6. With clvl < 20 and no merc, order a quest to Monastery: the refusal `route guard` must come before any `wp:` line.
7. Act 2+: rows 7 and 8 still need the icon column (V); a lit-by-passing waypoint in a later act appears as a new act tab only after the first activation (U: tab x positions extrapolated).

## Unverified / open

- U: that the cooldown clock starts at our first observation of the level (we assume it is never earlier than the game's; the retry round covers the opposite).
- U: Wine-hosted client sends the 0x49 packet exactly like native (the notes' open question).
- U: `portalFrom` is only an approximation of the portal's landing level (it ignores a portal read from a field area and then left by waypoint). A wrong guess costs one extra waypoint trip or the old wrong-level path, never a guard bypass.
- U: no bot-side way yet to learn a waypoint is lit except save, memory and own touches; `known=false` still tries the trip once (10 min ban on failure).
