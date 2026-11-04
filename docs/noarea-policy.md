# noarea: why the leveling picker found nothing at clvl 15-16 (Gate 2 seed 13) and what to decide

Tags: V = verified (log line or code read, say which), S = secondary (read from code, not run), U = unverified.
Every "Option" below is a POLICY DECISION FOR THE USER. Policy cards are paused; this card changes no code.
Evidence: the run logs of Gate 2 seeds 11, 12, 13 on the worker host (`bot-0.log`; line numbers below are log lines).
Follows `docs/gate2cursor.md` section B, which it corrects in one point (see 2.4).

## 1. The picker rules (S, code read)

Call chain: `LevelTo` (internal/bot/task.go:872) -> `target` -> `target0` (internal/bot/goal.go:807) -> `Plan.Pick`
(internal/bot/plan.go:390). If `Pick` returns nothing, `LevelTo` tries a fresh game (`freshGameToLevel`, task.go:911) and
only then returns `no eligible reachable leveling area for level N` (task.go:866 const, :915 use).

Level windows (data: internal/plandata/progression-normal.json; `Step.EnterLvl`/`LeaveLvl`, `LevelMargin` and `LeaveSlack`
are both 0 by default, internal/env/tuning_bot.go:73):
- A step is eligible when it is in an allowed act, not a town, not in `Exclude`, kit rules pass (`KitLvl`, `FallbackKit`,
  plan.go:403-413) and `enter + margin <= clvl` (plan.go:422).
- `best` = the last eligible step in progression order whose `clvl <= leave + slack` (plan.go:426). If none, `last` = the last
  entered step even though the character is past its window (plan.go:433).
- Floor rule (plan.go:434, `worthFloor = 4`, goal.go:804): that past-window `last` is REFUSED when
  `clvl - last.leave > 4` and `windowExcluded` is true (an area in the CLEARED set, same act, still has the level inside its
  window; plan.go:488-502). The `pickByXP` fallback has the same rule (plan.go:478). Result: Pick returns "nothing" and the
  caller starts a fresh game instead of farming for next to no XP.
- Normal Act 1 windows at clvl 15-16 (progression-normal.json): Cave 2 leaves 15, Crypt 17, Mausoleum 18, Outer Cloister 16,
  Jail 1 16, Catacombs 2/3/4 17/18/18, Tamoe 14, Black Marsh 14, Burial Grounds and Cave 1 10, Stony Field 12. Act 2 starts at
  Lut Gholein enter 18, and `Acts` defaults to `[1]` (internal/bot/config.go:101; `allowedActs` goal.go:611 adds the current act).
- No waypoint requirement exists in `Pick`. Waypoints matter only in the walk (`travelTo`); a character without town portal and
  waypoint walks home through the neighbour area (log line 41490), which is where nav faults come from.

Exclusions (`ex`, goal.go:862-878):
- areas CLEARED in the current game (`l.cleared`);
- every area with a live ban (`r.avoid`, goal.go:866-871); a ban is a hard exclude, the window does not matter;
- guarded areas (`guardedAreas`, routeguard.go:125): Monastery chain needs enough potions (log: `0 mana potions < 2`,
  `3 healing potions < 4`) and a town portal; Tower Cellar chain needs a merc or clvl 16; Den waits for its floor. (U: the exact area sets
  are `task.RouteGuarded`/`TowerGuarded`, applied to areas 7..37, routeguard.go:137; their contents were not read.)

Bans (`skipAreaLocked`, areaskip.go:31; defaults tuning_bot.go:74):
| reason | length | producer |
|---|---|---|
| death-streak | `DangerBanMin` 20 min | death.go:161, `banLength` death.go:207 |
| repeat-site | `RepeatSiteBanMin` 30 min (escalates) | deathrule.go:117 |
| unreachable (walk failed) | `NavSkipSec` 150 s, doubling per repeat to x8 (150/300/600/1200) | task.go:925, `targetFailSkip` areaskip.go:48 |
| unreachable (no way to the picked target) | 150 s | task.go:893 |
| nav-stuck (failsafe skip) | 150 s | loop.go:641 |
| route (hop banned) | 4 x NavSkip = 600 s | loop.go:808 |
| border-loop | `borderLoopSkip` | loop.go:1173 |
V (log): 150 s `unreachable` at line 40953, 300 s at 41188, 600 s at 6514, 150 s `nav-stuck` at 41324.

