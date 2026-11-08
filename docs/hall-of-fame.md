# Hall of Fame and the retire question (TASK-135)

> status: built offline (fixtures and fakes). Nothing here is verified against a live game. Ledger and UI only: no legacy charm item (TASK-136 waits for the overlay probe), no stat grant, no tally change, no Pact code.

Design: [research/fun-ideas.md](research/fun-ideas.md) section 5 (Legacy and the Hall), ADR 0018 item 7, decisions F7 and F10.

## What is built (V: `go test ./internal/watch ./internal/currency`)
- **Ledger kinds** `hall.retire` and `hall.death` (internal/currency/ledger.go). One entry per hero and kind, idempotent by key `hall:<kind>:<hero id>` (a replay returns `ErrDuplicate`, the Hall store treats it as "already booked"). Payload in `Entry.Data` (HallRecord: name, class, level, kills, kills of the monsters the title table counts, deaths, game and wall seconds, best drop, cause). They carry no currency and never move a balance. EventsMode now uses an explicit allow-list (`eventsAllowList`: tally credit/grant/spend plus the two Hall kinds with no currency); vanilla still refuses everything.
- **Titles are data**: `internal/watch/hall_titles.json` (12 rows): a rank, text, D2-voice flavour, optional class list and a rule (a status, or a monster and a kill count). The best earned rank is the title; all earned ones are listed. A row naming a class outside the table's class list, a rule-less row, an unknown status or a duplicate id fails the load naming the row; an unknown hero class fails Record/Title naming the class. No class check in Go.
- **Retire question** (`HallDesk`, internal/watch/hall.go): raised when a kill event names Andariel. Text: "Andariel is dead. Retire <hero> into the Hall of Fame, or keep playing?", default KEEP PLAYING. It is optional and has no timer: an absent player leaves it open, nothing ever retires without `POST /api/hall/answer {"answer":"retire"}` (away rule, tested with 500 simulated days). Retire books `hall.retire` then calls the bot's existing clean shutdown (`HallConfig.Stop`); a failed stop keeps the entry and a replay retries only the stop. Keep closes the question; the hero plays on.
- **Routes**: viewer GET `/watch/hall` (HTML) and `/api/hall` (JSON `watch.hall/1`), token gated by the viewer gate; answer POST `/api/hall/answer` (origin list, token, JSON only, small body). Return screen section "Hall of Fame" (question with two buttons while open, the account line, the title so far). Watch header: `HeroInfo.Title`, shown in the "Hero now" card.
- **Death memorial**: `hall.death` is booked on a death event when `HallFacts.Permanent` is true. Counters (kills, deaths, best drop) are fed from goals events and saved to `hall-state.json` next to the ledger (on Andariel, on death, every 25 kills; a crash can lose up to 24 kills, U).

## Wiring (cmd/d2bot/funwire.go `newHallConfig`, V: `go test ./cmd/d2bot -run Hall`)
- `newFun` passes `FunConfig.Hall` whenever the profile has an account ledger and a status source; state (`hall-state.json`) sits next to the ledger. The Hall rides the same goals event stream as the Codex and the unlock ladder (`feedHub` to `Fun.OnGoalEvent`); the retire question appears on the `EvKill` event naming Andariel, the same boss-kill source the Pacts feed to the ladder (`EvBossKill`), no second source.
- Facts come from the status poll: name, class, level; hero id is `<profile>:<name>` (U: no creation stamp, so a recreated name is the same hero); wall seconds are process uptime, game seconds stay 0 (U).
- `HallConfig.Stop` is `stopSelf`: SIGTERM to the own process, the same path `docker stop` and the updater take (docs/tester-live.md). Tests replace it.
- Tested with a synthesized Andariel kill through `startLadder` and `feedHub` (Sorceress and Paladin): the question appears, keep leaves the hero running and the Hall empty, retire books `hall.retire` and stops once (a replay does not stop again), the routes answer 401 without the viewer or answer token, and no tick auto-answers.

## Seams left
- `HallFacts.Permanent` is always false in the process: no goals event says a death is final and hardcore play does not exist (docs/pact-away.md: activation is refused). When a permanent-death signal exists, fill it in `newHallConfig` and nothing else changes; a recoverable death never enters the Hall (tested).
- The Pact death limit (internal/pact/botside.go `OnDeath`) only pauses the hero until the player takes it back, so it is NOT written as `hall.death` or `hall.retire`: both are booked once per hero and a later resume would leave a false memorial. If the owner wants a "retired by limit" line, it needs its own non-final kind (proposal, not built).

## Fake or missing
- **No permanence event**: `HallFacts.Permanent` is false in the process (see Seams left); unit tests use a Fake.
- **Retire is a desk question, not a decide catalogue point**: adding a point to internal/decide/catalogue.json touches the catalogue parity and risk tests (not owned). Proposal for the owner: point `t3.retire` with answers keep (default, noop) and retire, Irreversible. Until then the desk is the only asker and never auto-answers.
- Best item comes from drop events only (quality rank), not from the equipped gear (U). A started new hero is the wizard's job (not built). Account-wide codex and tier are shown as the account line and the title, not as a charm.

## Class generic (F10)
Everything class-specific lives in `hall_titles.json` (per-class retired and dead titles for sorceress and paladin, generic rows for the rest). Tests run both classes.
