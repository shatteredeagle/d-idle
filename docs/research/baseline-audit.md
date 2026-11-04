# Baseline audit: can the original-game baselines gate d2d? (card baselineaudit, 2026-11-01)

> status: current. Docs only; nothing was run on the game host or the worker host. Tags: V = read or recomputed in this audit from committed files (named), D = derived by arithmetic from V numbers (formula given), S = secondary (a doc's own number, not re-derived), I = inference, U = unverified. Inputs: `docs/contracts/fidelity-gate.md`, `docs/contracts/soak-gates.md`, `tools/d2dsoak/baselines.json`, `tools/d2dsoak/baselines-early.json`, `docs/d2d-soak.md`, `docs/d2d-baselines-early.md`, `docs/g1-log.md`, `docs/soaks/gate1-rerun.md`, `docs/d2d-xp-gap.md`, `docs/ab-design.md`, `docs/d2d-fidelity.md`, `docs/coverage/rule-coverage.md`, `docs/probes/soakfull/compare-baselines*.txt`.

## 0. Bottom line

The baselines are too thin to gate on as a hard pass/fail rule, and the one Gate 1 clause that uses them ("true XP/h at or above the baseline at the same clvl") compares things that are not the same. What they are good for today: a coarse "is d2d in the right half of the plane" check (the early d2d set is 2x below the original, far outside any noise, so that FAIL is real, V). What they cannot do: resolve a 20 percent gap from one seed, or even from eight seeds, because the baseline mean itself is uncertain by +-35 percent (XP/h) and +-52 percent (kills/h) at n = 3. The deaths baseline is biased low by how it was assembled. The non-baseline Gate 1 criteria (stability, belt window, no-kill window, portal, route guard, fault recovery) do not depend on any of this and can be gated on.

## 1. What the two baseline files hold

### 1a. `tools/d2dsoak/baselines.json` (the "late" file, clvl 11 to 14)

| field | value | tag |
|---|---|---|
| runs | a14, a16, a18 (3 runs; a14 is partial, its first 64 minute lines of a 2 h run) | V (file) |
| seeds | none: the original game has no seed control; 3 independent live runs on one character (Soaky Sorceress, instance i4), 3 different bot builds | V |
| virtual hours | 1.057 + 1.937 + 2.053 = 5.05 h total (hours = minute lines / 60, restart gaps excluded) | V (file, D sum) |
| class, clvl | Sorceress; clvl 11-12 (a14), 13-14 (a16), 14 (a18) | V (source lines) |
| kit, profile | honest play, profile m1; a14 started 0 gold at clvl 11, a16 gold 211 at clvl 13, a18 gold 16 at clvl 14; the planner and kit gates differ per build | V (g1-log) |
| date | a14 started 2026-10-29 23:57, a16 2026-10-30 03:07, a18 2026-10-30 06:15 | V (g1-log) |
| bot builds | a14 de9b597, a16 b2e010c, a18 b478bd9; these hashes are not in this public repo, so what changed between them cannot be diffed here | V (hashes in file), U (content) |
| backend | Game.exe 1.14d under Wine, offline, hidden instance | V |
| XP/h definition | net character XP, last minus first minute line, divided by minute-line hours; unsaved XP lost to a restart counts against it (a16 OOM restart). Same extractor as d2d (`d2dsoak extract`), so the definition cancels | V (docs/d2d-soak.md section 2) |
| weak flag | `note: PARTIAL` on a14 only; a16 carries an OOM-restart note. No `weak` field exists in the file | V |

Per-metric mean and spread over the 3 runs (D, recomputed from the file; sample SD, 95 percent interval = t(0.975, 2 df) = 4.303 x SD / sqrt 3):

| metric | a14 / a16 / a18 | mean | SD (CV) | min..max | 95 % interval of the mean |
|---|---|---|---|---|---|
| xp_per_hour | 12236 / 11226 / 9240 | 10901 | 1524 (14.0 %) | 9240..12236 | 7114..14688 (+-34.7 %) |
| kills_per_hour | 421 / 459 / 620 | 500 | 105 (21.1 %) | 421..620 | 238..762 (+-52 %) |
| deaths_per_hour | 0 / 0 / 0 | 0 | 0 | 0 | not defined (0 in 5.05 h) |
| deaths_per_1000_kills | 0 / 0 / 0 | 0 | 0 | 0 | 0 in 2606 kills |
| potions_per_hour (info) | 0 / 27.4 / 34.1 | 20.5 | 18 (88 %) | 0..34 | a14 has no counter: 0 is missing data, not a rate |
| gold_per_hour (info) | -191 / -177 / 571 | 68 | 436 (640 %) | | useless as a baseline |

The XP/h time-weighted mean is 10630 (total XP / total hours), 2.5 percent below the run mean the tool uses (V, D). The tool's per-run mean weights a 1 h partial run like a 2 h run.

### 1b. `tools/d2dsoak/baselines-early.json` (fresh clvl 1 to 7)

| field | Fresha | Soaky (early) | tag |
|---|---|---|---|
| backend, date | Game.exe, instance i5, 2026-10-30 | Game.exe, instance i4, 2026-10-29, old builds | V (file) |
| hours | 1.92 (115 minute lines) | 3.5 wall | V |
| XP/h, kills/h | 11743 / 389 | 6600 / 408 | V (Fresha: minute lines), S (Soaky: doc numbers, kills a lower bound) |
| deaths | 1 (0.52/h, 1.34 per 1000 kills over 746 kills) | 1 (0.29/h, 0.70 per 1000) | V / S |
| class, kit, profile | Sorceress, uncheated, fresh start, profile m24 (g1-log) | same class, old build, profile not recorded | V / U |
| seeds | none (live) | none | |
| weak | yes, by file text and by spec (fewer than 5 runs) | S quality | V |

Mean the gate uses: XP/h 9172, kills/h 398, deaths/h 0.41, 1.02 per 1000 kills. Two runs: SD of XP/h 3631 (40 percent of the mean, D), the two runs differ 1.8x. The 95 percent interval at n = 2 (t = 12.7) is wider than the mean itself, so no tolerance statement is possible. The Soaky run is wall-time XP (25 min pause flap, 3 restarts), the Fresha run is play-minute XP: they do not share a definition. The sensitivity is large: on the same Fresha logs, play-minute hours give 11743 XP/h and wall-clock hours give 9833 (-16 percent) and 326 kills/h (-16 percent) (V, docs/d2d-baselines-early.md section 1). A d2d run uses the virtual clock (game ms), so which of the two it matches depends on whether restarts and fresh games consume virtual time (U).

### 1c. Original-game logs that exist but are NOT in either file (V, docs/g1-log.md)

| run | bot | hours | XP/h | kills/h | deaths | clvl |
|---|---|---|---|---|---|---|
| G2 (m24, Soaky) | b478bd9 plus later merges | 8.06 | 4.1k overall (7.4k last hour) | about 880 (7100 kills) | 4 (0.50/h) | 14 to 15 |
| Fresha 4 h (m24) | a920fef | 4.05 | 13.4k | about 590 (2387 kills) | 3 (0.74/h) | 6 to 11 |

These are the only original runs longer than 2 h and the only ones that include the real death rate over a normal span. Their XP/h at clvl 14 (G2, 4.1k) is less than half of a18 (9.2k) on the same character and level (V), which says the spread at clvl 14 is far wider than the three runs of the file show (I: G2 spends hours with the level-up goal blocked by the route guard and a no-merc state; XP per kill at clvl 14 is low).

## 2. Against the spec minimums

| requirement (spec) | minimum | `baselines.json` | `baselines-early.json` | verdict |
|---|---|---|---|---|
| Minimum sample: seeds (fidelity-gate) | 5 | 3 runs (and no seeds as such) | 2 runs | FAIL both (the spec text binds the d2d set; the baseline files carry the same `minSeeds` 5 / 5 and the early file `minHoursPerSeed` 3, so the files state the bar they miss) |
| Hours per seed | 2 (late file) / 3 (early file) | 1.06, 1.94, 2.05: one of three reaches 2 h | 1.92 and 3.5: Fresha misses 3 h | FAIL (late), FAIL (early) |
| Total hours for death metrics | 10 (late), 15 (early) | 5.05 | 5.42 | FAIL both |
| Early baseline fields (m2-earlybase): at least 2 fresh runs with all four gated metrics and a source each | 2 runs, 4 metrics | not applicable | 2 runs, 4 metrics, source lines present | PASS (stays marked weak until 5 runs, which cannot happen on a frozen Game.exe) |
| Same character, same setup | class, clvl range, kit, profile m1 | Sorceress clvl 11-14 matches a clvl 11-14 d2d start; kit and gold differ per run | fresh clvl 1-7 matches a fresh d2d start up to clvl 7 only | PASS for a d2d run started at clvl 11-14; FAIL for any run that leaves the clvl 1-7 band (section 4b) |
| Source tag per run, extractor shared | | present, V | present, V and S | PASS |

## 3. Statistical power

Assumptions (all D, from the spreads above): per-run CV of XP/h about 14 percent for the original (SD of the 3 runs, inflated by build drift, so an upper bound for pure seed noise, I); about 10 to 14 percent for d2d (V: 8 fresh seeds x 3 h gives CV 13.7 percent; 12 soakfull seeds x 2 h gives 502/4985 = 10.1 percent; the xpgap off arm gives 594/5043 = 11.8 percent); two-sided 95 percent, 80 percent power where stated. Kills/h: original CV 21 percent, d2d CV 11 percent (V, early set).

Single d2d seed against the baseline mean. Two ways to count:

| treat the baseline mean as | noise (SD of one gap) | smallest gap seen at 95 % | smallest gap seen at 80 % power |
|---|---|---|---|
| exact (what `d2dsoak compare` does) | 12 % | 23.5 % | 33.6 % |
| exact | 14 % | 27.4 % | 39.2 % |
| uncertain (SE of the baseline mean 8.1 % added in quadrature, n = 3) | 14.5 % (12 % run) | 28.3 % | 40.5 % |
| uncertain | 16.2 % (14 % run) | 31.7 % | 45.2 % |

So a single seed cannot detect a 20 percent XP/h gap (even a gap of 25 percent is inside the noise), and the 20 percent tolerance is narrower than the noise of one run: a d2d that is truly equal to the baseline mean lands outside the +-20 percent band in 17 to 22 percent of single runs (D, normal model with SD 14.5 or 16.2 percent). The tool agrees by construction: fewer than 5 seeds is INCONCLUSIVE (V, spec "Verdict classes").

Many d2d seeds, 3 original runs. The d2d mean error shrinks as 1/sqrt n, the baseline error does not: with 8 d2d seeds the standard error of the gap is about 9.1 percent, and with only 3 original runs the Welch degrees of freedom are about 3.2, so the 95 percent interval of the gap is about +-28 percent (t about 3.1, D). The tool's PASS/FAIL ignores the baseline error (it treats the mean as exact, V, docs/d2d-soak.md section 3 says so), so a PASS means "no evidence of a gap above 20 percent" and a FAIL is only safe when the gap is large: the early set (-48 percent, 8 seeds) is a real FAIL against any plausible baseline mean; a gap between 20 and 35 percent would not be separable.

Baseline runs needed (D, t-quantile, SD 14 percent XP/h or 21 percent kills/h):

| goal: 95 % half width of the baseline mean at most | XP/h | kills/h |
|---|---|---|
| 20 % (equal to the tolerance) | 5 runs | 7 |
| 15 % | 6 | 11 |
| 10 % | 11 | 20 |

The spec's 5 seeds and the "10 to 15 percent spread" note in docs/d2d-soak.md section 3 size the d2d side and the baseline's spread, but not the baseline's own mean error; with 5 original runs the baseline interval would just reach the tolerance (+-20 percent) and no more.

Kills/h is the weakest gated metric: the baseline CV is 21 percent (min..max 421..620, which spans -16 to +24 percent of the mean), larger than the 20 percent tolerance, so a single original run can sit outside its own gate. The a14 value (421) is from an old build; a18 (620) is the best build but still below the report's own count (688) because the extractor counts per minute line and the report per 30 s sample (V, docs/d2d-soak.md section 2): the definition cancels only if both sides use the extractor.

Deaths: the Poisson part. The baseline has 0 deaths in 5.05 h; 0 events carry an upper 95 percent bound of 3 events / 5.05 h = 0.59 deaths/h (D, rule of three). The gate v2 cap (0.15/h) is below that bound, so the data cannot show a d2d rate under the cap, and 3 d2d deaths in 24 h (0.125/h) is the only reason the death metric passes in the early set (V, compare-baselines*.txt).

## 4. Confounds

### 4a. Bot drift since the baseline runs (V by merge log, effect size U)

The baselines were measured on bots de9b597 / b2e010c / b478bd9 (2026-10-29 and 10-30). The Gate 1 rerun used f4ca264 (2026-11-01). Between 2026-10-30 08:30 and 2026-11-01 13:10 there are 238 agent merges, 96 non-merge commits touching the bot, town, resource, combat and navigation packages (V, `git log` on this repo; the baseline build hashes themselves are not in this repo). The merged cards that plausibly move XP/h, kills/h or deaths (V: merge titles):

| area | cards (merge order) | likely direction on XP/h or kills/h (I) |
|---|---|---|
| potions and belt | fixbelt (F1 F3 F6-F9), fixbelt2, fixpickup, fixbelt (low kit bypasses the recovery cooldown), m1loot (buy order), kitgate | fewer empty-belt windows and fewer emergency trips; raises time in the field |
| town portal | fixportal, fixportal2, enterportal, fixportalmelee (no portal cast with 3+ adjacent hostiles), fixprogress (empty portal source marks the tome empty) | fewer on-foot walks; the no-cast-when-adjacent rule adds a safe delay |
| combat and casting | combatspin, fixmissile (CastGapMs 650), fixcastgap (per-skill gap: Lightning 890 ms on Game.exe, 650 ms on d2d), fixrecast, fixbuff, drills13/26/28 (armor recast), fixcrowd, fixmanamodel, fixskilllvl | changes the kill rate per minute; the bot now models d2d and Game.exe differently on purpose |
| melee and resist model | skmeleebot, skmeleewire, skmeleewire2 (melee blow fed from d2d gear stats), skresistbot (resist floor -100, pierce below 100) | affects the choice between melee and spell in the resource simulator (I) |
| navigation and town | fixnav, fixmouth/fixmouth2/fixmouth4 (cave-mouth bounce), roamleg, fixstall 1-4, townwedge, wpcooldown (waypoint cooldown removed), soakstall | fewer wedges and stalls; shorter walks |
| economy | fixprice (bot prices follow game.exe), xpgap knobs (default off) | small |

Net: the bot that Gate 1 judges is not the bot that produced the baseline, on a large number of behaviours that feed XP/h directly. This confound cuts two ways and cannot be separated without re-running the old binary on d2d (the paired recipe of docs/ab-design.md) or the new binary on Game.exe. A d2d-vs-baseline gap therefore mixes (engine difference) + (bot difference); the attribution rule of ADR 0009 ("a FAIL counts against d2d only when the same bot build's original-game baseline is inside the gate") is not met by any current comparison (V: the baseline builds are a14/a16/a18, every d2d set is a later build).

### 4b. Same kit, same profile, same clvl? The Gate 1 comparison is not like for like

- Gate 1 (and the early d2d set) starts a fresh uncheated clvl 1 Sorceress, class 4, per the recipe (V, docs/soaks/gate1-rerun.md). The Gate 1 rerun ran 120 wall minutes = 22.6 virtual hours, clvl 1 to 14 (V). The spec says a 2 h soak; the lockstep clock makes it about 11x longer in virtual time. The spec clause "true XP/h at or above the d2d baseline at the same clvl" was in practice evaluated as 7517 (whole run, clvl 1-14) against 9172 (the early mean, clvl 1-7 Game.exe) (V, gate1-rerun.md). That is not "the same clvl". XP/h rises steeply with clvl in this build (Game.exe: about 5 to 12k at clvl 1-7, 9 to 12k at clvl 11-14 for a14 to a18, 4.1k for the 8 h G2 at clvl 14-15 where progress is level-gated; d2d fresh: 4.8k at clvl 1-6, V), so the whole-run mean mixes bands that have different baselines.
- Which baseline applies to a fresh start? Only the early file, and only for the part of the run at clvl 1-7 (the first 2 to 3 virtual hours). A fresh d2d run compared with the late file is explicitly invalid (spec "Same character, same setup": fresh clvl 1 only against the early file, marked weak). A comparison of the full 22.6 h run has no valid baseline. The closest original data is the Fresha 4 h run (clvl 6 to 11, 13.4k XP/h), not in either file.
- Profile: the late file is profile m1 (2 h, death rule at m1), Fresha and G2 are m24. Kits differ per run (gold 0 / 211 / 16); the d2d fresh character starts with 0 gold and is "in recovery for the whole 2 hours by construction" (V, docs/d2d-xp-gap.md), which is also the Fresha pattern (56 percent recovery, V) but not the Soaky a14 to a18 pattern (0 to 12 percent recovery, V, states in the file). So the fresh-start baseline and the late baseline have different time-in-state profiles; the late file cannot be borrowed for a fresh start.
- The planned late baseline (card m2-latebase, deferred) would be built on d2d from a cheat-levelled clvl 12 character. It would calibrate the bot, not the engine: a baseline made on d2d cannot show d2d is faithful (I). Keep it as a regression baseline only.

### 4c. d2d known deviations that bias the compared metrics (V by the cited docs, direction I)

| deviation | source | metric affected | direction on d2d (I) |
|---|---|---|---|
| Blood Moor XP per kill +20 percent, Den +60 percent at clvl 4-6 | docs/d2d-fidelity.md check 4b (V-d2d against live numbers) | XP/h at clvl 1-6 (the whole early band) | d2d XP/h inflated, so the real engine gap in the early set is larger than the observed -48 percent |
| Blood Moor holds about 70 monsters on d2d against 90 on Game.exe | docs/research/per-kill-time-split.md (U for the density share) | kills/h, XP/h early | d2d lower |
| Cast period 14 ticks for every spell (Lightning and Chain Lightning -26 percent, bolts +8 percent) | d2d-fidelity check 7a (V-d2d) | kills/h of a Lightning build | d2d faster on Lightning; the bot now models the gap per skill (fixcastgap) but the engine does not |
| Tome of Town Portal 250 gold against 450, sell price ratio | d2d-fidelity check 5; fixprice | gold, trips, XP/h (62 percent of spend is portal scrolls, V, d2d-xp-gap.md) | d2d richer in scrolls, fewer poverty trips: XP/h up |
| Missing potion double-fill | check 1e | potions per hour, recovery trips | d2d heals less per potion, so more trips: XP/h down |
| Missile hit roll top-inclusive (SKMISS D1), freeze rule, curse on an immune target, plain-attack physical in 256ths, monster velocity, boss minion stats | docs/coverage/rule-coverage.md rows marked TESTED-DEVIATION (V by differential, effect on soak metrics not measured) | kills/h, deaths | unmeasured (U) |
| Monster base stats: 734 rows, none compared; XP award chain read, per-monster XP equal to d2d, no differential | rule-coverage.md (monster.stats 0 percent, F10 READ-ONLY) | XP/h, deaths | unmeasured (U) |
| Pacer watch interval up to 2 virtual s coarser than Game.exe | docs/d2d-soak.md section 5 | deaths/h | d2d deaths up |
| Wake radius only checked against d2d's own source | d2d-fidelity check 2a | kills/h, deaths | unknown (U) |

Several deviations have been fixed or are open PR candidates since the early set was taken, and the early set predates the fork rebase (V, gate1-rerun.md names the rebased binary), so the -48 percent figure is of unknown current size; the Gate 1 run's 7517 at a later clvl suggests it has shrunk (I, not comparable: section 4b).

### 4d. Other confounds

- No seed control on the original: each original run is a different dungeon, so spread mixes seed noise, build drift (3 builds) and clvl; the 14 percent CV is not a pure seed SD.
- a14 is truncated to its first 64 minutes (its XP/h of 12.2k excludes the idle failure that followed); a14 also carries the single highest XP/h and the lowest kills/h of the three (V), which are not typical of a full 2 h.
- a16 hours exclude the OOM restart window (111 of 126 wall minutes) while its XP loss counts (V, file note); a mixed definition.
- Hidden confound: the early d2d XP/h is `game_ms`-based and original runs are minute-line-based (section 1b), up to 16 percent apart by definition alone.

## 5. Verdict per gate criterion

| criterion | verdict | why |
|---|---|---|
| Gate 1 XP/h at or above the baseline at the same clvl (soak-gates) | DO NOT GATE ON (as a hard FAIL) | one seed, noise 24 to 32 percent; the clause is evaluated against the wrong clvl band (4b); baseline n = 2 (early). Report it as an indicator with its interval |
| Fidelity XP/h, early file, fresh clvl 1-7 start | TRUST WITH CAVEAT for large gaps only | the -48 percent early FAIL is real (8 seeds, interval +-11 percent); anything inside +-30 percent is not resolvable, and the file stays weak |
| Fidelity XP/h, late file, clvl 11-14 start | TRUST WITH CAVEAT | n = 3, baseline mean +-35 percent, one partial run, 3 builds; usable as "far outside the band" only; needs the baseline interval in the verdict |
| Fidelity kills/h (both files) | DO NOT GATE ON | baseline CV 21 percent is wider than the 20 percent tolerance; early kills are a lower bound (S); definition mismatch between report and extractor; engine deviations (density, cast period) bias it |
| Deaths per hour, deaths per 1000 kills, baseline-relative | DO NOT GATE ON | 0 deaths in 5.05 h because a14's death was cut and G2/Fresha are absent; pooled original rate from the g1-log runs (a14 1, a16 0, a18 0, G2 4, Fresha 4 incl. the early part, Soaky early 1) is 10 deaths in about 23.7 h = 0.42/h (D, S), nearly three times the 0.15/h cap |
| Deaths: the fixed budget (death-rule, gate v2: at most 1 per 6 virtual h, cap 0.15/h) | TRUST WITH CAVEAT | not baseline-derived, so sound; but stricter than the original's own measured rate, so a d2d miss is not by itself a fidelity bug (blame the bot first) |
| Stability, save semantics, fault recovery, route guard, portal, belt window, no-kill window (virtual minutes) | TRUST | measured per run from the minute lines, independent of the baselines; the 10 min and 5 min limits come from the G1 criteria and were never replayed on the original logs (U; recomputing BELT and NOKILL on the saved original minute lines costs nothing and would show if the limits are reachable on Game.exe) |
| potions, gold, state shares, repeat sites, clvl (info rows) | DO NOT GATE ON | already info-only; gold CV 640 percent, potions a14 is missing data |
| Same-setup rule (spec) | TRUST | correctly forbids fresh vs late, but the Gate 1 run breaks it once it passes clvl 7 |

## 6. What would fix the weakest spots

Wall-time numbers are inferences (I) from the soak tooling: an original run is real time (about 1 wall hour per virtual hour, up to 3 instances at once under the worker host capacity rule, 1 core each), a d2d run is about 11x faster (22.6 virtual h in 120 wall min, V) with 3 slots.

0. Free, do first (no new run): add the logged original runs that already exist. G2 (8.06 h, 4 deaths, clvl 14-15) and the Fresha 4 h run (clvl 6-11, 3 deaths) as `d2dsoak baseline` entries (the recipe is in docs/d2d-soak.md section 1), keep a14's full-run death by scoring it over the full 2 h instead of 64 minutes if its log is still on the worker host, and add the missing `weak` / `partial` fields. Raises total original hours from 5.05 to about 17 and the death evidence to 8 events, and gives the first clvl 6-11 mid band. Effort: a few minutes of extraction.
1. Deaths and a stable XP/h at clvl 11-14: 5 more honest-play original runs of 2 h on the current bot (8 runs total), different days, same kit rule (profile m1). Gets the XP/h mean interval to about +-17 percent at SD 14 percent (t df 7) and the original death rate to about 3 events per 8 h pooled with G2 (still wide: 10 hours per event at 0.1/h). Wall: 2 batches of 3 instances, about 5 h, needs the user's go because ADR 0013 retired Game.exe gates and the game host is shared (U).
2. Fresh clvl 1-7 baseline: 3 more fresh uncheated Sorceress runs of about 2 h each (5 total, ends the spec's weak flag), on the current bot, play-minute extraction, one definition for hours. About 2 to 3 h wall in one batch of 3 instances.
3. Kills/h: the CV of 21 percent needs 11 runs for +-15 percent. Not worth pursuing as a gate; demote kills/h to info until a baseline of 11 or more runs exists, or gate it only with the paired old-binary-on-d2d recipe (docs/ab-design.md: 8 seeds x 2 virtual h per arm, about 12 wall minutes per seed per slot, so about 40 minutes of wall for 8 seeds on 3 slots).
4. Bridge the bot drift without new original runs: run the baseline bot builds (a18 or the closest tag) on d2d next to the current build, 8 paired seeds x 2 virtual hours per arm (docs/ab-design.md; about 40 to 60 wall minutes on 3 slots). The difference separates "bot changed" from "engine differs" and is the one test that meets the ADR 0009 attribution rule. Needs the a18 build to be rebuildable (its hash is not in this repo, U).
5. The Gate 1 clause itself: replace "at the same clvl" by "per clvl band" (XP/h inside clvl 1-7 compared with the early file; later bands against the late file once built), or drop the XP clause from Gate 1 and keep it in the 8-seed fidelity run where the interval is meaningful. This is a spec change for the coordinator, not done here.

## 7. Limits of this audit

Everything is recomputed from committed files; the raw minute logs of a14, a16 and a18 were not re-opened (V only through the baseline file and docs/g1-log.md). The Gate 1 per-clvl XP breakdown was not available. The effect size of the bot drift and of the engine deviations on XP/h is not measured (U). Deaths pooled across builds assume one rate, which the builds do not share (I).

## Proposed text for the coordinator (not applied)

- fidelity-gate, `Minimum sample`: add that the baseline's own mean interval (n, SD, t) is printed with every verdict and that a baseline of fewer than 5 runs caps the verdict at INCONCLUSIVE for any gap under the baseline's own half width.
- soak-gates, `Gate 1 deaths and progression`: restate the XP clause per clvl band, or mark it indicative at one seed.

## After: baselines extended and the Gate 1 XP clause per clvl band (card baselinefree, 2026-11-01)

> Tags as above. V here = recomputed in this card with `d2dsoak` from the raw bot logs on the worker host (read-only copy of `soak24/runs` and `soak24-i5/runs` under the game host's live directory; the logs of 2026-10-29 and 2026-10-30 are all still there) or from committed files; D = arithmetic on those numbers. No new run was made. Reproduce: `d2dsoak baseline ...` (recipe in the "Reproduce" block below), then `d2dsoak bands -baselines tools/d2dsoak/baselines.json,tools/d2dsoak/baselines-early.json LOG`.

### What was added (V)

| row | file | log | hours | deaths | XP/h | kills/h | counted in whole-run means |
|---|---|---|---|---|---|---|---|
| a14-full (whole 2 h run, bot de9b597, 2026-10-29) | baselines.json | `soak24/runs/20261029-235659-d2bot.log`, 120 minute lines | 1.99 | 1 | 11336 | 395 | yes (replaces the 64-line a14 row) |
| a14 (first 64 lines) | baselines.json | same log | 1.06 | 0 | 12236 | 421 | no: `exclude_from_mean` (superseded) |
| g2-8h (m24, 2026-10-30, bot b478bd9 plus merges) | baselines.json | `soak24/runs/20261030-082115-d2bot.log`, 482 lines | 8.06 | 4 | 4074 | 506 | yes |
| fresha-1-11-chain (m24, 7527d07 then a920fef) | baselines-early.json | `soak24-i5/runs/` 095519 + 105216 + 120747, 353 lines, play-time hours | 5.90 | 4 | 12776 | 403 | no: bands only (it is the same run as fresha-fresh-1-7 plus its 4 h continuation) |

Every row now has `run_id`, `bot`, `backend`, `date`, `profile`, `weak`; the band tables are in `bands`. Corrections to the sections above, both V by the extractor: the audit's Fresha 4 h kills/h (590, from the g1-log total 2387) and G2's (880, from 7100 kills) used the counter totals of the g1-log; the extractor (sum of positive increments, the definition both sides share) gives 407 and 506 kills/h. The g1-log numbers stay as quoted there; the baseline rows use the extractor. a14 over its full 2 h: the death at about 01:40 and the 17 minute idle after it are now in, XP/h 11336 (the partial cut gave 12236, +8 percent).

### Counts and hours (V, D)

| file | runs counted | virtual hours (was) | deaths (was) | death rate |
|---|---|---|---|---|
| baselines.json (late) | 4: a14-full, a16, a18, g2-8h (was 3) | 14.04 (5.05) | 5 (0) | 0.36/h, 0.71 per 1000 kills over 7029 kills |
| baselines-early.json | 2, unchanged (Soaky S quality, Fresha) | 5.42 | 2 | 0.37/h |
| all original runs with minute lines, one count per run | a14-full, a16, a18, g2-8h, Fresha chain | 19.9 | 9 | 0.45/h |

The death evidence is now 9 events in 19.9 h of minute-line data (0.45/h), three times the 0.15/h cap, as the audit's pooled estimate (0.42/h, S) said; the late file alone is 0.36/h. The original game is not inside the death-rule cap, so a d2d miss of the cap is not by itself a fidelity bug (verdict table, row "Deaths: the fixed budget" stands).

### Spread and intervals recomputed (D, sample SD, t(0.975, 3 df) = 3.182)

| metric | before (3 runs) | after (4 runs, whole-run) | after without g2-8h (3 runs, a14-full) |
|---|---|---|---|
| XP/h mean, SD (CV), 95 % interval | 10901, 1524 (14.0 %), +-34.7 % | 8969, 3402 (37.9 %), +-60.4 % | 10601, 1180 (11.1 %), +-27.6 % |
| kills/h mean, SD (CV), 95 % interval | 500, 105 (21.1 %), +-52 % | 495, 95 (19.1 %), +-30.4 % | 491, 116 (23.5 %), +-58.5 % |
| XP/h, hours-weighted | 10630 | 6845 | |

Reading: G2 is a different regime (clvl 14 to 15, 6 of its 8 h at clvl 15 with the level-up goal gated by the route guard, m24 profile). Counting it as a fourth equal run drops the late XP/h mean by 15 percent and its CV to 38 percent, so the whole-run late mean is no longer a clean "clvl 11-14" number and is a worse gate than before for XP (the compare tool treats the mean as exact and does not see the wider interval). It does improve kills/h (the interval narrows because G2's kills/h sits near the mean) and the death evidence. Recommendation (I): judge XP per clvl band (below) and keep the whole-run late mean as information; the deaths and kills rows of the whole-run compare are better than before. The weak flags stay: still fewer than 5 runs in both files.

### Per-band baseline (V from the band tables, D for the statistics)

Bands are tiled (a level-up out of the band belongs to the band); XP/h is net XP over play time (restart gaps out); only runs with at least 0.5 h in the band count.

| band | baseline runs (n) | hours | deaths | XP/h per run | mean | CV | 95 % half width of the mean | single-seed tolerance (spec formula) | verdict class |
|---|---|---|---|---|---|---|---|---|---|
| 1-4 | 1 (Fresha) | 0.74 | 1 | 11089 | 11089 | floor 14 % | undefined | 38.8 % | ADVISORY |
| 5-7 | 1 (Fresha) | 1.80 | 0 | 13662 | 13662 | floor | undefined | 38.8 % | ADVISORY |
| 8-10 | 1 (Fresha) | 3.12 | 3 | 12531 | 12531 | floor | undefined | 38.8 % | ADVISORY |
| 11-14 | 4 (a16, a18, a14-full, G2) | 7.91 | 1 | 11807 / 9240 / 11336 / 7051 | 9859 | 22.1 % | +-35.2 % | 48.4 % | hard (FAIL possible) |
| 15-20 | 1 (G2) | 6.03 | 4 | 3074 | 3074 | floor | undefined | 38.8 % | ADVISORY |

Why these bands (I, from the data): XP/h is flat across clvl 1-10 on the Fresha chain (11.1k, 13.7k, 12.5k: the three early bands are within 20 percent of each other, so splitting them costs nothing and keeps the d2d early deviations visible per band) but falls at clvl 15 (3.1k) where the route guard gates progress, and the clvl 11-14 band is the level where a14 to a18 played. The old Soaky attempts 8 to 13 (2026-10-29, builds before the fixes of g1-log attempt 14) also cover clvl 8-11 (8-10 band: 6515 and 5274 XP/h in the two 2 h runs, 3 to 5 deaths in 2 h) but were NOT added: pre-fix builds, a different bot, and mixing them would put a 2x build effect into the band mean (this is the same bot-drift confound as section 4a, now visible inside one band). They are on the worker host if a later card wants them.

The single-seed tolerance uses the spec formula 1.96 x sqrt(CV^2 / k + CV^2 / n) with k = 1 d2d run, n the band's baseline runs and CV the larger of the band CV and 0.14; the 14 percent floor is the audit's section 3 value, the observed clvl 11-14 CV is 22 percent. At n = 4 the tolerance is wide (48 percent): a single Gate 1 run cannot resolve anything smaller there, which is the honest reading of the data.

Minimum sample per band (D, t-quantile, as in section 3): to bring the band mean to +-20 percent (the fidelity tolerance) the band needs 5 runs at CV 14 percent and 8 runs at the observed 22 percent (6 and 11 for +-15 percent, 10 and 21 for +-10 percent). The spec therefore marks a band ADVISORY below 3 runs (at n = 2 the interval is wider than the mean, section 1b) and gives a hard FAIL from 3 runs on with the wider tolerance above; "5 or more runs" ends the weak flag, not the hard/advisory split.

### Gate 1 re-scored per band (V, `d2dsoak bands -hours H`, the committed minute logs)

| d2d Gate 1 run (bot f4ca264 and earlier) | virtual h | 1-4 | 5-7 | 8-10 | 11-14 |
|---|---|---|---|---|---|
| gate1-rerun (2026-11-01, docs/soaks/gate1-rerun.md) | 22.6 | 7276 (-34 %, advisory) | 10539 (-23 %, advisory) | 11730 (-6 %, advisory) | 6191 over 15.8 h (-37 %, inside the 48 % tolerance: PASS) |
| gate1-20261101 (docs/soaks/gate1-20261101.md) | 24.67 | 6950 (-37 %) | 9760 (-29 %) | 3753 (-70 %, the stall windows) | 8371 (-15 %) PASS |
| postfix-soak2 (2026-10-31) | 21.25 | 5523 (-50 %) | 10037 (-27 %) | 9231 (-26 %) | 3324 over 13.1 h (-66 %): FAIL |

Reading: the old whole-run clause gave one number per run against the clvl 1-7 early mean (7517 vs 9172, -18 percent for the rerun; the same run is -6 to -34 percent per band and 11-14 is PASS). The per-band view shows where the gap sits: d2d is 23 to 50 percent below the Fresha chain in clvl 1-4 and 5-7 (consistent with the early -48 percent set, section 1b, and with the d2d deviations of 4c) but all of those rows are advisory because the early bands hold one baseline run each. The one hard band, clvl 11-14, fails only postfix-soak2, which spent 13 h there at 3.3k XP/h (the pre-fix run); the two later runs pass it. The honest summary (I): with the data we have, Gate 1 XP is a hard test only for clvl 11-14; the early-band verdicts need 3 fresh uncheated original runs (item 2 of the list above), which this card does not do.

### What the tools now do and do not do (V, `go test -p 1 ./cmd/d2dsoak ./internal/d2dsoakstat`)

- New `d2dsoak bands [-bands LO-HI,..] [-hours H] [-baselines F[,F]] LOG[+LOG]...` prints XP/h, kills/h, deaths per band and, with baselines, the band baseline (n, hours, mean, CV), tolerance, gap and verdict (exit 3 only on a FAIL, exit-zero overrides). The default bands are `1-4,5-7,8-10,11-14,15-20` and are the spec's.
- `d2dsoak baseline` takes `LOG+LOG` chains, a fifth `key=value` field for provenance and a `-bands` flag; rows with `exclude_from_mean` stay out of the whole-run means, the pooled death rate and the baseline-run count of `compare` (`Baselines.Counted`), but their bands count in `bands`.
- Not done: `compare` itself is unchanged (still whole-run, still treats the baseline mean as exact); `run` does not call `bands` yet; `gate-run.sh` (ops repo) still writes the whole-run XP row. Follow-up card: have `compare`/`run` print the band table and have the Gate 1 record generator use `bands` (the ops script is outside this repo's owns).

### Still missing

- Fresh clvl 1-10 original runs: bands 1-4, 5-7, 8-10 each rest on one run (Fresha); the Soaky early run has no minute lines. 2 more fresh uncheated runs make them hard.
- clvl 15+ rests on G2 alone; no original run reaches clvl 16.
- Bot hashes: a14, a16, a18, Fresha 7527d07/a920fef are recorded from the g1-log; G2's exact build (b478bd9 plus later merges) and the build of Fresha's first log are U.
- No original run has been repeated on the current bot (section 4a drift stays open).

Reproduce (on the worker host logs; recipe of docs/d2d-soak.md section 1 plus the new flags):
```
d2dsoak baseline 'a14-full|soak24/runs/20261029-235659-d2bot.log|SOURCE|NOTE|run_id=..,bot=de9b597,backend=game.exe,date=2026-10-29,profile=m1'
d2dsoak baseline 'g2-8h|soak24/runs/20261030-082115-d2bot.log|SOURCE|NOTE|run_id=..,backend=game.exe,date=2026-10-30,profile=m24'
d2dsoak baseline 'fresha-1-11-chain|soak24-i5/runs/20261030-095519-d2bot.log+soak24-i5/runs/20261030-105216-d2bot.log+soak24-i5/runs/20261030-120747-d2bot.log|SOURCE|NOTE|..'
d2dsoak bands -hours 22.6 -baselines tools/d2dsoak/baselines.json,tools/d2dsoak/baselines-early.json gate1-rerun-minutes.jsonl   # gunzip first
```
