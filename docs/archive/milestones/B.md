# B: Normal Act 1 complete (quests, Andariel, hop to Act 2)

Exit: a fresh character goes from clvl 1 to standing in Lut Gholein with the Act 1 required quests done, in one unattended run,
deaths <= 1 per hour. Design: docs/progression (tasklayer, quests, bosses, acts), checklists in docs/progression/verification.md.
Status today 40 % (rescored 2026-10-28 at master abf1195).

## Where it stands
- Exists and passes offline: `internal/task` (step graph, Act 1 planner, executor, sim; wired as the `campaign` goal), `internal/quest` (state reader, Interpret), `internal/questmem` (in-memory quest and waypoint source, **verified live**, docs/questmem.md), `internal/boss` (Blood Raven, Countess, Smith, Andariel; **all live behaviour unverified**, docs/boss.md), act-hop steps.
- Live: Den walk and 14 minute clear in the 31 minute run, Akara Den claim by skill point (on a cheated cleared state), 20 consecutive dungeon transitions, quest word positions, act arrival by cheat for acts 2-5.
- Not done: boss scripts and questmem are not wired into the bot, no boss was killed by a script, Blood Raven ended in a chicken exit at clvl 6, no Warriv hop, Kashya/Warriv/Charsi/Cain rows unknown, the campaign `acts` default is [1].


## Work packages still needed (order from docs/progression/packages.md P0-P4, Act 1 slice)
| wp | what | owner | effort (U) | parallel with |
|---|---|---|---|---|
| B-1 | Wire `questmem.New(reader)` first in the quest reader, `Invalidate()` on game start; record done/clear/reward words after one REAL Den completion (B.quest_state, B.questmem_wired) | d2-worker + d2-live (i5) | 1 | A |
| B-2 | Wire `boss.Runner` into `task.Actions.KillBoss` (adapter table in docs/boss.md section 3): poison/antidote state, unique number and name, loot by radius, Explore early stop | d2-worker | 2 | B-1 |
| B-3 | Live checks 1-7 of docs/boss.md section 5 on a cheated level (flagged), results table in docs/boss.md (B.boss_live) | d2-live (i1) | 2 | after B-2 |
| B-4 | Dialog rows for Kashya, Warriv, Charsi, Cain; confirm by effect (B.dialog) | d2-live (i3) | 1 | any |
| B-5 | Act hop A1->A2 (Warriv) and the Lut Gholein waypoint; `acts` config no longer [1] (B.hop) | d2-live | 1 | after B-4 |
| B-6 | Corpse retrieval samples inside bot runs, 5 (B.corpse) | d2-live | 0.25 | any |
| B-7 | Nav to the Tower Cellar 5 and Catacombs 4 with a fighting character (B.nav) | d2-live | 0.75 | after B-2 |
| B-8 | First Normal Act 1 clear to Andariel: `campaign` goal from clvl 1, then the unattended run (B.den, B.bosses, B.unattended) | d2-worker + d2-live | 1.5 + wall time | last |

Cheated states (quest bits, act, level) are allowed to test each handler and are flagged in the ledger; the exit run is clean.

## Risks
- Quest bit semantics unclear (U in docs/saves.md): fall back to proxies (area visited, boss killed, item in inventory) plus the
  save at menus. Do not block B-5 on V-05.
- Dialog row blindness (no text reads): recorded indices plus confirm-by-effect; never click an irreversible row without a check.
- The bot keeps dying to Blood Raven at clvl 6 (chicken exit): the level floor 6 is U; raise it before the first clear.
- Andariel at clvl 15 with Act 1 gear may need clvl 18-20 or more potions: the planner's level floor decides; measure deaths.
