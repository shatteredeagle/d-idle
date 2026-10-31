# Measured effect of the missile-damage-fine fix on casts per kill

> status: current (agent PRMEASURE, 2026-10-30). Tags: V-d2d = measured on private lockstep d2d builds, hidden, `--no-save`, throwaway Sorceress; nothing run on Game.exe; nothing pushed or posted.

## Method (V-d2d)

- Same probe and configuration as docs/d2d-casts-real.md: `cmd/d2dcasts2` (on agent/gaptime), Blood Moor (`-area 2`), Charged Bolt 1 / 4 / 8 at clvl 2 / 5 / 9, `-castgap 450`, seeds 1-4, `-virtual 15m`, god top-up per Clear. Real casts = mana spent / skill cost (3.0 / 4.5 / 6.5).
- Three binaries, all lab5 source (217dd20): `before` = the lab binary `d2d.lab5`; `after` = lab5 + PR commit 91a7cc92 (PRBUILD's `/tmp/prbuild-d2d.new`); `control` = unpatched lab5 source built by me with the same cmake flags as `after` (Release, `-DD2D_WERROR=OFF`, `/tmp/prmeasure-ctl-b`), to separate patch from build/load effects. 36 runs, 3 slots at once, `nice -n 10`, via `d2dslot.sh`. Raw rows: `docs/probes/prmeasure.csv` (`tag` + the d2dcasts2 columns).
- `before` and `after` ran interleaved (same seeds side by side); `control` ran afterwards, under heavier the worker host load (load average 12 to 22 vs about 3).

## Result (Fallen rows; n = Fallen kills)

| CB level | before: casts / cmds per kill (n) | control: casts / cmds (n) | after: casts / cmds (n) | prediction |
|---|---|---|---|---|
| 1 | 1.20 / 2.04 (198) | 1.82 / 3.39 (210) | 1.55 / 2.79 (182) | 1.0 |
| 4 | 0.75 / 1.26 (234) | 0.78 / 1.31 (206) | 0.75 / 1.23 (203) | 0.65 |
| 8 | 0.60 / 0.99 (199) | 0.60 / 1.01 (224) | 0.52 / 0.86 (211) | 0.57 |

- Sanity: no regression in killing. Kills per run, all monsters, before / control / after: CB1 349 / 362 / 325, CB4 383 / 348 / 354, CB8 353 / 390 / 376 (4 seeds); xp per kill identical (21.6, 21.5, 8.9 to 9.1). Every run exited 0.
- Casts per kill over all monsters, before / control / after: CB1 1.74 / 2.11 / 2.26, CB4 1.06 / 1.03 / 0.99, CB8 0.92 / 0.83 / 0.78.

## Reading it

- Only Charged Bolt 8 shows an effect of the predicted size: 0.52 vs 0.60 (control and before agree at 0.60), about 13 % fewer casts (prediction 0.57, a bit more than predicted).
- Charged Bolt 4: 0.75 vs 0.75 and 0.78, no measurable change (prediction 0.65 was not met; the bootstrap standard error per cell is about 0.05).
- Charged Bolt 1 is NOT a clean comparison: the unpatched control build gave 1.82 against the lab binary's 1.20, so something other than the patch (build or the worker host load while it ran; commands per kill 3.39 vs 2.04 point at command timing) moved it by 0.6 casts. Against its own control the patch gives 1.55 vs 1.82, but the two did not run under equal load. Do not quote CB1.
- Per-seed values swing 0.5 to 2.0 at CB1 and 0.5 to 0.8 at CB8, so the cells are noisy; whole-cast granularity and overkill cap the gain, as predicted.

## Rerun attempt (2026-10-30, blocked)

- Build recipe check: lab5 binary `d2d.lab5` is byte-identical to `build-lab5/apps/d2d/d2d` of the private fork; its cmake cache and the control's agree on build type (Release), `D2D_WERROR`, compiler (GCC 16.2.1, `/usr/bin/c++`), flags (none); source is the lab5 commit 217dd20 (clean tree) for both. Only the build directory path differs, so the 1.82 vs 1.20 gap is NOT explained by the recipe: load or timing noise is the remaining candidate (U).
- Interleaved 8-seed, 3-arm rerun (`tools/prmeasure/rerun.sh`, one slot at a time, load gate < 6 for up to 40 min) was started but only seed 1 CB1 ran (before/control/after, load 4.2 to 7.8): the host never went below load 6 again (10 to 28) and all 8 slots were then held by other agents' soaks, so the rest queued for the whole time box and I stopped it. No new numbers beyond the first table.
- To rerun: `bash tools/prmeasure/rerun.sh` on the worker host when slots are free (needs `/tmp/prbuild-d2d.new` and `/tmp/prmeasure-ctl-b`, both still present until /tmp is cleaned).

## Caveats

- Lockstep d2d is virtual-time, but the probe's pacer and the 6 minute `-wall` are real-time, so heavy host load changes command timing. A clean CB1 needs a rerun of all three binaries interleaved on a quiet host.
- Fallen only; monsters differ in life, so other types gain differently. Mana attribution is per kill row (lumpy), only means over many rows count.
- The patched d2d and control are private throwaway builds in `/tmp` (lost on reboot).

## Commands

```sh
# on the worker host, tools in ~/d2d-lab/prmeasure-tools (copies of tools/prmeasure/*.sh)
bash tools/prmeasure/runall.sh     # before + after, 24 runs, 3 slots
bash tools/prmeasure/runctl.sh     # control, 12 runs (build recipe: PRBUILD's draft, from git archive of wt-lab5)
```

## Measured effect (paragraph for the PR body)

On the same 36-run lockstep probe (Blood Moor Fallen, Charged Bolt 1 / 4 / 8, 4 seeds, about 200 Fallen kills per cell), the fix lowers real casts per kill at Charged Bolt 8 from 0.60 (lab5, and an unpatched rebuild) to 0.52, about 13 % fewer. At Charged Bolt 4 the change is within noise (0.75 vs 0.75 / 0.78, standard error about 0.05). Charged Bolt 1 was disturbed by host load (the unpatched rebuild read 1.82 vs 1.20 for the lab binary) and is not claimed. Kills and xp per kill are unchanged, so nothing regressed. The effect is small and only one level is clearly visible; the PR rests on the game.exe evidence, not on this measurement.
