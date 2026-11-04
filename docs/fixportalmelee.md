# fixportalmelee: a town portal cast under melee is not an escape (card fixportalmelee)

> status: current. Offline fix from the Gate 1 soak 2 death analysis ([probes/postfix-soak2-analysis.md](probes/postfix-soak2-analysis.md) section 3). Tags: VERIFIED OFFLINE = a test in this change; UNVERIFIED LIVE = needs the recipe below.

## The death timeline (from the soak log, VERIFIED in the analysis)
clvl 4, maximum life about 40, Burial Grounds, 3 to 5 undead adjacent. The belt held 4 healing potions at entry (not empty, the soak report's F1 wording is wrong). All 4 were drunk in about 16 virtual s. At 34% life with the belt empty the bot chose the town portal; the cast takes about 1.3 s, melee hits cut it, the scroll was spent, the character died 22 ms (wall) later.

## The escape logic (read)
- `combat.Decide` (internal/combat/safety.go) is the survival decision per tick: chicken at or under the chicken line, crowd rule, then below the drink line: potion, else `TownPortal` when no potion is left.
- `resource.AbortFight` (internal/resource/abort.go) runs after the potion step: `PortalOut` when a scroll exists at the chicken line (within 8 points of it), `escape()` for thin belt, no mana and dead merc cases.
- Both chose a portal whenever a scroll existed. Neither looked at how many attackers are adjacent (only the crowd rule did, and only with 8 or more hostiles in range).

## The rule
With `resource.CastAdjacentMax` (3, the crowd rule's existing `crowdAdjacentMax`, now defined once in internal/resource) or more hostiles within 3 subtiles in a fight, a cast that melee can interrupt is not an escape:
1. a potion that is ready (rejuvenation first, then healing): drink it;
2. else leave the game (chicken). `AbortFight` goes straight to `LeaveGame`.
A step away is deliberately not offered: melee monsters follow and keep hitting (INFERRED, UNVERIFIED live), and a retreat that fails leaves the character with neither cast nor chicken; the crowd rule keeps its own retreat. Unchanged: fewer than 3 adjacent, a quiet town trip (no fight), no scroll, and every numeric default. No knob was added; the death rule (docs/contracts/death-rule.md) only judges deaths, a chicken is not one.
Code: `meleeCastGuard` wraps `Decide`; `castCut` in AbortFight; `Adjacent` is now always filled in `Fighter.Safety` and `resInput`.

## Verified offline
`go test -p 1 ./internal/combat ./internal/resource`: `TestSafetyMeleeCutCastScenario` (soak shape through the fake game: 34% life, no potion, scroll, 3 adjacent plus a fifth: Chicken; 1 or 0 adjacent: TownPortal as before), `TestMeleeCastGuardOrder` (rejuv, healing, cooldown, 2 adjacent, no scroll, no fight), `TestAbortFightCutCast` (chicken line and thin-belt escape). Before the change the same shape returned TownPortal.
## Unverified live
That d2d cuts a cast on a hit and how long a cut cast takes (drill D2.d, cut cast, is BLOCKED); that 3 subtiles is the right adjacency for the monsters met.
## Live recipe (hidden slot)
`debug spawn` 3 or more melee monsters next to the character, `debug stat` life to about 34% of maximum, no potions in the belt, one scroll in the pack. Expect the bot log `chicken` (leaves the game) and the scroll unspent; with one monster adjacent expect the portal as before.
