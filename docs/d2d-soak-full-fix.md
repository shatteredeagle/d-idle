# d2d full-fix soak (change soakfull, 2026-10-30)

> status: current. Tags: V = measured live on hidden d2d (lab6, the worker host slots), U = inferred. Nothing was run on Game.exe.

## Method
- Tree agent/soakfull = master + livelockfix, astarfix, killfix (EmptyAreaSec and MouthGuard stay OFF), prochygiene, d2dfarm, slotscale, plus ONE hotfix (below). Bot `d2bot.soakfull`, d2d `d2d.lab6`, `D2D_RUN=on` explicit, killtime always on, no cheats.
- Fresh Sorceress `Loopy-sf-<seed>` per run, `--managed`, lockstep, 2 virtual hours, 4 concurrent in slots (`tools/soakfull/run.sh TAG SEED 2`, wrapped by `d2dslot.sh` under `agentrun.sh soakfull`, private dir `~/d2d-lab/soakfull/`). Load1 stayed 6 to 19. a Windows worker was not used (no node registry configured; the worker host was enough, a run takes 8 to 9 wall minutes).
- Judge: `d2dsoak compare -gate v2 -hours 2` vs `baselines-early.json` (clvl 1-7, XP/h 6600) and `baselines.json` (late, 10.9k XP/h, 500 kills/h, 0 deaths). Raw output in `docs/probes/soakfull/`.
- 12 valid runs (seeds 1-8, 12-15) = 24 virtual hours pooled; seed 11 hangs (see anomalies) and is excluded.

## Result (V)
| metric | d2d (12 runs) | early baseline | late baseline | verdict |
|---|---|---|---|---|
| XP/h | 4985 +- 145 SE (SD 502, 95% +-309) | 6600, gap -24.5% | 10901, gap -54.3% | early INCONCLUSIVE (interval straddles the 20% tol), late FAIL |
| kills/h | 195 +- 5 SE (SD 18.6) | n/a | 500, gap -61.1% | FAIL |
| deaths | 0 in 24.0 h | 0.29/h | 0 | PASS (budget 4, Poisson p=1.0) |
| deaths/1000 kills | 0 | no kill exposure | 0 | PASS vs late |
| potions/100 kills | 7.0 | - | - | proxy not comparable (baselines lack it) |
| min life fraction / dmg per kill | 0.70 / 3.1 | - | - | info |
Overall: INCONCLUSIVE vs early (exit 2), FAIL vs late (exit 3; but the late baseline is a clvl 11-14 character, not comparable, docs/d2d-soak.md 3b). Every run is under the tool's 3 h per seed minimum (warning); 24 h pooled meets the 6 h death exposure of gate v2.

clvl by virtual hour (minute-line index): 3.9 at 1 h (3 or 4), 4.9 at 2 h (4 or 5; seed 8 only 3 then 4). Per run in `runs.csv`: XP/h 3628 (s8) to 5606 (s2), kills/h 150 to 228.

## Kill time per kill (killtime, pooled 4727 kills): total 18.0 s = walk 13.8 + cast 0.9 + wait 3.3
By step (seed 1): "Recovery: farming gold" 64% of gap seconds (20 s/kill), "Quest: fighting" 21% (14 s), "Returning to the field" 7% (22 s). Gap classes: kills with gap <= 4 s are half of all kills and cost 0.7 s each overall; gaps over 30 s (about 13% of kills) carry most of the seconds. Longest gaps 240 to 350 s are almost all walk (about 90%: back and forth across Blood Moor / Cold Plains / Cave 1 borders, `nav exit`, `nav stuck`). Time in state: recovery 54%, town chores 28%, farming 15% (the character is poor and stays on the "Recovery: farming gold" step: poverty sell trips, "0 healing potions").

## Comparison with earlier measurements (same fresh-Sorceress setup, different bot builds, U: noise about +-10%)
| source | kills/h | XP/h | s/kill |
|---|---|---|---|
| KILLFIX baseline | 164 | 3830 | 22 |
| D2DLOOP replicated | - | 3013 | - |
| ECONOMY | - | 3869 | - |
| this soak | 195 | 4985 | 18.0 |
About +19% kills, +30% XP/h, -18% s/kill against KILLFIX. Walk share of kill time is still 77%; run policy (stamina) and the astar/killfix changes did not remove the walk, they moved it down (KILLFIX walk bucket not re-extracted here, U).

## Anomalies
1. **livelockfix 5f864c2 alone does NOT fix the clvl 3 livelock (V).** First batch (seeds 1-4, merged tree without hotfix): all four spun on "everything worth farming is cleared, starting a fresh game" (117 to 218 times in 3 minutes, virtual clock almost stopped at 21 to 46 minutes, clvl 2-3). Cause: on d2d the process restart is faster than the observer poll, `ob.newGames` never moves, `syncGames` never fires. Hotfix commit on this branch: `freshGameToLevel` clears `loop.cleared` itself after a successful NewGame (same idea as RUNAB 1f4b62f; mine also resets sweepKills/needBuff/stepArea; test `TestFreshGameForgetsClearedWithoutObserver`). After it: 0 to 2 "fresh game" lines per run. Merge note: the two fixes conflict textually, keep one.
2. **Seed 11 hangs deterministically (V)**: twice (same tag-independent result: clvl 5, 346 kills, 8808 xp, area 8) the bot freezes at virtual minute 109: `combat crowd: stepping away` then `combat move failed ... nav: no path: empty path`, over and over (25k lines in 4 min), lockstep never steps again. This is RUNAB's spin ("seed 42"). Logs `~/d2d-lab/soakfull/logs/hung-s11.log`, `hung-s11b.log` (18 MB). Both killed by their own process group. Runs with the same seed are deterministic (identical numbers), so this is a good offline repro: crowd of 8, life 34%, target (142,129).
3. Failsafe ladder fires: s8 20 (all "town: no town portal available (walk to town failed: the level changed during the walk)", poverty character with no TP scroll at clvl 3), s5 8, s1 s13 s14 1 each, others 0. Game restarts ("Save and Exit"): 11 to 36 per run, planned leave-game chores. "d2d already running" lines: 0 in all runs (ASTARFIX noise fix works, V). Panics: 0.
4. s8 is the weak run (150 kills/h, clvl 3 at 1 h): the poverty/no-TP failsafe loop above.

## What is still the gap and next suspects (docs/research/*.md)
1. Walk: 13.8 of 18 s per kill (per-kill-time-split.md, movement-and-recovery-trace.md): border ping-pong and nav stuck on the roam; route to dense spawns.
2. 54% of time in "recovery" step (farming gold with a poor character), only 15% real farming: economy/early-game policy (potions, TP scroll, poverty trips) rather than combat.
3. Casts per kill and cast period 14 ticks (gap-casts-per-kill.md, cast-trace-gameexe.md) bias kills/h by a few percent, not the 60% gap.
4. Spawn density per area (spawn-density-trace.md) and XP award (xp-award-trace.md, Den +60%) shift XP/h, not the kill rate.
5. Seed 11 stall: combat crowd step-away with an empty path must fall back to melee or hold position (owner: combat/nav).
