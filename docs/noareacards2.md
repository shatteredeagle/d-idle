# noareacards2: the noarea card set rewritten from the SME review (design text only, no Go)

Status: draft for the coordinator and the user. Supersedes `docs/noareacards.md`. No code was changed and nothing was run live. No OpenSpec card
(retired for new work, user 2026-11-04). Tags: V = verified (code read in this worktree with file:line, or a log number quoted from
`docs/noareameasure.md`), S = secondary (the SME review text, not re-derived), U = unverified. The SME review is input, not proof: every
claim below says whether the code agrees. `docs/noareameasure.md` lives on branch `agent/noareameasure` (not yet on main); line numbers
for it below are lines of that file as committed there.

## Verification of the SME claims against the code

| SME claim | result |
|---|---|
| `FarmAreas` = {2,3,4,8,9,13,17,18} | V internal/merc/farm.go:10 |
| the override is a HARD EXCLUDE of every other area | V routeguard.go:124-131 (`for a := 2; a <= 132` sets `ex[a]` unless in `FarmAreas`; only Normal, Act 1 (`Act == 0`), only while `MercGoldFarm(s)`) |
| cap 16 -> 18, reserve 150 -> 100 percent merged | V internal/env/tuning_merc.go:23 (`KitReservePct: 100, GoldFarmMaxLevel: 18`); the knob desc at tuning_merc.go:~52 still lists the 8 areas by name and must change with the data |
| fallback point goal.go:880 | V partly: the `pick` closure is goal.go:879-881 (the line 880 is the `r.d.Plan.Pick(...)` call); the first call is `step, ok := pick()` just below. The Den-floor retry directly after it already is a "re-pick after deleting one exclude" and is the pattern to copy. |
| revive runs after shopping (mercwire.go:618-627) | could not verify here (not re-read; the farm test is cmd/d2bot/mercwire_test.go:491, GoldFarm at mercwire.go:321-341) |
| `merc.GoldFarm` is a pure predicate | V farm.go:18-19: `GoldFarmMaxLevel > 0 && mercMissing && gold < need && clvl < cap`. It has no time input: a budget needs state in the wiring (mercwire.go:321), not in this function. |
| farm test pins DW out | V farm_test.go:86 asserts `FarmAreas[5]` is false; it must change with card 1 |
| Dark Wood / Black Marsh plan windows | V progression-normal.json: Dark Wood area 5 enter 9 leave 13; Black Marsh area 6 enter 10 leave 14; Cave 2 (13) 10-15; Crypt (18) 10-17; Burial Grounds (17) 6-10 |
| `xpLevelPct` bands | V recoveryxp.go:35-51: d >= 10 -> 1.95; d <= -10 -> 5.08. The SME is right and the measure doc is wrong (section 6 below) |
| 0 deaths in 315 min of BM, min HP 68-80 percent | S, SME computation from the minute logs; could not re-derive without the logs here |
| the 1,680 revive, kit 310, bars 2145/1990 | S (formula in mercwire); the kit constant is `KitRefill` farm.go:5 (V: 100 + (7)*30 uses defaults, U for the 7) |

## Card 1: farmfallback (NEW, first)

### Problem
While the merc is missing and gold is short, `guardedAreas` (routeguard.go:125-133, V) adds every area outside `FarmAreas` to the exclude
map; `target0` then calls `Plan.Pick` once (goal.go:879-881, V). If every farm area is cleared, banned (`r.avoid`, goal.go:862-868) or past
its window, `Pick` returns nothing and the bot idles in town even though Dark Wood, Black Marsh and the chain are open. Seed 13 (V, measure
doc section 2.1): 8 of 8 farm areas under a 150-300 s ban at 23:36:00, `no eligible reachable`, 616 virtual minutes of farm window
(23:35:06-00:31:44, S), gross gold in about 10.5k, out about 11.1k, total 2250 -> about 780 (S: net zero, never reached the bar). The farm set
is also the worst gold set in Act 1 (S, SME carried-gold pickup per virtual hour at clvl >= 13: Black Marsh 1.87k, Dark Wood 1.53k, Cave 2
0.94k, Cave 1 0.69k, Stony 0.56k, Crypt 0.40k, Blood Moor 0.12k).

