# Charged Bolt per-bolt damage on d2d (card chargedbolt, 2026-11-04)

> status: current. Evidence: docs/probes/sksmoke/2026-11-04-SK-cb.txt, .json, -raw.txt. Tag: compared to table on d2d (lab fork binary `d2d.lab-debug-skills`), NOT game.exe.

## Result
Charged Bolt (skill 38) PASS at levels 1, 4, 8, 13, 20: per-bolt mean 3.00, 4.52, 6.51, 9.00, 14.52 against the table means 3.0, 4.5, 6.5, 9.0, 14.5 (all within 0.5 percent), 42 to 120 hits per level (V ran), every single-bolt sample inside the table range +-0.5 (S for the table: docs/probes/sksmoke/2026-11-03-SK-design.txt).

## The method (V)
`monsters` gives the life of a monster as a whole point, so one cast shows only the sum of the bolts that hit. No new fork verb is needed: `debug missiles` lists one `chargedbolt` row per bolt and a bolt that hits leaves the list in that frame. Step one frame at a time (`step 1`) after `debug cast 38 unit <id>` and read `monsters` and `debug missiles` each frame:
- k = rows before minus rows after, d = life drop of the target in that frame.
- d > 0, k = 1: an exact per-bolt sample. d > 0, k >= 2: d / k per bolt (mean over all hits = total drop / hits, unbiased). d = 0: the bolt expired or hit nothing, discarded.
- Bolts per cast read from the first frame: 3, 6, 10, 15, 22 (= 3 + lvl - 1).
Script: `docs/probes/sksmoke/2026-11-04-SK-cb-gen.sh` (probe string), `-cb.sh` (runs on the worker host through `d2drill -probe`), `-cb-an.py` (analysis).

## Limits and dead ends
- Setup writes (stats, `points 38`, `mana full`, spawn, `mon set` big life and resists 0) come before the first cast; afterwards only the test action `debug cast` and reads. A first pass refilled mana between casts; that was a write after start and the numbers here are the rerun without it.
- Lockstep is deterministic: an identical invocation repeats exactly (V, two runs matched). Level 1 needed more hits (about 0.5 per cast), so the second invocation idles 37 frames first.
- Only about one in three bolts hits at level 1 and the rest expire (S, from the vanish events): the zombie is a small target and the wiggle path wanders. Not a damage question.
- Resolution is whole life points: a single sample is the true value +-1 at the range edge (none fell outside +-0.5 anyway).
- The roll deviation D1 (docs/research/sk-missile-trace.md: d2d rolls min + rand(max - min + 1), game.exe max - min) is far below this resolution; the PASS does not rule it out (S).
- `d2cheat runstart` belongs to the game.exe path; the d2d probe ledger (actor `drills:probe`) has no run marker (U, not tried).
