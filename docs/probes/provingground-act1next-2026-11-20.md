# Act 1 inventory and the next two scenario parts (card act1next, 2026-11-20)

> status: current. Tags: V verified (named run or test), S secondary, U unverified. Every scenario setup is CHEATED (see each `provenance`): mechanics evidence only, never balance or play evidence.

> verdicts: current verdicts per scenario and class are in [proving-ground-status.md](../process/proving-ground-status.md) (generated from the results ledger) and on the board page /pg. The tables below record these runs only; new runs are cited by run id, not added here (design 2026-11-21 Q10).

## 1. Inventory (S: read from `data/questboard/act1.json`, `internal/task/act1.go`, `data/questspec/act1.json`, `tools/provingground/scenarios`)

No chunk carries `unsupported` today (V: grep over `data/questboard`), so the board shows "not supported yet" for none; the real gaps are listed in the last column.

| quest | chunks | driven by board | scenario | gap |
|---|---|---|---|---|
| Den of Evil | clear_den | driven | act1.den.clear (this card) | none known |
| Sisters' Burial Grounds | blood_raven | driven | none | boss scenario; the reference hero already holds the title |
| Tools of the Trade | malus | driven | none | the Imbue item pick is `imbue:best` (docs/actbot.md: chooser NOT done); campaign skips imbue below clvl 20 or without a merc: out of scope |
| Search for Cain | scroll, read, stones, free_cain, report | driven | cain.* (9 scenarios, cain.chain green, #993) | none |
| Forgotten Tower | countess | driven | act1.forgotten.countess (this card) | runes are loot, not asserted |
| Sisters to the Slaughter | andariel | driven | none | clvl and potion gates; boss |
| Act 1 done | sail_east | driven | none | Warriv option name U |

## 2. The two parts and why (S)

- Tools of the Trade is not picked: its last step, the Imbue item pick, is `imbue:best` and the chooser is not built (docs/actbot.md), and the campaign skips imbue below clvl 20 or without a merc (`campaign.imbueMinLevel`, `campaign.imbueNeedsMerc`). Whether Normal offline offers it at all is U.
- Sisters' Burial Grounds: already done by the reference hero (title), no scenario needed first.
- Picked, in game order: Den of Evil (`act1.den.clear`) and the Forgotten Tower (`act1.forgotten.countess`). Both are in `tools/provingground/scenarios/act1.json`, driver board-taps (the tapper taps Start on the chunk, then Hand in), Paladin and Sorceress. `data/questspec/act1.json`: the tower chain gets the `return_town` step the handler already has (`home`); `go test ./internal/questspec ./internal/questboard` green (V).

## 3. Verdicts (V, builder main lane, lab d2d, bot of this branch; CHEATED setups = mechanics only)

| scenario | class/seed | verdict | job |
|---|---|---|---|
| act1.den.clear (level 8, 2000 gold) | Paladin/11 | RED: record 1 = 0x0004, chunk never started | 20261120-133340-b8dd |
| act1.den.clear (level 10, 8000 gold) | Paladin/11 | GREEN, 45 virtual min | 20261120-133950-f010 |
| act1.forgotten.countess | Paladin/11 | RED: hero reaches Tower Cellar 5 (area 25), record 5 = 0x0014, `boss: boss not found in the arena`, parked after 3 tries | 20261120-134630-0987 |
| act1.den.clear | Sorceress/3 | see the card result | 20261120-140056-626c |

Findings:

1. Den, first RED (V, log): the setup gold went to the stash by the bank chore, the purse recovery (`recoveryActive`, `internal/bot/questboard.go` boardPreemptsWhy) held the accepted chunk for 45 virtual minutes ("start gate: waiting for the recovery"); den_of_evil is deliberately not `noGold` (TestQuestBoardNoGoldChunkSkipsRecovery), so this is the specified behaviour for a hero without a purse, not a bug; the scenario purse was raised. Open question for the owner (U): a hero that keeps shopping its way below `resource.recoveryKitGold` (500) never starts the Den; the board shows "Starts after the hero has refilled the purse and restocked."
2. Forgotten Tower, RED (V, log lines of the run): the route works (Black Marsh waypoint, the Forgotten Tower, cellars 1 to 5). In Cellar 5 the monster preset 1/836 is not in the d2d preset list (U which verb lists monster presets), the Kolbot hint click {122 71} is refused by the nav mouth gate ("would cross the trigger area of an unwanted exit"), the roam finds no safe click ("nav mouth gate: click held", holds 1 to 3) and the boss run gives up in 10 s; Dread Wight the Dark stands at (117,88) in view, the Countess is never listed. Cause U: either the hint is wrong on the d2d layout or the gate refuses clicks near the arrival mouth. Fix not attempted (nav/boss, needs a trace of the level layout): next card `forgotten tower: locate the Countess when the preset is absent`.
3. Record 5 bit 0 after the kill is U (never reached); record 5 = 0x0014 after the entry to the cellars (bits 2 and 4 = started, V from the run state).
