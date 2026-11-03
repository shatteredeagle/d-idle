# XPHSHORT: why Gate 1 XP/h fell from 10042 to 7060, and the valid multi-seed Gate 1

> status: current. Analysis only (docs and one stdlib tool, no build, no game, no host load). Inputs: the minute archives of the five Gate 1 runs of 2026-11-02/03 (branches agent/gaterecords, agent/gate1run-20261103-0338, agent/gate1run-20261103-0628: `docs/probes/soak-logs/gate1-<run>-minutes.jsonl.gz`), read-only greps of the worker host run logs (`bot-0.log`), docs/research/beltfix4.md, specdelta-rejudge.md, docs/d2d-soak.md, the ops repo docs on gate-run and worker capacity.

Tags: D = computed from the minute lines by `tools/baseline/xph_budget.py` (rerunnable, below); V read = read in a log or script; I = inference; U = unverified. A minute line is one virtual minute and a SNAPSHOT of the bot's step, so category shares are samples (I). XP/h here is `final xp / (lines / 60)`; the records say 7060 for 0628, the script 7067 (0.1 percent, the hours rounding).

Rerun (any box, stdlib):
```
python3 tools/baseline/xph_budget.py LOG.jsonl.gz ...                       # per run: summary, budget per clvl band, farming areas
python3 tools/baseline/xph_budget.py --ref A.gz B.gz --cmp C.gz             # decomposition of mean(A,B) -> C
python3 tools/baseline/xph_budget.py --horizon 1000 --json LOG.gz ...       # cut every run at the same virtual minute
python3 tools/baseline/tests/test_xph_budget.py                             # fixture test
```

## 1. Answer

1. D: the XP/h drop is NOT a cost of the belt/shop/walk-home fixes. The town-related time (going to town, in town, returning to the field) is 8.8 percent of the minutes in run 0628 (the LOWEST of the five; 9.4 to 13.7 percent in the others) and costs 141 XP/h in 0628 against 640, 295, 291, 899 (1939, 2210, 0053, 0338). The newest run is the cheapest in town time. Trips: 4.2 per virtual hour (103 area-1 entries) against 4.4 to 4.9.
2. D: about 2300 XP/h of the gap (2839 to the first two runs, 2117 to the mean of the first four, 1396 to the two runs with the same fixes) is XP per kill at clvl 13 and above, i.e. WHERE the bot farmed: in 0628 no farming minute at clvl >= 12 was spent in Dark Wood or Black Marsh (0 percent; 27, 42, 18, 28 percent in the other four, 30 to 36 XP per kill there) and 909 field minutes ran at 14.6 XP/kill (89 XP/min) against 20 to 22 XP/kill (121 to 152 XP/min). Cold Plains alone took 229 minutes at 3.4 XP/kill. I: the planner's targets (Dark Wood, Black Marsh, Cave 2) were `unreachable: no activated waypoint on the way` or `nav-stuck`, so it fell back to "past every window, last eligible" areas (V read in `bot-0.log`). Not shown to be caused by the belt fixes; the same fix set (0053, 0338) reached those areas.
3. Cheapest remedy: area choice, not belt. Ranking the fallback areas by estimated XP per kill (never Cold Plains/Stony Field at clvl >= 13 while Crypt, Burial Grounds or Cave 1 are reachable) is worth about +1100 XP/h; making Dark Wood and Black Marsh reachable (activate their waypoints early) about +1200 to +1700 XP/h (I, section 5). Town remedies are worth at most 70 to 300 XP/h.
4. Gate 1 stays NOT PASSED for 0628 by the approved rules (7060 below the floor 7338) but one seed cannot show a cost of the fixes: the five runs have sd 1212 XP/h (13.8 percent; different bots). A valid verdict needs 5 seeds (section 6): about 3 to 3.5 wall hours with all five in parallel (U), Gate 2 after it.

## 2. Time budget per run (task 1)

D, `xph_budget.py` over each minute log. Category by `state`/`step`: farming (campaign "Quest: fighting"), gold farming (`Recovery: farming gold`: recovery mode FIGHTS, it earns XP), going to town, town (chores, settling, gear), return to field, walk/other, quest travel/wait. Share of virtual minutes, whole run:

