# Unlock ladder (TASK-153, unlockladder)

Shared framework behind the watch screen's unlocks: the senses tiers (`docs/research/info-unlocks.md`, TASK-149) and the ten unlockables (`docs/research/unlockables.md`). Design is the two notes; this file is what is built. Tags: V verified by the offline tests named, U unverified (needs a live run).

## Pieces

- `data/unlocks/ladder.json` (embedded by `data/unlocks/unlocks.go`): the table. An entry has `id`, `group` (`senses`, `ui` or `pacts`), `tier` (senses only, unique per group), `trigger {kind,args}`, optional `alt` triggers (any of them opens it), `reveal {title,text,teaser}` and `price` (tally, only with a `tally_spend` trigger). `unlock.Parse` rejects: an unknown trigger kind (error names the kind and the entry), a missing arg, a reveal without title or text (nothing unlocks silently), a price without `tally_spend` or the reverse, duplicate ids or senses tiers, an unknown group.
- `internal/unlock`: `Engine` (`Open(dir, profile, ladder, clock)`), file `unlocks-<profile>.json` beside the Codex and the ledger (atomic write, never relocks, an id the ladder dropped stays on file). `Observe(Event)` opens entries once and returns them; `Allowed(id)`, `SenseTier(n)`, `Met(sub, name)`, `States(group)` (locked entries carry the teaser and price, never the text), `Pending()`, `Buy(id, Spender)`.
- `internal/watch/unlock.go`: `Unlocks` (reveal, feed, routes, return section), `LedgerSpender`, `TallyStrip.UnlockSpender()`. `FunConfig.Unlock` (an `*unlock.Engine`) turns it on; nil means no ladder, no routes.

## Trigger kinds

