# Fresh Paladin first hour: where the 60 minutes went (2026-11-12)

> status: analysis, no code changed. Source: the sandbox bot log from the reset (build 0.0.261112021709), window 02:18:42Z to 03:18:42Z, parsed with a throwaway script over the JSON lines (`minute`, `killtime`, `combat fight`, `nav.trace leg`, `zone entry`, `arrival bounce`, `town trip`, `planner`, `area ban`). Tags: V read from the log (how given), S from docs or general game knowledge, U unverified. Headline row: docs/probes/firsthour.md (clvl 6, 528 kills, 0 deaths, 1007 gold).

## 1. Result and benchmark

| point | value | tag |
|---|---|---|
| clvl at 60 min | 6 (xp 17032 at minute 60) | V `minute` |
| level-ups | clvl 2 at 02:20, 3 at 02:23, 4 at 02:36, 5 at 02:45, 6 at 03:11 | V `minute` (matches the D2 normal XP table 500 / 1500 / 3750 / 7875 / 14175, D: Experience.txt via docs/research/xp-gold-trace.md Q6, corrected 2026-11-13; the levels above are unchanged) |
| kills | 528 (`minute` counter); 634 `combat kill` lines in the window (not explained, U; the repeat-suppressed lines go the other way) | V |
| xp per kill / per minute | 32 / 284 over the hour | V (derived) |
| earlier baseline | 2026-11-07 Sorceress, first realtime hour: 128 kills, clvl 3 (docs/fresh-start-pace.md) | V (doc) |

Against that baseline the hour is 4x the kills and 3 levels more (improvement, V). Against a human (S, not asked of the d2-sme, ask it to confirm): a competent player reaches clvl 6 in about 25 to 30 minutes (Den of Evil plus Blood Moor and Cold Plains) and is clvl 9 to 11 after an hour of Cold Plains / Burial Grounds / Cave. CORRECTION 2026-11-13: clvl 9 to 11 in an hour is at the edge of the model (25 kills/min with Stony Field gives clvl 9 to 10), and the SME's clvl 13 to 15 is unreachable from Act 1 Normal kills (V run, xp-gold-trace sections 5 and 6); the calibrated tiers are in docs/first-hour.md "Targets". U: no d2-sme benchmark was fetched; the questions to put to it are at the end.

## 2. Time budget of the hour (V unless noted)

Wall time per area from `zone entry` timestamps (sums to 3600 s):

| area | seconds | share | kills (combat kill) | xp gained per minute there |
|---|---|---|---|---|
| Rogue Encampment (town, incl. portal transit) | 495 | 13.8% | 0 | 0 |
| Blood Moor | 1478 | 41.1% | 211 | 337 early (to 02:24), 171 on the re-clear |
| Cold Plains | 620 | 17.2% | 198 | 558 (03:08 to 03:18) |
| Den of Evil | 1002 | 27.8% | 225 | about 400 (first pass), 223 (second farm) |

Inside the field time (3105 s), from the same log:

| bucket | seconds | share of hour | how |
|---|---|---|---|
| fight loops (`combat fight sec=`) | 1660 | 46% | sum of fight durations; includes the approach walk |
| of which walking legs called from `combat.Fighter.move` | 1085 | 30% | `nav.trace leg durMs` by caller; overlaps the line above |
| explore / roam between packs (`nav.Navigator.explore`) | 931 | 26% | legs by caller |
| exit and waypoint travel (`bot.travelTo`) | 235 | 6.5% | legs by caller |
| loot walks (`loot.pickOne`) | 126 | 3.5% | legs by caller |
| nav stall pauses (299 `stall` / `resume` pairs) | 311 | 8.6% | sum `pauseMs`; overlaps the legs |
| town (shop, repair, sell, heal, portal) | 495 | 13.8% | see section 4 |

`killtime` buckets (545 lines, 3582 s): walk 807 s (22.5%), cast/attack animation 534 s (14.9%), wait 2241 s (62.6%). The "wait" bucket (standing idle, target search, panels, being hit, town, link waits) is the largest and cannot be split from the log (U). Its definition and the mismatch with the leg sums (legs 2540 s against walk 807 s, so the displacement rule in killClock undercounts or the legs include the stalls, U) make it the first measurement to fix.

