# Sandbox log analysis, 2026-11-27 (sandbox Doofer, Paladin; sandbox Questy, Paladin)

> status: research, read-only (card sandboxlogs). Window: the last 12 h before 2026-11-27 16:45 EST, i.e. from 04:45 EST; times below are EST (UTC minus 5 h) unless a `Z` is shown. Target: classic 1.14d, offline sandboxes. No game verb was run, no knob flipped, nothing restarted, no token printed.
>
> Tags: **V** read in the sandbox files or in this tree (named); **S** derived from a V source by arithmetic; **U** not known (cause or intent unproven). Cause classes A to K are those of `docs/roadmap-fable.md` ("Operating rule").

## 0. What could and could not be read

| source | state |
|---|---|
| `docker logs` of the bot (structured stdout, the `inv=` and tick lines) | V EMPTY for the window. The last bot line in Questy's container is 2026-11-26T15:50:14Z, in Doofer's 2026-11-25T14:15:14Z; both bots have run since (Questy's `d2bot` restarted 2026-11-27T19:02Z). The `--since 12h` call returns 0 lines. So the `inv=` counts and `bot.tickLogSec` lines named in the brief are not readable; see problem 3. |
| game process logs (`d2d.out` and its rotations) | V read: Questy 2026-11-27T13:26Z to now, Doofer 14:27Z to now (older rotations are gone). Area changes, town portals, kills, relaunches. |
| incident files (`incidents/*.json`, written by the invariant monitor and at deaths) | V read, 21 on Doofer, 25 on Questy. They carry the `invariant` name, state, activity and belt facts: the only invariant evidence for the window. |
| `deaths.jsonl`, `goals.json` (hourly XP, kills, deaths, potions), survival `ledger.json` (per-zone chickens, deaths, XP), `state/quests/*` (runs, latches, audit), `state/tuning.json` (knob overrides) | V read. |
| `/api/explain`, `/api/quests`, `/api/hero` | not read: no curl in the image; a raw socket request to the bot port answered 404 for the `/api/hero` and `/api/explain` paths I tried (the real routes or a token are needed). Skipped as the brief allows. |
| the handoff report "no path to Dark Wood" | not available to me (not in the tree: `grep -r "no path to Dark Wood"` finds nothing). What I could verify about Dark Wood is in problem 2 and the note under it. |

Releases running (V `image.json`, `state.json`): Doofer 0.0.261127145257 (built 09:52 EST), Questy 0.0.261127060042 (built 01:00 EST). Most of today's merges (section 5) are newer than both builds, so a log line below may already be fixed on main.

## 1. Sandbox Doofer (Paladin, `paladin-normal30-v2`)

Note on the start: V `goals.json` stats `since` 2026-11-27T01:37:54Z (= 20:37 EST on 11-26): the hero was archived and recreated; the first quest start was 01:39Z by the player (audit). So the 12 h window is hour 8 to 20 of this hero.

Timeline (V incident `pre.level` is a lower bound for "first seen at"):

| EST | event |
|---|---|
| (before window) 23:35 on 11-26 | death, level 4, Cold Plains, killed by Fallen, 11 hostiles near, belt 0 hp, activity "Recovery: farming gold", 15 gold. The only death of this hero (V `deaths.jsonl`). |
| 09:16 | level 14, Dark Wood, first arbiter-starved |
| 09:48 | `field-entry-no-kit` Cave Level 1 (3 of 4 healing potions) |
| 10:04 | `resupply-in-field` Cold Plains, 54 s, 6 hostiles; emergency town portal 1 s later |
| 12:06 | `town-thrash` entries=17 in 3600 s (Stony Field) |
| 12:38 | level 15 (Tower Cellar 2) |
| 12:38 | `resupply-in-field` Tower Cellar 2, 124 s, 11 hostiles, life 68 %; chicken at 12:39:21 (ledger) |
| 13:34 | `refusal-storm` refusals=10, hard, activity "Failsafe: restarting the game"; the Forgotten Tower (Countess) chunk is parked `gave_up` after 3 failures at 13:34 (V quests.json, release e19701a) |
| 13:37 | "Quest: travelling (Go to the Tower Cellar)", the Countess walk again (V incident) |
| 14:12 | level 16 |
| 16:23 | level 17 (Outer Cloister) |

Counts (S from V files, window): 12 `arbiter-starved`, 4 `field-entry-no-kit`, 2 `resupply-in-field`, 1 `town-thrash`, 1 `refusal-storm`, 0 deaths, 13 chickens (ledger `recent`: Stony Field 2, Dark Wood 5, Black Marsh 1, Cave Level 1 1, Underground Passage 1 2, Tower Cellar 2 1, area 27 1), 112 town portals and 63 town arrivals in the 7.2 h covered by `d2d.out`, 26 `d2d` launches (about one per 20 min, problem 10). XP per hour (V `goals.json` hourly rows, EST 05:00 to 11:00 file hours): 12.2k, 11.6k, 9.5k, 13.7k, 38.7k, 29.3k, 21.6k, 22.8k, with 500 to 850 kills per hour.

Residency in the covered 7.2 h (S `d2d.out` area changes): Black Marsh 98 min, Dark Wood 91, Stony Field 48, Tower Cellar 1 to 3 38, Tamoe Highland 37, town 34, Cave Level 1 29, Cold Plains 20, Outer Cloister 15, Cave Level 2 9. The hero moved on up the area ladder as it levelled (Black Marsh, Tamoe, Tower, Outer Cloister): zone choice is not the problem here.

Quests (V `state/quests`): Den of Evil, Sisters' Burial Grounds, Cain scroll, read and stones chunks finished, all started BY THE PLAYER through the UI (audit agent `firefox`, 18:11Z to 18:22Z = 13:11 to 13:22 EST); the player also set auto-start on for Forgotten Tower and Tools of the Trade. The campaign goal "Campaign: Normal Act 1" still reads `current 0 / target 6, percent 0` (V goals.json) after those five finished chunks: see problem 7.

Recovery-mode gold loops: none in the window. The one death (23:35 EST) happened while "Recovery: farming gold" at level 4 and 15 gold, so the loop that is meant to rebuild the kit killed the hero: see problem 8.

## 2. Sandbox Questy (Paladin, `paladin-normal30-v2`, created 11-25 13:18 EST)

Run of the whole hero (V `goals.json` totals): 22 723 kills, 24 deaths, 970 potions, 15 level-ups, 51 h active; the campaign goal still reads `0 / 6`, `deaths 20` since level 8.

Deaths (V `deaths.jsonl`; all before the window): 22 of the 24 happened at level 12 or 13 in Stony Field, Cold Plains and Cave Level 1; the last three:

| EST | area | level | killer | state at death |
|---|---|---|---|---|
| 11-26 17:51 | Stony Field | 13 | Skeleton | clearing, "Level up to 14", drink-life, 0 potions drunk in the last minute |
| 11-26 22:57 | Cold Plains | 13 | Dark Hunter | clearing, "Level up to 14" |
| 11-27 00:12 | Cold Plains | 13 | Fallen | traveling "Walking to Blood Moor", crowd-retreat, life 8 %, 17 hostiles near, 5 potions drunk |

Zero deaths in the window, one chicken (09:59 EST, area 9). That is the good news; the cost shows in the XP rate.

Timeline: level 15 first seen 09:13 (Dark Wood), level 16 at 12:37, "Level up to 17" active at 16:41 (still level 16; V incident `pre.level`). 20 `arbiter-starved` incidents, all with activity "Quest: fighting (Level up to N)", the hero in Dark Wood (16 of 20), Cave Level 1 (3) and town (1). The `d2d.out` residency in the covered 8.8 h: Dark Wood 373 min, Cave Level 1 106, town 28, Cold Plains 16; one 2-minute Black Marsh and Tower Cellar visit at 14:27 to 14:31 EST (V `level:` lines 19:27Z to 19:30Z). 102 town portals, 63 town arrivals, 25 `d2d` launches since 08:26 EST.

XP rate (V `goals.json` hourly rows, EST hours 11 to 16): Questy 5.4k, 5.9k, 6.0k, 5.0k XP per hour at 830 to 870 kills per hour; Doofer in the same hours 38.7k, 29.3k, 21.6k, 22.8k. Both level 14 to 17.

Quest board (V `state/quests`, `quests.json`): the last run record is 11-25 18:32 (Cain scroll, `stopped`); the last audit line is 2026-11-26T03:19:55Z = 11-25 22:19 EST, "pause: 2 deaths in one chunk" from the operator quest loop (`questloop try n/3`, agent `ops-cli`). Nothing has started a quest on this hero for about 42 h. No quest is opted into auto-start (no `auto` row in the audit, unlike Doofer). Knobs on Questy (V `state/tuning.json`): `quests.board` 1, `quests.boardSM` 1, `quests.stepExec` 1 (ledger: 2 h window 14:02 to 16:02 EST closed clean, row enabled, V commit 18ffa16c5), `combat.chickenPct` 22, `bot.zoneModel` 1, `bot.zoneDial` 1.

Recovery gold loops: 2 of the 24 deaths were in "Recovery: farming gold" (11-25 23:54 and 11-26 00:45, Stony Field; see problem 8); none in the window; the hero holds 5.6k stashed gold and 1.1k to 1.5k on hand all window, so no gold-poverty state occurred.

"no path to Dark Wood": I cannot see today's bot log (section 0). What is V: (a) the last logs (11-25 to 11-26 15:50Z) carry `quest board: search_for_cain/scroll: needs_attention: quest.search_for_cain/to-wood: too weak for Go to the Dark Wood; 3 failed attempts in 1h30m0s, parked` and `task step quest.search_for_cain/to-wood outcome=blocked kind=danger err="area not allowed now: Underground Passage Level 1 (10): route guard (1 healing potions < 2 for the underground_passage road)"` and `(no town portal for the underground_passage road)` (Questy bot log, 2026-11-26T00:02Z to 00:10Z); the walk to the Dark Wood goes through the Underground Passage guard row. (b) Today the hero stands in Dark Wood 373 min, arriving by town portal 37 times and by waypoint 24 times, so Dark Wood itself is reachable; what fails is the quest-walk step (the to-wood node) and the quest board never retries it (problem 1). Whether a literal "no path" error exists today is U.

## 3. Top 10 problems, ranked by time lost or risk

Cause class per the operating rule. "Row" is the roadmap row or card that covers it today (`docs/roadmap-fable.md`).

### 1. Questy: the Cain chain has not run for about 42 h; the quest board waits for a human and the 2 h `quests.stepExec` window proved nothing

- Evidence: audit last line `2026-11-26T03:19:55Z ... cancel search_for_cain ... pause: 2 deaths in one chunk`; no audit line after it (V). Every incident of the window says `activity: "Quest: fighting (Level up to 15/16/17)"` and `invariant: arbiter-starved ... nothing else wanted: level the hero` (20 times, 09:13 to 16:41 EST, V). `quests.stepExec` is 1 on Questy (V tuning) and its ledger window closed clean at 16:03 EST (V 18ffa16c5), yet no chunk ran in that window, so no step was executed under it (S).
- Cost: the hero is level 16 at 5 to 6k XP/h; a quest chain that gives Cain, a mercenary (Blood Raven) and a stash of rewards sits idle (S).
- Class: A (two deciders: leveling target vs quest board; the board only offers, a human or an ops loop starts) with H.
- Row: P9 `quests.stepExec` (TASK-998 flipwave2), BSM `quests.boardSM` (TASK-983). They cover how a chunk runs, not WHO starts it when auto-start is off.
- Fix proposal, card `questresume`: owns `internal/questboard/`, `internal/bot/questauto.go`, `data/questboard/`. ACs: (1) a parked Search-for-Cain chunk whose park reason was "too weak" or "2 deaths in one chunk" is re-offered by the board itself when the hero is at or above the legacy floor plus a margin and has had no death for N hours, with the offer visible in `/api/explain`; (2) an offline scenario (the existing `cain.scroll.stepexec`) starting from a parked state with level above the floor reaches the chunk start with no operator call; (3) the stepExec ledger row gets a criterion "at least one chunk started under the knob in the window" so a clean empty window cannot close it; (4) `arbiter-starved` carries the list of parked sources and their reasons (see problem 7).

### 2. Questy farms Dark Wood and Cave Level 1 at level 15 to 16: 5 to 6k XP/h against 22 to 39k XP/h on the same class

- Evidence: residency Dark Wood 373 of 530 min, Cave Level 1 106 min (S `level:` lines); hourly XP 5.0 to 6.0k (V goals.json). Doofer in the same hours moved to Black Marsh, Tamoe Highland, Tower Cellar and Outer Cloister. The survival ledger holds Questy's old hardness only for area 10 (Underground Passage 1, `hard 11.3` since 11-25 13:30, never decayed, V ledger.json) and zero seconds in Black Marsh (6), Tamoe (7), Outer Cloister (27) for Questy while those rows exist for Doofer (V).
- Cost: the lost rate is about 15 to 30k XP per hour (S); a level at this point costs about 3 h instead of under 1 h.
- Class: G (a threshold moved by one incident: the 24 deaths at level 12 to 13 raised margins; nothing lowers them after 24 h with no death).
- Row: ZONE `bot.zoneModel`, `bot.zoneDial` (on in Questy's tuning), R6 survival model; P4 admission (TASK-970 admit2 shadow merged today 15:34 EST). No card tracks "decay the margin after a clean day".
- Fix proposal, card `zonepace`: owns `internal/bot/zonemodel.go`, `internal/survival/`, `data/` zone rows. ACs: (1) the zone picker reads per-zone hardness with a half-life (a single death dims, a clean 12 h restores), threshold from data, not code (P10); (2) an offline replay of Questy's ledger.json (hardened rows) picks Black Marsh or Tamoe at level 15 to 16 for the paladin build and still picks Stony Field for level 12; (3) a second build (Sorceress) in the same test (build-flexibility rule); (4) the zone choice line gives the XP per minute it expected so the next review can compare to the hourly table.

### 3. The sandboxes' structured bot log is silent for 31 h (Questy) and 55 h (Doofer)

- Evidence: `docker logs` last timestamps 2026-11-26T15:50:14Z (Questy) and 2026-11-25T14:15:14Z (Doofer), while `d2bot` ran (pid started 19:02Z today on Questy; `ps` shows it at 48 % CPU). No `inv=` or tick line is available; only the 21 and 25 incident files remain (V).
- Cost: any review of a flip (the 2 h windows, `bot.tickLogSec` night, `arbiter-replay -session`) has no input; roadmap "Next up" item 5 says one recorded night is still missing (V `docs/roadmap-fable.md`).
- Class: K (not visible). Cause U (the bot's stdout is a pipe to the launcher; the reader or the driver stops delivering lines; a full pipe would stall the bot, which it does not, so the lines are dropped or the level changed).
- Row: T4 `bot.tickLogSec` (trial, "one recorded night (U)"); P1 invariant monitor ("read the `inv=` counts" next action). Both assume the log is reachable; none covers delivery.
- Fix proposal, card `logsink`: owns `deploy/d2d-container/entrypoint.sh`, `internal/logx/`, `internal/didleboot/` if present (U). ACs: (1) bot log goes to a rotating file under the instance directory (size and count cap) as well as stdout; (2) a unit test fills the stdout pipe and shows the bot still logs to the file; (3) `tools/` has a one-line reader for the last N hours; (4) after the fix a sandbox hour shows both `inv=` and tick lines (verified live on a sandbox, not offline).

### 4. Doofer: "Returning to the field" with 6 to 11 hostiles for 54 to 124 s (twice), ending in an emergency portal or a chicken

- Evidence: `10:04:05 resupply-in-field: state=resupplying step="Returning to the field" area=3 for=54s hostiles=6` then `10:04:06 ... Going to town (emergency town portal)` (belt 2 hp 0 mp); `12:38:54 resupply-in-field ... area=22 for=124s hostiles=11`, life 68 %, belt 1 hp, then a chicken at 12:39:21 (V incidents, ledger `recent`).
- Cost: two near-deaths; the hero fought with the resupply state on, so the fight policy and the potion floors of "resupplying" apply instead of "clearing" (U which policy).
- Class: A or E (the state machine says resupplying while the hero stands in a field with a crowd; stale state after a portal).
- Row: P6 arbiter control (families 255 are on Doofer), P2 outcome exits. `townguard` TASK-1057 (merged 14:45 EST) changes the guard refusal, not this state. None for this.
- Fix proposal, card `fieldstate`: owns `internal/bot/invariants.go`, `internal/arbiter/`. ACs: (1) with hostiles within the crowd radius in a field area the arbiter overrides "Returning to the field" with the fight or the retreat lane within one tick; (2) a scenario from the incident pre-state file (`d2scenario incident`) replays the 17:38Z file and the hero does not stand 124 s; (3) the invariant `resupply-in-field` stays as the monitor but its threshold is read from data.

### 5. Doofer: four `field-entry-no-kit` and a `town-thrash` with 17 town entries in an hour although gold was 500 to 1600

- Evidence: `missing=healing potions 3/4 area=9` (09:48), `3/4 area=5` (12:39), `3/4 area=7` (13:26), `1/4 area=2` (14:56); `town-thrash: entries=17 window=3600s` (12:06); gold 521 to 1313 at those moments (V incidents). The belt facts at the 13:26 file: belt 3 hp, pack 1 (S: the hero left town with 3 of 4 and a pack potion).
- Cost: the field time lost to the 17 entries is not measurable from these files (U); the risk is entering with a short kit.
- Class: C (supply and economy).
- Row: TOWN, P5 `town.plan` (stage 3 on Doofer, 45 min window closed clean 09:25 EST), P4 admit (TASK-970), townguard TASK-1057. Doofer's build (09:52 EST) is before admit2 and townguard, so this may already be changed; recheck after the next deploy.
- Fix proposal (only if it recurs after the redeploy), card `kitwhy`: owns `internal/bot/invariants.go`, `internal/town/`. ACs: the incident file for `field-entry-no-kit` and `town-thrash` carries WHY the kit was short (gold, seller stock, belt slots, purchase refused) so the next cause is read, not guessed; one test per reason.

### 6. Doofer: `refusal-storm` (10 refusals with no kill, XP or area change) forced a game restart and the Countess chunk parked as `gave_up`

- Evidence: `13:34:18 refusal-storm: refusals=10 ... hard=true`, state `recovering`, activity "Failsafe: restarting the game"; `quests.json` run `forgotten_tower/countess status parked code gave_up failures 3 at 13:34:21 release e19701a`; a `d2d` relaunch at 13:34:09 (V). The hero walked the tower again at 13:37 and again at 14:47 (Tower Cellar 3 reached 14:46 EST in `level:` lines, V).
- Cost: one failsafe restart, a parked quest, repeated long walks (S).
- Class: B (refusal counted as a failure) and D (Tower Cellar navigation).
- Row: P2 typed outcomes (enabled), TASK-766 countessnav (Tower Cellar 5 walk regression, merged 15:09 EST, offline). BSM for the park.
- Fix proposal, none new: attach to TASK-766 and check on the next build whether the Countess chunk completes; if the storm repeats, the refusal that repeated needs naming in the invariant (the incident file lists only the count; ask for the refusal code list). AC: `refusal-storm` incident lists the top three refusal reasons.

### 7. The campaign goal reads 0 of 6 although quest chunks finished; `arbiter-starved` fires every 25 min and hides what the board holds

- Evidence: Doofer `Campaign: Normal Act 1, current 0, target 6, percent 0` with five chunks finished and latched (V goals.json, quests.json); Questy the same after 51 h. 32 `arbiter-starved` incidents in the window (12 on Doofer, 20 on Questy) with the single summary "nothing else wanted: level the hero" (V).
- Cost: the progress bar cannot tell the player that a quest is parked, and the monitor's most frequent line carries no cause (S).
- Class: F (a fact the backend lacks: the campaign counts a different completion than the board latches) and K.
- Row: R5 questspec, BSM; invmon P1 (the monitor exists). None for the campaign counter. Whether the target counts quests (not chunks) is U.
- Fix proposal, card `campaigncount`: owns `internal/campaign/`, `internal/bot/invariants.go`. ACs: (1) the campaign progress reads the board's latches; a test with Doofer's quests.json gives progress above 0; (2) `arbiter-starved` lists parked or never-auto quests with their reason; (3) the invariant is info when a level goal is active by choice and the board offers nothing (no rate limit reset by every relaunch).

### 8. Deaths during "Recovery: farming gold": Doofer at level 4 and two Questy deaths at level 12

- Evidence: Doofer 23:35 on 11-26, Cold Plains, killed by Fallen, 11 hostiles near, belt 0 hp / 1 mp, 15 gold, activity "Recovery: farming gold"; Questy 11-25 23:54 (Stony Field, Dark Ranger) and 11-26 00:45 (Stony Field, Skeleton); 2 of 24 Questy deaths carry that activity (V `deaths.jsonl` autopsy). All outside the 12 h window; the Doofer one is the most recent.
- Cost: the loop meant to rebuild a kit kills a hero with no potions; each such death loses the corpse and starts the loop again (S).
- Class: C with G.
- Row: `povertyfarm` data (`internal/bot/povertyfarm.go`) and P4 admission rows; `resource.recoveryEndOnKit` is on. No card for "recovery picks its zone by survival margin with an empty belt".
- Fix proposal, card `recoveryzone`: owns `internal/bot/povertyfarm.go`, `data/`. ACs: with belt 0 and under 2 portals the recovery zone is chosen by the survival model with the "no potion" kit; a scenario at level 4 with 15 gold in Cold Plains no longer picks a zone with 11 Fallen in reach; two builds in the test.

### 9. Doofer: `tuning.json` was written as zero bytes twice (`tuning.json.corrupt-1795731200` and `...597`)

- Evidence: two 0-byte files dated 11-26 17:13 and 17:19 EST (V directory listing); the live file is intact and the loader moved the bad ones aside.
- Cost: small now (the loader recovered); a lost tuning file would reset the sandbox's knobs silently (S).
- Class: J or K. Cause U (a non-atomic write, a truncate before the write, or a kill during the write).
- Row: none.
- Fix proposal, card `tunerwrite`: owns `internal/bot/tuner.go`. ACs: write to a temp file and rename; a test that kills the writer mid-write leaves the old file; a log line names the recovery.

### 10. About 2 `d2d` relaunches per hour on both sandboxes; 7 of Doofer's 13 chickens fall in the same second as a save at a relaunch

- Evidence: 26 launches in 7.2 h (Doofer) and 25 in 8.3 h (Questy), each preceded by a town-portal return; the gap between the last line of one file and the first of the next is 6 to 10 s, once 37 s (S). Doofer chicken times 14:45:53, 15:02:06, 16:39:20, 16:40:48, 17:24:13, 17:35:01, 17:39:21 and 18:53:32Z match `d2d.out` save lines at the same second (V ledger `recent`, `d2d.out.*` ends).
- Cost: 6 to 40 s per relaunch (S), tiny; the question is whether a chicken (leave game) is counted as a death risk or a recycle, because the same event ends a game that was working (U).
- Class: K. Cause U (game recycle policy per `docs/knobs.md` 676: 30 min minimum between recycles; or leave-game on chicken then relaunch).
- Row: none; the recycle policy is documented, not measured.
- Fix proposal, none: add the reason of each relaunch (recycle, chicken, failsafe) to the incident or a one-line log so a count by reason exists. Card `relaunchwhy`, owns `internal/launch/`; AC: each relaunch writes its reason; a test per reason.

## 4. Compact numbers

| | Doofer | Questy |
|---|---|---|
| level at window start to end | 14 to 17 | 15 to 16 |
| deaths in the window | 0 | 0 |
| chickens in the window (ledger) | 13 | 1 |
| invariant incidents (window) | 20 (12 starved, 4 no-kit, 2 resupply-in-field, 1 town-thrash, 1 refusal-storm) | 20 (all starved) |
| hard invariant breaches | 1 (refusal-storm) | 0 |
| XP/h at EST hours 11 to 16 | 38.7k, 29.3k, 21.6k, 22.8k | 5.4k, 5.9k, 6.0k, 5.0k |
| failsafe restarts | 1 | 0 |
| quest chunks started in the window | 5 (by the player) + 1 auto-started (Countess, parked) | 0 |

`writes after start` does not apply (no measured run; both are live sandboxes whose cheats ledger is outside this read).

## 5. Compared with what merged today (`git log origin/main` since 00:00 EST)

Already merged and relevant, so NOT re-proposed: hostilespin TASK-1048 (hostile scan friendly set, 10:28 EST; the quest-board slice vs watchdog), akarafix TASK-1051 (Akara hand-in logs and stale request), townguard TASK-1057 (guard refusal hand-over, route path cost), admit2 TASK-970 step 2 (admission shadow wiring, knob `bot.admission`), townrest (town plan view, arbiter upkeep source), `ledgerenabled`/`ledgertrial` (stepExec row), diffgo/diffready (difficulty hop), vaultperf, countessnav TASK-766 (offline). Neither running build contains these except those older than 06:00 / 09:52 EST (V `image.json` against `git log --format=%cI`). The ten problems above are not covered by any of them: problems 1, 2, 3, 7, 8, 9 have no card (none), 4 and 10 no row, 5 and 6 sit on rows that may change after the next deploy.

## 6. Open questions for the coordinator

1. Should a human click stay the only starter of a quest chunk in "train" mode, or should the campaign goal start its own chain (problem 1)? ADR 0030 and `quests.auto*` knobs suggest the opt-in is by design; the proposal assumes the player's opt-in is respected and only fixes the stuck re-offer after a park.
2. Where does the bot's stdout go between the bot and the container log (problem 3)? I did not read the launcher.
3. The handoff text for "no path to Dark Wood" (section 2).
