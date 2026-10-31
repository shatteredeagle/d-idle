# Quest layer: internal/quest (agent QUESTS, offline)

Package `internal/quest`, stdlib plus `internal/saves` only. Nothing here touches the game; every claim that needs it is tagged U and
listed in "Live checks" below. Tags: V verified live, S secondary source, U unverified.

## What exists

| file | content |
|---|---|
| `state.go` | `Interpret(word)` (the one place bit meaning lives), `Source` interface, `SaveSource` (saves.Progress), `MapSource` (memory/test), `Reader` (sources in order, freshest first), `Status` Needed/Started/Done, `Evidence` None/Proxy/Save/Memory, `Skippable` (Done on Save/Memory only), `ActOpen` |
| `dialog.go` | `Dialog` interface (`Talk(npc)`, `Choose(row, of)`, `Close()`), `Menus` table (per NPC, per level main/talk, rows with Tag V/U and state-dependent visibility), `RowIndex`, `Pick`, `PickPath`, `DoTurnIn`, NPC and object class constants |
| `fake.go`, `fakeenv.go` | `FakeDialog` (validates row index and line count against the table), `FakeEnv` (area, pack, scripted travel and failures) |
| `steps.go` | `Step` data, `Env` interface, `RunSteps` |
| `hops.go` | `Hops` A1->A2, A2->A3, A3->A4, A4->A5 as step lists with `Pre`, `FromArea`, `To`; `RunHop` detects success by `Env.Area() == To` |
| `act1.go` | `Act1` handlers (Den of Evil, Blood Raven, Search for Cain, Countess, Smith, Andariel) as data; `RunHandler` |

Tests: `go test -p 1 -count=1 ./internal/quest`.

## Decisions the task layer must know

- Bit meaning: Done = bit 0 or bit 15 (V: both read "completed in a previous game"); Started = bit 2, 3 or 4 (V for 0x000c and 0x0010 alone);
  Pending = bit 1 (U). Changing it is one function.
- A source that knows a quest wins in list order: `NewReader(memSource, SaveSource{p})`. A future memory reader just implements `Source`
  (`Word(diff, questID)`, `Evidence()`); quest ids are the `saves` table ids.
- Skip only on `Skippable` (Done with Save/Memory evidence). After a handler runs, `Result.Confirmed` is false until the save or memory says Done:
  that is expected (the save lags), the caller records T3 and re-reads after the next Save and Exit.
- `Dialog.Choose(row, of)` takes the line count because the box is bottom-anchored (town.entryPoint needs n). Rows are picked by LABEL
  through `Menus`, so one corrected table row fixes every handler. Rows with `When` disappear/appear with quest state; `Env.Status`
  (FakeDialog.Status) filters them the same way as the real adapter must.
- `Env` is the one interface the task layer implements over nav, combat, loot and town (GoNPC, GoArea, Clear, Kill, Pickup, UseObject,
  TouchWaypoint, WaitArea, ToTown plus Dialog and Has).
- Handlers never persist progress; re-derive from the Reader and the pack (execution.md section 5).

## Wired by INTEGRATE (2026-10-28)

