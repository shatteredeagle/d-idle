# G1 (24 h) and G2 (72 h): soaks

Harness: `cmd/d2soak` + `tools/soak/soak.sh` (V offline, tools/soak/CHECKLIST.md). Profiles m1 (2 h), m3 (24 h), v1 (72 h); H
uses v1 limits over 168 h. Soaks run continuously from milestone A on: every fix is judged by the next report, not by a demo.

## Where it stands (rescored 2026-10-28, G1 37 %, G2 0 %)
- Done: the soak judge handles warm-up and the fd plateau (G1.judge, docs/soak.md). fd growth is a per-area plateau, not a leak (docs/leakhunt.md, live).
- Partial: sentinel (drills live in the setcap topology), recycler (policy offline, 22 s manual recycle live, not wired), dashboard visibility, recovery rungs per hour (baseline 13 in 31 minutes, cause fixed).
- Todo: rung 4 restart wiring (cmd/d2bot has no Deps.Restart), save cycle knob, the 24 h run, zero touch. No soak longer than 45 minutes has run on the product path.

## G1 work packages
| wp | what | owner | effort (U) |
|---|---|---|---|
| G-1 | Wire bot ladder rung 4 (`Deps.Restart`) to `launch.Life` restart in cmd/d2bot, and one `launch.Recycler` per managed instance called between goals (docs/soak.md requests; emits a `recycle` event) | d2-worker | 1 |
| G-2 | Save cycle knob: leave and re-enter every N minutes (default 20-30) so a kill loses at most one cycle; snapshot at the menu (saves.Snapshot) | d2-worker | 1 |
| G-3 | Sentinel in the product topology: in-process goroutine tree in d2bot --managed, or out-of-process via memfd; rerun the drill table plus a real death and a forced Wine crash | d2-live | 1.5 |
| G-6 | Soak report prints recovery rungs per hour, chicken exits and sentinel actions as one table (G1.rungs) | d2-worker | 0.25 |
| G-4 | Check the soak process match strings once (`pgrep -af 'Game.exe|wineserver|d2input|Xvfb|d2bot'`), fix soak.sh for one-bot-per-instance PIDs | d2-live | 0.25 |
| G-5 | The 24 h run itself with an operator log; fault injection at hour 2 (kill -9) and hour 6 (SIGSTOP) | d2-live / main | 1 + wall time |

## G2 additions
- Log rotation and a disk guard (`GaugeFunc` for df and log size in /metrics, added to `leakGauges`), gotchas 56/58.
- Phone notification for a human-needed state (self-pause, crash loop): ntfy or Home Assistant notify; U which.
- Mixed goals through the goal manager (leveling, farm runs, idle windows, an act or difficulty change).

## Risks
- A soak occupies an instance for its whole length: plan instance use (next-3-weeks.md) so iteration does not stall.
- The worker host is the user's desktop machine and is loaded by other jobs; a reboot or heavy job invalidates a run. Record load in the report.
- "0 interventions" needs agents to stay off the soak instance: put a notice on the board and in /tmp/d2lod-live.notice style.
