# Cain's notes, the bestiary (TASK-158, unlockbestiary)

Sits on the unlock ladder (`docs/unlock-ladder.md`); copied from `docs/unlock-chronicle.md`. Tags: V verified by the offline tests named, U unverified.

## What it is

A card per monster kind that fills in as the hero kills it. Panel "Cain's notes" on the watch page. It is not tied to Cain's quest (the rescue comes late for a Sorceress); "Cain's notes" is only the name.

| ladder row | trigger | opens |
|---|---|---|
| `bestiary` | `kills_of_kind` `*` min 10 (any kind reaches 10 kills) | the panel |
| `bestiary_resists` | `kills_of_kind` `*` min 50 | resist details, for kinds that have 50 kills |

A card has four states by that kind's own kill count: `silhouette` (not met: only a family such as demon, beast or undead), `met` (1 to 9: the name and the count), `card` (10: type, attack pattern, drops, flavour and a veteran tip with its tag), `full` (50: resists on Normal, Nightmare and Hell, 100 or more is immune).

Spoiler rule: no resist or immunity field is sent before the 50th kill of that kind and the row `bestiary_resists` being open (V `TestBestiarySpoilerRule` checks the raw JSON).

## Pieces

- `internal/watch/unlockbestiary.go`: `Bestiary`, `GET /api/bestiary`, schema `watch.bestiary/1`, the return screen section "New in Cain's notes" (cards and resist pages opened since the viewer left).
- `data/bestiary/notes.json`: the kinds (name, family, flavour, drops, tip, tipTag S or U). Type, attack pattern and resists come from the extracted tables (`data.Monster`), through the `MonsterTables` interface (`FunConfig.Monsters`). With no tables the cards carry flavour only (`tables:false`). With tables, a kind missing from the monster table is an error naming it.
- `internal/watch/web/unlockui.js`, `watch.html`: the panel.
- Kills are counted per kind in memory from the goals kill events (a replayed `Seq` never counts twice); a restart counts again, the ladder rows stay open (U).

## Open items

- U: the kind names in `notes.json` match monstats display names (a real-table check is a follow-up; the tests use a fixture table). Tips are secondary (S: `docs/research/class-early-game.md`, the SME) or U.
- `cmd/d2bot` does not set `FunConfig.Monsters` yet (needs the `data.DB` plumbed into `newFun` and the names checked against the real table first), so the live bot shows flavour-only cards.
- Drops are free text, not the treasure class tree (U).

## Verified

V offline: `go test ./internal/watch -run Bestiary` (golden payloads for Sorceress and Paladin at 10 and 50 kills, locked view, spoiler rule, replay, return section, errors naming the row or kind, no-table cards).
