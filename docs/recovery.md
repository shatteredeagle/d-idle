# Recovery phase (broke character)

Running out of gold is not a failure. A human drops back to easy monsters, farms gold and sellable loot, restocks and
returns to the harder goal; dying is normal. Tags: V = verified in the code or in the cited log, S = SME advice
(d2-sme, docs/sme/play-economy.md, play-bot-pitfalls.md row 3), U = unverified.

## What went wrong (G1 attempt 9, i4, Soaky clvl 10)
- V (log 20261029-195315): "poverty mode: on" fired at 19:56 (gold 15), farming Cave Level 1. For the next 90 minutes gold
  stayed 5-84, kills 40 in 10 minutes, XP 2-4k/h. Poverty mode was on, but it never earned anything:
  1. Poverty's lean plan engages only packs of danger 0 (resource.PovertyAdvice), so Cave Level 1 packs (danger 1-2) were
     refused: "combat fight danger=2 kills=0".
  2. choresDue sent an emergency potion trip whenever healing < threshold and gold >= 30 (minPotionPrice), although 37 gold
     buys one potion: town trip, walk to the waypoint, nothing bought ("broke":true), back, again (4 trips in 10 min).
  3. The campaign's step "Level up to 14" owned the planner; poverty only picked the area inside LevelTo.
  4. Nothing counted gold per hour or ended the state.

## Design (internal/bot/broke.go, internal/resource/recovery.go)
- State machine `resource.RecoveryCheck`. Start: `recoveryOn`, gold < `recoveryKitGold` (500) AND the missing kit
  (KitShortfall: portal + HPFirst healing + MPFirst mana) costs more than the gold, outside `recoveryCooldownMin` (15).
  End: gold >= `recoveryTargetGold` (900) and fewer sellable items than `bot.sellMinItems` (the sell trip is done), or
  `recoveryMaxMin` (45) elapsed. S: enter below ~400-600, leave at 800-1000, 30-60 min per cycle.
- Log lines: `recovery: start (gold N < kit M)`, `recovery: end (gold N, reason ...)`, `recovery: gold +N in M min`
  (once per minute while running and at the end, with goldPerHour, xpGained).
- While on: the campaign's quest step is not run (recoveryStep: one leveling slice in the recovery area instead); a gear-gated
  character keeps the old gate. Applies to campaign and reach_level goals, not to farm runs.
- Area: plan area of a character `povertyLevels` (4) lower, restricted in Normal act 1 to Blood Moor, Cold Plains, Stony
  Field, Den (only with 2+ potions), Cave 1/2 (S). Without a portal only the first field (walk home works there). Cleared
  and death-banned areas are excluded as before.
- Potion trips are suppressed while recovery runs and gold < recoveryKitGold (a trip with an empty purse buys nothing);
  the existing trip gold floor already limits portal purchases to one. Sell trips fire for a single sellable item
  (inPoverty is true during recovery); loot rules are the existing gold-based poor rules (loot.PoorGoldBelow).
- Engagement: poverty pushes MaxPackDanger 1 (trash) instead of 0 during recovery, so the character actually kills things.
  This makes it less timid, not more.
- Blood Raven budget: `brAttempts` (3) attempts counted at KillBoss; the fourth within `brLevelGap` (2) levels of the last
  one is refused as a danger fault (the quest blocks, the planner levels) and renews at clvl+2. Log
  `blood raven: attempt N/M`, `blood raven: budget spent (N/M), waiting for clvl X`. 0 = unlimited. A merc is not
  considered (U).
- `recoveryOn 0` restores the old behaviour (U for the rest of the paths below).

## Correction from later research (docs/sme/blood-raven-caster.md, economy-data.md)
- The earlier "use cold on Blood Raven" advice was wrong: she resists 50% of every magic element, 0% physical. Treat her as
  skippable; the attempt budget (2-3, then clvl+2 or a Rogue merc, hire unlocks about clvl 8-9) stands and supplies are not
  sized on killing her.
- Prices used by the kit cost match economy-data.md: TP scroll 100, minor healing 30. Gold per hour by area is U, hence
  every threshold is a knob and the per-minute `recovery: gold +N in M min` line (goldPerHour) is the calibration data.