| kind | args | event that opens it |
|---|---|---|
| `codex_first` | `sub` (Codex kind), `ids` (Codex ids; empty = any) | a new Codex entry (`decide.Tracker.Observe` result, so a replay or reload fires nothing) |
| `quest` | `quest` | a goals quest event with `Done` |
| `level` | `min` | a level-up event with `Level >= min` |
| `kills_of_kind` | `kind` (or `*`), `min` | `Event{Type: kills, Name, N}`: the source reports the running count (U: no emitter feeds it yet) |
| `retire_count` | `min` | `Event{Type: retire, N}`: running count (U: no emitter yet) |
| `boss_kill` | `boss` (monster name), `difficulty` (`normal`/`nightmare`/`hell`; absent = any) | `Event{Type: boss_kill, Name, Sub: difficulty}`: a kill event, idempotent (opens once). Fed by `startPacts` (cmd/d2bot/pactwire.go) |
| `quest_stage` | `quest` (id), `min` (stage 1 started, 2 cleared, 3 done) | a goals quest event with `Stage >= min` (the bot's quest reader, `pollQuests`) |
| `item_quality` | `qualities` | an identified-item event of that quality (idreveal) |
| `seen` | `what` | a viewer-side moment (`Unlocks.Seen`): `expedition_card`, `near_death_survived` |
| `tally_spend` | none; the entry's `price` | `POST /api/unlocks/buy` |

Counts carry the running total, so a repeated event can never count twice. Replay safety for the rest comes from "open once".

## Rows in the shipped table

| row | group | opens when | doc |
|---|---|---|---|
| `chronicle`, `chronicle_tape` | ui | first kill; the first expedition card shown | `docs/unlock-chronicle.md` |
| `voice` | ui | level 2, or the first death | `docs/unlock-voice.md` |
| `questlog` | ui | Den of Evil started (`quest_stage` 1) | `docs/unlock-questlog.md` |
| `idreveal`, `idreveal_later` | ui | first magic or rare item; first unique or set | `docs/unlock-idreveal.md` |
| `bestiary` | ui | any kind reaches 10 kills | `docs/unlock-bestiary.md` |
| `bestiary_resists` | ui | any kind reaches 50 kills (resist details per kind) | `docs/unlock-bestiary.md` |
| `atlas` | ui | the first waypoint taken (Codex `waypoint`) | `docs/unlock-atlas.md` |
| `risk` | ui | `near_death_survived`: life under 25 percent, then back to 50 percent | `docs/unlock-risk.md` |
| `orderqueue` | ui | Den of Evil cleared (`quest_stage` 2) | `docs/unlock-orderqueue.md` |
| `orderqueue2` | ui | Blood Raven's reward quest done (`sisters_burial` 3), or Tools of the Trade done, or 30 tally | `docs/unlock-orderqueue.md` |
| `orderqueue3` | ui | Deckard Cain rescued (`search_for_cain` 3), or 75 tally | `docs/unlock-orderqueue.md` |
| `senses_map` .. `senses_hero` | senses, tiers 1 to 7 | map: first area or level 2; dots, bars: first monster; elites: first champion or a named boss; loot (40 tally) and danger (80 tally) are bought; hero: level 12 or Andariel | `docs/senses-unlock.md` |
| `pact_andariel_first_kill`, `pact_baal_normal` | pacts | the first Andariel kill; Baal killed on Normal | `docs/pact-ui.md` |
| `alerts` | ui | bought with 60 tally (`tally_spend`) | `docs/unlock-alerts.md` |

## Wiring in the live bot (TASK-185, `cmd/d2bot/funwire.go`)

| feature | what the bot passes | stays missing |
|---|---|---|
| bestiary | `FunConfig.Monsters` = the `data.DB` the backend opened (nil without game tables: flavour-only cards, one log line); `BestiaryPath` = `bestiary-<profile>.json` in the state dir, so kill counts survive a restart; a shipped kind the table lacks is skipped and named in `Fun.Errors` and the log, not fatal | the shipped kind names are not checked against the real extracted table (U live) |
| risk meter | `FunConfig.Risk` with the proposer table, the tally zone ranges and `bot.Runner.RiskFacts` (kit via `ladder.KitFor`, potions, life percent, dead, engage lines of the resource tuning, the route guard reason for the zone); "no zone" means the first pending proposal, else the area the hero stands in. The `near_death_survived` moment comes from `RiskMeter.Tick`, run by `Fun.Settle` on the 2 s status poll, once per episode | `ResistGap` is always 0 (no zone resist table); clvl and class come from the hub's hero (the status poll) |
| order queue | `FunConfig.OrderQueue` with `TallyStrip.Quote` (charter check) and `Halt` = the Pact controller's `Paused` (Pact and low-life pause); the sequencer places through `Hub.place`; a death event stops it (`EvDeath`); slot rows open from the bot's quest reader events: Den cleared (`den_of_evil` 2), Kashya's reward (`sisters_burial` 3), Charsi (`tools_of_the_trade` 3), Cain (`search_for_cain` 3) | the live quest reader has not been run against a game (U); the queue is in memory only |
| atlas | nothing: it reads the Codex and the first pending proposal | needs the Codex point (`-watch` with firsts) for lit nodes |

V offline: `go test ./cmd/d2bot -run 'RiskMeterWired|OrderQueueWired|BestiaryWired|BestiaryWithout|PactHalt'` (Sorceress and Paladin, through `startLadder` and `feedHub` with fake facts, a fake monster table and goals events).

## Pacts group

Entries `pact_<unlock_trigger>` open the Pacts of the catalogue (`data/pacts/pacts.json`) with that trigger; `pact.Controller` reads `Allowed`. `Engine.Adopt(id)` opens an entry from an older record without a reveal (the Pact state migration). See `docs/pact-ui.md`.

## Reveal

Every entry that opens (by trigger or by buying) writes one Chronicle line (`Hub.Note("story", "Unlocked: Title. Text")`) and one reveal record (`AwayUnlock`, kept in the `Unlocks`' own log so a late reveal in camp cannot open a phantom expedition on the return screen). The return screen section `unlocks`, "Unlocked since you left", lists the away events newer than the viewer's last look and is hidden when there are none. Opening raises no question.

## API

- `GET /api/unlocks` (viewer, same gate as the other viewer routes): `{schema: watch.unlocks/1, open[], locked[], balance?}`. A locked entry has `title`, `teaser`, `price`; its `text` is absent.
- `POST /api/unlocks/buy {id}` (answer listener): same origin list, token, JSON-only and small-body rules as the other answer routes. 200 with the entry; 402 `insufficient` when the tally is short (nothing booked); 404 unknown entry; 400 `not-for-sale`; 409 `already-open`; 503 `tally-off` without a ledger or in vanilla mode. The spend is a ledger `spend` entry (key `unlock/<profile>/<id>`, upgrade `unlock`), so the ledger's own check keeps the balance from going negative and a retry after a crash books once.

## Rules kept

Basics are never entries (nothing in the table may gate a view the first session needs); the ladder is view-only and the bot never reads it; trigger values are data, not Go; monster and area firsts are world data, so the same table serves every class (tested with Sorceress and Paladin).

## Not done here

- (Done in TASK-154) `cmd/d2bot` newFun opens the shipped ladder into `cfg.Unlock`; `Unlocks.OnGoalEvent` feeds `kills` (per monster name, in memory). New trigger kind `seen` (arg `what`) for viewer-side moments.
- No emitter feeds `retire` events. (pactladder) `startPacts` feeds `boss_kill` events. (TASK-185) the bestiary, risk meter and order queue are wired into `cmd/d2bot` (section above).
- The shipped table holds the Chronicle rows (`docs/unlock-chronicle.md`), the seven senses rows (TASK-149, `docs/senses-unlock.md`) and the Pacts rows (TASK-181, `docs/pact-ui.md`); the other unlockable cards add theirs.

## Verified

V offline: `go test ./internal/unlock ./internal/watch` (table-driven trigger kinds, replay, reload, profiles apart, buy short/accepted/again/after crash, ledger entry, token/origin/method gates, return section, two classes). U live: nothing run against a game.
