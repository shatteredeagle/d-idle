# Death spiral guards (G1 attempt 1, 2026-10-29)

Evidence (V, bot log i4 12:32 to 12:44): Soaky clvl 8, gold 0, no potions, no weapon, no body armor; the walk toward the
Barracks went through Underground Passage Level 1 (mlvl 5, so clvl-2 = 6 would NOT have refused it) and died four times at
the same spot. Why the old gates did not stop it: the kit gate only faulted on potions (chores blocked, then the walk went
on), corpse value counted only equipped items on the newest body (0 when already naked), the spiral ban needed 3 deaths.

What is in place
- `resource.GearGate` (pure): from clvl 5, missing weapon or body armor, or no potions and no gold, gates the character.
  Gated: only towns and the safe farm area (first field of the act, `Runner.safeFarmArea`) are allowed. `CanRebuy` when
  naked and gold >= 150 (U) sends it to town for the poverty rebuy (`rebuyDue`, asks again at +50% gold).
- Task layer: `World.Gated/SafeArea/Banned`; `StepTravel/StepClear/StepKill` carry `fieldReady`, a danger fault, so the
  executor blocks the quest and the planner levels instead (`task/gate.go`).
- Bot layer, the backstop: `travelTo` checks every hop of nav's route (`routeAllowed`), walks to town are always open;
  `target()` returns the safe area for a gated leveler (or nothing when it is banned); `gateField` redirects trips.
- Ban: 2 deaths in one area within `SpiralMinutes` (15) or 3 within an hour, for `DangerBanSec` (default 60 min). A
  `SpiralDeaths` above 3 still turns the breaker off. No alternative-route search: nav.Route has no avoid list (U).
- Corpses: a naked character whose newest body is worthless goes for the newest unretrieved body worth the threshold
  (`deathLog.gearCorpse`, max 2 h old), unless that area is banned, or its monster level is above clvl-2, or retries are
  used up. Whether an older body survives later deaths or a restart is U. The decision log line now carries area, x, y,
  value, naked and whether the target is an older body.
- Corpse route (LIVE G1 attempt 12, 2026-10-29, V from the log): the town portal in town stands where the LAST scroll was read
  (Cold Plains), not at the corpse (Monastery Gate, area 26). `BackToField` entered it, landed in 3, then called the
  single-hop `ExitTo(26)` which fails in nav ("no exit to that level: level 26 from 3", nav only knows neighbours).
  Fix: `BackToField` returns `town.ErrWrongLevel` as soon as the portal lands in another field level (no single-hop try),
  and `retrieveCorpse` then runs `travelTo(area)` (route, activated waypoints, exit by exit with nav exploring), twice.
  Offline-tested with fakes only; the real walk is U until a live death.
- Gold: death took all carried gold (1125 to 0). `town.stashGoldBase`/`stashGoldPerLvl` (300 + 50/lvl) keep the pack
  light; the stash chore runs after the shop in every full chore plan, including the emergency town portal trip.
- Town: `Rebuy` buys weapon and body armor before potions.

Not done / U: gate levels (clvl 5, 150 gold) are judgment; Normal fresh characters have no body armor and are exempt below
clvl 5 (U: classic starting kits). Needs a live check on a naked throwaway character.

## Area ban is a planning block, not a fault (AREABAN, LIVE Tcont 2026-10-29)

V (log): Tcont (clvl 3, town) had Blood Moor banned, the only way out of the Rogue Encampment. Every target's walk was
refused with `area not allowed now: Blood Moor (2): banned after repeated deaths`; `do` counted five action failures,
the failsafe ladder ran Esc, town, re-entry, restart and ended in `failsafe: paused` (ERROR). Fixes (internal/bot):

- `do` treats `errAreaGated` as a deliberate refusal: failure count reset, no ladder, no restart, no error. It shows
  the wait text and sleeps 5 s (`gatedWait`).
- `target()` skips a target whose route (`Mover.Route`) crosses a banned area (`routeBan`) by avoiding that target until
  the blocking ban ends, then picks the next best. When none is left it returns no target.
- With no target, `noTarget` and `planningBlock` show `Waiting: <area> banned for N more min` in town (state in_town),
  never `setError`, never counted toward the planning-block pause.