### Parts and the minimal first change
(a) Fallback. Minimal change: split the farm exclusion out of `guardedAreas` into `farmExcluded(s) map[int]bool` (same loop). `target0`
adds it to `ex` as today, and when the first `pick()` is empty it deletes exactly those keys from `ex` (not the guards, bans, cleared), logs
`farm: nothing left in the farm set, picking unrestricted` once per arming, and calls `pick()` again. The Den fallback at goal.go:882-893
(V) is the template. Keep the entry-gate loop after it unchanged.
(c) Data. Add areas 5 and 6. `FarmAreas` is a Go `map[int]bool` literal in internal/merc/farm.go:10, which is a literal, not data. Proposed
home: a list in the plan profile (`internal/plandata/`, a new `farm-areas-normal.json` or a `farm: true` flag per step in
`progression-normal.json`, so the farm set reads the same windows as the picker). The flag option is preferred: the farm set then follows
the area's own enter/leave and cannot drift. `merc.FarmAreas` becomes a loaded set; the `farm_test.go:80-87` assertions are rewritten to read
the data. Alternative if plandata is rejected: a `merc.farmAreas` string knob in internal/env/tuning_merc.go. (U which one the coordinator
prefers; see questions.)
(b) Budget (second change, not in the first commit): the farm stays a PREFERENCE after (a), but also time-boxed: at most `FarmBudgetMin`
virtual minutes per arming, then `FarmCooldownMin`, ended earlier when gold >= need. State lives in the wiring (mercwire.go:321-341), the
predicate in farm.go gets two ints (armed-since, last-end) so it stays testable. Hysteresis: re-arm only when gold < need minus a margin
(`FarmRearmPct`).

First change = (a)+(c): one fallback re-pick plus the data edit. (b) after the first rerun shows time on farm.

### Files
internal/bot/routeguard.go (split `farmExcluded`), internal/bot/goal.go (fallback after the first `pick()`), internal/merc/farm.go (set from
data), internal/plandata (flag or file), internal/env/tuning_merc.go (knob desc text updates; new knobs for (b)), tests: internal/merc/farm_test.go,
internal/bot/goal_test.go, cmd/d2bot/mercwire_test.go.

### Knob defaults (data)
`merc.farmFallback` 1 (0 = today, rollback). For (b): `merc.farmBudgetMin` 40, `merc.farmCooldownMin` 60, `merc.farmRearmPct` 90 (all S: SME numbers,
U as tuning). Cap: `goldFarmMaxLevel` 18 -> 17 belongs to card 3b below, not here. No class literal: the farm set and windows are the same for
all builds; a melee build enters Stony earlier through `melee_enter_clvl` (progression-normal.json:45), which the picker already reads.

### Tests (table-driven, two builds)
Sorceress and Barbarian at clvl 15, farm on, all farm areas banned: expect the unrestricted pick (Cave 2 / next window area, guards still
excluded), one log line; farm set non-empty: restricted pick wins, no fallback; knob 0 equals today; DW and BM members of the set at clvl 13
(`enter` differs per build only through the plan data); a guard-closed Monastery chain stays excluded in the fallback; bans and cleared stay
excluded in the fallback; budget rows (b): armed 41 min -> off, cooldown 59 min -> off, gold >= need -> off at once, re-arm after 60 min.
A farm-flag data test: every flagged area exists in the steps and has a window.

### Risks
- Deaths in Dark Wood / Black Marsh at clvl 13-16: S, SME: 0 deaths in 315 min of BM, average minimum HP 68-80 percent; BM area level 6. Not a
  proof for a merc-less caster (U: the logged characters had mercs or not? not stated by the SME). Dark Wood and Black Marsh were already the
  plan's own areas before the farm was added (V plan windows), so this is a return to the normal path, not a widening past it.
- The fallback re-opens everything not guarded; `routeGuardWhy`/`towerGuardWhy` still close the Monastery chain and the tower (V
  routeguard.go:133-141), so the SME's "redundant for safety" holds for the chain (S, not re-proved for areas 19-20 outside the guard: could
  not verify; Mausoleum 19 is not guarded and was not in the farm set, so the fallback can pick it, which the pre-farm picker also did).
- Gold per kill ranking is U: the carried-gold per hour figures are the SME's computation, the repo has no per-area gold log (see measure).
- A "clvl >= 13" sample, 3 seeds (S).

### Size and live rerun
S for (a)+(c), about 40 lines of Go plus data; M with (b). Live: soak seed 13 (the policy doc rerun): count `gold farm on` periods and `farm:
nothing left` lines, minutes in 5/6 during farm, deaths per hour no worse, `no eligible reachable` lines gone.

