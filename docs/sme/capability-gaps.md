# Capability table: what is filled and what is still open (card fix8)

> status: current (2026-11-27). Reader: the card that next touches `data/capabilities/capabilities.json` for the Amazon or the Assassin, and `TestBuildTablesConsistent` (internal/plandata), which checks the rows against the build tables.

Fable review 8 (docs/design/fable-review-2026-11-28/08-builds-as-data.md, C1) found 53 Amazon and Assassin rows without `reqLevel` and `reqSkills`, and seven skills with no row.

## Filled by card fix8

- 51 rows (26 Amazon, 25 Assassin) got `reqLevel` and `reqSkills` from the skill tables of `docs/sme/class-amazon.md` and `docs/sme/class-assassin.md` (columns clvl and prereqs; those files tag the columns T, read from the install's tables). Tag here: S, copied from a repo doc, not re-read from skills.txt by this card.
- Six passive rows were added in the shape of the other passives, because the shipped tables buy them: Critical Strike, Dodge, Avoid, Evade (Amazon), Claw Mastery, Weapon Block (Assassin). Same source and tag.
- The SME tables use display names, the capability table uses the skills.txt keys. Mapping used, each named in the repo: Fire Blast = Fire Trauma, Shock Web = Shock Field, Burst of Speed = Quickness (`docs/sme/class-assassin.md` section 2), Decoy = Dopplezon (`internal/plandata/unlocks-amazon.json`), Wake of Fire = Wake of Fire Sentry (same mana and damage row in the skill damage table).
- V: `go test ./internal/plandata -run TestBuildTablesConsistent` now checks the required level and the prerequisites of every Amazon and Assassin row of every build table and finds none broken; `go run ./tools/capgen -check` is clean.

## Closed by card cls2 (2026-11-28)

Source: `internal/power/skilldamage.json`, the skills.txt extract the repo carries (131 damage skills with `id`, `req_level`, mana columns, element, hit shift). V = read from that file; S = matched to a display name through `docs/sme/class-assassin.md` section 2.

| skill | class | filled | tag |
|---|---|---|---|
| Inferno Sentry | assassin | `reqLevel` 24 (extract id 272); `reqSkills` Wake of Fire Sentry | level V; the display name Wake of Inferno and its prerequisite are S (the only clvl 24 fire trap with 20 mana and hit shift 4 in both tables) |
| Royal Strike | assassin | `reqLevel` 30 (extract id 280); `reqSkills` Cobra Strike, Blades of Ice | level V; the display name Phoenix Strike and its prerequisites are S (the only clvl 30 martial art with 4 mana) |
| Fire Golem | necromancer | `reqLevel` 30, `reqSkills` IronGolem, same as the `FireGolem` row | the key `FireGolem` is V (the extract carries it, id 94); `Fire Golem` stays as an alias row because the golem ladders in `profiles.json` list both spellings, so no rename |

`go test ./internal/plandata -run TestCapabilityReqLevelMatchesSkillExtract` compares the `reqLevel` of every capability row the extract knows (all classes) and fails on an Amazon or Assassin row without a level. Passives and buffs (17 Amazon and Assassin rows) are not in the extract and stay S from the SME tables.

## Still open (U)

| skill | class | what is missing | why |
|---|---|---|---|
| Dragon Flight | assassin | the whole row | S values exist (`docs/sme/class-assassin.md` section 2: clvl 24, prerequisite Dragon Tail, 15 mana) but the extract has no row for it, no build table buys it and no delivery is known (a teleport kick: target unit, lands next to it); a row without a driver would be new in the table. `internal/combat/rules.go` names it as a fire fallback, which is wrong (it is a physical kick, S) and belongs to the audit's skill-literal rows |

The equality test against the full skills.txt on the worker host (review 8 section 3.1) is still not written: `reqSkills` and the 17 passive and buff rows have no in-repo extract to compare with.

## Per-level skill cap

`data/build-cap-ratchet.json` lists the rows where a table holds more points in a skill than `clvl - reqLevel + 1`. Whether Game.exe refuses such a point is U (`docs/points.md`). The list only goes down: a new row fails the test, a fixed row must be deleted from the list.