- A gate refusal also skips the refused target for 15 min (`skipTarget`, the gated safe area is exempt), otherwise a
  planner choice the gear gate refuses (hysteresis hold) would be retried forever.
- Bans expire (`banned` logs `area ban lifted`). When the planner finds the ban of a first field of an act (the way out
  of town, `isStarter`) in its way while the character stands in town, what is left is cut to `starterBanMax` (10 min)
  instead of up to `dangerBanSec` (1 h), so a character that cannot leave town cannot deadlock. V (code read): `r.avoid` is in memory only, it is not loaded from the death log at start; the dashboard
  already lists active bans in the death report (`avoidedAreas`).
- Not covered (U): the task layer's step `Ready` gate (`fieldReady`) still checks only the target area, not the route;
  a route refusal there arrives as a danger fault from `Travel`, which `planningFault` already handles.

Tests: `internal/bot/area_ban_test.go` (fakes only). Not run live.

## Death tolerance (SOFTGATE, user rule: "dying is normal, sometimes it takes a few deaths")

Changes (tests `internal/bot/death_tolerance_test.go`):
- A death alone never raises gear risk and never bans. `bot.spiralDeaths` default 3 -> 4: the area is avoided on the 4th death in the window (`bot.spiralMinutes` 15), on the 3rd when the area was clearly unfavorable (monster level >= clvl+3) or the character has no potions and no gold. Slow rule: 1.5 x `spiralDeaths` deaths in one area within an hour (was 3). The old "2 deaths in 15 min" rule is gone. U: the numbers are judgment; EXP: a boss goal (Andariel, Duriel) that needs 2-3 tries now keeps retrying.
- Ban length: `bot.dangerBanMin` default 20 (was 60 via `dangerBanSec`; `Config.DangerBanSec > 0` still overrides). Starter-area cap `bot.starterBanMin` default 3 (was a 10-minute constant).
- Response ladder: deaths 1-3 recover (respawn, restock, corpse retry as before) and retry the same goal; the 4th (3rd if unfavorable) bans the area for 20 min so the planner picks an easier one; 8 deaths per hour still pauses (`bot.pauseDeathsPerHour`).
- Corpse policy (docs/sme/play-safety.md section 6): unchanged; the retrieval check `recent >= spiralDeaths-1` follows the knob (now 3 recent deaths abandon the retrieval). The "oldest corpse only" open item (audit X1) is untouched here.
- G1 soak criterion "deaths <= 2 in 2 h" is a harness metric and was not touched (superseded: the death rule of plan-v2 s10.2 replaced it).

## G1 attempt 11: deaths in alternating areas (DEATHCHAIN)
Five deaths in 7 minutes were split over Tamoe Highland (7) and Monastery Gate (26), so the per-area spiral count (4) was reached only late and bans covered Cold Plains/Stony Field/Blood Moor, the recovery farm areas; the recovery slice then had no area ("no eligible reachable leveling area") and the failure ladder restarted the game. Now that result is a planning block (waiting in town, ETA from the ban) and a streak of 3 deaths in 30 min with gold under the kit price starts recovery despite its cooldown. See docs/recovery.md.

## NOIDLE: never idle in town on area bans (G1 attempt 14)
V (read-only log, soak24 attempt 14, Soaky clvl 12, 0 deaths): a nav-stuck failsafe skip (`stuck at {5459 4549}`, ladder rung 1) banned Cold Plains for 15 min, more skips followed, and from 00:38 to 00:56 the bot sat in town on "Waiting: ... banned for N more min", XP flat, until all bans ended together. The bans were bot faults, not danger.
Code (`internal/bot/areaskip.go`, tests `areaskip_test.go`, fakes only, U live):
- Every producer goes through `skipArea` and logs `area skipped ... seconds=N reason=nav-stuck|unreachable|gate|death-streak|repeat-site` (route skips inherit the reason of the banned area on the route).
- Nav faults (failsafe rung 1, unreachable target, no exit) skip for `bot.navSkipSec` (default 150 s, was a constant 15 min). Death-streak bans keep `bot.dangerBanMin` (20 min).
- `idleBreak` runs wherever the bot would wait in town for bans (`noTarget`, `gatedWait`, `planningBlock`). After `bot.areaIdleSec` (60 s) of one continuous wait, it lifts the ban of the least bad area and logs `area wait: all areas banned/skipped, farming <area> anyway (remaining Ns)`; the planner then farms it with normal combat. Order: bot-fault skips before death bans; among death bans the starter field first; then the shortest remaining. If only death bans exist the cap is max(areaIdleSec, `bot.starterBanMin`) (3 min), so a death streak still waits a little but never stands in town for the full ban. If the planner still finds nothing, the next area is lifted about 10 s later.
- The stall detector (`stuckCheck`, "no progress for 600s (state in_town)" while Waiting) reaches `escalate`, which first calls `idleBreak(true)` when the character is in town in state in_town: it lifts a ban at once and rearms instead of climbing the ladder.
Unverified: whether the lifted area is eligible for the character level (if not the next lift follows in ~10 s, and a final fallback is the existing planning block); behaviour live.

