# Live check drills: stash (codefix67) and tome stat 70 (tomefix)

> status: current, offline only, NOT run live. Tags: V = verified (code read or fake-lab test here), S = secondary, U = unverified live.

Two groups for `d2drill`, written to the pattern of D8.h and RT: setup cheats go through the logged cheat tool before the bot code runs, PASS conditions are numbers, skills/classes/prices come from the tuning and item tables. Fake-lab tests: `go test -p 1 ./internal/drill ./cmd/d2drill` (V; the old withdrawal and a qty dropout make DS.1 and TM.a fail).

## How to run (same recipe as docs/probes/livedrills2/2026-11-04-run.sh)
On the worker host, from the synced worktree, with the lab binary that has the test cheats, class Sorceress clvl about 11 in town (the recipe's character; U which level the keep line then gives):

    docs/probes/livedrills2/2026-11-04-run.sh LABEL GROUP ONLY REPEAT
    docs/probes/livedrills2/2026-11-04-run.sh livecheck1 DS DS.1,DS.2,DS.4 2
    docs/probes/livedrills2/2026-11-04-run.sh livecheck2 TM TM.a 1

(The script builds `cmd/d2drill`, runs it through `tools/d2dslot.sh`; the run script's `--ttl 200` is too short for TM.a: copy it and use `--ttl 2400 -t 2300`, U.)

## Group DS (P5, docs/codefix67.md)
Setup per drill: ClearPack, DrainBelt, `debug stat 14 <purse>`, `debug stat 15 <stash>`; then the bot's town chores `potions stash` once (`town.Chores`). Keep line and kit cost are read from the run's tuning (`StashGoldBase + level*StashGoldPerLvl`, `KitTPGold + max(HPFirst,1)*KitPotionGold`), the rejuvenation code from the item table (lowest tier).

| id | scenario | PASS |
|---|---|---|
| DS.1 | 600 purse, 1400 stash | stash not lower afterwards, Spent > 0, belt grew |
| DS.2 | 6 purse, 1400 stash | stash fell by exactly keep line minus 6, belt grew |
| DS.3 | 6 purse, stash 0, shop cannot sell | at most 1 emergency trip line and 1 stash round over 4 Runner rounds. BLOCKED: needs (a) `TripLab.RunnerRounds`, the bot Runner loop in town (the futility latch is Runner state, `Chores` cannot show it), (b) a d2d verb that empties or blocks a vendor stock; neither exists |
| DS.4 | one rejuvenation in the pack (`debug give`, pickup), 6 purse, stash 1400 | stash unchanged. BLOCKED by the drill itself when the potion is not in the pack after setup |

Verbs: `debug stat 14`, `debug stat 15` (V live, docs/probes/drills/README.md), `debug give` (V), `debug belt clear`/belt drink (V). DS.1 is BLOCKED if kit cost is above 600; DS.2 if keep-6 is above 1400.
Risks: the character's real level decides the keep line; a withdrawal and the shop may need two chore runs on a slow window (U); DS.2 compares the withdrawal with the keep line exactly (S: `needWithdraw` tops up to the keep line, docs/codefix67.md).

## Group TM (P10/P12, docs/tomefix.md)
| id | path | PASS |
|---|---|---|
| TM.a | ground ids beside the tome (path 11) and any qty dropout seen in the field | tome bought and filled by the `scrolls` chore (D3.tp, V), at least 13 charges else BLOCKED; 10 ground items dropped with `debug give`; 60 looks 750 ticks apart (about 30 virtual minutes, U tick rate); 0 looks with the bot reader (`PView.Tome`) at 0 or no tome; no log line `dump items failed`, `items verb`, `no70` or `tp-debounce-flip`. On FAIL the reason names the path: empty read next to `dump items failed` = path 10, next to `items verb` = path 9, with neither = fork qty dropout (then capture the raw `dump items` JSON) |
| TM.b | path 9, stock `items` verb never sets qty | BLOCKED: needs a d2d verb or option that makes the producer use the stock `items` verb for one look |
| TM.c | path 10, `dump items` fails | BLOCKED: needs a fault-injection verb that makes `dump items` return an error for one look |

Verbs used: `debug give` (V), `debug stat 14`, goto (V), `dump items`/`ground` by the bot's own reader. TM.a only observes natural occurrence of paths 9/10; the forced cases are TM.b/TM.c. U: that `cap` stays on the ground under the drill's pickit (a filtered item is needed so the ground list is non-empty), that the field pickup in Blood Moor does not take the tome out of the pack.
Risks: 30 virtual minutes in Blood Moor with a clvl 1 to 11 character may meet monsters; the drill does not fight, so a death ends it as FAIL (setup) and is not a verdict on the tome.

## Gaps for the merger
Proposed BACKLOG line: livecheckdrills: groups DS (4 stash scenarios, DS.3 BLOCKED) and TM (tome stat 70, TM.b/TM.c BLOCKED) ready for the live recheck, docs/livecheckdrills.md
