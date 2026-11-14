# Table-dependent tests that rotted (TASK-401)

Tags: V verified by a run on the worker host with the game tables (`D2_MPQ_DIR=$HOME/d2d-lab/data`), U unverified.

## Finding
The town sim tests call `geartest.Need(t)` and SKIP without game tables. CI and the merge gate have no tables, so three tests were never run after they were written.

Whole repo with tables (`go test -count=1 -p 2 ./...`, V): only `internal/town` failed; every other package (loot, resource, bot, env, leveler, invdesk, plandata) passed. A panic aborts the package, so the town list was completed by re-running after each fix. Without `-race` and with `-race` the same three tests fail (the nil-Env panic is not race related).

| test | cause | verdict | fix |
|---|---|---|---|
| TestPlayerStashMovesAndRefusals, TestPlayerStashNoRoomInThePack | the sim rig's `Env.Read` is nil (the world is wired through `town.Source`); the test called `r.Env.Snapshot()` | stale test | use `r.W.Snapshot()` |
| TestTidyStashesByPolicyAndNeverTouchesEquipmentOrMarks | plan `Stash: true` makes the pickit stash chore bank every rune BEFORE tidy, the marked one included, so tidy never ran | stale test, NOT a product bug: with `Sell: true` tidy runs alone, stashes the unmarked rune and leaves the keep-marked one in the pack, equipment unchanged (V, passes for Sorceress and Paladin). Stashing a keep-marked item is harmless (keep forbids selling, not stashing) | plan `Sell: true` |
| TestUnidentifiedJewelryWaitsWhileThePackHasRoom | counted rings in the pack only; tidy now stashes rare jewelry by design (docs/tidy.md), none was sold | stale test | count pack plus stash |

No product code changed. Tidy never dropped or sold an item in these runs (V).

## Proposal: go-tables gate (card to open, not built)
Goal: these tests must run somewhere on every merge. The merge script lives in the ops repo, not edited here.
- Add an extra gate `go-tables` to the merge step, run on the worker host after the normal gates, from inside the merge worktree:
  `D2_MPQ_DIR=$HOME/d2d-lab/data go test -count=1 -p 1 ./internal/town/ ./internal/loot/ ./internal/resource/ ./internal/bot/ ./internal/plandata/`
- Fail the merge on a non-zero exit; fail also when the tables dir is missing (the tests would silently skip: check `test -d "$D2_MPQ_DIR"` first, and optionally grep `-v` output for `SKIP` counts above a baseline).
- Cost (U): the town package took about 3 s with tables, the other four are small; the gate adds well under a minute.
- A skip guard: `tools/ci/flaky.txt` style allowlist for any test that is known to need the game itself.
