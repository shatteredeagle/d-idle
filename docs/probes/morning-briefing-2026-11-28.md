# Morning briefing, 2026-11-28

> status: current at 06:50 EST (card morning, TASK-1138). Docs only: nothing was run against a game or a sandbox. The coordinator fills "Final numbers" at 07:30. Sources: `docs/probes/needs-owner-2026-11-28.md`, `docs/probes/night-log-2026-11-28.md`, the reviews in `docs/design/fable-review-2026-11-28/`, `git log origin/main`, `python3 -I tools/darklaunch/report.py`. Tags: V = read or counted in this run, S = taken from another doc or a card result, U = not checked. All times EST.

## 1. Needs you NOW (7 decisions, most urgent first)

| # | Decision | Options | My suggestion |
|---|---|---|---|
| 1 | **Questy's character file is lost** (0 bytes since 04:08; the bot there has been in a restart loop since). Your Search for Cain progress went with it. | (a) restore the level 1 backup of Nov 25 and let the bot level again; (b) create a fresh Questy with the OOG tooling and replay the quest state; (c) leave it down. | (a): fastest, no new character, bot state and vault are intact (S). |
| 2 | **Fork: atomic save, request A1** (temp file, fsync, rename; a failed write keeps the old file). Needs your go, and the branch choice B0 first. | go / wait. Text: `docs/design/fork-save-atomic.md`, order: `docs/design/fork-requests-2026-11-28.md`. | Go. Until it lands a full disk can cost a hero again; the bot's copies only limit the loss to about 10 minutes. |
| 3 | **`VaultDirectEquip` interim switch-off** (and hide Carry/Stow). Asked while N1 to N3 were open. | switch off / leave on. | Leave on: fix1b merged and names N1 to N3 (S, offline tests only). One medium stays open from review 14 (M2, lost-reply equip). Switch off if you will not look at the vault today. |
| 4 | **Cain reward on Questy**: the game pays a quest reward on first contact with a reward NPC, and the Act 1 potion seller counts. | bot may claim / only you claim (then no potion buying while a reward waits). | Moot until decision 1; then: let the bot claim. |
| 5 | **`town.plan` 4, your look (ADR 0028)**. Lab night: Paladin GREEN on belt refill and sell-keeps-need; Sorceress belt refill RED (the assertion asks 3 healing, the class fills 2 + 2); malus errand RED (V `docs/probes/lab-night-results-2026-11-28b.md`). | look now / after the two RED rows are judged. | After: not ready, card townnight (TASK-1108). |
| 6 | **Tick-line criteria**: the seconds-based rule of ticklog-t4 can never hold (ticks come once per goal pass, median gap 16 s). | accept: "ticks at least twice the minute lines, no gap longer than the longest action timeout" / keep. | Accept. |
| 7 | **Doofer repair mode**: gear hit 0 percent durability; the hero's repair mode decides, and it was not read. | check `ops status` on Doofer. | 1 minute look. |

Smaller owner rows, no hurry (S `docs/probes/review-sweep2-2026-11-28.md`): one flake rule and the scenario matrix floor (review 06); deaths over the life of a run (10 B1); when the difficulty go is consumed (10 B4); the escape rule "and" against "or" and the stance floor (SME); an "I checked, close it" desk action (10 N4).

## 2. Overnight in numbers (00:00 to 06:45)

- **418 commits** on main (V `git log --since='2026-11-28 00:00' --oneline origin/main | wc -l`).
- **57 branch merges from 44 branches** (V, `Merge agent/` subjects): cls1 cls2 drv2 dur2 evid1 fix14 fix1b fix4 fix7b fix8 fix8b flipfix2 forkrebase forkreq grad1 labops2 labtriage led1 ledenable live1 nf1 nf1b nf2 nf3 nf3b nf4 nightlog ops2 ops4 road4 road5 roadfresh rv1 to rv6 savesafe sec2 sweep1 sweep2 vault2 zone5.
- **15 review docs**, four rounds; every HIGH of rounds 1 to 3 has a merged fix except the rows marked open below. All fixes are offline (fakes); none is verified live.

### Reviews, one line each (HIGH found / fixed / open)

| # | Topic | HIGH | State |
|---|---|---|---|
| 01 | item flows, vault | 4 (V) | fixed by fix1 (S); F6 twin Keep open, now medium |
| 02 | quest board | 4 (V) | fixed by fix2 (S) |
| 03 | combat, nav, survival | 3 (V) | fixed by fix3 (S) |
| 04 | security, privacy | 1 blocker + 2 (V) | fixed by fix4 (S); volume prune is a procedure doc; 4 medium/low open |
| 05 | dark ledger, flip tool | 4 (V) | 1.5 fixed by led1 (ledger split); 1.2 to 1.4 open in the flip tool (S sweep 1, U since) |
| 06 | Proving Ground as evidence | several (U count) | OPEN: the flip tool checks no evidence, no control arm, no exercise counter |
| 07 | arbiter, deciders | 3 (V) | fixed by fix7 (S); slice budget (A4) open, design rows in TASK-973 |
| 08 | builds as data | data gaps (U count) | cls1, cls2, fix8 merged; skill cap needs one live read |
| 09 | change pipeline | none (process) | fed led1 and the merge speedups |
| 10 | round 2: fix1, fix2 | 3 (V) | fixed by fix1b, vault2, sweep2 (S); N8 open |
| 11 | round 2: fix3, fix7 | 2 (V) | fixed by fix7b (S) |
| 12 | round 2: security, PG, pipeline | 6 (V) | autodeploy rows by ops2 (S night log); B1, B2 by fix8b (U); C2 flip run check open |
| 13 | both sandboxes, live logs | 3 (V) | L1, L2 fixed by live1 and sweep2; **L3 open, needs live** (nav start cell, Tower Cellar 3) |
| 14 | round 3: merged since 00:30 | 5 (V) | all 5 fixed by fix14 (V card result); M2, M3, M4, M6 open |
| 15 | round 4: deploy readiness | 2 noted (V) | verdict GO for the 06:30 canary deploy; H1, H2 open (U) |

