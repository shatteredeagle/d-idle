# Feature soak campaign, accelerated lab (2026-11-26, card featuresoak)

> status: current. Everything here is a CHEATED/lab result: lockstep d2d lab slots, a prepared hero (declared SETUP verbs before the start, none after: `writes after start: 0` in every row), the lab Andariel has 1024 life. Use it for mechanics and relative A/B between arms only, never as live parity or balance evidence. Tags: V verified in this run (how named), S secondary, U unverified.

## 1. How to run the accelerated soak (for every new dark feature)

Files: campaign `tools/provingground/campaigns/features-2026-11-26.json` (copy it, edit `arms` and `features`), runner `tools/provingground/featuresoak.py`, one row `tools/provingground/featuresoak-row.sh` (a derived `abrun.sh`: prep hero, lockstep d2d without lab verbs, `d2bot --goals campaign`, `D2DSOAK_HOURS` virtual hours).

1. Build once on the worker host (from a checkout synced there; the lab binary is named in the campaign): `python3 tools/provingground/featuresoak.py build CAMPAIGN` (writes `~/d2d-lab/featuresoak/bin`). A feature that is a code or data change without a knob needs an arm with `"_bot": "<name>"` and a binary `bin/d2bot.<name>` built from a tree with that commit taken back (the `bots` key says how the three used here were made).
2. Look at the plan: `python3 tools/provingground/featuresoak.py plan CAMPAIGN [--features f1,f2] [--classes paladin] [--seeds 201,202] [--hours H] [--ratio R]` (rows and the wall cap per row: hours*60/R*1.6+10 minutes).
3. Run on the worker host, always through the process registry, detached: `tools/agentrun.sh <agent> --ttl SECS -- python3 tools/provingground/featuresoak.py run CAMPAIGN --out ~/d2d-lab/featuresoak/out-DATE [filters]` (at most `parallel` rows at once, default 6; a row whose log is complete is skipped, so a rerun resumes). Orion needs nothing else; slots come from `tools/d2dslot.sh` (7, one per row).
4. Read: `python3 tools/provingground/featuresoak.py report CAMPAIGN OUTDIR [--md FILE]` prints the per-row table and the per-feature table.

How to read the tables. Per row: virtual hours reached (minute lines), wall minutes, ratio (virtual over wall), clvl start to end, deaths, xp per virtual hour, gold plus stash gold per virtual hour, healing/mana potions bought (`shop: bought` lines), potions used, hard invariant hits (names in `cmd/d2scenario/invariants.json` `hard`), soft hits, stalls (`stall: goroutine stack` and `campaign stalled` lines), exit code, `writes after start`. Per feature and class: medians of ON and OFF over the seeds, the noise (the largest distance of an OFF seed from the OFF median) and the verdict: RED = more hard hits ON than OFF, a failed or written row, or a regression beyond max(noise, 5 percent) in xp/h or gold/h, or deaths up by more than max(noise, 1); GREEN = no RED reason and an improvement beyond the noise; NEUTRAL = no RED reason and no difference beyond the noise (safe to flip, unproven benefit). With 3 seeds the noise is a rough range, not a confidence interval.

`no-earn` and `town-deadlock` are not invariant rows on origin/main (they were planned names); the nearest monitor rows are `xp-frozen` and `mode-no-exit` (hard), `economy-stall` and `town-thrash` (soft), and they are in the tables.

## 2. Stage 0: calibration, the highest stable speed

There is no speed knob in the lockstep lab: the bot's pacer steps the game, so a row runs as fast as the bot computes. `SPEED` in `wave.sh` only sizes the wall cap. What was measured (one fixed row: Paladin, seed 201, clvl 12 hero, 200 gold, defaults, 1 virtual hour, lab d2d devp29, Orion with other users' jobs on it, load 8 to 12 of 16; V from the row logs):