`status` samples (every 30 s, 120 samples): clearing 100, resupplying 12, in_town 4, traveling 3, idle 1 (V).

Other counters in the hour (V): 6 `arrival bounce`, 63 `nav stuck`, 41 `nav stalled by monsters, yielding`, 10 `clear killed nothing; moving on`, 9 `loot: pack full`, 3 "Save and Exit" save points (about 3 s each), 0 chickens, 0 deaths, 0 poverty stalls, about 43 `loot: potion skip` repeats.

Dead minutes (at most 1 kill in a minute, from `minute`): 12 of 59: 02:25 to 02:26, 02:28 to 02:30, 02:38, 02:43, 02:46, 02:48, 02:49, 03:02, 03:04.

## 3. The 35 zone entries and the 7 to 9 "shuttles"

The firsthour tool reports 33 entries and 7 shuttles; counting `zone entry` lines in the window gives 35 and the A,B,A rule over the non-town entries gives 9 (small difference of window edge and restart handling, U). Sequence (minutes:seconds after 02:00, T = town, BM = Blood Moor, CP = Cold Plains, D = Den of Evil):

`18:46 T, 18:58 BM, 30:46 CP, 31:05 BM, 31:42 D, 34:20 T, 35:16 D, 38:34 T, 38:55 T, 39:07 BM, 39:28 D, 42:51 T, 44:35 D, 46:24 BM, 46:25 D, 48:12 T, 48:37 D, 48:55 BM, 48:57 D, 48:57 BM, 48:59 D, 49:00 BM, 49:02 D, 49:04 T, 50:22 BM, 51:15 T, 52:08 BM, 56:16 T, 57:16 BM, 63:45 T, 64:13 T, 64:24 BM, 64:38 D, 68:05 T, 68:41 CP`

| # | when (Z) | what | reason | cost |
|---|---|---|---|---|
| 1 | 02:30:46 to 02:31:05 | BM, CP, BM | journey chapter `cold_plains_wp` (take the waypoint) before the Den quest | 19 s, planned (V `journey: chapter done`) |
| 2 | 02:38:55 to 02:39:28 | T (re-entry), BM, D | quest done, "save point: saving and re-entering" dropped the portal, so the walk back runs Town to BM to Den | about 33 s per save point (V; also 03:04) |
| 3 | 02:46:24 to 02:49:02 | D, BM, D, BM, D, BM, D, BM, D | border-loop: the Cold Plains exit in Blood Moor sits at (145,145) beside the Den cave mouth (145,141); the walk "left level 2 for 3 but ended in 8"; 6 `arrival bounce` (stay 0.5 to 2.2 s) | 2.7 min of walking, 0 useful xp (V) |
| 4 | 02:49:04 | D, T | "everything worth farming is cleared, starting a fresh game" because every target route crosses the banned Blood Moor | Save and Exit, 78 s idle in town "Waiting: Blood Moor banned for 3 more min" (V) |
| 5 | 02:50:22 to 02:52:08 | BM, T, BM | repair trip (`town trip` repair, 154 gold) | 53 s |
| 6 | 02:56:16 to 02:57:16 | BM, T, BM | pack 85% full, sell + a portal scroll | 61 s |
| 7 | 03:03:45 to 03:04:38 | BM, T, T, BM, D | `Going to town to claim a quest reward` (claim trip, goal "Farm Den of Evil"); the claim failed "no skill point after talking to Akara" because it was already claimed at 02:38:49; save point; walk back to the Den | 39 s town + 33 s walk, then a 4 min Den re-farm at 223 xp/min (V) |
| 8 | 03:08:05 to 03:08:41 | D, T, CP | `no town portal scroll`; sold 7 items; then waypoint to Cold Plains | 36 s (V) |

The other town entries are ordinary trips (section 4). So the "7 shuttles" are: 1 planned (1), 2 save-point walks (2), 6 genuine bounces in one 2.7 minute burst (3). Only (3) is a defect of nav/planner; (4) and (7) are decisions made because of (3) and a stale quest state.

