# The Act 1 Codex and its firsts (TASK-130)

> status: built and tested offline (fixtures and a synthesized run); NOT wired into the bot loop and NOT verified live. Design: docs/research/fun-ideas.md section 4, ADR 0018 item 5.

A **first** is the hero's first meeting with something in Act 1. It fills one **Codex entry** (per profile, persistent) and, when its row has a question, raises ONE thought through the decision-point store with a default and a visible consequence. The table is data; no monster, area, skill or class name is Go.

## Files

| File | What |
|---|---|
| `internal/decide/firsts_act1.json` | the table: 37 rows, the `first` catalogue field (one enum value per row) and the `t0.first` point |
| `internal/decide/firsts.go` | `Firsts` (table), `Catalogue.WithFirsts`, `Codex` (file `codex-<profile>.json` beside the decide profile), `Tracker.Observe/Settle` |
| `internal/watch/codex.go` | `CodexHandler` (`GET /api/codex[?window=<minutes>]`), `CodexReader`, `CodexSection` (the return screen line) |

## The `first` context flag

`Catalogue.WithFirsts(tbl)` returns a COPY of the catalogue with the field `first` and the point `t0.first` added (context key: `first`, i.e. the row id; answers `yes` / `no`; risk normal). The embedded `catalogue.json` is untouched, so the 23-point parity tables stay valid. The runner that wants firsts builds its store from the copy. Because the key holds the row id, every first has its own key: the store dedupes it, a rule is never learned from it (a key is answered once), and the answer lives in the Codex entry (`Codex.Decided(id)`), which the consumer reads.

Row fields: `kind` (monster, area, waypoint, shrine, drop, champion, death), `match` (names compared case-insensitively with `FirstEvent.Name`, or a champion mod; `*` matches any), `title`, `question` + `yes` / `no` labels + `default` (+ `defaultByClass`, validated against the catalogue's class table), `teaches` (the rule), `consequence` (what the player will see), `tip` + `tipTag`. A row without `question` is quiet: the Codex fills, no thought opens. An event claims the FIRST matching row in table order (so the specific champion row sits before the general one) and fires it once.

## Events and readers

`Tracker.Observe(FirstEvent{Kind, Name, Area, Mods, Clvl})` is the one entry point. State of each reader, honestly (V = the reader exists in the tree today; nothing below is wired to `Observe` yet, the bot is outside this task's paths):

| Kind | Reader that exists | Missing / U |
|---|---|---|
| monster | monster class to name through `data.DB`; `MonsterInfo.IsUnique/IsChampion` (internal/d2); `debug nearby` on d2d | the "first sight" edge (nearest hostile newly in range) is not a bot event; superunique names (Corpsefire, Bishibosh, Rakanishu, Blood Raven, Griswold, Countess, Andariel) need the class-to-name map from `MonStats`/`SuperUniques` (U that the name matches the table) |
| area | `Runner.areaName(level id)` (internal/bot) | the edge "first time in area" |
| waypoint | `questmem.Source.Waypoint(diff, area)` and the d2d `dump quests` masks (V, memory 2.10) | no "waypoint lit" event is emitted (U): needs a poll that diffs the mask |
| shrine | none | NO reader: the bot sees only a gold change near a shrine (areametrics, S). Write the trigger against `FirstEvent{Kind:"shrine"}` when an object reader exists |
| drop | pickit item kinds (rune, unique, set, rare) | the edge "first of this kind picked up" |
| champion | `MonsterInfo.IsChampion`; enchant ids in the death log | mod id to mod name table; the table lists names (U) |
| death | death log `Killer` (best effort, U in internal/bot/deathlog.go) | none beyond that |

Hook sites to wire (for the owner of internal/bot, with `cat.WithFirsts`, `decide.OpenCodex(dir, profile)`, `NewTracker` and a `Settle()` after each `Tick`): the area-change handler, the nearby-monster scan (new class id), the waypoint-mask poll, the pickup path, the death handler.

## Questions queue for the return (D2)

A first's thought opens like any other: it waits for the player until the fade (`config.fadeSec`, 45) and the default applies afterwards; the Codex entry shows `source: default` and `open: true` while the thought is still open, then `answered` or `default` after `Tracker.Settle`. The fade logic is unchanged here (TASK-115 deferred).

## /api/codex

`watch.codex/1`: `total` (rows), `filled`, `asked`, `byKind`, `recent` (newest first, every entry or those within `window` minutes), `missing` (titles not met), `line` (the return screen text, e.g. "3 new firsts: Fallen Shaman, Corpsefire, Cold Plains waypoint (3 of 37 Act 1 entries)"). Mount `CodexRoute` with `CodexHandler(CodexReader{...})`. TASK-127's `ReturnSection` registry has not landed on main: the return screen calls `watch.CodexSection(summary)` until it does. `firstSec` is virtual seconds since the run started.

## Honesty tags

- V: Den of Evil gives a skill point (ADR 0004); the density and kill criteria are measured by `TestFirsts_ReplayDensity` on a SYNTHESIZED run (13 events, 12 firsts, 5 in the first 10 minutes, 12 in the first 30), which is not a recorded soak.
- S: most tips and the zone order come from the SME's table (fun-ideas.md); tagged `S` in the rows.
- U: tips tagged `U` (Quill Rats, archers, Griswold, Forgotten Tower level, notable champion mods, shrine, drops, death), the `defaultByClass` for a paladin at Corpsefire (stand, a melee build cannot kite), the champion mod names, and every default flagged as a design choice. The SME's level numbers (Blood Raven 6 vs 8) are printed as he gave them, UNVERIFIED against the tables.
- NOT verified live: no game, no bot wiring, no real soak. A live check needs the hooks above and one run of a fresh hero; the first 10 minutes at 1x should show at least 3 firsts (kill criterion of TASK-130).

Build flexibility: class data are `defaultByClass` and the catalogue's class table (an unknown class is an error naming it); tests run the default Sorceress and a Paladin.
