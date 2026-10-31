# Next 3 weeks: recommended sequence (refreshed 2026-10-28, master abf1195)

Goal: alpha (M0, A, B, G1; 47 % today) by the end of week 3 if nothing slips (risks.md). Owners are agent types from AGENTS.md
(`d2-worker` offline Go, `d2-live` one live instance, `d2-researcher` docs, `d2-reviewer` Opus singly) or the main agent / user.
Live capacity: at most 3-4 live agents at once on the worker host; one instance (i4) is the soak instance. "||" = can run in parallel with the other
rows of the same week. Effort in agent-days (U). What changed against the first version: managed mode, the task layer, the quest
reader, nav seed independence, loopback join and class-agnostic combat are done, so the plan is now wiring, live verification, then soaks.

## Week 1: wire what exists, first Normal Act 1 clear
| wp | owner | inst | effort | parallel | needs |
|---|---|---|---|---|---|
| WP-WIRE-BOSS: `boss.Runner` into `task.Actions.KillBoss`, adapters (poison, unique number/name, loot, Explore early stop); raise Blood Raven floor (B-2) | d2-worker | - | 2 | first | - |
| WP-WIRE-QMEM: questmem first in the quest reader, `Invalidate` on start; campaign `Verified` by memory (B-1) | d2-worker | - | 1 | \|\| | - |
| WP-SAVEWIRE: saves.Snapshot/Verify in launch.Life, save cycle knob, ladder rung 4 restart, one Recycler per managed instance (M0.saves, G-1, G-2) | d2-worker | - | 2 | \|\| | - |
| WP-SOAK1: first 2 h soak on i4, managed, no cheats, triage by cause, recovery rungs per hour in the report (A-1, G-6) | d2-live | i4 | 0.5 + 2 h | \|\| | - |
| WP-CIRUNNER: register a forge runner on the worker host (M0.ci) | user (token) + main | - | 0.25 | \|\| | admin token |
| WP-BOSS-LIVE: docs/boss.md section 5 checks 1-7 on cheated levels, results table (B-3) | d2-live | i1 | 2 | after WIRE-BOSS | WIRE-BOSS |
| WP-DIALOG: Kashya, Warriv, Charsi, Cain rows and the Lut Gholein hop (B-4, B-5) | d2-live | i3 | 2 | \|\| | - |

## Week 2: first Normal Act 1 clear to Andariel, the 2 h soak passes
| wp | owner | inst | effort | parallel | needs |
|---|---|---|---|---|---|
| WP-CLEAR1: `campaign` from clvl 1 through Den, Blood Raven, Countess, Smith, Andariel, Warriv hop; fix the top causes after each attempt (B-7, B-8) | d2-live + d2-worker | i4 | 3 + wall time | first | WIRE-*, BOSS-LIVE, DIALOG |
| WP-A-ITER: rerun m1 until it passes twice, once on a second character/seed (A-4, A.seeds) | d2-live | i1 | 2 | \|\| | SOAK1 |
| WP-ECON: earned-gold economy: starting gold, pickup, repair seen, starvation rule seen (A-2, A-3) | d2-worker + d2-live | i3 | 1.5 | \|\| | - |
| WP-SENTINEL-PROD: sentinel under `--managed`, drills again incl. a real death and a Wine crash (G-3) | d2-live | i2 | 1.5 | \|\| | SAVEWIRE |
| WP-REVIEW-ADAPTERS: Opus look at the boss and questmem adapters and the task escape policy (S5) | d2-reviewer | - | 0.5 | after WIRE-* | - |

## Week 3: soaks
| wp | owner | inst | effort | parallel | needs |
|---|---|---|---|---|---|
| WP-SOAK2H-FINAL: m1 passes twice (A.soak2h) | d2-live | i4 | 2 h x 2 | first | CLEAR1 |
| WP-SOAK24: 24 h soak, managed, sentinel, fault injection at hours 2 (kill -9) and 6 (SIGSTOP) (G-5) | d2-live / main | i4 | 1 + 24 h | after SOAK2H-FINAL | SAVEWIRE, SENTINEL-PROD, A passed |
| WP-PALADIN-A: Paladin second class: leveler pass live, Normal Act 1 leveling run to clvl 15, then m1 (H.class_leveler, H.second_class) | d2-live | i1 | 3 | \|\| SOAK24 | CLEAR1 for the campaign |
| WP-ACT2-PREP: Act 2 prep: merc reader/revive, waypoint tabs for acts 2-5, cube spike (C-1, C-2, C-3) | d2-worker + d2-live | i2 | 3 | \|\| | CLEAR1 |
| WP-ALPHA-REVIEW: Opus review of the alpha evidence; update milestones.json | d2-reviewer | - | 0.5 | last | everything above |

## After alpha (beta order)
Act 2 handlers (C1), the 72 h soak with the phone notification and log/disk guard (G2), the real game in the container (F-1, F-2; the managed mode removes
the old blocker), the installer viewport (F-3). Soak 24 h, then 2 h, then 24 h again after each material change: the plan's reliability gate is the soak, not a demo.

## Notes
- Every live result updates milestones.json in the same commit (evidence rule); the /milestones page shows it at once.
- Stream and public tooling (S) is not on this plan; do not take a worker host instance from a soak for it.
- If week 1's soak shows many new failure classes, push WP-BOSS-LIVE and WP-DIALOG to week 2 and keep A first: the gate is reliability.
- Do not start C1 code before WP-CLEAR1 shows the task layer holds; do not start game-layer code before H.