| run (bot) | vmin | XP/h | farming | gold farming | going to town | town | return to field | walk/other | town-related XP/h lost |
|---|---|---|---|---|---|---|---|---|---|
| 1939 (8c7d657) | 1182 | 10051 | 52.4% | 35.9% | 0.3% | 5.9% | 5.3% | 0 | 640 |
| 2210 (cd4c7cb) | 1147 | 9766 | 71.0% | 17.6% | 0.7% | 5.1% | 3.7% | 1.9% | 295 |
| 0053 (cbf6905) | 1239 | 8170 | 52.4% | 34.4% | 1.3% | 5.4% | 6.1% | 0.4% | 291 |
| 0338 (30710e1) | 1205 | 8759 | 56.9% | 28.8% | 1.3% | 4.8% | 7.6% | 0.3% | 899 |
| 0628 (a45b9af) | 1473 | 7067 | 71.9% | 19.3% | 0.6% | 4.5% | 3.6% | 0 | 141 |

"town-related XP/h lost" = minutes in those three categories times the farming XP/min of the same clvl band, minus their own XP, per run hour. Recovery (gold farming) shares of 18 to 36 percent did not decide the XP/h: the best run (1939) had the most.

XP/min by clvl band (D; band share of the run in brackets), farming category only at the right:

| run | 1-6 | 7-9 | 10-12 | 13+ | farming XP/min at 13+ |
|---|---|---|---|---|---|
| 1939 | 180 (9.9%) | 277 (10.8%) | 175 (26.7%) | 138 (52.6%) | 172 |
| 2210 | 174 (10.8%) | 224 (13.7%) | 187 (25.5%) | 132 (50.0%) | 136 |
| 0053 | 128 (14.0%) | 297 (9.3%) | 145 (35.3%) | 109 (41.4%) | 149 |
| 0338 | 127 (14.6%) | 285 (10.2%) | 175 (25.7%) | 118 (49.5%) | 142 |
| 0628 | 120 (12.6%) | 215 (10.7%) | 197 (19.2%) | 77 (57.5%) | 81 |

The full per-band category table (minutes, share, XP/min, kills/min, XP/h lost against farming) is the script's default output.

## 3. Split of the gap, run 0628 against the earlier runs (task 2)

D, `--ref ... --cmp 0628`. Gross XP/h = sum over bands of (share of minutes at the band) times (XP/min); the terms are level-lag (band time mix), category mix, kills/min, XP per kill, injected-kill loss. The sum differs from the net delta by the interaction and rounding.

| reference | gap (net) | level-lag | category mix | kills/min | XP per kill | injected kill |
|---|---|---|---|---|---|---|
| 1939 + 2210 (10042, 9754) | -2839 | -239 | +428 | -447 | -2298 | -153 |
| 1939, 2210, 0053, 0338 | -2117 | -271 | +755 | -220 | -2312 | +10 |
| 0053 + 0338 (same fixes) | -1396 | -223 | +951 | +26 | -2326 | +172 |

- Time on the new behaviours: not a cost in 0628 (section 1; category mix is POSITIVE, because 0628 spent more minutes farming and fewer in recovery and town). Per trip: 1.32 vmin in area 1 plus the walk (return-to-field 53 minutes over 103 trips: 0.5 min), so one trip costs about 1.9 vmin; measured against the farming rate of the same band, the town-related minutes cost 141 XP/h in total (D; town minutes still carry some XP: kills at the portal and on the way). Walk-home minutes (state `Returning to the field`, area <> 1 in 7 of 53 lines): not separable from the portal return in the minute lines (I).
- Kills per minute: 5.83 in 0628 against 6.25, 6.22, 5.50, 5.60 (D). Not the gap (-447 at most, +26 against the same fix set).
- XP per kill by area at clvl >= 12 (farming and gold farming minutes; D): Black Marsh 6: 49.5 / 36.3 / 39.6 / 50.5 / (absent in 0628); Dark Wood 5: 16.0 / 31.3 / 26.8 / 24.9 / (absent); Stony Field 4: 15.0 / 12.7 / 21.1 / 16.6 / 19.4; Cold Plains 3: 2.9 / 2.2 / 6.7 / 3.8 / 3.4 (0628: 229 minutes); Burial Grounds 17: about 5 everywhere; Crypt 18: 38.4 in 0628 (54 min). Weighted XP per kill at clvl >= 12: 21.4, 21.8, 19.8, 20.0, 14.6; XP/min there: 145, 152, 121, 123, 89.
- Injected kill -9: lost XP 0, 280, 7197, 6354, 3928 = 0, 15, 349, 316, 160 XP/h (D). It decides the d2d save timing, not the bot: the fault is at wall minute 60 and a longer virtual run dilutes it.
- Early-game loot divergence: XP at vmin 100 was 19400, 14955, 11403, 11403, 9296; clvl 6 at vmin 88, 91, 117, 117, 131; clvl 12 at 406, 500, 540, 417, 500; pack plus stash gold at vmin 100: 129, 340, 250, 250, 35 (D). The level-lag term (time in the bands) is only -220 to -270 XP/h: the early lag is real but small (about 10 percent of the gap), because the high-clvl rate, not the pace to get there, differs (I). Same seed, same first 6 lines, then single-kill differences (beltfix4 section 3).
- Seeds/luck (I): the five runs are one seed and five bots; their spread is 13.8 percent (sd 1212 XP/h). The XP/h of the SAME bot depends strongly on the horizon: mean over the five runs at 6, 10, 12, 16.7, 19 virtual hours = 12203, 10952, 10920, 9464, 9115 XP/h (D). Rates fall with clvl; a longer run is judged lower. 0628 is the longest (24.6 vh, the others 19.1 to 20.7), so its whole-run mean is biased down by about 300 to 400 XP/h against the others (D: at 1140 vmin 7940 against 8766 to 10326; the baseline 9172 has its own horizon, U which).

