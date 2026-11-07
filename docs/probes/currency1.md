# Probe: currency slice 1, the Raven's Mark rows (TASK-63)

> status: UNVERIFIED live, 2026-11-06. Offline checks passed; the live run was not started because Gate 2 seeds were running on the worker host (rule: no live work while a gate seed runs). Tags: V verified, S read in source, U unverified.

## Offline (V, worker host, 2026-11-06)
- `go test -count=1 -p 1 -v -run Didle ./internal/overlay`: the three fragments merge onto the user's own MPQ tables (Misc appended 1, CubeMain appended 1, TreasureClassEx appended 1 and replaced 1); `mrk` unused; Blood Raven class `Picks` 5, `NoDrop` 19, total weight 42 with our entry; per kill 0.02703 (1 in 37.0); Monte Carlo of the fork's roll within 5 sd.
- `go test -count=1 -p 1 ./internal/loot`: rule `currency marks` keeps `mrk` for three builds.

## Live acceptance (U, ready to run)
Binary: the fork branch `d2d.lab4-overlay` (tip `e66d0fdf`, private remote) built on the worker host (`build/apps/d2d/d2d` in the overlay-fork worktree; it contains `--lab-verbs`, `f75816c6` is an ancestor). Record its sha256 in the run line.

```
D2_MPQ_DIR=/tmp/d2mpq nice -n 19 python3 docs/probes/currency1-drill.py <d2d> data/overlay/didle-v1 300 15
```
(through `tools/agentrun.sh currencyrows --ttl 900 --`). A throwaway headless character (`--no-save`, private HOME, classic), `--lab-verbs` for `debug goto 17`, `debug spawn bloodraven 15`, `debug kill` only; no write touches the measured quantity. Pass: `info` ends with `overlay=<hash8>`; the d2d starts (a bad Misc/TC/Cube row would stop it); `ground` shows `mrk` items; the count of `mrk` is inside the printed 99.9 percent interval around 300 x 0.0270 = 8.1 (sd 2.8); each label reads `Raven's Mark`. Then, by hand or an extension of the script: pick one up (`cmd pickup`), note `quantity` (expected 1), put it in the Cube, transmute, see `transmute: DI bank: mark (1 made)` in the log and the stack gone (the pack-loss observation).

Risks (U): `debug goto 17` may need `debug act` first; `spawn` makes a plain monster so only the `TreasureClass1` path is exercised (same class); a TC defined after its user is resolved at roll time in the fork (S: `tables.treasure.find` at roll), game.exe order is not tested.

## Result
None yet. Fill: run id, binary sha256, overlay hash8, kills, mrk count vs expected, d2d accepted the code (yes/no), quantity, name text, bank recipe result.
