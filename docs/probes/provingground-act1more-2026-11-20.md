# Act 1 boss scenarios: Blood Raven, Tools of the Trade, Andariel (card act1more, 2026-11-20)

> status: current. Tags: V verified (named run or test), S secondary, U unverified. Every setup is CHEATED (level, gold, waypoints, potions by SETUP verbs): mechanics evidence only, never balance or play evidence.

> verdicts: current verdicts per scenario and class are in [proving-ground-status.md](../process/proving-ground-status.md) (generated from the results ledger) and on the board page /pg. The tables below record these runs only; new runs are cited by run id, not added here (design 2026-11-21 Q10).

## 1. What was added (V: `go run ./cmd/d2scenario validate`, `go test -race -count=20 ./internal/questspec`)

- `tools/provingground/scenarios/act1more.json`: `act1.burial.blood_raven` (quest record 2), `act1.tools.malus` (record 3, U), `act1.andariel.slaughter` (record 6). Board-taps driver, Paladin and Sorceress, seeds 11 and 3; death and chicken bounds as `log_line` assertions (`min: 0`).
- `internal/questspec/act1boss_test.go`: the three boss chains kill their boss in an area entered first, with a kill-or-cleared post and `park` on failure; the Malus (`hdm`) is picked before the hand-in; the title "Bane of Blood Raven" exists in `internal/watch/hall_titles.json` as a kill rule on Blood Raven x1 (data check only: no Hall reader in a run); the scenario file covers each boss quest once.
- Not used: `api_reply` / `trigger.api` (only on the unmerged pgcover2 branch, not in main).

## 2. Runs (V, builder main lane, bot of this branch, lab d2d; CHEATED setups, mechanics only; Paladin seed 11 only, the Sorceress/3 runs were not made for lack of builder time)

| scenario | verdict | job and what the state shows |
|---|---|---|
| act1.burial.blood_raven (bank default) | RED | 20261120-184924-bb3e: record 2 never written, chunk parked 28 min on chores (finding 1) |
| act1.burial.blood_raven (bankFirst 0, mixed belt) | RED | 20261120-192148-b1a8: purse kept (1020), still chores "still needed" (belt 3 healing of 4) |
| act1.burial.blood_raven (four healing in belt) | GREEN | 20261120-195436-d0c5: record 2 = 0x2011, no death, no chicken, writes after start 0, 905 s wall; the board reached "ready to hand in to Kashya", the tapper made no Hand in tap before the run ended, so the Kashya hand-in is U |
| act1.tools.malus | BLOCKED | 20261120-200952-46d2: record 3 not observed (never written); the Barracks step refused: "route guard (3 healing potions < 4)" with 1220 gold and 8 potions in the pack (finding 2) |
| act1.andariel.slaughter at level 30 | RED (scenario defect) | 20261120-204236-684a: `reach:30` is already met at level 30, the bot idles in town; level set to 26 |
| act1.andariel.slaughter at level 26 | RED | 20261120-204437-d3bd: no Start was ever offered in 90 virtual minutes (no taps), no `quest board` line; the hero trained through Outer Cloister and Jail (finding 3); record 6 not observed |
| determinism pair act1.burial.blood_raven Paladin/11 | same verdict GREEN, digest differs (warning, wall-clock decisions) | 20261120-210945-74a3 against 20261120-195436-d0c5 |

Scenario defects fixed on the way (mine, not bot): the `log_line` format cannot express `min 0` with a positive `max` (0 means the default 1; the death bound is `max: 0`, the chicken bound counts the single "starting chunk" line in its `max`); a level equal to the `reach:` goal ends the run at once.

## 3. Findings (bot defects, each a separate card, none fixed here)

1. Purse and stash (V, 20261120-184924-bb3e): with the default `town.bankFirst` the bank chore deposits 6900 of 8000 gold in the first minute; the chunk's chores step then could not pay and sat in `needs_attention` for 28 wall minutes. Same family as Den finding 1 of `provingground-act1next-2026-11-20.md`. Card: `town: withdraw stash gold when the quest chores are short`.
2. Route guard against the belt (V, 20261120-200952-46d2 and -b1a8): the route guard and the kit gate want 4 healing potions in a 4-slot belt; the shop plan reports `wantHP 0` while 8 potions sit in the pack, so the belt is not topped up from the pack and the chunk waits with gold in hand. U whether the guard should count the pack or the refill should move pack potions to the belt. Card: `route guard: belt refill from the pack before refusing a quest step`.
3. Andariel never offered, and Outer Cloister nav (V log, 20261120-204437-d3bd): at level 26, Normal, no Start was offered for 90 virtual minutes (readiness reason U: the tapper does not log why no chunk is startable; a cheap tooling follow-up is to log the offer's `readiness` when no Start is possible); the level plan sent the hero toward Jail and Catacombs 3 and `travel to Jail Level 1` failed 3 times with `nav: no exit to that level: level 27 from 26 (nothing new since the last search)` from the Outer Cloister. Cards: `quest board: why Andariel is not startable at clvl 26` and `nav: Outer Cloister to Jail exit not found on the d2d layout`.

## 4. Not done

Sorceress seed 3 for all three; Tools of the Trade and Andariel past their first blocker (the kill, the Malus and the Charsi imbue pick were never reached, so the imbue chooser gap is untested); the Andariel fight itself, so the milestone assertions are written for the kill word (record 6 bit 1) and are UNPROVEN.