## 4. Town trips (8 in the hour, 495 s of town = 13.8%)

| time | reason (V `town trip`) | town seconds | spent / gained |
|---|---|---|---|
| 02:34:19 | 8 items to sell, 22 gold | 57 | +650 gold (two chipped gems at 250), then 450 on a Tome of Town Portal |
| 02:38:34 | heal below the resume line (life 61 of 88) | 33 | 0; quest reward claimed and save point |
| 02:42:49 | inventory 90% full, 8 items to sell | 106 | 0; "2 item(s) no vendor would buy"; portal back needed 2 tries |
| 02:48:10 | inventory 95% full, valuable item left | 27 + 78 wait | +200 (a Scepter) |
| 02:51:13 | repair trip | 53 | -154, 2 scrolls (200) |
| 02:56:15 | pack 85% full | 61 | +190 sold, 100 for a scroll |
| 03:03:45 | heal (claim trip) | 39 | -100 |
| 03:08:03 | no town portal scroll | 36 | +535 sold (gem 250) |

Five of eight trips are the pack filling with white and low junk (9 `loot: pack full` lines); the hero picks up armour, bows and clubs at clvl 3 to 5 and then walks home to sell them for 4 to 50 gold each. Portal scrolls are bought one at a time at 100 gold each, 8 trips need 8 scrolls (V `shop: bought`, 6 purchases).

## 5. Time lost, ranked (minutes, V from the segment table)

Segment table (from `minute` lines): xp and kills per segment.

| segment | window | minutes | xp | xp/min | kills |
|---|---|---|---|---|---|
| Blood Moor, fresh | 02:19 to 02:24 | 5 | 1683 | 337 | 77 |
| Blood Moor, exhausted | 02:24 to 02:31 | 7 | 324 | 46 | 13 |
| Den pass 1 plus 4 trips | 02:31 to 02:45 | 14 | 5607 | 400 | 133 |
| bounce plus ban plus fresh game | 02:45 to 02:50 | 5 | 213 | 43 | 9 |
| Blood Moor re-clear (forced) | 02:50 to 03:03 | 13 | 2223 | 171 | 96 |
| Den re-farm (stale claim) | 03:03 to 03:08 | 5 | 1114 | 223 | 43 |
| Cold Plains | 03:08 to 03:18 | 10 | 5577 | 558 | 146 |

Cold Plains is the best field the hero stood in (558 xp/min, 14.6 kills/min), and it was entered only at minute 50. Counting the loss against that rate:

1. **Blood Moor held past exhaustion, 6.2 min** (02:24:32 to 02:30:46, 11 kills, 250 xp). V: `clear killed nothing; moving on remaining 0` three times, killtime gaps of 144 s and 198 s, `route guard` lines. The campaign chapter `start` does not end until Blood Moor is "cleared"; the sweep walks the whole map for stragglers. Fix: end a campaign/leveling clear when K consecutive empty sweeps or coverage 100% (`roam: coverage points`) or kills/min below a knob over 2 min; at clvl 3 advance to the Den (planned next anyway). Expected saving about 5 min (to Den at 400 xp/min is +1600 xp, almost one clvl at that stage).
2. **Border-loop cascade, then a forced re-clear of Blood Moor, about 4.7 min plus 9 min of lost rate = 12 to 13 min.** V: 6 bounces from 02:46:24, `area skipped Blood Moor 180 s border-loop`, then every target "route crosses a banned area Blood Moor" (13 planner picks refused), "starting a fresh game", 78 s waiting, then 13 min of Blood Moor at 171 xp/min instead of Cold Plains at 558 (lost (558-171)/558 of 13 min = 9 min). Two fixes: (a) a border-loop skip must not ban a transit area that every route goes through (ban the target or the edge only; `internal/bot/loop.go` `tripLoopLocked`, `skipAreaLocked`), (b) the Blood Moor to Cold Plains exit walk must not step on the Den mouth at (145,141) (`nav: stepping away from a cave mouth` already fires, 8 times, and rewrites the click, but the exit gate is 4 subtiles from the mouth, U root cause). Expected saving 8 to 12 min per such occurrence.
3. **Stale quest claim, about 3 min** (03:03:43). V: `claim den_of_evil` failed, claimed already at 02:38:49 (`Quest done: den_of_evil`). `questPending` (`internal/bot/goal.go` claim trip) was non-empty again 25 minutes later and the goal switch put "Farm Den of Evil" in front, 4 min at 223 xp/min instead of 558. Fix: remember the claim in the run state (or read the Akara claimed bit) and drop it from `questPending`. U why it re-entered the list. Saving about 3 min.
4. **Town trips for junk, about 3 min.** 5 pack trips at 27 to 106 s. Fix: a pickup floor for the first hours (skip white weapons and armour under a sell value, keep gems, potions, charms, runes; there is a `loot: potion skip` rule already, the item rule is the knob), and no trip when the sellable list is "no vendor would buy". Expected saving 3 min (2 fewer trips) plus fewer stalls from the full pack (9 left items).
5. **Nav stall pauses, 311 s total** (299 stalls, 63 `nav stuck`). Not all removable; U how much is monster contact. A single log line with the cause class would show it. Expected saving unknown (target 2 min).
6. **Second Den pass and Den-first at clvl 3 to 4**: the Den was planned again after the quest was done ("window 2-6"). The second pass earned 415 xp/min, so the loss versus Cold Plains is small, about 2 min. A plan data change (Cold Plains window start 4 instead of 5, docs/earlygame.md says the window is a data decision).