## Fixes after G1 attempt 10 (agent recoveryfix)
Evidence (V, log 20261029-202908, Soaky clvl 10, i4): 14 minutes of recovery gave gold -26 (goldPerHour -120, sellable 0) while XP
ran at about 7k/h; "loot: pack full" for a superior Cap and a Club; trips at 97 and 100 percent full sold nothing; two minor
healing potions took the last 60 gold (69 -> 9); a town portal scroll took 100 of 118.
- V (the character's own .d2s, parsed read-only at 20:45, the file is only written on Save and Exit): the pack's 40 cells held about 38
  one-cell items: roughly 12 healing, 9 mana and 2 stamina potions, 12 identify scrolls, rejuvenation and a Tome, a Key. The
  starter pickit keeps them (rules "spare supplies" and "spare scrolls": tier C, prio 0, "already in the pack"), so they are
  never sold, and nothing moves pack potions to the belt, so the belt read 0 healing and the bot bought more. (S: that no
  code belts pack potions is from grep of internal/town and internal/bot; U live.)
- Fix 1, pack: town `sellableIf` also sells spare potions (hpot, mpot, spot, apot, wpot) and identify scrolls beyond `town.spareKeep`
  (4) per item code, only when the pack is over `town.tripPackUsedPct`. Needed potions and scrolls (tier S) are not touched.
  Selling stops being "futile", so the quiet period after a full-pack trip no longer hides the next one. U live: what the
  vendors pay for a minor potion (small, likely 5-15 gold) and that the vendor accepts them (the code tries each act vendor
  in turn and reports refusals).
- Fix 2, loot while recovering (`Looter.SetRecovering`, set by `updateRecovery`): the poor line is `max(loot.poorGoldBelow, recoveryKitGold)`,
  so at 300-500 gold every gold pile and junk gear piece is still taken; the per-cell value bar becomes `loot.poorMinSell`;
  sell pickups no longer leave `sellKeepExtra` (4) extra cells free, because the sell trip is what frees the pack.
- Fix 3, potions: `resource.recoveryPotionFloor` (2) caps a recovering visit at that many healing potions in the belt, no mana or rejuvenation
  (`ChorePlan.PotionFloor`, set in `advanceTrip` while recovering with gold under `recoveryKitGold`). 0 = the old belt plan.
  The portal scroll is still bought first (the kit needs it, SME). Veteran practice (S): with 0 potions buy 1-2 minor ones
  and no more until the purse is back; do not top up mana.
- Fix 4, calibration: a once-per-minute `recovery: loot +N items, +G gold in piles, K left for lack of room` line (with
  inventoryPct) joins the `recovery: gold +N` line; K counts every skipped sighting, so it overcounts repeated looks at one item.
- Tests: internal/town/spare_test.go (pack of spares is sold down and not while it has room, potion floor),
  internal/loot/poor_test.go (recovery line, last cells), internal/bot/broke_test.go (potion floor knob).
- Not done / U: moving pack potions to the belt (needs a verified shift-click, a separate job and the real fix for "buys potions
  while carrying twelve"); the income itself (the +2..+10 gold piles at clvl 10 are what Normal drops, so 1k gold/h may simply
  be out of reach in Cave 1/Cold Plains, the loot line will tell); no live run.

## Farm area by XP per kill (agent recoveryxp)
Evidence (V-d2d, docs/d2d-fidelity.md, 7885 kills; consistent with our live logs): at clvl 12 Cold Plains, Den and Blood Moor pay
under 3 XP per kill, Stony Field about 15, Dark Wood about 35. Gold per kill (0.13 piles, 0.29 items) is area independent, so the
old pick (plan area of a character 4 levels lower, G1 attempt 17: clvl 14 in Cold Plains for 45 minutes) paid the same gold as any
safe area at about 0 XP.
- Rule (`recoveryXPArea`, internal/bot/recoveryxp.go): in Normal act 1 with a town portal, candidates are the plan steps in
  Blood Moor, Cold Plains, Stony Field, Dark Wood, Black Marsh, Den, Cave 1/2 and Burial Grounds (the old safe tier plus Dark Wood,
  Black Marsh, Burial Grounds) that pass the level picker's gates (entry level + `bot.levelMargin`, melee entry, kit level, allowed
  acts) and are not excluded (cleared, death-banned, route guard, Den without 2 potions or below its class floor). The area monster
  level (levels.txt MonLvl1Ex, V via `data.Level`; a Normal fallback table when no install data is loaded) may be at most
  clvl + `resource.recoveryMaxAreaAbove` (1). Pick = highest expected XP per kill, ties to the lower monster level.