Budget of the 2839: XP per kill (area choice) 81 percent; kills/min 16; early game 8 (level-lag); category mix -15; injected kill 5 (D, rounded; terms interact).

## 4. Is the lost XP/h a cost of the fixes? (task 3)

No, with the evidence above: (a) the town-related time and its XP cost are the lowest in the run with all five fixes; (b) 0053 and 0338 carry most of the same fixes and lose 1000 to 1900 XP/h to the first two for reasons that sit in the early game, the fault (316 to 349 XP/h) and the 13+ area mix, with town-related shares of 12.8 and 13.7 percent against 11.6 and 9.4 (D); (c) the 0628 gap is area selection. Upper bound for what the new town behaviour costs, taking 1939 as "before": town-related lost XP/h is 141 in 0628 against 640 in 1939, so no extra cost shows (noise between runs is larger than any effect, I); the worst town figure is 0338 (899), before STASHFIELD and BELTFIX5. The only town-side signal is in 0338 (899 XP/h, `Returning to the field` 7.6 percent of its minutes) and 0053 (`walking the level chain` 46 times, V read); in 0628 the chain walk occurred 0 times (V read, `grep -c`).

## 5. Remedies (task 3, ranked by XP/h and by cost; expected gains D, causes I)

1. Fallback area ranking by XP per kill (bot, planner): at clvl >= 13 the "past every window, last eligible" fallback picked Stony Field and Cold Plains (3.4 to 19.4 XP/kill) with Crypt (38.4), Cave 1 (19.1) and Burial Grounds reachable. Replace the order by estimated XP/kill (the planner already prints `xp/kill est`). Expected: if the 229 Cold Plains minutes had run at the Crypt/Cave-1 rate (about 137 XP/min against 18) +1100 XP/h. Does not touch belt protection.
2. Make Dark Wood and Black Marsh reachable: both were `unreachable: no activated waypoint on the way` (Dark Wood 6, Black Marsh 3, Cave 2 16 times) and Stony Field, Burial Grounds and Cold Plains `nav-stuck` 17 times in 0628 (V read). A one-time walk to activate the Black Marsh waypoint at clvl 9 to 10 with the kit in hand. Expected: 0628 at the 0053/0338 rate for clvl >= 12 (122 XP/min against 89) +1200 XP/h; at the first two runs' 135 to 150 +1700 to +2100 XP/h (I, one seed).
3. Town trips (cheap, small): each trip costs about 1.9 vmin; cap non-emergency kit trips at 3 per hour (0628 4.2) and bank gold only when the stash visit is on a trip that happens anyway (the trips that bought nothing: `shop: visit bought {}`). Expected 70 to 150 XP/h (D: town-related loss 141 to 899, a quarter to half recoverable). Skipping the shop at gold below a potion price is already the rule (30 gold, V read in beltfix4) and avoiding the walk home needs a portal at hand, which the bot already prefers (0628: 0 level-chain walks): nothing to gain there.
4. Not a remedy: relaxing recovery (gold farming 17 to 36 percent of minutes) - 1939 had the most and the best XP/h (D).
Merc: revived only twice in 0628 and `revive skipped` 188 times (gold or cap, V read); same in 0053/0338 (165, 170); the merc dies at clvl 7 to 8 each time. I: this interacts with the 20 to 30 percent "gold farming" minutes, not with XP per kill. Out of scope.

## 6. A valid multi-seed Gate 1 (task 4)

