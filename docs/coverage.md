# Test coverage and unattended-run gaps

Measured on the worker host: `labrun worker -- go test -short -count=1 -cover ./...` (V: ran 2026-10-29 on master d05f560).
New tests live in `zz_cover_*`-style files: `internal/{fleet,goals,resource,loot,town,launch}/zz_cover_test.go`,
`internal/bot/zz_cover_{task,loop}_test.go`. No non-test code was changed.

## Critical packages (before -> after)

| package | before | after | notes |
|---|---|---|---|
| internal/bot | 83.9% | 85.4% | loop/failsafe already dense; added task-act fault mapping, KillBoss, waitArea/travelTo |
| internal/resource | 91.3% | 92.1% | abort decision sweep (no portal without scroll, no retreat away from town) |
| internal/town | 66.0% | 66.4% | ToTown/BackToField/Chores cancellation; found a nil-deref (below) |
| internal/loot | 77.8% | 79.8% | bounded pickup of an unhoverable item, cancel, item stolen |
| internal/launch | 58.0% | 58.5% | StopGame, signal/spawn errors, helper never listens |
| internal/goals | 90.8% | 91.8% | Manager.Run cancel+save, clock jump not credited, save failure |
| internal/fleet | 81.9% | 84.1% | wedged Start vs Remove, login retry, HTTP restart/unlease |

Other packages (same run): canary 87, cheat 50, combat 79, d2 70, data 93, event 97, gear 90, goals/sim 92, input 43,
leveler 47, mem 73, nav 81, quest 90, sentinel 46, setup 67, status 44, store 84, task 91, viewport 69,
town/hand 12, town/belt 52, town/panel 73. cmd/ and tools/ mains: 0 (live tools).

## Real bug found

- BUG (skipped test `TestCoverToTownCancelledContext`): `internal/town/trip.go:69` and `:89` assign `s` from
  `waitArea`, which returns a nil snapshot on ctx cancel or a Snap error. A later iteration (scroll already spent) gets
  `ErrNoPortal` from `openPortal` and calls `walkToTown(ctx, s, err)` at `trip.go:80` with `s == nil`;
  `trip.go:114` (`s.Player.Area`) panics. Reproduced on the worker host (SIGSEGV). Effect: cancelling a goal or stopping the bot while
  a town-portal trip is in flight can crash the whole process. Fix: use a separate variable for the waitArea result
  (or re-read `t.current()` before walkToTown).

## Remaining gaps (untested, ranked by 24 h risk)

1. internal/launch (58%): `Launch/Attach/Shutdown`, `managed.go` Start/WaitUp/Stop/Reap, `proton.go` spawner,
   `install.go` Run, OSSystem. All need real processes; only the fake-System lifecycle is covered. Zombie/orphan handling
   (`subreaper.go`) has a separate proof tool.
2. internal/town: `repair` (0%), `buyKeys` (14%), `Hire/Revive` merc (0%), `openNPC` timeouts (54%), `openStash` (48%),
   `enterPortal` fallbacks (57%), `PickLine`. Vendor UI waits are the likeliest place for a stuck chore.
3. internal/bot: `KillBoss` hook path (Task.Boss), `SmashOrb`, `scanQuestItems`, `campaign` (0%), `wakeUp` (empty hook),
   `starveArea/avoidedNow` (leveling avoid list), `lowLifeGate`, `ensureDifficulty`.
4. internal/fleet: real `Runtime` adapters (`runtime.go` 0%), `bot.go` process wrapper (Start/Signal/Kill/Wait 0%),
   `Cores`/cgroup reads. Fake runtimes cover the supervisor logic only.
5. internal/goals: `Catalog`, `chainFail`. internal/loot: `evaluate` custom-pickit branch, `SetPickit`.
6. internal/sentinel 46%, internal/status 44%, internal/input 43%, internal/town/hand 12%: not reviewed here.

## Notes

- `internal/bot` failsafe tests were flaky once under load on the first full run (ran green on rerun, 28 s); tools/ci/flaky.txt
  is empty. Watch it.
- Tests use virtual clocks where the package has one (bot `vclock`, fleet `clock`, town sim); the ones that could spin
  carry a 5-20 s wall-clock guard.
