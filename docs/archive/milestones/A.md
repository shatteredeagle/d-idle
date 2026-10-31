# A: Act 1 self-leveler (clvl 1 to 15, 2 h unattended)

Exit: two 2 h runs on two characters (different seeds) pass `tools/soak/soak.sh ... --profile m1`, clvl 12 inside the 2 h,
clvl 15 within a further 2 h, all on the managed path (M0). Status today 50 % (milestones.json, rescored 2026-10-28 at master abf1195).

## Where it stands (V docs/bot.md, docs/launch.md 9, docs/combat.md, docs/nav.md, docs/leakhunt.md, docs/town.md)
- Done live: managed run without setcap, 90+ minutes of the leveling loop (A.run90); points and hotkeys (A.points).
- Live in part: the 31 minute campaign run (clvl 6, 7,600 XP/h, 203 kills, 1 death with corpse retrieved, 1 chicken exit, 5 portal trips,
  13 rung-3 events all from one nav cause that is fixed); death mechanics (3 real deaths); chores and the full-pack fix with cheat gold;
  seed-independent nav; fd growth is a plateau, not a leak.
- Not seen: any d2soak m1 report, an earned-gold economy (the run used a 500 gold cheat), repair effect, the starvation rule, 3 bot-run deaths,
  clvl 12 inside 2 h. The two causes that ended earlier runs (helper deaths, dead-overlay waits) are fixed and held for 90 minutes.
- A 45 minute soak exists only as the fd numbers quoted in docs/leakhunt.md; there is no report file for it (U where it lives).

## Work packages still needed
| wp | what | owner | effort (U) |
|---|---|---|---|
| A-1 | First 2 h soak on i4 in managed mode (M0.managed is done); triage report by cause (owner per failure) | d2-live | 1 |
| A-2 | Early economy: start-of-life gold (sell starting items, pick up all gold below clvl 10, skip the Den without potions), starvation rule seen live, no abandoned trips | d2-worker + d2-live | 1.5 |
| A-3 | Gear pass inside the loop with earned gold; repair effect verified once (damage gear on a cheated character) | d2-live (i3) | 1 |
| A-4 | Fixes from A-1 and repeat runs until m1 passes twice (expect 2-3 iterations) | d2-live + d2-worker | 2 |
| A-5 | Death drills inside d2bot: 3 real deaths in one run (1 seen); record fixtures with d2rec | d2-live | 0.5 |
| A-6 | Report recovery rungs per hour and chicken exits in the soak report (G1.rungs) | d2-worker | 0.25 |

## Live checks
1. `tools/live.sh --bg --max 7500 --nocap bot d2bot --managed --goals reach:15` on i4 (managed mode is merged).
2. In parallel `tools/soak/soak.sh start i4 --for 2h --profile m1 --min-xp-per-hour 8000` for the first run (roadmap's 15,000 XP/h
   is an after-clvl-5 average; the harness averages over the whole run, CHECKLIST.md), then the plain m1 profile.
3. `tools/soak/soak.sh report` and copy report.json next to a dated line in docs/bot.md; update milestones.json A.* evidence.
4. Second character: a fresh Sorceress on i1 or a recreated Booti (main agent creates it; agents never create characters).

## Risks
- The worker host load 30-40 makes timing knobs flaky (R5): run the soak when fewer agents are live, note load in the report.
- The 0-gold start can dominate the first hour; if A-2 is not enough, allow a logged cheat of starting gold as a temporary
  crutch and mark the run as cheated (the soak still counts for reliability, not for XP/h).
- The earned-gold economy is the main unknown: the last run needed a cheat for the tome. Do not cheat in the soak.
- A 2 h run occupies i4 for the whole time; LIVEBOT cannot iterate on code during it (use i1 for quick checks).
