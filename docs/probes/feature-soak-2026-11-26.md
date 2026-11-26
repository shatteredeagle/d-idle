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
## 3. Results (pass 1: 1 virtual hour per row, 19 of 69 rows complete)

### 3.1 Per feature and class

| feature | class | n on/off | xp/h on vs off (median) | gold/h | deaths | hard hits on/off | stalls on/off | short rows on/off | noise (off spread) | verdict |
|---|---|---|---|---|---|---|---|---|---|---|
| arbiter-control-families-3 | paladin | 3/3 | 4451 vs 4482 (-31) | 274 vs 3048 (-2773) | 0.0 vs 0.0 | 0/0 | 19/26 | 1/0 | xp 7792 gold 2566 deaths 0.0 | RED (gold/h regress -2773) |
| arbiter-control-families-3 | necromancer | 3/3 | 1121 vs 1349 (-228) | -97 vs 2310 (-2407) | 0.0 vs 0.0 | 24/0 | 19/26 | 0/0 | xp 819 gold 2069 deaths 0.0 | RED (hard hits 24>0; gold/h regress -2407) |
| arbiter-control-families-3 | sorceress | 3/3 | 1580 vs 936 (+645) | -119 vs -131 (+12) | 1.0 vs 0.0 | 0/0 | 27/26 | 0/0 | xp 409 gold 1433 deaths 2.0 | GREEN |
| arbiter-control-families-255 | paladin | 3/3 | 6295 vs 4482 (+1813) | 864 vs 3048 (-2184) | 0.0 vs 0.0 | 0/0 | 5/26 | 3/0 | xp 7792 gold 2566 deaths 0.0 | NEUTRAL |
| arbiter-control-families-255 | necromancer | 3/3 | 1254 vs 1349 (-95) | -52 vs 2310 (-2362) | 0.0 vs 0.0 | 0/0 | 15/26 | 2/0 | xp 819 gold 2069 deaths 0.0 | RED (gold/h regress -2362) |
| arbiter-control-families-255 | sorceress | 1/3 | 144 vs 936 (-792) | -360 vs -131 (-229) | 0.0 vs 0.0 | 0/0 | 2/26 | 1/0 | xp 409 gold 1433 deaths 2.0 | RED (xp/h regress -792) |
| kit-floor-tiers-poverty-farm | paladin | 3/0 | no data | | | | | | | NO-DATA |
| kit-floor-tiers-poverty-farm | necromancer | 3/0 | no data | | | | | | | NO-DATA |
| kit-floor-tiers-poverty-farm | sorceress | 3/0 | no data | | | | | | | NO-DATA |
| admission-early-route-guard | paladin | 3/0 | no data | | | | | | | NO-DATA |
| admission-early-route-guard | necromancer | 3/0 | no data | | | | | | | NO-DATA |
| admission-early-route-guard | sorceress | 3/0 | no data | | | | | | | NO-DATA |
| typed-outcomes | paladin | 3/0 | no data | | | | | | | NO-DATA |
| typed-outcomes | necromancer | 3/0 | no data | | | | | | | NO-DATA |
| typed-outcomes | sorceress | 3/0 | no data | | | | | | | NO-DATA |
| reflex-drink-portal | paladin | 3/0 | no data | | | | | | | NO-DATA |
| reflex-drink-portal | necromancer | 3/0 | no data | | | | | | | NO-DATA |
| reflex-drink-portal | sorceress | 3/0 | no data | | | | | | | NO-DATA |
| paladin-without-sacrifice | paladin | 3/0 | no data | | | | | | | NO-DATA |
| necromancer-summoner-defaults | necromancer | 3/0 | no data | | | | | | | NO-DATA |


### 3.2 Per row

| arm | class | seed | virtual h | wall min | ratio | clvl | deaths | xp/h | gold/h | potions bought | potions used | hard hits | soft hits | stalls | exit | writes after start |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| arbiter255 | necromancer | 201 | 0.98 | 9 | 6.7 | 12>12 | 0 | 1254 | -109 | 18 | 23 | 0 | 26 | 7 | 0 | 0 |
| arbiter255 | necromancer | 202 | 0.37 | - | - | 12>12 | 0 | 2005 | 158 | 12 | 14 | 0 | 11 | 4 | None | None |
| arbiter255 | necromancer | 203 | 0.37 | - | - | 12>12 | 0 | 843 | -52 | 7 | 10 | 0 | 10 | 4 | None | None |
| arbiter255 | paladin | 201 | 0.22 | - | - | 12>12 | 0 | 6295 | -563 | 10 | 12 | 0 | 4 | 3 | None | None |
| arbiter255 | paladin | 202 | 0.08 | - | - | 12>12 | 0 | 3660 | 22476 | 5 | 3 | 0 | 1 | 1 | None | None |
| arbiter255 | paladin | 203 | 0.08 | - | - | 12>12 | 0 | 18144 | 864 | 5 | 3 | 0 | 1 | 1 | None | None |
| arbiter255 | sorceress | 201 | 0.08 | - | - | 12>12 | 0 | 144 | -360 | 3 | 2 | 0 | 2 | 2 | None | None |
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
