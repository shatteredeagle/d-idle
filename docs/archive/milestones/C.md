# C1 and C2: Normal Acts 2-5

Exit C1: Acts 2-3 in one unattended run (merc, cube, Duriel, Mephisto). Exit C2: Acts 4-5, Baal, Nightmare entry; fresh
clvl 1 to Nightmare within 40 h wall time. C1 8 %, C2 0 % today (only the merc hire dialog is live-probed, commit 3c020b1). Plan: docs/progression/packages.md P4-P8, quest mechanics in
docs/goals/execution.md section 3.

## Work packages (rough, U)
| wp | what | owner | effort |
|---|---|---|---|
| C-1 | Merc: reader (owned monster), revive, equip; combat treats it as friendly (R5 in packages.md). Hire dialog is done live | d2-worker + d2-live | 2 |
| C-2 | Waypoint tabs acts 2-5, cross-act travel (V-06) | d2-live | 1 |
| C-3 | Cube UI and transmute; staff assembly | d2-worker + d2-live | 2 |
| C-4 | Act 2 handlers: Radament (optional), cube, amulet, staff, Summoner, correct tomb, Duriel; Meshif hop | d2-worker + d2-live | 4 |
| C-5 | Act 3 handlers: Khalim's relics, Council, Mephisto, red portal | d2-worker + d2-live | 4 |
| C-6 | Act 4: seals executor, seal bosses, Diablo | d2-worker + d2-live | 4 |
| C-7 | Act 5: Ancients (no-portal constraint), Worldstone Keep, Baal waves | d2-worker + d2-live | 5 |
| C-8 | Difficulty screen (V-08) and `normal_playthrough` campaign run | d2-live | 2 |
| C-9 | One arena fixture per boss with d2rec; offline tests per handler | d2-live | 1 |

About 26 agent-days of work (U); several handlers can run in parallel once the task layer (B-2) exists, limited by live
instances (one per live agent) and the worker host load.

## Risks
- Expansion flag or merc readiness off on test characters (V-02 first).
- Random tomb/seal layouts: preset anchors only, never coordinates.
- Cold immunes in Act 3+ with a pure Blizzard Sorceress: the plan's Static/merc answer must work by C1, not D1.
- Wall time: a 40 h campaign needs a quiet instance for two days; book it.