Relief valves (all S):
- `idleBreak` (areaskip.go:93): while waiting in town, after `AreaIdleSec` = 60 s (starter/death-only bans: `StarterBanMin`
  = 3 min) the ban of the "least bad" area is lifted: faults before death bans, shortest remaining first.
- `routeBan` shortens a starter-field ban to 3 min while standing in town (goal.go:691).
- `target0` yields the Den floor when nothing else fits (goal.go:883).

## 2. Seed 13, clvl 15 at 23:35-23:36 EST 2026-11-03 (V, log)

State at the block (lines 39934-41596): clvl 15, no merc, tome of town portal empty from line 41434, caster.
Guards: `blocked monastery (0 mana potions < 2)` / later `(no town portal)` and `blocked tower (no merc, clvl 15 < 16)`
(lines 39934, 40291, 41055, 41434). So Outer Cloister, Jail 1, Catacombs and the tower chain were EXCLUDED by guard.
That left Act 1 open field and cave areas. In order:

| area (window) | outcome at clvl 15 | log line |
|---|---|---|
| Mausoleum (11-18) | picked 23:34:11 (earlier ban just expired); no later pick in the block | 39942 |
| Cave 2 (10-15) | picked; walk failed `trigger area of an unwanted exit`, banned `unreachable` 150 s | 40953 |
| Crypt (10-17) | picked 23:35:10; failsafe skipped it `nav-stuck` 150 s (no town portal, could not walk home) | 41324 |
| Stony Field (leave 12) | past window, picked as last eligible, banned `unreachable` 150 s | 40962 |
| Burial Grounds (10) | picked (Blood Raven); `nav-stuck` 150 s after the failed walk home | 41504 |
| Cave 1 (10), Den (6), Cold Plains (10), Blood Moor (6) | past window, each picked then `unreachable` 150 s | 41348-41366 |

Why Pick returned nothing at 23:36:00.76 (line 41596): the in-window areas Cave 2 (banned 300 s at 23:35:36, line 41188) and
Crypt (banned `nav-stuck` 23:35:48) were out, Mausoleum was not picked again (S: it was probably cleared or banned; the log
block does not say), Outer Cloister/Jail/Catacombs/Tower Cellar were guarded, and Burial Grounds (the only other entered area not
yet banned, leaves 10, 5 below) had just been banned `nav-stuck` 150 s (line 41504). Everything else (Cold Plains, Cave 1, Den,
Blood Moor, Stony Field) was banned `unreachable` at 23:35:48 or 23:35:36. So the empty result is the ban set, not the floor
rule; S: the floor rule (plan.go:434) would also have refused Burial Grounds, since Cave 2 or Crypt (a CLEARED step) still
covered clvl 15. LevelTo's fresh-game restart did not fire for it (U: why; it fired at 23:35:50, line 41405, just before).
Dashboard text: `Waiting: Burial Grounds banned for 3 more min` (line 41598).
Root cause chain: the bans are almost all `unreachable`/`nav-stuck`, created by the same fault: the walk from Cave 2 / Burial
Grounds toward Cold Plains is refused by nav (`the click would cross the trigger area of an unwanted exit`, lines 40954, 41494)
and the character has no town portal (empty tome), so it cannot shortcut. The picker saw bans that were nav faults, not danger.

It self-cleared in 2 slices: 0.15 s later `idleBreak` lifted Cave 1 / Cold Plains (lines 41606-41607) and the plan picked Cave 1
(`past every window, leaves at 10`). The permanent form of the block is later: 20743 `no eligible reachable` lines in the whole
log, first line 41596 and a flood from 01:06 on (line 86940, `area gate refused the walk`, level 16), when gold was 26 and no
tome (gate2cursor A, the cursor fault). U: this doc does not re-derive that second phase.

