# Pursuers: the hero that walks on with monsters behind it (card pursuers)

> status: rule built and tested offline (internal/nav/pursuit_test.go); NOT run live. V = counted in the container log of d2play-20261108 or proven by a test here, S = read from code, U = unverified.

## Reports (2026-11-10)

1. Paladin ran away from two enemies at full life and mana and did not turn to fight (Blood Moor, wandering, not heading to a waypoint or town).
2. Later, same area: the hero stood still while enemies hit it.

## Findings

- S (code): the three wander paths (`nav.explore`, `nav.roam`, exit walks) stop for the bot's `hostile()` check only (`bot.sweepPass`, `hostileNear`, ClearRadius 30). Inside a walk the only monster rules were (a) the stall rule `packNear` (8 subtiles, only after the stuck timer ran out, max 4 yields per walk) and (b) the low-life `stuckFight`. A hero walking at full life with followers behind it never stalls, so neither fired. Same for `bot.travelTo` (route walks): only `Fighter.Safety` runs beside it (`guardOnce`), no fight.
- S: `hostile()` is false while the 90 s / 40 subtile futile window is open after a `Clear` that killed nothing, and for units the fighter remembers as ignored (`IsIgnored`). During that window followers are not a stop reason for the roam.
- V (6 h of log, 11:26 to 17:27 UTC): 22906 `nav.trace move` lines, 1270 kills; `status` samples while `traveling`: 18 of 34 had `hostileNear > 0`, `resupplying` 9 of 72, `clearing` 191 of 380. `clear killed nothing; moving on` 24 times (all with `fight.tick` ticks 0, maxHostiles 0, i.e. the bot counted a hostile the fighter scope did not). `fight.tick` lines: 469; 127 had zero kills and zero attacks (87 with lastNoCast `approach: no shot`). `ignoring target` only 2.
- V: 17:23:53 to 17:27:09 UTC the bot ran one uninterrupted roam (`nav.trace plan/move/click`, callers `roam < explore < Explore < sweepPass`), no `status`, no `fight.tick`, no kill, no retreat or backing-off line. So the fight loop was not running and no retreat logic owned the hero: the wander owned it for over 3 minutes. Stamina low (the coordinator's tail) only slows the hero.
- U: why `hostile()` stayed false during those 3 minutes while something was hitting the hero. Candidates: the futile window, a unit class that `r.hostile` rejects (Killable or IsFriendly), or a unit absent from the snapshot's monster list. The log carries no monster ids or positions for the walk, so this cannot be decided from it. Next live step: log, in the wander walk, the nearest monster (class, mode, distance) when `LifePct` drops during a walk.

## The fix (class and build generic, nav layer)

`nav.pursuit` (internal/nav/pursuit.go, `walkOpts.pursuit` set by explore, roam and exit walks): a living non-NPC monster within `nav.pursuitRadius` (10) for `nav.pursuitPolls` (4) observations in a row, at life at or above `nav.pursuitMinLifePct` (50), hands the walk to the `Yield` hook (a bounded Fighter clear that filters friends and lifts earlier ignore decisions) and then replans. It uses the nav monster list (no friend or killable filter beyond town NPCs), so it also covers the U cases above whenever the monster is in the snapshot. Budget: shares `maxYields` (4) per walk, 3 s cooldown after a hand-over. The fight itself still decides attack or retreat (`resource.AttackChoice`, safety), so a caster still keeps distance. No class check: the nav layer has no class data; the test varies life and knobs. 0 on `nav.pursuitRadius` switches the rule off.

Not changed: `bot.travelTo` waypoint and exit-by-exit routes through `ExitTo` are covered only for the exit walk; the other travel walks (GoTo, waypoint) stay as they were. Risk U: a deliberate flee (stale pack, low kit) at life of 50 % or more through an exit walk now turns to fight; the Safety guard still preempts at low life.
