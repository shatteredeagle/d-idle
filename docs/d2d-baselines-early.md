# Early baseline: Game.exe fields and the d2d fresh-character set (card m2-earlybase)

> status: current. Tags: V = computed from logs in this run, S = read from a doc, U = unverified. Gate files: `tools/d2dsoak/baselines-early.json`, spec fidelity-gate `Early baseline fields`. Raw output: `docs/probes/m2early/`.

## 1. Game.exe side (the baseline), 2 runs, weak
| run | XP/h | kills/h | deaths/h | deaths per 1000 kills | quality |
|---|---|---|---|---|---|
| Fresha (i5, 2026-10-30, clvl 1 to 7, 115 play minutes, 746 kills, 1 death) | 11743 | 389 | 0.52 | 1.34 | V (minute lines) |
| Soaky (i4, 2026-10-29, clvl 1 to 7, 3.5 h wall, 1 death) | 6600 | 408 | 0.29 | 0.70 | S (no minute lines; kills = sum of the kills column of docs/soak-analysis.md section 2, 1427, a lower bound) |

Mean used by the gate: XP/h 9172, kills/h 398, deaths/h 0.41, deaths per 1000 kills 1.02. The two runs differ by 1.8x in XP/h (Soaky's is wall time with a 25 min pause and three restarts), so the verdict stays weak until 5 runs, which cannot happen (Game.exe is frozen). The Soaky minute lines of clvl 1 to 7 do not exist (its logs start at clvl 8); only Fresha has them.
Extraction (V, on the worker host, logs copied read-only): minute lines of `soak24-i5/runs/` 20261030-095519 and 105216 whole plus the first 6 minute lines of 120747 (up to the first clvl 7 line) are concatenated, then `d2dsoak extract -hours 1.916667 all.log` (hours = 115 lines / 60, restart gaps excluded; without `-hours` the wall timestamps give 2.29 h and 9833 XP/h, 326 kills/h). Kills are the persistent per-character total of the minute line, so the 16 to 37 percent restart undercount of the death-rule spec does not apply to Fresha; a residual undercount (monsters never seen alive) is U. Fresha spent 56 percent of its minutes in state `recovery`.

## 2. d2d side: 8 fresh seeds x 3 virtual hours (V)
Fresh uncheated Sorceress (Loopy-me-N), hidden d2d lab6, lockstep, run policy on, `--goals campaign`, seeds 1 to 8, 3 slots at a time, current main (35ce4c2 plus this card). Per seed (d2dsoak extract, `docs/probes/m2early/extract.txt`):

| seed | XP/h | kills/h | deaths | clvl at end |
|---|---|---|---|---|
| 1 / 2 / 3 / 4 | 5421 / 5177 / 4180 / 5202 | 214 / 209 / 186 / 206 | 0 / 0 / 2 / 0 | 6 / 6 / 5 / 6 |
| 5 / 6 / 7 / 8 | 5205 / 4974 / 4435 / 3544 | 204 / 171 / 191 / 153 | 0 / 1 / 0 / 0 | 6 / 6 / 5 / 5 |

**Mean +- SE (n = 8): XP/h 4767 +- 230 (SD 651, CV 13.7 %, 95 % interval +- 544 = 11.4 %); kills/h 192 +- 7.5 (SD 21.1, CV 11.0 %); deaths 3 in 24 h (0.125/h; 0.65 per 1000 kills over 4599 kills).** Exclusions: none (8 of 8 seeds ran the full 3 h, exit 0, no wedge, no stall-ladder exit, no "no eligible area"). Flags kept, not excluded: every seed had 2 to 12 bot restarts and 2 to 5 fresh games (chicken exits, the run policy), the same in all earlier d2d sets.
Verdict against the 2-run baseline (`compare -gate v1 -hours 3`, exit 3): xp_per_hour FAIL (-48 %), kills_per_hour FAIL (-52 %), deaths_per_hour PASS, deaths_per_1000_kills INCONCLUSIVE (interval straddles the tolerance); baseline marked weak. Gate v2 gives the same XP and kills verdicts. So the M2 early gate is not met by the current bot: the gap to Game.exe (about 2x on both metrics) is far above the 20 % tolerance and far above the noise.

## 3. Sample size (ab-design rule, 95 % confidence, 80 % power)
For a 10 % effect on this setup the unpaired formula `2 * (2.8 * SD / (0.1 * mean))^2` gives 29 seeds per arm for XP/h and 19 for kills/h; with pairing (ab-design measured correlations 0.03 to 0.85) 8 to 34 seeds per arm depending on how much the change reshuffles the run (SD of differences must come from the A/B pilot). With 8 seeds the 95 % half width of one arm is 11 % (XP/h) and 9 % (kills/h) of the mean, so a single 8-seed set only proves effects of about 11 %; the observed -48 % gap needs 2 to 3 seeds. A 20 % tolerance verdict needs n with half width < 20 %: already met at 8.

## 4. Reproduce (on the worker host; fork and lab binary are private)
```
labrun <worker> -- go build -o /tmp/m2e-bin/d2bot ./cmd/d2bot          # also ./cmd/d2dsoak
# ship d2bot as ~/d2d-lab/m2early/bin/d2bot.m2early, tools/{d2dslot,agentrun,agentreap}.sh beside it
tools/agentrun.sh m2early --ttl 14400 -- ./all.sh 3 1 2 3 4 5 6 7 8     # all.sh: at most 3 runs at once, each:
  d2dslot.sh run -- d2bot --backend d2d --managed --d2d-class 4 --char Loopy-me-S --seed S --toggle autoloot=off,autoparty=off \
     --d2d-args --lockstep --goals campaign --run-for 120m            # env D2DSOAK_HOURS=3 D2D_RUN=on; stops at 3 virtual h
d2dsoak extract -hours 3 bot-base?.log
d2dsoak compare -gate v1 -hours 3 -baselines tools/d2dsoak/baselines-early.json bot-base?.log
```