## 3. Dark ledger

| | midnight | now (06:45) |
|---|---|---|
| dark | 23 | 20 |
| trial | 6 | 5 |
| enabled | 10 | 12 |
| graduated (default-on) | 0 | 1 |
| closed clean, not yet committed | 0 | 1 (`quests.boardSM`) |

V: midnight from `data/darklaunch.json` at the last commit before 00:00 (the page `docs/probes/dark-queue-status-2026-11-28.md` named in the brief does not exist); now from `tools/darklaunch/report.py` (25 open rows, 0 overdue) and the per-knob files. 3 by-design rows and 1 retired row are not counted.

- Graduated: `bot.tickLogSec` (V `docs/probes/graduations-2026-11-28.md`).
- Enabled today: `quests.refusalPark`, `quests.resumeAfterDeaths` (both sandboxes, 2 h windows closed clean 04:10).
- Waiting for a one-line ledger commit: `quests.boardSM` (window on Doofer 04:30 to 06:30; the report prints the command).
- The ledger is now one file per knob with runtime state outside git (led1).

## 4. Sandboxes

- **Doofer**: healthy at the last tick (05:40, S). On the 04:27 build while the `quests.boardSM` window was open; review 15 says GO for the 06:30 autodeploy. Whether that deploy ran is U: see Final numbers. Flip windows tonight: refusalPark and resumeAfterDeaths 02:11 to 04:10 (clean), boardSM 04:30 to 06:30.
- **Questy**: DOWN. Restart loop on the 0-byte character file since 04:08; nothing was stopped or deleted. Copy of the broken save folder: `~/d2play-q-rescue-20261128/save`. Waits for decision 1.

## 5. Incidents

1. **01:51, merge queue file wiped by a test run** (ops4 suspected, S). The merge worker crashed. Rebuilt by hand from the worker log with the ids kept, worker back at 02:01. Guard: ops4 merged a queue guard against overwriting (S; review 14 did not read it, U).
2. **04:05 to 04:10, root disk of the worker host full**, a Go build cache of about 155 GB under many agents (S). Cost: Questy's character file (decision 1) and seven lab result files. How the space came back is not written down in this repo (U; the lab log only says 155 GB were free again at 04:10): see Final numbers. Guards now (S card savesafe, TASK-1132, NOT verified live): a copy of the character file every 10 minutes and at town arrival, repair from the newest valid copy at startup, a warning under 2 GB free, no backup writes and no planned save point under 200 MB (`docs/ops/save-backups.md`). Not guarded: the fork still writes in place (decision 2); nothing caps the build cache itself (U).

## 6. Open risks

- Nothing merged tonight is verified live. The reviews found new HIGH defects in each fix round (rounds 2 and 3), so round 4 code (fix14, savesafe, dur2, drv2, road5, zone5) is likely to hold more.
- savesafe's startup repair runs before the bot knows whether a d2d still writes (15 H1).
- A hole hold waits in town with no status and a fed heartbeat (15 H2); `bot.zoneModel` is 1 on Doofer by a tuning override.
- Flips still close on "no breach in 2 h", not on proof the feature ran (05, 06). grad1 checked presence by hand.
- 13 L3 can still park a quest chunk next to stairs.
- The disk can fill again until the cache has a cap.

## 7. What I would do next

1. Decisions 1 and 2, then bring Questy back and re-flip its knobs.
2. Cap or relocate the Go build cache on the worker host; add a free-space check to the coordinator tick.
3. Commit the `quests.boardSM` ledger line; run `python3 tools/flip/autoflip.py graduate-list`.
4. One Opus review of the round 4 merges before the next unattended deploy.
5. Live checks: 13 L3 on the recorded cellar grid, savesafe restore on a throwaway hero, the skill cap read (08).
6. Evidence rule in the flip tool (`--evidence <run id>`, exercise counter): closes the largest open HIGH group (05, 06).

## Final numbers (coordinator, 07:30)

- Commits on main since 00:00: TODO (06:45: 418)
- Branch merges / branches: TODO (06:45: 57 / 44)
- Ledger dark / trial / enabled / graduated: TODO (06:45: 20 / 5 / 12 / 1)
- `quests.boardSM` ledger line committed: TODO
- Doofer build and time of the last deploy; rollback yes/no: TODO
- Questy state: TODO (06:45: down)
- Disk: how the 155 GB were freed, free space now, cache cap in place: TODO
- Merge queue: waiting / failed: TODO
- Agents still running: TODO