- `cmd/d2bot/task.go`: `saveQuestReader` (a `quest.SaveSource` over the character's .d2s, re-read when mtime/size change, maps task quest ids to save ids) and `dialogAdapter`: `quest.PickPath` over a `townDialog` (`town.OpenForTest` to open, new `town.PickLine(idx, n)` to click a bottom-anchored line, `CloseAll` to leave). `talkPaths` holds the label path per (NPC, pick): Akara quest/reward is `Talk` then `Den of Evil`; Kashya/Charsi/Warriv rows are U.
- The save is prior knowledge only (it lags the game until Save and Exit): Done and Started are reported, an untouched quest is left to proxies.

## Requests for other owners (not done here)

- internal/town: a `Dialog` adapter (Talk = openNPC incl. greeting wait; Choose = entryPoint(idx, of) click; Close = CloseAll/Cancel + flag 0x08 clear). `town.MenuOrder` only knows the main menu; it could be generated from `quest.Menus`.
- internal/bot: an `Env` adapter and a `quest.Reader` fed from the save (and later memory); `Env.WaitArea` polls Player.Area.
- A memory quest reader (docs/goals/tracking.md items 1-4) as a `quest.Source`.

## Live checks (main agent; one dated line per check in docs/progression/results.md)

Run on a throwaway character, Rogue Encampment unless stated. For each: write what was seen and flip the Tag in `dialog.go`.

1. **Warriv (class 155) main menu.** Talk to him before Andariel and after (cheat Andariel quest words 6 and 7 to 0x0001 with `d2cheat`). Record row count and labels. Expect Talk / Go East / Cancel only after word 7 is set; confirm `Go East` is row 1 of 3 and that before word 7 the menu is Talk / Cancel (our table then makes `Go East` absent). Click it: Player.Area must become 40 (Lut Gholein).
2. **Kashya (150) main and Talk menus.** Record rows. Confirm the existing `town.MenuOrder[150] = Talk, Cancel` is not the Hire layout; confirm whether `Hire` is a main row.
3. **Charsi (154).** With Horadric Malus (`hdm`) in the pack (cheat or Smith kill): does the Talk alone consume it and add an `Imbue` row? Record the row index of Imbue, and what follows (item pick). Without the Malus: Talk / Trade / Cancel (the repair is inside the trade window).
4. **Akara (148) beyond Den.** After Den of Evil is cleared (word 1 = 0x0001 plus kill): does Akara's Talk menu lose the `Den of Evil` row (our table assumes Cancel moves up to row 2 of 3)? Does re-talking grant the skill point (stat 5 +1)? With Scroll of Inifuss (`bks`) in the pack: does the Talk alone consume it and give `bkd`?
5. **Cain (town class 245).** After the rescue: main menu rows (expect Talk / Identify / Cancel), and whether talking ends Search for Cain (word 4 -> 0x0001). Confirm class 245 with `townlive units`.
6. **Item codes.** Pack dump with the Scroll (`bks`), Cairn key (`bkd`), Malus (`hdm`).
7. **Object classes.** Tree of Inifuss (30), Cairn Stone (17), Cain's cage (26), and the area ids in `act1.go` (Den 8, Burial Grounds 17, Barracks 28, Catacombs 4 = 37, Tristram 38) via nav state.
8. **A2->A3.** Meshif class (210 A2, check A3), the `Sail East` row index after Duriel; Jerhyn (201) talk effect on word 15; arrival area 75.
9. **A3->A4 / A4->A5.** Red portal object class (60) in Durance of Hate 3 (area 102) and arrival area 103; Tyrael (367) rows after Diablo and arrival area 109 (expansion character only).
10. **Quest-word bits per handler.** After each handler, Save and Exit and read the word: Done should be 0x0001 (or 0x8001) for every quest above. Note any quest that ends at another value; `Interpret` is the only place to change.

## Dead ends / not done

- Cairn stone order, Tristram clearing and the Cain cage are single steps with a note; a real handler needs the order verified first.
- Imbue (item pick after Charsi's Imbue row) is not implemented.
- Acts 2-5 quest handlers are not in scope here.

## Akara dialog by quest state (TOWNFIX, V live on i4, 2026-10-28)

| Den word | Talk | result |
|---|---|---|
| 0x0000 needed | topic list Introduction / Gossip / Den of Evil / [Sisters' Burial Grounds if started] / Cancel | click Den of Evil: quest starts (0x0004) |
| 0x0004, 0x001c | no list, a speech | nothing paid |
| 0x001e reward pending | no list, pays at the first click | +1 skill point, word 0x8001 after Save and Exit, quest log "completed" |
| 0x8001 done | no list | speech |

The main menu has 3 lines, 4 once the free respec exists ("Reset Stat/Skill Points", word 41 set by the claim): Talk and Trade then sit one line higher (bottom-anchored). `quest.Menus` models it with the pseudo quest `quest.RespecRow`, `quest.HasSubmenu` models "Talk opens the list only while Den is Needed" (`PickPath` returns `ErrNoSubmenu` otherwise). cmd/d2bot `dialogAdapter`: the Den reward claim is the single label "Talk", verified by the skill point, retried with the other line count (3 attempts, dialog reset between). `town.pickEntry` also tries +-15 px so Trade works with either count. Open U: whether the free respec row disappears after use.

## EARLYFIX (2026-10-29): early-game fixes after the live Paladin
Source: docs/classes/paladin.md "Live results" (V live as symptoms; the fixes below are offline-tested only, U live).
- Den of Evil "cleared": an exhausted explore is not "all monsters dead". `bot.sweepEndsQuest`: with a live quest reader (`TaskDeps.QuestsLive`, questmem) the run only finishes when the den_of_evil word reads Cleared (reward pending); otherwise the bot walks to a leftover monster up to 3 sweeps, then gives the area up without claiming. The campaign world no longer counts the Den cleared by the sweep proxy while the live word says open.
- Claim: `claimStep` drops a pending claim when the word says the objective is open; a failed claim or claim trip sets a 5 minute backoff (`claimSkipUntil`) instead of retrying.
- Claim trip without a scroll: `town.ToTown` walks Den -> Blood Moor -> town (and Cold Plains -> Blood Moor -> town when the waypoint fails).
- First Akara talk (about 2.5 min live): `waitGreeting` waited one whole `greetingWaitS` in 20 s slices with Esc but never re-clicked until it ran out; now one 6 s slice, Esc, click again (U: that the speech is skipped by Esc and plays only once).
- Planner flip Blood Moor / Cold Plains: `bot.holdTarget` keeps a chosen area 3 min at the same level (10 min after 3 flips in 10 min); a level change, cleared, avoided or town releases it.