Sum of the removable time: about 25 min of the 60 (sections 1 to 4 overlap by a few minutes; do not add up the savings, at least 15 min is safe). Where the saved time lands: at Cold Plains rate (558 xp/min) 15 min is 8400 xp, which puts the hero at clvl 7 (23000) at minute 60 rather than clvl 6; at the Den rate it is smaller. The hero is not slow in a fight (killtime cast 15%); it is slow in choosing where to fight.

## 6. Comparison with older runs

| run | kills / 60 min | clvl@60 | notes |
|---|---|---|---|
| 2026-11-07 Sorceress, fresh start (docs/fresh-start-pace.md) | 128 | 3 | recovery loop, mana limited, empty field |
| 2026-11-12 Paladin (this) | 528 | 6 | no poverty stall, no death; time lost to routing |

No other first-hour row exists in docs/probes/firsthour.md (one row), so there is no regression to name (V). The three Sorceress runs planned in docs/smoke.md / sorceress-early.md are not in the file (U).

## 7. Proposed cards (ranked, each with the knob or file)

| rank | fix | where | saving |
|---|---|---|---|
| 1 | transit-area exemption for the border-loop ban; add a Cold Plains fallback that does not cross the banned area | `internal/bot/loop.go` (`tripLoopLocked`, `skipAreaLocked`), planner `internal/bot/plan.go` | 8 to 12 min per occurrence |
| 2 | end a campaign clear when sweeps come up empty (K sweeps or coverage done) | `sweepPass`, `internal/bot/goal.go`; knob proposal `bot.clearEmptySweeps` | about 5 min |
| 3 | claim memory: do not re-queue a claimed quest | `internal/bot/goal.go` `questPending`, `internal/task/claim.go` | about 3 min |
| 4 | early pickup floor and "nothing sellable" trip rule | loot filter, trip triggers (`trip budget`, `goldFloor` 300) | about 3 min |
| 5 | split `killtime.wait` into idle, panels, hit, town, link; log stall cause | `internal/bot/killtime.go`, `nav.trace stall` | measurement |
| 6 | Cold Plains window start 4 (data) | plan table, docs/earlygame.md | about 2 min |

## 8. Questions for the d2-sme

(1) Expected clvl and kills at 60 min for a Zeal-less Paladin (plain Attack, Might, Sacrifice, Prayer rest-only) in normal Act 1 with Den of Evil, Blood Moor and Cold Plains? (2) Is Blood Moor worth more than one clear after the Den? (3) At which clvl does a human move to Cold Plains / Burial Grounds? (4) Is picking up white armour and weapons at clvl 3 to 5 worth the town trips?

Not verified live; no game, no bot code, no sandbox change. Throwaway script lives in the scratchpad, not in the repo.
