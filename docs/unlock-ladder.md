# Unlock ladder (TASK-153, unlockladder)

Shared framework behind the watch screen's unlocks: the senses tiers (`docs/research/info-unlocks.md`, TASK-149) and the ten unlockables (`docs/research/unlockables.md`). Design is the two notes; this file is what is built. Tags: V verified by the offline tests named, U unverified (needs a live run).

## Pieces

- `data/unlocks/ladder.json` (embedded by `data/unlocks/unlocks.go`): the table. An entry has `id`, `group` (`senses` or `ui`), `tier` (senses only, unique per group), `trigger {kind,args}`, optional `alt` triggers (any of them opens it), `reveal {title,text,teaser}` and `price` (tally, only with a `tally_spend` trigger). `unlock.Parse` rejects: an unknown trigger kind (error names the kind and the entry), a missing arg, a reveal without title or text (nothing unlocks silently), a price without `tally_spend` or the reverse, duplicate ids or senses tiers, an unknown group.
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
| `tally_spend` | none; the entry's `price` | `POST /api/unlocks/buy` |

Counts carry the running total, so a repeated event can never count twice. Replay safety for the rest comes from "open once".

## Reveal

Every entry that opens (by trigger or by buying) writes one Chronicle line (`Hub.Note("story", "Unlocked: Title. Text")`) and one away event (`AwayUnlock`). The return screen section `unlocks`, "Unlocked since you left", lists the away events newer than the viewer's last look and is hidden when there are none. Opening raises no question.

## API

- `GET /api/unlocks` (viewer, same gate as the other viewer routes): `{schema: watch.unlocks/1, open[], locked[], balance?}`. A locked entry has `title`, `teaser`, `price`; its `text` is absent.
- `POST /api/unlocks/buy {id}` (answer listener): same origin list, token, JSON-only and small-body rules as the other answer routes. 200 with the entry; 402 `insufficient` when the tally is short (nothing booked); 404 unknown entry; 400 `not-for-sale`; 409 `already-open`; 503 `tally-off` without a ledger or in vanilla mode. The spend is a ledger `spend` entry (key `unlock/<profile>/<id>`, upgrade `unlock`), so the ledger's own check keeps the balance from going negative and a retry after a crash books once.

## Rules kept

Basics are never entries (nothing in the table may gate a view the first session needs); the ladder is view-only and the bot never reads it; trigger values are data, not Go; monster and area firsts are world data, so the same table serves every class (tested with Sorceress and Paladin).

## Not done here

- (Done in TASK-154) `cmd/d2bot` newFun opens the shipped ladder into `cfg.Unlock`; `Unlocks.OnGoalEvent` feeds `kills` (per monster name, in memory). New trigger kind `seen` (arg `what`) for viewer-side moments.
- No emitter feeds `retire` events.
- The shipped table holds the Chronicle rows (`docs/unlock-chronicle.md`) and the seven senses rows (TASK-149, `docs/senses-unlock.md`); the other unlockable cards add theirs.

## Verified

V offline: `go test ./internal/unlock ./internal/watch` (table-driven trigger kinds, replay, reload, profiles apart, buy short/accepted/again/after crash, ledger entry, token/origin/method gates, return section, two classes). U live: nothing run against a game.
