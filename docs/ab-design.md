# A/B protocol for bot changes on d2d (change m2-abdesign)

> status: current. Tags: V = computed from committed data with `tools/ab/paired.py`, U = inference. Contract: openspec/specs/fidelity-gate "A/B protocol for bot changes" (delta in openspec/changes/m2-abdesign until merged). Evidence base: docs/d2d-xp-gap.md (xpgap, 2026-10-31) and its raw output `docs/probes/xpgap/ab*.txt`. Nothing here was run on Game.exe; shared-code effects are d2d-only until a Game.exe soak says otherwise.

## Why a protocol
xpgap compared three knobs with 4 seeds per arm and 2 virtual hours per run and found nothing beyond noise: SE is 7 to 9 % of the mean, so only effects of 15 % or more were visible. Two flaws of that recipe are fixed here: it compared arm means as if the arms were independent (the seeds are shared and the fork is lockstep-deterministic, so the per-seed difference is the unit), and it never stated how many seeds a claim needs.

## The protocol
1. **One binary, one knob toggle.** The before arm and the after arm differ only in the knob (or in the code path behind it); same character spec, same toggles, same d2d fork. No off-by-default knob may be the end state (docs/roadmap.md): the toggle exists for the A/B and is removed or defaulted on with the change.
2. **Paired seeds.** Both arms run the same seed list. The analysis unit is the per-seed difference `after - before` of XP/h and of kills/h (`tools/ab/paired.py`, input = the `tools/xpgap/an.py` run lines).
3. **Interleaved arms.** Per seed the two arms start back to back on the slot pool (`tools/ab/run.sh "off on" 1 ... 8`, which wraps `tools/xpgap/ab.sh` and refuses fewer than 8 seeds), so load drift of the host hits both arms alike. Never run all of one arm first.
4. **At least 8 seeds per arm, 2 virtual hours per seed** (the fidelity-gate minimum hours). A seed that is excluded in either arm (d2dsoak "excluded seeds": crashed, short, empty) is dropped from both, so the pairs stay complete.
5. **The statistic.** Paired mean of the differences with a two-sided 95 % Student-t interval (`n - 1` degrees of freedom, `mean +- t * SD_diff / sqrt(n)`). An effect is **claimed only when the interval excludes 0**; the size reported is the paired mean over the before-arm mean. Anything else, and any run with fewer than 8 seeds whatever the interval, is reported as `no effect shown` together with the seeds per arm the SD requires. `paired.py` exits 0 only on a claim, 1 on `no effect shown`, 2 on bad input.
6. **Both metrics.** XP/h is the headline, kills/h is reported beside it. A change that raises one and lowers the other is a trade-off to explain, not a win. Deaths are reported but too rare per run to test (0 in all 24 xpgap runs); the Poisson death gate of docs/d2d-soak.md is the tool for them.

## Sample-size rule
Seeds per arm for a relative effect `e` (usually 10 %): `n = ((1.96 + 0.84) * SD_diff / (e * mean_before))^2`, 95 % confidence, 80 % power, with one Student-t correction for the small n (`paired.runs_needed`; the unpaired variant, `2 * (...)^2` with the per-run SD, is printed beside it for comparison). `SD_diff` is the SD of the per-seed differences, measured from a pilot of at least 4 seeds and then re-measured on the full run; the floor is 8 seeds. A 4-seed SD has 3 degrees of freedom (about 40 % uncertain), so plan with 1.3 times the pilot SD when the cost allows.

Pairing helps exactly as much as the two arms' seeds stay correlated, and that depends on how much the change reshuffles the run. Measured (V, XP/h, seeds 1-4 or 5-8, `paired.py`):

| after arm | corr of the arms across seeds | SD of differences | seeds per arm, paired, 10 % | unpaired formula |
|---|---|---|---|---|
| on (recoveryXPEarly + povertySellMin, changes the first field) | 0.03 | 1052 | 34 | 33 |
| sell (povertySellMin) | 0.59 | 639 | 17 | 35 |
| tp (town.tpMinGold, 1 purchase in 8 runs differs) | 0.85 | 402 | 8 | 39 |
| norec (recovery off, large effect) | 0.95 | 276 | 7 | 32 |

So a local change (one decision in town) needs about 8 to 17 seeds, a change that alters the early route about 35, and the unpaired recipe about 35 to 40 either way. With 4 seeds per arm nothing under about 15 % is visible, which agrees with xpgap. U: the figure xpgap quotes ("about 35 runs per arm for 5 %") matches this formula's 10 % row; for 5 % the same SD needs about four times as many (about 130 for the changes that reshuffle the run), so 5 % effects are not worth chasing with this tool and should be argued from the mechanism (trips per hour, s per kill) instead.

## Worked example: the xpgap A/B re-analysed (V)
Seeds: the off arm ran seeds 1 to 8; `on` used seeds 1-4, `sell`, `tp` and `norec` seeds 5-8, so each after arm is paired with the same-seed off runs (checked from the run names in `docs/probes/xpgap/ab*.txt`; `bot-tp5` and `bot-off5` are even identical, the knob never fired on that seed). Commands: `python3 tools/ab/paired.py --arm on docs/probes/xpgap/ab1.txt`, and `--arm sell|tp|norec` with `ab2-sell.txt`, `ab3-tp.txt`, `ab4-norec.txt`.

| arm vs off | n | XP/h paired mean | 95 % interval | kills/h paired mean (interval) | protocol verdict |
|---|---|---|---|---|---|
| on (1-4) | 4 | -6 (-0.1 %) | -1679 .. +1668 | -7.5 (-32 .. +17) | no effect shown (4 < 8 seeds) |
| sell (5-8) | 4 | +176 (+3.6 %) | -842 .. +1193 | +8.8 (-22 .. +40) | no effect shown |
| tp (5-8) | 4 | -161 (-3.3 %) | -800 .. +479 | -4.8 (-20 .. +10) | no effect shown |
| norec (5-8) | 4 | -728 (-15.1 %) | -1168 .. -289 | -23.0 (-43 .. -3) | no effect shown by the rule (4 < 8 seeds), but the interval already excludes 0 and the mechanism agrees: re-run with 8 seeds before quoting it |

What changes against the xpgap reading: the verdict on the three fixes stands (no effect shown), the pairing exposes how different they are (tp is nearly the baseline run again, on is a different run per seed), and norec, the only visible effect, is a regression whose paired interval is much tighter than Welch's p 0.18 suggested. For a 10 % effect on this setup the runs per arm are in the table above: 34 for the `on` change, 8 to 17 for the town-only ones.

## Not done here
- `cmd/d2dsoak compare` and `tools/d2dloop` were not changed: the card owns only `tools/ab/**` and this doc. `paired.py` reads `an.py` lines; a native `d2dsoak` paired mode would need its own card (U: small, the statistic is 30 lines).
- The protocol was not run live; the arms of the worked example are the existing xpgap runs.
