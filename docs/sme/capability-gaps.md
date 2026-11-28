# Capability table: what is filled and what is still open (card fix8)

> status: current (2026-11-27). Reader: the card that next touches `data/capabilities/capabilities.json` for the Amazon or the Assassin, and `TestBuildTablesConsistent` (internal/plandata), which checks the rows against the build tables.

Fable review 8 (docs/design/fable-review-2026-11-28/08-builds-as-data.md, C1) found 53 Amazon and Assassin rows without `reqLevel` and `reqSkills`, and seven skills with no row.

## Filled by card fix8

- 51 rows (26 Amazon, 25 Assassin) got `reqLevel` and `reqSkills` from the skill tables of `docs/sme/class-amazon.md` and `docs/sme/class-assassin.md` (columns clvl and prereqs; those files tag the columns T, read from the install's tables). Tag here: S, copied from a repo doc, not re-read from skills.txt by this card.
- Six passive rows were added in the shape of the other passives, because the shipped tables buy them: Critical Strike, Dodge, Avoid, Evade (Amazon), Claw Mastery, Weapon Block (Assassin). Same source and tag.
- The SME tables use display names, the capability table uses the skills.txt keys. Mapping used, each named in the repo: Fire Blast = Fire Trauma, Shock Web = Shock Field, Burst of Speed = Quickness (`docs/sme/class-assassin.md` section 2), Decoy = Dopplezon (`internal/plandata/unlocks-amazon.json`), Wake of Fire = Wake of Fire Sentry (same mana and damage row in the skill damage table).
- V: `go test ./internal/plandata -run TestBuildTablesConsistent` now checks the required level and the prerequisites of every Amazon and Assassin row of every build table and finds none broken; `go run ./tools/capgen -check` is clean.

## Still open (U)

| skill | class | what is missing | why |
|---|---|---|---|
| Inferno Sentry | assassin | `reqLevel`, `reqSkills` | no repo doc names the display name of this key (Wake of Inferno is a guess, U) |
| Royal Strike | assassin | `reqLevel`, `reqSkills` | same (Phoenix Strike is a guess, U) |
| Dragon Flight | assassin | the whole row | no build table buys it and no delivery is known; a row without a driver would be new in the table |
| Fire Golem | necromancer | row name | the review reads the key as `FireGolem` in skills.txt; not checked by this card, a rename touches profiles and tables |

The equality test against skills.txt on the worker host (review 8 section 3.1, last sentence) is not written: this card had no tables cache in its inputs.

## Per-level skill cap

`data/build-cap-ratchet.json` lists the rows where a table holds more points in a skill than `clvl - reqLevel + 1`. Whether Game.exe refuses such a point is U (`docs/points.md`). The list only goes down: a new row fails the test, a fixed row must be deleted from the list.