| run | mode | virtual min reached | wall s | ratio (virtual over wall) | note |
|---|---|---|---|---|---|
| calib-a | lockstep, nav trace on | 58 | 1149 | 3.0 | bot at about 125 percent of a core, d2d 45 percent (V: ps) |
| calib-b | lockstep, nav trace off | 58 | 1184 | 2.9 | trace off does not speed it up; the rows use trace off |
| calib-b2 | same as b | 56 at the cut | n/a | about 3 | repeat for determinism |
| calib-ts8 | d2d `--time-scale 8`, no lockstep, `D2BOT_TIMESCALE=8`, 0.25 h asked | 12 | 788 | 0.9 | slower than lockstep, and stopped short of the asked length: not usable |

Reading: the stable speed is lockstep at roughly 3 virtual minutes per wall minute for a Paladin row in the field with a loaded host (the earlier 18x smoke was town and an empty field). Speeds of 8, 16, 32 or 64 are not reachable: the limit is the bot's own compute, not the game. Caster rows run faster (the first necromancer rows read about 6x). 24 virtual hours per row would be 8 wall hours; with 69 rows and 6 in parallel that is about 90 wall hours, so this pass is 1 virtual hour per row (U whether a longer run changes a verdict: the slow invariants `economy-stall` (7200 s) and `xp-frozen` (1800 s) mostly cannot fire in 1 h).

Determinism (V, minute lines of calib-a, b, b2): the same seed and arm gives identical xp, gold and area at virtual minutes 10, 20 and 30 in all three runs (xp gained 1180, 1672, 2037), a and b stay identical to minute 45 (2989), b2 differs from minute 45 (2068), and a and b differ at minute 58. So a seed is a repeatable start but not a repeatable run; the A/B noise below is the spread between seeds and reruns, and one seed per arm is never a verdict.

Hero names must be 15 characters or fewer (the first attempt with 18 characters failed the prep save: "the written save didn't read back"). Class indexes of the lab d2d (V: prep and bot started each class): Paladin 2, Necromancer 1, Sorceress 4.

<!-- results -->
## 3. Results (pass 1: 1 virtual hour per row, 58 of 69 rows complete)

### 3.1 Per feature and class

