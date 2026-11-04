# Gate 1 rerun (2026-11-01): NOT PASSED

> status: current. One finished hidden d2d soak (exit 0, 120 wall min, one injected d2d kill), n=1 seed, not a statistical claim. Previous attempt: [../probes/postfix-soak2.md](../probes/postfix-soak2.md). Criteria: the Gate 1 requirements of docs/contracts/soak-gates.md, minutes are virtual (game) minutes (decision 2026-11-01). Tags: V = counted from the log.

## Setup (V)
- Binary: `bin/d2d.lab2-freezeresume` on the worker host, fork branch `lab2-freezeresume` `ee5e06d8ef99` (the fork stack rebuilt on upstream main 52362d52, so the merges #152 #153 #155 #156 #157 #158 are in; vetting in docs/forkrebase.md of agent/forkrebase: differentials 0, live drill legs PASS), file sha256 prefix c526394770f9.
- Bot: origin/main f4ca264 (clean worktree, built on the worker host), sha256 prefix 07b7e6ef73c8; `d2dsoak` built from the same tree.
- Character: fresh uncheated Sorceress `Gate1a` (class 4, seed 11), no cheat of any kind. Recipe of docs/d2d-baselines-early.md section 4: `--backend d2d --managed --goals campaign --toggle autoloot=off,autoparty=off --d2d-args "--lockstep" --d2d-linkwait 60s --run-for` the remaining seconds, `D2D_RUN=on`, nice 10, restart supervisor (never needed: 0 bot restarts), slot 0, through `agentrun.sh` and `d2dslot.sh` with `D2D_AGENT=gate1`. Capacity before start: idle 56 percent, no other soak.
- Times (worker clock): start 13:10:01, fault 14:10:04, supervisor exit 15:10:05 (limit 7200 s). 1359 minute lines = 22.6 virtual h.

## Fault and recovery (V)
At 14:10:04.9 `kill -9` of the slot's d2d (pid 2762879, child of the registered bot, pgid checked equal to the registered slot group; nothing else touched). The bot logged `d2d process exited ... signal: killed`, `game lost, restarting it` at 14:10:05.8 and `re-entered the game` at 14:10:07: back in game in about 3 s (limit 3 min). Minute-line `restarts` 8 to 9 (+1 exactly). Saved XP: 118060 in memory, 3885 at risk, the save held 114176 and the character had 114176 after the restart (not lower than the saved value before the kill). After the run `agentreap --list` was empty.

## Gate 1 criteria
| criterion (soak-gates spec) | result | evidence |
|---|---|---|
| Stability: only the injected exit, 0 panics, bot and d2d alive at 2 h | PASS | exit 0, 0 panics, 0 bot restarts, one `d2d process exited` (the fault), both alive until the bot's own limit stop. `saveXp == charXp` after a graceful stop: UNVERIFIED (the end is a d2d kill by the bot, as in postfix-soak2; `xpRisk` 230 at the last line) |
| Deaths follow the death rule | PASS | 1 death in 6633 kills (0.15 per 1000), clvl 14, Black Marsh, 14:27:46, killer Black Claw (cold enchanted boss pack), belt 0 hp 4 mp, 1 portal scroll; invariants report no death-rule code |
| True XP/h at or above the d2d baseline | FAIL (indicative) | 7517 vs 9172 (-18.0%, tolerance 1834), INCONCLUSIVE at one seed (soak 2 had 5437, -40.7%); clvl 14 reached |
| No single state over 10 min without progress | FAIL | d2dsoak NOKILL: longest no kill and no XP 43 virtual minutes (limit 10) |
| Belt: no window over 5 min without healing outside town | FAIL | d2dsoak BELT: 6 virtual minutes (limit 5) |
| At least one town-portal trip | PASS (wording U) | 143 `back to field: portal in town`; the old counter text `arrived in town through an open portal` no longer exists in the log (11 plain `arrived in town`); 105 waypoint fallbacks, 167 `d2d town portal: no portal source (empty)` |
| Route guard: no boss run when refused | PASS | 0 `blocked: not ready`, no boss-run start; route guard lines present (monastery and the Countess area blocked, no merc, clvl 14 < 16) |
| Injected fault recovery | PASS | about 3 s, restarts +1, saved XP not lower, no orphan |

d2dsoak (`compare -hours 22.6`): `INVARIANTS: VIOLATED(BELT,NOKILL)`, OVERALL INCONCLUSIVE (exit 2, one seed); hang 0, stall 0, wedge 1m, no-progress (wide) 5 m, unrecovered 0.

## Verdict
Gate 1 is NOT PASSED on this build (lab2-freezeresume ee5e06d8, bot f4ca264). Failing criteria: no kill and no XP 43 virtual minutes (limit 10) and belt window 6 (limit 5); XP/h -18 percent is indicative only. Better than soak 2 on XP/h (5437 to 7517) and on the belt window (27 to 6).

## First evidence and best cause (U)
- No kill and no XP: minute lines 14:58:14 to 15:02:03 (wall), area 17 then area 3, steps `Going to town ([no town portal scroll])`, state `town chores`/resupplying, gold 603, belt stocked. The bot had no town portal scroll and walked back through Cold Plains on foot: 103 `nav stuck`, 25 `nav: stuck again at the same spot`, 26 `nav: blacklisted cell` in that window, 0 kills. Best cause: poverty or loss of the portal scroll after the 14:27 death (gold went to potions; 167 `no portal source (empty)` in the run) combined with the nav stuck loop on the Cold Plains exit while walking to town. Other shorter windows: 24 min at 14:42 and 11 min at 14:56 in area 17 under `Quest: fighting (Level up`.
- Belt window 6: not located line by line; d2dsoak reports the length only.
- Fix queue (one agent per criterion): (1) keep one town portal scroll as a reserve before spending gold on potions and avoid the on-foot walk (nav stuck on the Cold Plains exit); (2) belt window: find the 6 minute window first (U where it sits).

## Files
`docs/probes/soak-logs/gate1-rerun-minutes.jsonl.gz`: every `minute` line, the death, fault and recovery lines, the supervisor and fault logs (the full 10 MB log stays on the worker host in the run directory). Reproduce: `go build ./cmd/d2bot ./cmd/d2dsoak` at f4ca264 on the worker host.

## Shared-file lines for the merger
- docs/BACKLOG.md: `- GATE1: Gate 1 rerun on lab2-freezeresume ee5e06d8 + bot f4ca264 NOT PASSED: no kill and no XP 43 vmin (limit 10, on-foot walk to town with no portal scroll), belt window 6 (limit 5), XP/h 7517 vs 9172 indicative; fault recovery 3 s PASS (docs/soaks/gate1-rerun.md)`
- docs/INDEX.md: `- docs/soaks/gate1-rerun.md: Gate 1 rerun 2026-11-01 (NOT PASSED), criteria table, fault recovery, first evidence`
- CHANGELOG.md (unreleased, docs): `- docs: Gate 1 rerun on the rebased d2d binary: NOT PASSED (docs/soaks/gate1-rerun.md).`