## Repeat-site action (DEATHRULE-ACTION, plan-v2 10.2, user approved)
Bot side of the approved death rule; the verdict side (FAIL-1..3) is in cmd/d2soak. Code `internal/bot/deathrule.go`, tests `deathrule_test.go` (fakes only; nothing here is verified live, U).
- Repeat site = 2 deaths in one area within `bot.repeatSiteWindowMin` (30), OR a death within `bot.repeatSiteGapSec` (180 s) of the previous death (any area). Town deaths never count. `bot.repeatSiteBanMin` (30, 0 = off) bans the area with reason `repeat-site` (`area skipped ... reason=repeat-site`). It runs after the spiral breaker, so a spiral and a repeat death end with the longer ban. Because the window rule fires on the second death, the old tolerance ("three deaths on the same goal never ban") now holds only with `bot.repeatSiteBanMin` = 0; the spiral tests pin that.
- Log line (the harness greps `death: repeat-site ban`): `death: repeat-site ban area=<id> name=<n> deaths=<k> window=<m>min planner dump=<path>`, level WARN.
- Planner dump: `plannerdump-<UTC ts>.json` next to `deaths.json` (the instance data dir; skipped, path `none`, when no `DeathLogPath`). Contents: trigger, the death record, the active goal (Params, Progress, Note), runner state/activity/message, planner step area and level, kit (belt/pack potions, portals, gold), every plan-step area and every banned area with remaining ban, reason and the `areaAllowed` refusal text, the last 5 deaths, and the last 40 decisions (state/activity changes and area skips, ring in `Runner.decisions`). U: the quest step is not a field; the goal and the activity name it.
- NOIDLE interplay: `repeat-site` is not `death-streak`, so `idleBreak` counts it like a bot fault. If the banned area is the only farm area the bot waits at most `bot.areaIdleSec` (60 s) in town, then the least-bad lift farms it with normal combat; it never stands in town for 30 min (test `TestRepeatSiteBanOnOnlyAreaStillFarmsAfterWait`). Faults are lifted before death bans, so with a death-streak ban elsewhere the repeat-site area is lifted first.
- Corpse: the repeat-site ban also blocks the corpse run into that area (existing "area avoided" case in `retrievalDecision`).
- `bot.corpseMinValue` (50 gold equivalent, 0 = off): below it the in-field corpse run is skipped before any other check and logged `corpse: skipped (value N < M)`. It is a floor under the existing `bot.corpseValueMin` (1000), which already abandons everything below 1000, so with defaults the new floor only changes the log text; it matters once `corpseValueMin` is lowered. The value is the existing `CorpseValue` estimate on the death line.
- Potion bug: `Autopsy.Bug` (beltHP >= 1 and no potion drunk in the last 60 s, M0B) is now in the WARN `death autopsy` log line (`bug=true beltHP=N`) and the card summary gets `BUG: N healing potions in the belt and none drunk in the last minute`.
- Retreat at 50 % life via portal: NOT added. The hook already exists: `combat.Decide` returns TownPortal when life < `combat.drinkLifePct` (55) and there is no healing potion and no rejuvenation (any hostile count, needs a portal scroll), or when hurt, quiet and short of potions. Adding a stricter 2+-hostiles rule would only narrow it; a real 50 % retreat with potions left means a combat redesign (kiting vs leaving), out of scope.