### Measure first (SME list, S)
1. Count `gold wait need=` lines (the mercrevive logging) per run. 2. Log time on farm and gold at farm on/off. 3. Check the revive quote against
the formula (price 1680 at merc level 15). 4. Log XP per kill by monster, and gold per area (neither is measured now).

## Card 2: nav fault ranking penalty = HOLD

Keep the old card 2 text (docs/noareacards.md section "Card 2") as the design, but do not start it. Reason: real and separate fault (one
nav fault with an empty tome banned every area, S measure 2.1), but it carries the G1 attempt 14/17 repeated-failing-walk risk (the
doubling ban exists to stop it, areaskip.go:45, V as cited in the old card), and the farmfallback card removes the symptom seen in seed 13
(Pick empty) with far less risk. Re-open only if `no eligible reachable` still appears after card 1 on the rerun. Dependency: the idleBreak
helper of the dropped card 1 is NOT needed any more.

## Card 3: kit trip when a guard is kit-closed and gold covers the kit (rewrite of gold-for-kit)

Old card 3 (recovery start when gold is short) DROPPED: the seed-13 character held 1.6-2.2k gold against a kit of 310 (S, SME; V in measure doc
section 3 for 1074-810 carried at 23:35). The real gap: a guard closed by a kit shortfall that gold already covers. Today `RecoveryCheck`
starts only when `Gold < RecoveryKitGold` (500) and `Shortfall > Gold` (resource/recovery.go:45-54, V as cited in the old card), and `kitTrip`
(kitgate.go:96) runs only from `KitGateLevel` 8 with a way home. New change: when `routeGuardWhy`/`towerGuardWhy` is non-empty for a KIT
reason (potions, mana, tome/TP; not merc, not clvl) and gold >= the minimal guard kit cost, trigger one `kitTrip` (town, shop, chores) before
the next pick, capped by `KitTripCap` per hour. No farming, no recovery state. Cost uses the existing `KitRefill` data.
Knobs: `GuardKitTrip` 1 (0 = off), cap reuses `KitTripCap` 2. Tests (two builds): Sorceress (needs mana) and Barbarian (no mana need) with gold
1500, healing 0 -> trip, bought, guard opens; gold 100 -> no trip (recovery path); merc/clvl guard -> no trip; cap reached -> none. Size S-M.
Risk: a trip loop if the shop lacks stock (U): the cap bounds it. Live rerun: seed 13 block: `0 healing potions < 4` guard lines followed by
a trip line, not by the flood (measure 2.2).
3b (data, depends on card 1): `merc.goldFarmMaxLevel` 18 -> 17 (tied to the merc-less Andariel floor 17, S), revive reserve 50 percent when the
kit is already full, optional skip of the revive when price > 3x the Act 2 quote (S, medium confidence). Reason: at clvl 16-17 only Crypt (leave 17)
is in window in the farm set (V plan data: Cave 2 leaves at 15), cap 18 buys nothing.

## Card 4: idleBreak best-XP = DROPPED

Evidence: the block it addressed cost about 0.15 s (S, SME: idleBreak lifted at 23:36:00, 0.15 s after the empty pick; V in the measure doc
section 2.1 that the lift came at 23:36:00 and the plan picked Cave 1 at once). The 8 of 8 empty set was a farm-rule effect, not an idleBreak
order effect. Also the XP table it ranks by is the weak part (traced XP/kill 7-29x under measured, S; the table has no row for Crypt,
Mausoleum or the Cloister areas, V recoveryxp.go:69-70 as cited in the old card). Nothing to implement; the old section 0 caveat about the XP
table stays true.

## Card 5: plan-data trial, Black Marsh leave 16 and Dark Wood leave 15