| feature | class | n on/off | xp/h on vs off (median) | gold/h | deaths | hard hits on/off | stalls on/off | short rows on/off | noise (off spread) | verdict |
|---|---|---|---|---|---|---|---|---|---|---|
| arbiter-control-families-3 | paladin | 3/3 | 4451 vs 4482 (-31) | 274 vs 3048 (-2773) | 0.0 vs 0.0 | 0/0 | 19/26 | 1/0 | xp 7792 gold 2566 deaths 0.0 | RED (gold/h regress -2773) |
| arbiter-control-families-3 | necromancer | 3/3 | 1121 vs 1349 (-228) | -97 vs 2310 (-2407) | 0.0 vs 0.0 | 24/0 | 19/26 | 0/0 | xp 819 gold 2069 deaths 0.0 | RED (hard hits 24>0; gold/h regress -2407) |
| arbiter-control-families-3 | sorceress | 3/3 | 1580 vs 936 (+645) | -119 vs -131 (+12) | 1.0 vs 0.0 | 0/0 | 27/26 | 0/0 | xp 409 gold 1433 deaths 2.0 | GREEN |
| arbiter-control-families-255 | paladin | 3/3 | 3363 vs 4482 (-1119) | 1119 vs 3048 (-1928) | 0.0 vs 0.0 | 0/0 | 27/26 | 0/0 | xp 7792 gold 2566 deaths 0.0 | NEUTRAL |
| arbiter-control-families-255 | necromancer | 3/3 | 1254 vs 1349 (-95) | 321 vs 2310 (-1989) | 0.0 vs 0.0 | 0/0 | 26/26 | 0/0 | xp 819 gold 2069 deaths 0.0 | NEUTRAL |
| arbiter-control-families-255 | sorceress | 3/3 | 2482 vs 936 (+1546) | 2615 vs -131 (+2746) | 0.0 vs 0.0 | 0/0 | 27/26 | 0/0 | xp 409 gold 1433 deaths 2.0 | GREEN |
| kit-floor-tiers-poverty-farm | paladin | 3/3 | 4482 vs 3538 (+944) | 3048 vs 4922 (-1874) | 0.0 vs 0.0 | 0/0 | 26/25 | 0/0 | xp 7845 gold 4801 deaths 0.0 | NEUTRAL |
| kit-floor-tiers-poverty-farm | necromancer | 3/3 | 1349 vs 1104 (+245) | 2310 vs 76 (+2234) | 0.0 vs 0.0 | 0/1 | 26/27 | 0/0 | xp 715 gold 1402 deaths 1.0 | GREEN |
| kit-floor-tiers-poverty-farm | sorceress | 3/3 | 936 vs 947 (-11) | -131 vs 656 (-787) | 0.0 vs 0.0 | 0/0 | 26/24 | 0/0 | xp 794 gold 650 deaths 1.0 | RED (gold/h regress -787) |
| admission-early-route-guard | paladin | 3/3 | 4482 vs 7008 (-2525) | 3048 vs -28 (+3076) | 0.0 vs 0.0 | 0/0 | 26/30 | 0/0 | xp 5648 gold 829 deaths 1.0 | GREEN |
| admission-early-route-guard | necromancer | 3/3 | 1349 vs 1008 (+341) | 2310 vs 264 (+2046) | 0.0 vs 0.0 | 0/0 | 26/28 | 0/0 | xp 374 gold 1650 deaths 0.0 | GREEN |
| admission-early-route-guard | sorceress | 3/3 | 936 vs 1381 (-445) | -131 vs -131 (+0) | 0.0 vs 0.0 | 0/0 | 26/29 | 0/0 | xp 1051 gold 2106 deaths 0.0 | NEUTRAL |
| typed-outcomes | paladin | 3/3 | 4482 vs 4180 (+302) | 3048 vs 3048 (+0) | 0.0 vs 0.0 | 0/0 | 26/26 | 0/0 | xp 8287 gold 2971 deaths 0.0 | NEUTRAL |
| typed-outcomes | necromancer | 3/3 | 1349 vs 1349 (+0) | 2310 vs 1896 (+414) | 0.0 vs 0.0 | 0/0 | 26/24 | 0/0 | xp 819 gold 2483 deaths 0.0 | NEUTRAL |
| typed-outcomes | sorceress | 3/3 | 936 vs 527 (+409) | -131 vs -131 (+0) | 0.0 vs 0.0 | 0/0 | 26/25 | 0/0 | xp 409 gold 1433 deaths 1.0 | NEUTRAL |
| reflex-drink-portal | paladin | 3/2 | 4482 vs 4541 (-59) | 3048 vs 1393 (+1655) | 0.0 vs 0.0 | 0/0 | 26/18 | 0/0 | xp 781 gold 246 deaths 0.0 | GREEN |
| reflex-drink-portal | necromancer | 3/3 | 1349 vs 1164 (+185) | 2310 vs -119 (+2429) | 0.0 vs 0.0 | 0/0 | 26/26 | 0/0 | xp 503 gold 1012 deaths 0.0 | GREEN |
| reflex-drink-portal | sorceress | 3/3 | 936 vs 2455 (-1519) | -131 vs 464 (-595) | 0.0 vs 0.0 | 0/0 | 26/19 | 0/3 | xp 1330 gold 624 deaths 1.0 | RED (xp/h regress -1519) |
| paladin-without-sacrifice | paladin | 3/0 | no data | | | | | | | NO-DATA |
| necromancer-summoner-defaults | necromancer | 3/0 | no data | | | | | | | NO-DATA |


### 3.2 Per row