- XP model (S): `10 * mlvl * levelPct(clvl, mlvl)`, levelPct = the penalty table of docs/sme/mechanics-progression.md 2.1/2.2 (V).
  The base of 10 per monster level is a fit that reproduces the d2d means at clvl 12 (Cold Plains about 1, Stony 17, Dark Wood 31);
  only the ranking is used. Typical result: clvl <= 9 Cold Plains/Blood Moor/Cave 1, clvl 12 Dark Wood or Black Marsh, never Cold
  Plains from clvl 11 on.
- Log: `recovery: farm area <name> (xp/kill est N at clvl M)` when the area changes.
- Knobs: `resource.recoveryXPPick` (1; 0 = the old pick), `resource.recoveryMaxAreaAbove` (1; 0..4). No pick (everything excluded or banned) falls
  back to the old pick and then to the waiting state (`noAreaMsg`, errAreaGated), so NOIDLE and the death and corpse policies are
  unchanged: only the area choice changed, nothing about how an area is entered or cleared.
- Tests: internal/bot/recoveryxp_test.go (model against the d2d means, clvl 14 never Cold Plains, clvl 6 early areas, exclusions, all
  banned, the cap knob).
- U: no live run. Black Marsh and Dark Wood are farther from town than Cold Plains and unlike it have no town-side shortcut for a
  walk home; a character without a portal still uses only the first field. The fallback monster levels for Den, Caves and Burial
  Grounds are S (only used without install data).

## Not verified / limits
- U: no live run. The 1.5-4k gold/h income (S, SME unsure) is untested; the time cap and cooldown bound the cost if it is lower.
- U: the Blood Raven refusal rides the executor's danger strikes; how long the quest then stays blocked in a live run is unseen.
- U: the budget counts the campaign KillBoss path only, not the RunBoss path of plain reach_level goals.
- Tests: internal/bot/broke_test.go (fakes), internal/resource/recovery_test.go.

## Fixes after G1 attempt 11 (agent DEATHCHAIN)
Timeline (V, soak24 log 20261029-211013): recovery ended by the 45-min time cap at 21:55 (gold 86); the quest walk to the Barracks died 5 times 22:02-22:09 (Steel Grumble at Monastery Gate), gold 0 after each death; the 15-min recovery cooldown (from 21:55) held until 22:10 and recovery restarted at 22:11.
- Correction (V, "shop: potion plan" lines): the belt still held 2 HP + 2 MP potions, so SOFTGATE's "no potions and no gold" did not fire; gear risk only reached medium after the 5th death (deaths +3). The "potions":0 of the minute line is the drink counter, not the belt.
- Force start: `RecoveryInput.Force` (3 deaths in 30 min, `resource.RecoveryForceDeaths/RecoveryForceMin`, any area) skips only the cooldown; gold must still be under the kit price and the kit unaffordable. Deaths 3+ on a quest walk with an empty purse now start recovery at once.
- A recovery slice whose planner finds no eligible area (bans after deaths) returns an `errAreaGated`-wrapped error: `do()` counts no failure, never climbs the ladder or restarts the game, goes to town and waits ("Waiting: <area> banned for N more min", or "Waiting: no eligible farm area" without a ban). Before, 5 such failures per rung walked rung 1 to 4 (game restart) in 16 s.
- Unverified: live behaviour of the wait (offline tests with fakes only: TestRecoveryAllBannedWaitsWithoutLadder, TestDeathStreakStartsRecoveryInsideCooldown, TestRecoveryForceBypassesCooldown). Not done: a quest-walk check that avoids a boss area on repeated deaths (the area ban does that after SpiralDeaths; here the deaths were in alternating areas 7/26, so inArea never reached 4).