A data edit in internal/plandata/progression-normal.json: `leave_clvl` of area 6 from 14 to 16 and area 5 from 13 to 15 (V current values
14 and 13). Evidence: BM left at clvl 14 because of the window and the level-up came 13 s after the pick, not danger (S measure section 4:
the pick at 23:33:58 and the clvl-15 level-up at 23:34:11, V). Measured BM XP/h (V measure section 1): seed 11 6.9k (99 min), seed 12
9.2k (157 min), seed 13 31.1k (59 min). Uncertainty: the 31k is a 59 minute sample, 21 of them at clvl 13, with the top 5 minutes giving 37
percent of the XP (S, SME); plan on 7-9k, not 31k. Rank BM >= DW > Cave 2 held in 3 of 3 seeds (S). Against it: the repo's `xpLevelPct` (V
recoveryxp.go:35) gives 62/43/24/5 percent at clvl 13-16 for a mlvl 6 area, yet logs at clvl 16 earned 6.9k and 9.2k, so the table
is contradicted (S) and cannot decide this. Medium-low confidence. Test: a plan-data test (the window shifts, nothing else), two builds. Risk:
slower leveling past 16 if BM/DW stay on the list; deaths U. Live check: seeds 11-13 rerun, XP/h at clvl 15-16 versus the measure doc
section 1 (seed 13 clvl 16: 558 XP/h, seeds 11/12 3.3k-4.8k). Independent of card 1 but measure after it lands so the farm does not confuse
the A/B. Size XS.

## Corrections to docs/noareameasure.md section 5 (list only; that file is not edited here)

Lines are of `docs/noareameasure.md` on branch `agent/noareameasure`:
1. Line 195-197 (Caveat, last sentence): "in the 1.95% band for Cave 2, Stony Field and the Burial Grounds" is wrong. `xpLevelPct` gives 1.95 for a
   monster 10 or more levels ABOVE the player (d >= 10) and 5.08 for 10 or more levels BELOW (d <= -10) (V recoveryxp.go:35-51). Those areas are
   below a clvl 15 player, so the band is 5.08%.
2. Line 190 (Cave 2 comparator): "traced 27.2 x 1.95% = 0.5 XP/kill" -> 27.2 x 5.08% = 1.4 (V arithmetic, same file). The 0.5 traced vs 14.6
   measured (also in line 193) -> 1.4 vs 14.6; apply the same to Stony (0.7), Burial Grounds (0.8), Cave 1 (0.7), Cold Plains (0.5) if they use
   the same band (U for each: only Cave 2's mlvl was re-derived by the SME).
3. Line 176-178: Catacombs 2 is closed by the route guard (stated) but the "about 25.5k XP/h" (lines 187, 188 region) is not a prediction: it
   is arithmetic of a plain-monster mean times the stay's kill pace. Mark line 187 "not a forecast" and remove "25500" from any summary. Also
   section 4 line 171 context: "the earlier economy cause" for the first block was wrong; the kit and gold story fits only the 01:06 flood
   (gold 26, 0 healing potions, section 2.2), not the 23:35 block.
4. Section 3 table (line ~149-161): "tome 6 -> 0 in 50 s" suggests town-trip churn (U, SME). Add the U tag and the gold line: 2022 at 23:35:06,
   1616 within 3 s after chores (S, SME), the revive runs after shopping (could not verify the mercwire.go:618-627 order here).
5. Add after line 197: gold per area and XP per kill by monster are unmeasured (the SME gap).

## Order of work and dependencies

1. Card 1 farmfallback (a)+(c), then its rerun and the measure list. Independent of everything else.
2. Card 1 (b) budget, and 3b (cap 17, reserve) after the rerun shows time on farm.
3. Card 5 data trial: after card 1 so the A/B is clean (can be run in parallel as a separate soak seed).
4. Card 3 kit trip: after card 1 (the farm no longer hides the guard story).
5. Card 2 HOLD, card 4 DROPPED.
Only goal.go/routeguard.go (card 1, 3) and the plan JSON (card 5) overlap; card 1 and 3 are serial.

## Questions for the user

1. Farm set home: a `farm` flag per plan step (preferred), a separate JSON, or a string knob?
2. Is the farm a pure preference after the fallback, or also time-boxed (b) in the first release? Default: (b) one rerun later.
3. May a merc-less character farm Dark Wood and Black Marsh at clvl 13-16 on the evidence of 0 deaths in 315 min (S, 3 seeds)?
4. Cap 17 (tied to the merc-less Andariel floor) instead of 18: confirm.
5. Skip the merc revive when its price exceeds 3x the Act 2 mercenary quote, or leave the revive always on?
6. Card 5: accept the trial although the 31k figure is a 59 minute outlier?
7. Do you want a per-area gold and XP-per-kill log added before any further tuning? (Measure first list.)

## Shared-file lines for the merger
See merge-lines in the commit body (none needed: no card).
