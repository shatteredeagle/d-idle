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