Rule: `d2dsoak compare` needs at least 5 valid seeds, at least 2 virtual hours per seed and 10 virtual hours in total (docs/d2d-soak.md section 3, fidelity-gate "Minimum sample"); PASS needs |gap| <= tol AND the 95 percent half width hw <= tol; FAIL when |gap| - hw > tol; else INCONCLUSIVE. With baseline 9172 and tol 20 percent = 1834 (V, the gate records): PASS means mean in 7338..11006 and hw <= 1834; with n = 5, hw = 2.776 sd / sqrt(5) = 1.24 sd, so sd must be at most 1479 (16 percent); our five runs have sd 1212 (13.8 percent) and at a common horizon 8 to 13 percent (D), so a PASS is reachable when the mean is at least 7338; FAIL needs the mean below about 5800 to 6100. The approved delta (specdelta-rejudge.md): one seed inside the tolerance passes only "one seed, inconclusive"; a claim "at or above the baseline" needs the 5-seed d2dsoak verdict. Minimum for a pass: d2dsoak XP/h PASS over 5 seeds, plus every seed passing the other Gate 1 criteria (stability, deaths, NOKILL, belt with the stash rule, portal trip, fault recovery: one fault per seed at wall minute 60) (U: whether a pooled belt rule is wanted; the spec reads per run).

- Does `tools/gate-run.sh` take `--seed`? At the committed HEAD of the ops repo the seed is fixed at 11 and a second gate run is refused ("another gate run is active"; V read). The ops working tree holds an uncommitted change (not by this card) that adds `--seed N`, `--char`, `--tag T`, `--parallel-ok`, `--collect-multi` (multi-seed compare and one branch) and `GATE_MAX_PARALLEL` (default 3) and `GATE_PAR_LOAD_MAX` (load1 per core 1.1) (V read). U: it is neither committed nor tested by this card; commit and test it first, and raise `GATE_MAX_PARALLEL=5` for the set. Seeds: 1 to 5 are tuning seeds; use 16 to 20 (holdout, docs/d2d-soak.md section 3d) for the verdict (I: the gate has been tuned on seed 11 only; any set the bot has not been tuned on is better).
- Parallelism: pool D2D_SLOTS default 7 (knee 8, ceiling 12), slots 0 to 3 admission-free, slot >= 4 a 10 s admission (idle+nice, iowait, load per core; V read worker-compute.md). One gate run costs about 1.2 cores (d2d lockstep 1.0 plus the bot 0.2; cost table). Five in parallel = 6 cores of 16 plus other agents' runs; keep K (all d2d holders) at or below 7 (U: the speed cliff sits between K=8 and K=10 in the one-off measurement). Check with `tools/capacity2.sh --add 6` (OK needs idle+nice >= 25 percent after the add) and look at load1 per core <= 1.1 before each start; start the five 30 s apart (the instance id uses the minute stamp, the record name a seed tag).
- Wall time and virtual hours: one run at low load went 12.3x (24.6 vh in 120 wall min). The slotscale table gives 6.2x at K = 4 to 8 and 4.9x at K = 8 on an older build (U for the current build). So 5 parallel seeds run at about 6 to 10x: 20 virtual hours per seed need 2.0 to 3.3 wall hours; plan `--hours 3.5` (wall limit) per seed and let the supervisor end it. Total exposure 5 x 20 = 100 vh (the 10 h death floor is met 10 times over).
- Horizon: XP/h falls with run length (section 3: 12.2k at 6 vh to 9.1k at 19 vh). Seeds run to different virtual hours because the limit is wall time, so cut all five minute logs to the same virtual minute (the shortest, at least 1140) with `xph_budget.py --horizon` for the report and pass `-hours` of that horizon to `d2dsoak compare`. U: at which horizon the baseline 9172 was taken (clvl 11 to 14 runs, 8 to 24 h of the old protocol); if it was shorter, the comparison is biased against longer seeds.
- Schedule with Gate 2 after it: Gate 1 set 3.5 wall hours, collect and judge 0.5 h (`--collect-multi`, d2dsoak compare, record), then Gate 2 only on the same passing build: 8 wall hours (spec: 8 h soak of profile m24; at 12x that is about 98 vh; the m24 floor is 1000 XP/h, a stability and economy soak, not the 9172 baseline), one seed (11) or three seeds in parallel inside the admission-free slots 0 to 3. Total about 12.5 wall hours for Gate 1 set plus Gate 2 (U on the speed). Gate 3 (24 h) needs the user's go and is not scheduled here.
- Remedies first: running the set before remedy 1 or 2 of section 5 judges the current planner; 0628 shows a seed can land 1200 to 2000 XP/h low on area choice alone (that variance is part of what the five seeds average).

## 7. Not verified

No game or host load; the worker host was only read (`grep` and `ls` of three run directories, a few seconds). XP/h per horizon and the counterfactual gains are arithmetic on one seed. The causes in sections 1 and 5 are from log text (V read) but not reproduced. Parallel speed at K = 5 is an extrapolation (U).

## Shared-file lines for the merger
See openspec/changes/xphshort/merge-lines.txt.
