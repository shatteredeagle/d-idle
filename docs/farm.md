# Farm runs (FARMRUNS, 2026-10-29)

Post-campaign play: repeat boss runs for XP or drops. Everything is **U** (offline fake tests only, never run live); the
xp points in `farm-runs.json` are V arithmetic from `docs/merc-farm.md` 2.1, all other numbers are estimates.

## Pieces
| Piece | File | What |
|---|---|---|
| Run table | `docs/plan/data/farm-runs.json` (copy in `internal/plandata/`, kept equal by `TestMatchesDocs`) | per run: steps (KillBoss names), secs, xp points per difficulty, drop weights (runes, uniques), `tp_mult`, `class_mult`, `needs` flags. Floors and act come from `runs.json` (reuse) unless `min_clvl` overrides. |
| Scheduler | `internal/boss/farm_sched.go` | `LoadFarmTable`, `FarmTable.Rank/Pick(FarmChar, goal, skip)`; score = value per hour (xp interpolated by clvl, or drop weight) divided by run time, slowed when the character cannot teleport or the class is penalised. |
| New scripts | `internal/boss/farm_scripts.go` | Pindleskin (Harrogath, Anya red portal object 60, area 121), Eldritch (Frigid Highlands wp). `FarmRunner` = `Act45Runner` plus these two; Mephisto, Andariel, Countess, Shenk, Baal (waves) are the existing scripts. |
| Loop | `internal/boss/farm_loop.go` | `FarmLoop`: per run `Char` -> pick -> `NewGame` -> `Chores` -> steps -> loot (scripts loot and ToTown themselves). Stops: `Limits{Runs, Duration, Deaths, ConsecFails}`, no allowed run, ctx. |
| Goal | `internal/goals/runs.go`, `goal.go` | type `runs`, text `runs:NAME:N` (`runs:auto:N` = scheduler, `_` = space); progress = boss kills (one per completed run) of N, optional `UntilMins`. |

## Interfaces the host must supply
`FarmOps` (adapter in bot/cmd): `Char` (level, class, difficulty, can teleport, flags such as `anya_portal`), `NewGame`
(leave + create game, arrive in town; OOG tooling), `Chores` (heal, repair, sell, restock, merc revive via internal/town).
`BossKiller` = `FarmRunner`. A completed run must emit one boss-kill event so the goal progress moves.

## Selection rules (from the table; U)
- Hell clvl under 82: Mephisto (XP), Baal from 82 (floor), Baal stays best at 85 although the 256/1024 ratio bites.
- Runes goal: Countess. Uniques goal: Pindleskin (needs `anya_portal`), then Mephisto/Baal.
- A run failing twice in a row is skipped for the rest of the loop (not when `Fixed`); `ErrTooWeak` skips without counting.

## Wiring (WIREFARM)
Done offline, tests only (no live run): `enqueueGoals` parses `runs:NAME:N` (goals.ParseRuns); `bot.Runner.compile` accepts `goals.Runs` when `TaskDeps.Farm` is set; `farmStep` (internal/bot/farm.go) runs the whole loop as one action through `TaskDeps.Farm`, with a `farmOps` adapter (Char from the snapshot, NewGame through the Cycler inside the OOG bracket, Chores = ToTown if needed + full chore plan). `boss.FarmOpts.OnRun` fires after every counted run; a killed run emits one boss-kill event, which moves the goal's progress (Count kills = done). `cmd/d2bot/boss.go farmLoop` builds boss.FarmLoop over the game and `actRunner` (now embeds boss.FarmRunner so Pindleskin/Eldritch resolve). Fixed run name unless "auto". Stop reasons fails/none/deaths/error fail the goal action; "runs"/"time" end it cleanly (and the goal restarts the loop if kills < Count).
Gaps: FarmChar.Has (anya_portal etc.) is empty, so only `Fixed` runs that need a flag are reachable; the dashboard goal list is not touched.

## Live checks (none done)
1. Pindleskin: red portal object class (60 is S), arrival area 121, Pindleskin unique number 52 and name string as read.
2. Eldritch: unique 49, location relative to the Frigid Highlands waypoint; Shenk afterwards by walking 111 -> 110.
3. Measure `secs` per run (event log) and replace the estimates; check Countess/Andariel xp rows (guesses).
4. NewGame + Chores cost per run (time) and whether bosses really respawn in a fresh game.
5. Mephisto Durance 2 -> 3 hop and Baal waves with a real character at the floors.
