# Smoke: a one-line verdict per bot change (TASK-111)

> status: built and unit-tested offline (U live: never run on the game yet). It is the regression floor while Gate 2 waves are paused (decision D1, [ADR 0017](adr/0017-direction-fun-first-2026-11-07.md); review section 7 step 5). It does not replace Gate 1 or Gate 2; it is a cheap tripwire.

## Run it
```
tools/smoke.sh [--commit REF] [--class paladin[,sorceress]] [--minutes 45] [--profile tester|gate] [--seed 7] [--record] [--dry-run]
```
Defaults: `origin/main`, Paladin and Sorceress, 45 virtual minutes, profile `tester`, seed 7. Output is ONE line per class and the exit code is 0 (all pass), 1 (any fail) or 2 (could not run or no verdict):
```
SMOKE PASS|FAIL <class> <commit> clvl@45=N kills=N deaths=N notes=<notes or failed rules, spaces as _>
```
`--record` also files `docs/soaks/smoke-<stamp>-<class>.md` (verdict, failed rules, at most 12 sample log lines; a few KB, far under the 256 KB guard). Commit it yourself if it matters.

## What it does
1. `tools/smoke.sh` makes a detached worktree of the commit and runs `tools/smoke-worker.sh` through `labrun $D2_LABRUN_HOST` (local.env) (the worker host) from it, so the bot under test is the one built.
2. The worker half is the recipe of `docs/probes/classpacing/run-class.sh` (V there): fresh character through the normal path, `d2bot play --backend d2d --managed --d2d-args "--lockstep" --goals campaign`, lab binary with lab verbs OFF (no `--lab-verbs`), `nice -n 19`, no setup writes. `D2BOT_RUN` is set so Pacts stay off, and `d2cheat runstart` is written before the bot starts.
3. Profile `tester` adds `--ladder` (decision ladder on, session record under `<data>/sessions`); profile `gate` leaves it off, as the gate runs do, and skips the session-file rules.
4. The wall time is `minutes/12 + 3` minutes of `--run-for` (lockstep is about 18 virtual minutes per wall minute, V in classpacing), about 10 to 15 minutes per class with the build and the game start. Classes run one after the other (one live job at a time).
5. `cmd/d2smoke judge` reads the bot log and prints the verdict. It is a plain Go package (`internal/smoke`) with table-driven tests on synthetic logs.

## The rules (data/smoke/rules.json)
| Rule | Default | Source |
|---|---|---|
| No crash line (`panic:`, `fatal error:`, `goroutine `, `SIGSEGV`) | 0 | |
| No log event at level ERROR except the allow list | 0 | allow list empty |
| At least the judged number of `minute` lines (one per virtual minute) | 45 | short run fails |
| clvl floor per class at minute 20 and 45 | baseline minus 1 (Paladin 2 / 4, Sorceress 2 / 3, others in the file) | `docs/soaks/classpacing.md` (V, one seed) |
| kills at the judged minute | >= 1 | |
| equip refusals (`slot click refused`, `did not reach slot`, `equip retry/failed`) | <= 5 | TASK-182 |
| `did not come up` aura lines | <= 3 | TASK-180 |
| `question` events; no `key` (or `thought`) repeated | <= 12, no repeat | TASK-137 (U: the event and attribute names of the live log) |
| session files present (`session.jsonl`, `decisions.jsonl`, `summary.json`, `summary.md`) and no secret in them | tester profile only | `docs/session-record.md` |
| writes after start (`d2cheat ledger --since-run`) | 0; unknown fails | ADR 0014 |

Deaths are reported, not judged. Change a threshold by editing the file; a class must exist there (an unknown class is an error naming it). The clvl floors are a tripwire for "the bot got much slower", not a pace gate: one seed, and the classpacing runs are confounded by the Blood Moor recovery loop.

## How it differs from Gate 1 and Gate 2
Gate 1 is a long soak with invariants (`internal/d2dsoakstat`); Gate 2 is a 24-hour wave of many characters. Both are paused or expensive (D1). The smoke is 45 virtual minutes, one seed, one class, a handful of rules and no statistics: it catches crashes, a dead hero, spam loops and gross pace regressions, not subtle ones. A PASS is not a gate pass.

## For a merge agent
After a bot merge: `tools/smoke.sh --commit <merged sha> --class paladin,sorceress`; read the exit code; on 1 paste the lines in the merge report. Calling it from the merge worker is NOT done here (the ops repo is untouched). It needs the worker host free of other live jobs.

## Built vs verified
- Built, V offline: the judge, the rules loader, the record writer and the exit codes (`go test ./internal/smoke`), `tools/smoke.sh --dry-run`.
- U (never run live): both shell scripts end to end; the exact log field and event names assumed for questions (`msg` `question`, attribute `key`/`thought`) and for level (`level` `ERROR` of slog JSON); the session directory (`<data>/sessions`); the wall-time estimate; that `d2cheat runstart` plus `ledger --since-run` works with a smoke character name.
First live checklist: `tools/smoke.sh --class paladin --record`, then read the log under `~/d2d-lab/smoke/logs` on the worker host for the minute lines and fix any name in the rules file.

## Sorceress first-hour measurement (sorcready, TASK-267)
The Sorceress rows above (clvl 2 at minute 20, 3 at minute 45) are unchanged. The live plan for three fresh Sorceress first hours (reset, 60 realtime minutes, `tools/firsthour.sh --class sorceress`) is in [sorceress-early.md](sorceress-early.md) section 6; not run (U). The lightning profile now avoids Chain Lightning and Lightning, so a smoke run past clvl 19 no longer exercises them.