| arm | class | seed | virtual h | wall min | ratio | clvl | deaths | xp/h | gold/h | potions bought | potions used | hard hits | soft hits | stalls | exit | writes after start |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| arbiter255 | necromancer | 201 | 0.98 | 9 | 6.7 | 12>12 | 0 | 1254 | -109 | 18 | 23 | 0 | 26 | 7 | 0 | 0 |
| arbiter255 | necromancer | 202 | 0.97 | 29 | 2.0 | 12>12 | 0 | 1613 | 682 | 35 | 42 | 0 | 33 | 10 | 0 | 0 |
| arbiter255 | necromancer | 203 | 0.97 | 22 | 2.7 | 12>12 | 1 | 591 | 321 | 34 | 32 | 0 | 19 | 9 | 0 | 0 |
| arbiter255 | paladin | 201 | 0.97 | 29 | 2.0 | 12>12 | 0 | 3363 | 1119 | 28 | 54 | 0 | 27 | 10 | 0 | 0 |
| arbiter255 | paladin | 202 | 0.98 | 37 | 1.6 | 12>12 | 0 | 3281 | 3482 | 27 | 30 | 0 | 11 | 9 | 0 | 0 |
| arbiter255 | paladin | 203 | 0.97 | 25 | 2.4 | 12>12 | 0 | 5326 | 12 | 25 | 38 | 0 | 16 | 8 | 0 | 0 |
| arbiter255 | sorceress | 201 | 0.97 | 22 | 2.6 | 12>12 | 0 | 624 | -80 | 19 | 17 | 0 | 18 | 9 | 0 | 0 |
| arbiter255 | sorceress | 202 | 0.97 | 18 | 3.2 | 12>12 | 0 | 2707 | 4649 | 33 | 37 | 0 | 35 | 9 | 0 | 0 |
| arbiter255 | sorceress | 203 | 0.97 | 21 | 2.7 | 12>12 | 0 | 2482 | 2615 | 54 | 59 | 0 | 26 | 9 | 0 | 0 |
| arbiter3 | necromancer | 201 | 0.98 | 11 | 5.5 | 12>12 | 0 | 650 | -97 | 6 | 9 | mode-no-exit:23 | 8 | 2 | 0 | 0 |
| arbiter3 | necromancer | 202 | 0.95 | 29 | 2.0 | 12>12 | 0 | 1159 | 1392 | 41 | 38 | 0 | 25 | 9 | 0 | 0 |
| arbiter3 | necromancer | 203 | 0.98 | 15 | 3.9 | 12>12 | 0 | 1121 | -141 | 22 | 28 | mode-no-exit:1 | 18 | 8 | 0 | 0 |
| arbiter3 | paladin | 201 | 0.97 | 28 | 2.0 | 12>12 | 0 | 4451 | 274 | 32 | 47 | 0 | 25 | 8 | 0 | 0 |
| arbiter3 | paladin | 202 | 0.98 | 38 | 1.6 | 12>12 | 0 | 3971 | 3157 | 44 | 48 | 0 | 17 | 8 | 0 | 0 |
| arbiter3 | paladin | 203 | 0.30 | 8 | 2.2 | 12>12 | 0 | 11617 | -273 | 15 | 24 | 0 | 5 | 3 | 0 | 0 |
| arbiter3 | sorceress | 201 | 0.97 | 29 | 2.0 | 12>12 | 1 | 689 | -122 | 18 | 26 | 0 | 33 | 10 | 0 | 0 |
| arbiter3 | sorceress | 202 | 0.98 | 28 | 2.1 | 12>12 | 0 | 1876 | -55 | 28 | 37 | 0 | 41 | 10 | 0 | 0 |
| arbiter3 | sorceress | 203 | 0.98 | 18 | 3.4 | 12>12 | 1 | 1580 | -119 | 28 | 35 | 0 | 29 | 7 | 0 | 0 |
| base | necromancer | 201 | 0.97 | 21 | 2.8 | 12>12 | 0 | 1349 | 2310 | 47 | 46 | 0 | 33 | 9 | 0 | 0 |
| base | necromancer | 202 | 0.98 | 33 | 1.8 | 12>12 | 0 | 530 | 4379 | 21 | 27 | 0 | 33 | 8 | 0 | 0 |
| base | necromancer | 203 | 0.97 | 17 | 3.4 | 12>12 | 0 | 2077 | 251 | 48 | 50 | 0 | 33 | 9 | 0 | 0 |
| base | paladin | 201 | 0.97 | 28 | 2.1 | 12>12 | 0 | 4482 | 482 | 28 | 47 | 0 | 22 | 9 | 0 | 0 |
| base | paladin | 202 | 0.97 | 38 | 1.5 | 12>12 | 0 | 4180 | 3048 | 42 | 47 | 0 | 8 | 8 | 0 | 0 |
| base | paladin | 203 | 0.98 | 41 | 1.4 | 12>12 | 0 | 12275 | 4119 | 26 | 60 | 0 | 29 | 9 | 0 | 0 |
| base | sorceress | 201 | 0.97 | 30 | 1.9 | 12>12 | 2 | 1291 | -132 | 30 | 36 | 0 | 34 | 10 | 0 | 0 |
| base | sorceress | 202 | 0.97 | 27 | 2.2 | 12>12 | 0 | 527 | 1301 | 27 | 32 | 0 | 30 | 10 | 0 | 0 |
| base | sorceress | 203 | 0.98 | 17 | 3.5 | 12>12 | 0 | 936 | -131 | 18 | 28 | 0 | 17 | 6 | 0 | 0 |
| earlyroute0 | necromancer | 201 | 0.97 | 20 | 2.9 | 12>12 | 0 | 1008 | 1915 | 39 | 39 | 0 | 32 | 11 | 0 | 0 |
| earlyroute0 | necromancer | 202 | 0.97 | 22 | 2.6 | 12>12 | 0 | 633 | -144 | 36 | 38 | 0 | 32 | 9 | 0 | 0 |
| earlyroute0 | necromancer | 203 | 0.98 | 25 | 2.3 | 12>12 | 0 | 1220 | 264 | 31 | 32 | 0 | 21 | 8 | 0 | 0 |
| earlyroute0 | paladin | 201 | 0.95 | 27 | 2.1 | 12>12 | 0 | 10663 | -131 | 36 | 63 | 0 | 35 | 10 | 0 | 0 |
| earlyroute0 | paladin | 202 | 0.98 | 35 | 1.7 | 12>12 | 1 | 1360 | -28 | 20 | 31 | 0 | 33 | 9 | 0 | 0 |
| earlyroute0 | paladin | 203 | 0.97 | 18 | 3.2 | 12>12 | 0 | 7008 | 801 | 53 | 51 | 0 | 29 | 11 | 0 | 0 |
| earlyroute0 | sorceress | 201 | 0.98 | 19 | 3.1 | 12>12 | 0 | 1381 | -131 | 38 | 45 | 0 | 53 | 10 | 0 | 0 |
| earlyroute0 | sorceress | 202 | 0.98 | 23 | 2.6 | 12>12 | 0 | 331 | 1975 | 40 | 41 | 0 | 34 | 10 | 0 | 0 |
| earlyroute0 | sorceress | 203 | 0.98 | 26 | 2.3 | 12>12 | 0 | 1898 | -139 | 18 | 39 | 0 | 23 | 9 | 0 | 0 |
| nokit | necromancer | 201 | 0.97 | 21 | 2.8 | 12>12 | 1 | 1104 | 76 | 30 | 32 | life-zero-alive:1 | 22 | 9 | 0 | 0 |
| nokit | necromancer | 202 | 0.98 | 32 | 1.9 | 12>12 | 0 | 711 | -112 | 17 | 26 | 0 | 21 | 9 | 0 | 0 |
| nokit | necromancer | 203 | 0.98 | 23 | 2.5 | 12>12 | 0 | 1818 | 1478 | 41 | 41 | 0 | 30 | 9 | 0 | 0 |
| nokit | paladin | 201 | 0.98 | 36 | 1.6 | 12>12 | 0 | 2895 | 121 | 20 | 33 | 0 | 14 | 9 | 0 | 0 |
| nokit | paladin | 202 | 0.98 | 35 | 1.7 | 12>12 | 0 | 3538 | 4922 | 24 | 31 | 0 | 9 | 8 | 0 | 0 |
| nokit | paladin | 203 | 0.98 | 37 | 1.6 | 12>12 | 0 | 11383 | 9461 | 33 | 62 | 0 | 22 | 8 | 0 | 0 |
| nokit | sorceress | 201 | 0.97 | 31 | 1.9 | 12>12 | 1 | 947 | 656 | 15 | 23 | 0 | 20 | 9 | 0 | 0 |
| nokit | sorceress | 202 | 0.97 | 27 | 2.1 | 12>12 | 0 | 818 | 869 | 21 | 28 | 0 | 28 | 9 | 0 | 0 |
| nokit | sorceress | 203 | 0.98 | 23 | 2.6 | 12>12 | 0 | 1741 | 6 | 28 | 40 | 0 | 26 | 6 | 0 | 0 |
| reflexdrink0 | necromancer | 201 | 0.98 | 28 | 2.1 | 12>12 | 0 | 1667 | -123 | 26 | 39 | 0 | 38 | 8 | 0 | 0 |
| reflexdrink0 | necromancer | 202 | 0.97 | 26 | 2.3 | 12>12 | 0 | 1049 | 893 | 37 | 40 | 0 | 29 | 10 | 0 | 0 |
| reflexdrink0 | necromancer | 203 | 0.97 | 15 | 3.8 | 12>12 | 0 | 1164 | -119 | 14 | 27 | 0 | 23 | 8 | 0 | 0 |
| reflexdrink0 | paladin | 201 | 0.98 | 26 | 2.2 | 12>12 | 0 | 5323 | 1147 | 28 | 36 | 0 | 17 | 9 | 0 | 0 |
| reflexdrink0 | paladin | 202 | 0.95 | - | - | 12>12 | 0 | 3760 | 1639 | 37 | 50 | 0 | 21 | 9 | None | None |
| reflexdrink0 | sorceress | 201 | 0.63 | - | - | 12>12 | 0 | 1124 | 1088 | 15 | 22 | 0 | 22 | 7 | None | None |
| reflexdrink0 | sorceress | 202 | 0.65 | - | - | 12>12 | 1 | 2797 | -23 | 19 | 25 | 0 | 19 | 7 | None | None |
| reflexdrink0 | sorceress | 203 | 0.57 | - | - | 12>12 | 0 | 2455 | 464 | 15 | 21 | 0 | 11 | 5 | None | None |
| typedoutcome0 | necromancer | 201 | 0.97 | 21 | 2.8 | 12>12 | 0 | 1349 | 1896 | 47 | 46 | 0 | 33 | 9 | 0 | 0 |
| typedoutcome0 | necromancer | 202 | 0.98 | 33 | 1.8 | 12>12 | 0 | 530 | 4379 | 21 | 27 | 0 | 33 | 8 | 0 | 0 |
| typedoutcome0 | necromancer | 203 | 0.98 | 19 | 3.1 | 12>12 | 0 | 1454 | -108 | 22 | 33 | 0 | 21 | 7 | 0 | 0 |
| typedoutcome0 | paladin | 201 | 0.97 | 23 | 2.5 | 12>12 | 0 | 4079 | 77 | 26 | 35 | 0 | 21 | 9 | 0 | 0 |
| typedoutcome0 | paladin | 202 | 0.97 | 37 | 1.6 | 12>12 | 0 | 4180 | 3048 | 42 | 47 | 0 | 8 | 8 | 0 | 0 |
| typedoutcome0 | paladin | 203 | 0.97 | 40 | 1.5 | 12>12 | 0 | 12468 | 4190 | 26 | 59 | 0 | 29 | 9 | 0 | 0 |
| typedoutcome0 | sorceress | 201 | 0.97 | 27 | 2.1 | 12>12 | 1 | 318 | -142 | 22 | 29 | 0 | 43 | 9 | 0 | 0 |
| typedoutcome0 | sorceress | 202 | 0.97 | 28 | 2.0 | 12>12 | 0 | 527 | 1301 | 27 | 32 | 0 | 30 | 10 | 0 | 0 |
| typedoutcome0 | sorceress | 203 | 0.98 | 18 | 3.3 | 12>12 | 0 | 936 | -131 | 18 | 28 | 0 | 17 | 6 | 0 | 0 |
