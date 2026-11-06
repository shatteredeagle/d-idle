# A/B protocol for bot changes on d2d (change m2-abdesign)

> status: current. Tags: V = computed from committed data with `tools/ab/paired.py`, U = inference. Contract: docs/contracts/fidelity-gate.md "A/B protocol for bot changes" (delta in openspec/changes/m2-abdesign until merged). Evidence base: docs/d2d-xp-gap.md (xpgap, 2026-10-31) and its raw output `docs/probes/xpgap/ab*.txt`. Nothing here was run on Game.exe; shared-code effects are d2d-only until a Game.exe soak says otherwise.

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

## Variants
The standard recipe (fresh clvl 1 character, 2 virtual hours) only reaches about clvl 8. An effect that lives at a higher level cannot be seen there. A variant changes the START state, never the statistic: pairing, interleaving, 8 seeds and the interval rule stay as above. Tags: V = run in this form on the worker, U = not yet measured.

### Late start (the merc revive reserve, task abmerc)
Why: a merc dies and a revive matters from about clvl 12, and g2f reached clvl 16 only after about 25 virtual hours, so a clvl 1 run of 2 hours cannot test merc.reviveReserve*.

- **Start state, identical in both arms.** A prepared hero: Sorceress, clvl 11 with the experience of level 11, all stat and skill points unspent (the bot's own leveler spends them), 500 gold in the purse, empty stash, the Act 1 Rogue hired for free (`debug merc hire 0`), starting kit as a fresh character. 500 gold is below the revive quote at clvl 11 (about 900, S), so the first death makes the reserve matter. Made by `tools/ab/prep` (V, ran in the dry run): the lab d2d runs WITH `--lab-verbs`, the verbs go through a ledger file (`prep-ledger.jsonl`, reason `SETUP`), then `save` and `quit`.
- **No write after the start.** The arm loads that save with a d2d WITHOUT `--lab-verbs`; the fork refuses gated debug verbs there (docs/lab-verbs.md), so a gameplay write is impossible by construction. Each run log ends with `writes after start: N` (count of refused setup verbs seen, 0 expected).
- **Horizon.** 4 virtual hours per seed (the 2 hour fidelity minimum is for clvl 1 runs; here the interesting part starts after the first death). A run that has not reached clvl 17 by then reports XP gained instead of the time to clvl 17.
- **Seeds.** 11 13 17 19 21 23 25 27 (8 pairs), arms interleaved: the two arms of a seed run side by side on two slots (`tools/ab/mercab.sh`). Arm names in the logs are `new` (A) and `old` (B).
- **Per seed record** (`tools/ab/mercan.py`): XP/h and kills/h over the elapsed virtual time (headline, read by `paired.py --base old --arm new`), merc alive share of the minutes, merc deaths, revives, `revive skipped` lines, player deaths, clvl reached, virtual minute of the first minute line at clvl 17.
- **Statistic.** XP/h and kills/h by the paired interval. The merc columns are mechanism evidence, reported as paired means, not tested separately.
- **Limits (U).** The start is a cheated hero, so a result applies to the clvl 11 to 17 stretch with this kit; it says nothing about the climb to clvl 11. The prepared character has the starting life of a fresh Sorceress until it levels or rests (V in the dry run: life 40 at the first minute, 136 on the first zone entry), the same in both arms.

Arms of the merc A/B, exactly (`tools/ab/mercrun.sh`, written into the instance `tuning.json`):

| arm | merc.reviveReserveLow | merc.reviveReserveHigh | merc.reviveReserveClvl |
|---|---|---|---|
| A `new`: the SME rule of #276 (the defaults) | 0 | 50 | 13 |
| B `old`: the old policy, emulated | 155 | 310 | 13 |

B is the closest the knobs express, not the 2026-11-02 rule. The old rule kept `merc.kitReservePct` (100 percent) of the live full-kit cost, which changes with the shop prices and the potion tier (S: about 155 at the low levels and 310 later, from the g2f seed 17 log), and halved it while the kit was full. B uses those two sums as fixed gold per clvl band. Also not reproducible by a knob: the old spending order (potions first, then the merc) is code; in both arms the revive runs before potions and repair. So B tests "a reserve of one kit refill versus almost none", not the old ordering. Name that limit in any claim.

Run plan, dry-run result and numbers: docs/ab-merc.md.

## Standing rule: when a paired A/B is required
A change that claims to be a strategy improvement (a rule that changes WHAT the bot does to earn more XP or lose less, not the repair of something that does not work) gets a paired A/B (at least 8 seeds, paired mean with its 95 percent interval) before its long gate. A bug fix gets a short drill in its exact failing scenario instead. If the effect lives at a level a clvl 1 run does not reach, use a variant above (late start). Checked in the review of the task: the brief or result names the A/B record, or says "bug fix: drill <id>".