## Blood Raven for the free Rogue (agent BRKILL, plan-v2 10.1)
Decision (user): take the free Rogue, so the bot kills Blood Raven early, then claims the reward at Kashya. All offline with fakes; nothing here ran live (U).
- Gate (V in tests, numbers U: SME judgment, docs/sme/blood-raven-caster.md section 7): `task.BRLine` logs `blood raven: gate clvl=N kit=ok|short(why)`. Level floor: Sorceress/Druid/Amazon `boss.brCasterLevel` 9, Necromancer and melee `boss.brMeleeLevel` 8, raised to the wait level of a failed attempt. Kit: `boss.brPotions` 4 healing (belt and pack), `boss.brManaPotions` 2 (casters, belt only), a town portal with `boss.brTPCharges` 2 charges or scrolls (the bot cannot read charges or scroll counts yet, `World.TPKnown` false, so any portal counts: U), gold `boss.brGold` 250 for the refill of a retreat trip.
- Planning blocks (no new mechanism): the planner's Blood Raven entry floor is `brFloor`, so a low level is a level step; a short kit is a `KindResource` fault with a chores insert on the travel step; short gold alone is a `KindDanger` fault (a chores trip buys nothing, recovery and leveling earn the gold). The same gate runs again inside `KillBoss` (kit used up on the way): the bot goes to town and the planner restocks.
- Attempts (`brAttempt`/`brEnd` in internal/bot/broke.go): `resource.brAttempts` 3 kept; every attempt needs the gate; a failed attempt (retreat by town portal, chicken, death, abort) sets the wait level clvl+1, after the second failure clvl+2 (`World.BRWaitLevel`, so the planner levels in Cold Plains/Stony Field through the normal level picker); never a same-level retry; a cancelled run costs no wait and refunds the attempt. After 3 attempts the old `brLevelGap` rule still applies. Logs: `blood raven: attempt N/M start` and `end(reason)` with `secs`, `goldBefore`, `goldAfter`, `waitLevel`.
- Her HP: `boss run` log line carries `bossMaxLife` (stat 7 of the unit when first seen). U: the client may scale monster life (docs/sme says 113 vs 323); read it before trusting it.
- Tactic (internal/boss, `bloodRaven()` script, best effort U): `OuterClear` sweeps radius 28 around the arrival point before she is looked for; `RetreatPct` 40 walks back to the arrival point (the fence gap we came in by: no area geometry is available offline, so the entry is the only chokepoint known) and drinks; the `boss.brMaxRetreats`-th (2) retreat of one run is a town portal and ends the attempt (`ErrEscaped`). The cast-first loop is untouched. Static Field is not opened on her (`combat.staticCastsBR` 0; she resists everything by half, U whether it would pay). Charged Bolt: the existing skill pick already ranks it for the Sorceress; no change (U).
- Reward: after the kill the campaign goes home and talks to Kashya (existing reward step), then the optional entry `sisters_burial_merc` (`task.Campaign.WithRogueClaim`, applied by the bot to every campaign) runs `claim free Rogue` when the Burial quest word is at least Cleared, the act is 1, Normal, and no merc is employed (unknown proceeds). It calls `task.MercClaimer.ClaimFreeMerc`; the real bot has `TaskDeps.ClaimMerc` nil, so it logs `merc: claim step skipped (no backend)` and finishes. The step is Optional with proxy evidence: unsupported, failed or unverified never blocks. To wire the backend set `TaskDeps.ClaimMerc` (return `task.ErrClaimUnsupported` to skip).
- Soft rule: `mercHirer` (cmd/d2bot/acts.go) refuses act 1 outright (`merc: no paid Kashya hire`); `mercRows` is still empty and `HirePolicy` is Act 2 only, so no paid Kashya path exists. `task.KashyaHireAllowed` states the rule for a future backend.
- Not done: the boss script floor stays 6 (RunBoss path of plain boss goals ignores the class gate); mana potions in the pack are not counted; Blood Raven's raised-adds hazards still clear zombies although the SME advises ignoring them once at her (not redesigned).