### 2.4 Correction to gate2cursor
gate2cursor says the Act 1 windows leave at clvl 10-12. In the plan data several Act 1 windows run to 16-18 (Crypt 17,
Mausoleum 18, Catacombs 18). At clvl 15-16 those WERE in window; they were unavailable because of guards (no merc, mana
potions, no portal) and bans, not because the character outgrew them.

## 3. Seeds 11 and 12 at the same clvl (V, log)

- Seed 12 (clvl 15-16, 0 `no eligible` lines): a merc was alive (`route guard: forgotten tower floor=14 (merc=alive)`, line
  112639), so the tower guard opened; the picker alternated Jail 1, Catacombs 2/3/4, Mausoleum, Outer Cloister, Forgotten Tower
  (lines 108117-120599; all "in window"). When the guards closed again (`1 mana potions < 2`, line 114626) other windows were
  still open. It also spent a long stretch at clvl 15 in `Stony Field (past every window)` (about 20 `level plan` lines
  01:03-03:24); that is the picker working as designed with the floor rule not triggered (nothing CLEARED covered the level).
- Seed 11: 12 `no eligible` lines, all at clvl 6 at 20:04 (line 5787), none at clvl 14-15; the picker bounced Cave 2 / Crypt /
  Cold Plains / Cave 1 and reached Jail 1 (in window 13-16, line 66492). Same shape as seed 13, but nav kept working and no
  tome emergency developed (U for the reason; the log was only sampled).
- Difference in one line: seeds 12 and 11 had open in-window areas (merc, potions, nav success); seed 13 had them guarded
  or nav-banned all at once and then lost its town portal.

## 4. Options (all POLICY DECISIONS FOR THE USER)

1. Do nothing. The block self-clears in seconds (idleBreak, 60 s). Cost: Gate 2 time is spent in town and in fresh games, as
   seed 13 shows. The permanent form came from the cursor fault (gate2cursor A), already fixed.
2. Do not count nav faults as bans for the picker. Treat `unreachable`/`nav-stuck` (150 s+) as a soft penalty (a lower
   rank) instead of a hard exclude. Pro: the cause here was the nav fault, not danger. Con: repeated failing walks (the doubling
   exists to stop exactly that, G1 attempt 14/17) come back.
3. Shorten / cap the fault bans (cap the doubling, e.g. at 300 s) or run `idleBreak` after 15 s for fault-only bans. Pro: smaller
   stall. Con: more attempts on a broken exit; does not fix the nav fault.
4. Low-ban fallback: when Pick returns nothing and only faults (not death bans) exclude areas, lift the shortest fault ban
   at once (idleBreak already does this after 60 s). Same as 3 with no new knob.
5. Widen the windows (`leave_clvl` of Cave 2 / Burial Grounds / Cave 1 plus 2-3) or relax `worthFloor` (4 -> 6). Pro: the
   past-window fallback stays alive instead of returning nothing. Con: XP per kill falls with the level gap (LIVE: Den at
   clvl 12 gave 8 XP in 12 min, plan.go:370); data change, shifts every build.
6. Change the guard: open the Tower Cellar chain for clvl 15, or allow mana-potion shortfall when healing potions are fine. Pro: more
   in-window areas. Con: deaths (the guard exists from play-safety evidence, docs/sme/play-safety.md).
7. Move on instead of waiting: allow Act 2 / Nightmare when Act 1 is exhausted (`Acts` default `[1]`, config.go:101;
   Lut Gholein enter 18). Pro: the real fix for being clvl 16 in Normal Act 1 with no merc. Con: needs quest steps the bot
   cannot do yet (config.go:24 comment); a larger card.

Recommendation: option 1 now (no change), then 2 or 4 only if a rerun shows the block outliving idleBreak with a fault-only ban
set. The fix with real effect is the nav exit fault and the empty tome (A), not the picker. Do not widen windows (5): the picker
did what its XP rule asks. 7 is the long-term answer and is the user's call.

## Rerun check (live, not run here)
Soak seed 13, same settings. Pass: every `no eligible reachable` line is followed within 2 minutes by a `level plan` line; list
the `area skipped` reasons around each; XP rises past clvl 16.

## Shared-file lines for the merger
See openspec/changes/noarea/merge-lines.txt.
