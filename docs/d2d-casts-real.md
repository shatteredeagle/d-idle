# d2d casts per kill, real casts (agent GAPTIME, 2026-10-30)

> status: current. Confirms agent/gapcasts 21f0dd4 (`docs/research/gap-casts-per-kill.md`, cause 1). Tags: V-d2d = measured on the private lockstep fork (`d2d.lab5`, Normal, hidden, `--no-save`, throwaway Sorceress, under `/tmp/d2d-lab.lock`); U = not checked. Nothing run on Game.exe.

Probe: `cmd/d2dcasts2` (copy of `cmd/d2dcasts` of docs/d2d-casts.md, plus `-castgap` and two time columns), driver `docs/probes/run-d2d.sh`, tables `~/d2lod-live/combat/data`. Per level (Charged Bolt 1 / 4 / 8, clvl 2 / 5 / 9) and per `castGapMs` (450 as shipped, 650): seeds 1 to 4, 15 virtual minutes each (a run ends early when the Blood Moor is cleared; the Fighter's own `Clear` radius 60). Raw rows: `docs/probes/d2d-casts-real.csv` (`area,seed,cb_level,clvl,monster,boss,xp,cast_cmds,mana_spent,castgap_ms,t_ms,dt_ms`; `t_ms` is virtual time since the first Clear, `dt_ms` since the previous kill row).

A real cast = mana spent / skill cost. Cost from Skills.txt (GAPCASTS, V): 3.0 / 4.5 / 6.5 mana at Charged Bolt 1 / 4 / 8. Mana is the Meter's sum of drops between samples, attributed to the row of the kill (a fight's casts are split across its rows, so only means over many rows count).

## Result (V-d2d, Fallen rows)

| CB level | castGapMs | Fallen kills | commands per kill | real casts per kill (mana / cost) | commands per real cast |
|---|---|---|---|---|---|
| 1 | 450 | 156 | 2.03 | **1.16** | 1.75 |
| 1 | 650 | 220 | 1.24 | **1.19** | 1.04 |
| 4 | 450 | 224 | 1.21 | **0.73** | 1.66 |
| 4 | 650 | 215 | 0.77 | **0.75** | 1.03 |
| 8 | 450 | 232 | 0.97 | **0.61** | 1.60 |
| 8 | 650 | 225 | 0.64 | **0.62** | 1.02 |

- **Confirmed (V-d2d):** at 450 ms the fighter issues 1.6 to 1.75 commands per real cast, the rest are dropped by d2d mid-cast. At 650 ms (longer than the 560 ms cast) commands per cast fall to 1.02 to 1.04: a command is a cast. Commands per kill at 650 ms, 1.24 / 0.77 / 0.64, are GAPCASTS' predicted "about 1.4 / 0.7 / 0.6".
- **Real casts per kill do not depend on castGapMs** (1.16 vs 1.19, 0.73 vs 0.75, 0.61 vs 0.62): the dropped commands cost nothing but log noise. Raising castGapMs to 650 would not speed d2d up (and is not recommended: it only delays the next real command).
- Real casts per Fallen kill, 1.16 / 0.73 / 0.61, against the original's commands per kill 0.96 / 0.90 / 0.69 (docs/orig-casts.md, branch agent/origprobes 90b09fb, commands on the original may also be dropped, U): the gap at Charged Bolt 4 and 8 is gone or reversed (d2d needs fewer casts), and at Charged Bolt 1 it is 1.16 vs 0.96 (about 0.2 casts, 0.1 s per kill), within the noise of 51 original kills. The "2.7x more casts" of docs/d2d-casts.md was the command artifact.
- Time per kill row in this probe (virtual seconds to the last kill / rows, 4 seeds): CB1 6.2 (450) / 4.7 (650), CB4 5.5 / 5.6, CB8 5.5 / 4.6. Original probe, real seconds per row 5.1 / 6.8 / 4.0 (GAPCASTS, 7 minutes per level). The Fighter alone on d2d is about as fast per kill as on the original; GAPCASTS' "d2d 10.9 s per row" assumed 3600 s per level, but runs end after 260 to 650 s (area cleared, no respawn), so the real figure is about half (V from `t_ms`). The per-kill gap is outside the Fighter: docs/research/per-kill-time-split.md.

## Caveats

- Mana and life topped up each Clear (as in docs/d2d-casts.md), so no mana waits. Area depletes within a run: per-kill time is an average over the cleared part, not steady state.
- Mana is sampled at pacer sleeps, so a cast between two samples with regeneration can be missed; the three CB levels agree with the expected 1 command per cast at 650 ms, which bounds the error to a few percent (U).
