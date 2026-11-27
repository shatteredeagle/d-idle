# Input layer I5 to I7: chores, waypoint and drink on verbs; SemanticInput; hotkey map

> status: bot side partly built offline (TASK-989, card inputl5, 2026-11-27); everything stays dark behind `input.planner` (0) and `input.hotkeyMap` (0). Live: U. Tags: V verified (how is said), S secondary (read from code), U unverified.

Roadmap R17, rows I5 to I7 of [07-input-layer](fable-review-2026-11-19/07-input-layer.md) section 6; rule of [ADR 0021](../adr/0021-input-rule-verb-key-click.md). The verbs themselves are in [input-i3.md](input-i3.md) (use-item), [input-i4.md](input-i4.md) (buy, sell, repair, waypoint, menu-choose) and [input-cast-verbs.md](input-cast-verbs.md).

## I5: what is on verbs

| chore | verb | state |
|---|---|---|
| buy, sell | `input.buy`, `input.sell` | adopted in `internal/town/shop.go` (S, `AsWorldVerbs`) |
| waypoint | `input.waypoint` | adopted in `internal/nav/waypoint.go` (S) |
| repair | `input.repair` | adopted by TASK-988 (`town.pressRepairAll`, V by `internal/town/repairverb_test.go`) |
| town portal | `input.use-item` | adopted by TASK-986 |
| drink from the belt | `SemanticInput.Belt` (`cmd belt`) | exists in `internal/combat/packdrink.go` and the d2d town (S); no caps name, the ledger row `input.planner` governs it |
| menu choice | `input.menu-choose` | no adopter: see input-i4.md section 4 |
| item, gold, identify verbs | `input.item-*`, `input.gold-stash` | listed in the caps set; promotion waits for their differentials (U) |

Open ledger rows named by the card (`invdesk.verbs`, `mgmtfast`): not decided here; no knob was added or changed.

## I6: SemanticInput and the pointer count

`input.SemanticInput` (`internal/input/semantic.go`) is still optional: behaviour packages type-assert it. Making it the only interface of behaviour packages is a refactor of about 107 call sites (below) and is NOT done; the ratchet is.

CI check `tools/ci/pointercount.py` (run by `tools/ci/check.sh`, unit test `tools/ci/test_pointercount.py`): counts text matches of `In|in|inp|ctrl` `.Click|MoveMouse|ClickWorld|Key|Hold|Release|MoveTo(` in non-test files under `internal/`, outside the adapters `internal/input`, `internal/d2dinput`, `internal/d2dctl`, `internal/d2dtown`. It fails when a package is above `tools/ci/pointercount-baseline.json`; after a migration run `python3 -I tools/ci/pointercount.py --update` to lower the baseline. Baseline at this commit: 107 sites (combat, town, nav, arbiter, leveler, launch and the drill packages lead). S: it is a text match, not a type check; it also counts packages that never run on the d2d path (launch/oog, the drills), so the target "0 on the d2d path" needs a path allow-list when the migration reaches them. `cmd/` is not scanned.

## I7: hotkey map

Already built and read back before this card (V by `cmd/d2bot/hotkeymapwire_test.go`): knob `input.hotkeyMap` (0, dark-launch row in `data/darklaunch.json`), `hotkeymap.Generate` from the agency plan's slots, `hotkeymap.Apply` writes the table and reads it back (`written`, `already`, `beyond` in the log line `hotkey map applied`); step `d2dHotkeyStep` in `cmd/d2bot/d2dwire.go` runs it at prepare time. Live: U (needs a hidden instance).

## Enable criteria and checklist

Pointer count 0 on the d2d path (not met: 107); `invdesk.verbs` and `mgmtfast` rows decided (open). Live drill (not run): hidden instance, `input.planner=1`, `input.hotkeyMap=1`, a town trip with repair and a waypoint, read `hotkey map applied` and the minute line `input=v/k/c`, leave the hero in town with no panels open.
