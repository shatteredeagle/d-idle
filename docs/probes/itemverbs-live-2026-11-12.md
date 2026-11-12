# Item verbs: live click-vs-verb comparison (TASK-383, 2026-11-12)

> status: result record. Tags: V = run and seen on the worker host. Driver: `tools/itemverbs/live_compare.py`; design: [../design/item-verbs.md](../design/item-verbs.md).

## Setup (V)
- Throwaway heroes `Ivp` (a Paladin, d2d class index 2, and a Sorceress, index 1) created fresh in a headless d2d on the worker host, own temp data dir and ports; the sandbox container and its volume were not touched. Real game files mounted read-only. Logged in `test-characters.md`.
- SETUP writes (declared, before the bot starts, `debug give` and `debug stat` through the lab verbs of the harness d2d): 11 items in the pack (two Hand Axes, a Cap, a Plate Mail, three unidentified Hand Axes, two Scrolls of Identify, a Short Staff, a magic Cap), gold 5000, stash gold 2000. The bot then ran against that save with the real `d2bot` (managed d2d, `--item-verbs`, farm goal Blood Moor, knobs `town.sellJunk 0` and `town.stashGoldBase` high so the town chores leave the test items alone). No write after the bot started: `writes after start: 0`.
- Two modes per class from the identical save: click (`invdesk.verbs` 0) and verb (`invdesk.verbs` 1). The steps were queued through the same watch API the inventory page uses (`/api/inventory/*`), run by the bot at its safe moment, and the item view was recorded after each step.

## Result table (V)
| Step | Paladin click | Paladin verb | Sorceress click | Sorceress verb | Verdict |
|---|---|---|---|---|---|
| equip Cap to head (empty slot) | done 26.6 s | done (verb) 23.6 s | done 26.6 s | done (verb) 25.2 s | same state |
| equip Plate Mail to torso (strength too low) | refused `requirements` | refused `requirements` | refused `requirements` | refused `requirements` | same (bot pre-check, the verb is not reached) |
| equip Hand Axe to right hand (swap) | done 11.5 s | done (verb) 11.5 s | done 11.5 s | done (verb) 10.1 s | same state |
| equip Short Staff to right hand | refused `requirements` | refused `requirements` | done 10.0 s | done (verb) 10.0 s | same (class rule, both modes alike) |
| identify unidentified Hand Axe 1 (scroll) | done 10.1 s | done (verb) 10.1 s | done 10.0 s | done (verb) 10.1 s | same state, one scroll used |
| identify Hand Axe 2 | done 10.1 s | done (verb) 10.0 s | done 11.6 s | done (verb) 31.2 s | same state |
| identify Hand Axe 3 (no scroll left) | refused `requirements` (no tool) | same | same | same | same |
| drop (trash) a Hand Axe | FAILED "the item could not be dropped and is back in the pack" (34.1 s, 26.6 s) | done (verb) 16.1 s, 10.1 s | failed | done (verb) | DIFFERENT: the click path failed on both classes, the verb dropped the item; state afterwards equal except that one axe |
| stash a Cap | done 43.2 s | done (verb) 23.7 s | done 23.6 s | done (verb) 16.1 s | equal once the dropped axe is left out |
| take the Cap back | done 7.0 s | done (verb) 11.5 s | done 7.1 s | done (verb) 11.6 s | equal ignoring the axe |
| gold deposit 300 | done 13.0 s | done (verb) 8.5 s | done 14.6 s | done (verb) 10.0 s | gold 4500 / stash 2300 in both |
| gold withdraw 100 | done 7.1 s | done (verb) 11.6 s | done 7.1 s | done (verb) 10.1 s | gold 4600 / stash 2200 in both |

The driver's strict step-by-step verdict says DIFF from the drop on (10 of 24 rows); the difference is the one dropped item (the click path could not drop it) carried through the later rows. Re-reading those rows with the Hand Axes left out: equipment, pack, stash, gold and stash gold are identical in every row (V, the python check over the saved traces).

## What it shows
- Every verb step ran as `how=verb` through the real bot, the same safe-moment rule and queue; nothing fell back to the click path. Equip, swap, identify (the tool is chosen by the fork: a Tome with charges, else a loose Scroll), stash and take at the chest, gold deposit and withdraw match the click path's end state on both classes. The stash and gold verbs ran while the stash panel was open at the chest (the strict rule: `far` otherwise).
- Latency is dominated by the bot's own safe-moment and queue polling (10 s steps); the verb saves the panel work on stash and drop (stash 43 s vs 24 s, 24 vs 16 s). Equal elsewhere. Small n (one run per cell).
- The click path's drop failed live on both classes (the item stays in the pack). Cause found afterwards (TASK-385): on d2d `ClickWorld` is `cmd move`, a walk order, so a held item never dropped; fixed with a real mouse press (`DropClick`) and verified live on both classes (click-path trash: done, 25 s).
- Not driven through the bot (no API for them, or unreachable): unequip (no request kind), cube moves, stash too far (the town walks to the chest first), stash full / backpack full. Covered by the fork's headless e2e and unit tests (`item_verbs_d2d.py`, `test_item_verbs`): unequip, cube grid, `far` for stash moves and gold away from the chest, `nospace`, `overlap`, `bounds`, `notool`, `gold`, `limit`.
- Bug found and fixed by this run: the first rule for "at the chest" used the walk target (`interact_npc`), which is cleared once the panel opens, so every stash and gold verb was refused `far` right after the bot opened the stash. Now: stash panel open and a stash object within 3 cells. The refused requests failed cleanly (request failed with the reason, nothing moved) and the next run passed.

## Repeat
`python3 -I tools/itemverbs/live_compare.py --bot BIN --d2d FORK_D2D --game GAMEDIR --out DIR --classes 2,1` on the worker host (about 25 minutes). Prerequisite: the fork build with `--item-verbs`.