## Gold on death and the gold sweep (GOLDSWEEP)
Source caveat: S (secondary), docs/plan/plan-v2.md 10.6 and docs/sme/d2decomped-findings.md (d2decomped notes). Our own logs only show "gold 0 after the death" (V, G1 attempt 12: 1125 to 0) and in-field corpse runs that failed, so the pile was never swept. A live probe (U) must confirm: die with a known purse, read `GoldNextMinute`, look at `Ground` at the death spot.
- Mechanics (S): loss is min(clvl,20) percent, never below clvl x 500 in total; the rest of the purse drops as a ground pile at the death spot; the corpse holds only worn items (and the cursor item); the stash is safe. Open point (U): whether the percentage applies to purse alone or purse plus stash. The code only needs the pile estimate `carried x (100 - min(clvl,20)) / 100`, an upper-ish guess (`expectedPile`, `internal/bot/goldsweep.go`).
- Value decision: `retrievalDecision` now uses `corpseWorth` = `CorpseValue` + expected pile (carried gold from `autopsy.goldCarried`) against `bot.corpseMinValue` and `bot.corpseValueMin`, so a big purse justifies the trip with cheap gear. A fresh death only; an older gear body does not add its long gone pile. Log text "corpse worth about N gold" shows the sum.
- Sweep (`sweepDeathGold`, end of `finishRecovery`): after the corpse run reached the spot, the leftover piles within 40 subtiles are walked onto (a corpse run already walks onto the nearest one, V live) and the looter runs once. When no corpse run happens, `sweepDecision` allows a trip back to the death spot from town unless: hardcore, death in town, expected pile < 100, boss kill, a second death on the way, the area is banned (spiral or repeat-site: never back into a just-banned area), monster level >= clvl+3, or SpiralDeaths-1 recent deaths in the area. A failed or older-body corpse run does not sweep. It reuses `gotoDeathArea` (the portal / travelTo logic of `retrieveCorpse`).
- Log: `death: gold sweep area=<id> picked=+N gold (carried A -> B)` (INFO) or `death: gold sweep skipped why=...`. `autopsy.goldRecovered` and a follow-up jsonl line, see docs/deathautopsy.md.
- Banking (CORPSENAV rule `town.stashGoldBase` 300 + `stashGoldPerLvl` 50 per level): GAP found and fixed (V by reading code): `cmd/d2bot/adapters.go` set `Gold: p.Stash`, so chores without the stash flag (heal-only trips, which are the trips that return to the field) never banked gold; it is now always on and the town chore only opens the stash when carried gold exceeds the keep line. Full chores (after death, farm run, forced) already banked. After a sweep that leaves the purse more than max(keep, 500) above the keep line, a `fullChores("bank swept gold")` trip is queued so the big purse does not stay in the field. U: not run live.
- Tests (fakes): `internal/bot/goldsweep_test.go`.

## Repeat-site escalation (agent ROUTEGUARD, 2026-10-30; U live)
- `bot.repeatSiteEscalate` (default 1): the repeat-site ban covers the whole AREA and grows. First ban in an area `bot.repeatSiteBanMin` (30), the second within 12 h x4 (120 min), the third and later x12 (cap 6 h). The count is read from the death log (`DeathRecord.repeatBanMin`, written when a ban is applied), so it survives a restart. 0 keeps every ban at the base length. Test `TestRepeatSiteEscalates`.
- Why: G1 attempts 11, 12 and 15 died on the same Tamoe Highland / Monastery Gate border spot within hours; a 30 min ban only delayed the next walk into it.
- The corpse run respects it: `retrievalDecision` refuses a corpse in an area banned (`banned`), in an area the route guard closes, or after a crowd death (hostiles near >= `combat.crowdN`) in an area that already took another death within 12 h. `gotoDeathArea` no longer walks on to the corpse area after a wrong-level portal landing when `areaAllowed` refuses that area (the 02:30 run landed in Monastery Gate, walked back into the same crowd and aborted in 2 s).
- Tower guard (TOWERGUARD, U live): the same rule covers areas 20-25. A corpse there is abandoned with `corpse: skipped (guarded area N)` when no merc and clvl below `route.towerMinLevel` or the kit is thin; `gotoDeathArea` refuses a guarded corpse area up front for the waypoint-and-walk path (WPUSE, G1 attempt 17 went to Tower Cellar 5 that way) and the portal path alike (test `TestCorpseInTowerAbandonedAndWaypointRunRefused`). A character that ends up in a guarded area anyway leaves first (docs/plan/plan-v2.md 11.2).
