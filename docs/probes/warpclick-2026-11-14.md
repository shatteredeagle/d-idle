# Warps on the rebased d2d need a Select-box click (TASK-412, 2026-11-14)

> status: result record. Tags: V = run and seen on the worker host (headless throwaway heroes, seed 7 unless said); S = read in source. Follows docs/fork-rebase-2026-11-13.md and docs/probes/killdrop-2026-11-13.md.

## Symptom

On d2dbot:devp6 (fork `dd70428d`) the hour-4 hero walked to the Den of Evil's mouth four times and never entered (`nav exit tile did not fire`, then `nav: no exit to that level: level 8 from 2`); on the old fork lineage (iv1, `a42dba25`) it entered.

## Cause (V)

1. Upstream `66aece1a` (2026-11-07) takes a warp only for a click inside LvlWarp's Select box round the warp's tile unit (`World::warp_clicked`, FUN_00463ee0), as game.exe does; before, any click within 2 cells of the tile's centre took it. The bot's final push clicked the tile's corner subtile (`d2map.Exit.Pos`), which is outside the box at most cave mouths. Headless, hero placed beside the tile, the bot's click against the new click point (`tileclick.sh`, `dirs.sh`): the tile click fails for Blood Moor to the Den (2 to 8), Cold Plains to the Cave (3 to 9), Stony Field to the Underground Passage (5 to 10), Dark Wood to the Hole (6 to 11) and the Crypt's return (18 to 17); it worked for 4 to 10 and 17 to 18. The click point takes all of them. Class (a): an upstream change towards game.exe that the bot must follow.
2. Upstream `0ddb6fe7` (2026-11-06) walks a clicked warp toward its tile unit and takes it within unit distance 5. Where the unit stands on a blocked subtile the walk stops short and the warp never fires: Black Marsh to the Countess's tower, level 20 (6 to 20) 0 of 8 start directions with the click point, 7 of 8 when the walk aims at the free spot nearest the unit; the old lineage 3 of 8 with its own rule. A d2d deviation from game.exe (game.exe enters there), so a fork shim plus a draft upstream candidate (docs/upstream/pr-queue.md row 9, NOT READY: no game.exe differential of the approach walk).

Not the rebase: neither change is in a conflict resolution of TASK-395, and the bot's `debug players` fix (TASK-396) was already in.

## Fix

- Fork `didle/rebase-20261114b` on `dd70428d`, head `75891650457803a7de2522b7a0d83512250ff279` (private forge, pushed 2026-11-14): `debug warps` appends the tile unit's cell and a subtile-centre click point that `warp_clicked` accepts (`-1 -1` when none within 3 subtiles; the first four columns unchanged) (060c8901, c3e24fcb); a clicked warp's walk aims at `nearest_free(unit)` (75891650; c0555966 and its revert a1e50a79 stay in the history: the revert came from a Den-only test, level 20 showed it is needed).
- Bot (`agent/warpclick`): `d2dctl.Warp` parses the four new columns (`HasClick`), `d2dworld` carries the click point on `d2map.Exit`, `nav.gateCands` pushes it instead of the tile. An older d2d (four columns) keeps the tile click. Tests: `TestParseWarpClickColumns`, `TestTileGatePushesTheClickPoint`. Not build-specific.

## Checks (V)

- Headless click point, hero 2.5 cells off the unit in 8 directions, monsters killed first (`dirs.sh` on the fork build): Den 7 of 8, Hole 8 of 8, level 20 7 of 8, Burial Grounds to the Mausoleum 7 of 8, to the Crypt 6 of 8 (the misses: start spots in pockets that no walk leaves). Inside level 20 (20 to 6, 20 to 21) no click takes the warp on either build (0 of 8 new, old 2 of 8 for 20 to 6 and 0 of 8 for 20 to 21; the click point there is at the map edge, y below 0.2): U, a separate card if the journey needs level 21 by walking.
- ctest on the fork build: `test_warp_arrival`, `test_game`, `test_rules`, `test_devctl`, `test_click_keys`, `lab_verbs_d2d`, `item_verbs_d2d`, `coop_d2d`, `pad_use_d2d`: 9 of 9 pass.
- Image `d2dbot:devp7` (`sha256:d46e16a59a32...`): d2d `75891650`, bot = agent/warpclick (the tree before a comment-only amend that renumbered the task in two comments). SOURCE.txt names the published URL and the ref. GPL source tarball `d2d-didle-75891650457803a7.tar.gz` (worker host, in the fork's rebase output folder), sha256 `f0e4904ec9ec3ba793bb753ed0af05dc0cd9a886cea398f537a52f300e902e7e`.
- Bot runs on devp7 (fresh Paladins, 22 minutes, realtime, seeds 7 and 11): both entered the Den of Evil (seed 7 twice), 0 `exit tile did not fire`, 0 `no exit` lines, 0 unreachable, 0 deaths, 204 and 181 kills, bounces 1 and 0. Level 20 was not reached in 22 minutes (U in a bot run).

## Swap (NOT run; the sandbox runs devp6)

Back up first as for the devp6 swap (save dir, state, settings), then on the worker host:
```
sh ~/play-build-20261108/rb/swap-image.sh d2dbot:devp7
```
The bot inside comes from the image only if no newer dev-update release is active; after a dev-update, deploy agent/warpclick (or main once it merges) with `tools/dev-update.sh --commit <ref> --to d2play-20261108`. Rollback: `sh ~/play-build-20261108/rb/swap-image.sh d2dbot:devp6`.
