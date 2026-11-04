# retreatlive2: RT.drink and RT.portal rerun after retreatfix (2026-11-04)

d2d, not game.exe. Lab binary `d2d.lab-cheats3`, slot 3 (soak slots 0 to 2 untouched), lockstep, throwaway character, `d2drill -group RT -only X -repeat 2`. Recipe: docs/probes/retreat/2026-11-04-RT2-run.sh. Logs: docs/probes/retreat/2026-11-04-RT2-{drink,portal}.{txt,json}. Runs took seconds (V: live).

| drill | verdict | runs | decision | ticks act / effect | life floor | belt / scrolls |
|---|---|---|---|---|---|---|
| RT.drink | PASS | 2, identical | drink-life | 0 / 4 | 120 (start 120 of 300, end 121) | belt 4 to 3 |
| RT.portal | PASS | 2, identical | town-portal | 0 / 1 | 102 (start 102 of 300) | scrolls 1 to 0, portal listed |

- RT.drink setup life is 120 of 300 (40 percent), between ChickenPct and DrinkLifePct; the earlier FAIL (chicken at 20 percent) is gone (V). Life did not drop in the window (no hit), so the floor equals the start.
- RT.portal effect at 1 tick, well inside the 10 tick bound. Act steps the cast inside lockstep, so the count is ticks after Act (S).
- Bounds: drink 4 of 50 ticks, portal 1 of 10. BOUND-QUESTION: none; the 50 tick drink bound is loose (proposal 10, U).
- Writes: setup cheats only (clearinv, goto, kill, spawn, stat 6/7, belt clear and give tsc for portal), each ledgered by the drill before the Act. `d2cheat runstart` is not called on the d2drill path (cmd/d2drill has no runstart), so no run marker exists to query `ledger --since-run`; writes after start: 0 by construction (S: drill code only calls Tick, Snapshot, PView after Act). Not a measured run.
- Character ended alive in town (life 125/300 in the last drink run, 102 after portal).
- RT.leave stays skipped (Fighter.Leave unwired).
