# Per-class chore defaults (TASK-146, card classchores)

> status: built and tested offline; not run against a live bot. Tags: V = proven by a test in this tree, EXP = veteran experience (docs/research/class-early-game.md section 4, not measured), U = unverified.

ADR 0018 Amendment 2026-11-07: chore defaults are per-class data. They live in `internal/plandata/chores-defaults.json` (copy in `docs/plan/data/`, kept equal by `TestMatchesDocs`), loader `plandata.Chores(class)`. Unknown class, unknown field, bad cadence or an empty potion mix is an error naming it (V `TestChoresErrorsNameTheCause`, `TestChoresTwoClasses`).

## Rows (all seven classes, EXP)

| class | potions health:mana | repair | ammo | summons | buffs | class trip trigger |
|---|---|---|---|---|---|---|
| amazon (bow) | 1:1 | rare | arrows, quiver 250-350, restock under 20% | none | none | health under 3, ammo under 20% |
| assassin | 1:1 | rare | none | traps | none | mana under 3 (S, section 4 does not list the assassin by name) |
| barbarian | 1:0 | every trip, under 25% | none | none | Battle Orders, Shout | health under 3, durability under 25% |
| druid | 1:1 | rare | none | Raven, Creeper, Wolves | Oak Sage, Cyclone Armor | mana under 3 |
| necromancer | 1:1 | rare | none | Skeleton, Golem | none | mana under 3 |
| paladin | 4:1 | every trip, under 25% | none | none | Might, Holy Fire, Prayer, Resist Fire | health under 3, durability under 25% |
| sorceress | 1:1 | rare | none | none | none | mana under 3 |

All classes also go home on a belt under half, a full pack and a town portal tome under 1 (`all_classes.town_trip`). Only the Amazon bow has an ammo trigger (V `TestChoresTwoClasses`).

## How the policy sheet reads it

Each class row has `start`: policy row id to option id. On the first read of the sheet for the hero's class (`PolicySheet.rows`, so View, Set and the section) the rows the player never set are put on those options (`applyStart` in `internal/watch/policysheet.go`): knobs through the tuner and the decide log line, but NOT a recorded change (no card on the return screen). A row the player already set is left alone; a start equal to the row default does nothing; a row or option id the sheet does not have is an error naming it (V `internal/watch/policystart_test.go`: Sorceress and Paladin, knob call once, player choice kept, errors).

## Decisions and gaps

- Every shipped `start` is empty, so today's behaviour is unchanged for every class (regression of old defaults). The veteran differences (Paladin and Barbarian repair every trip, Amazon ammo) are in the data rows but no sheet option acts on them yet; flipping a class start is a one-line data change once an option exists (U which options the user wants as class starts: open choice).
- The numeric trip triggers, ammo, summon and buff rows are data for future consumers; no town or combat code reads them yet (U).
- The decide catalogue knows only sorceress and paladin (`internal/decide/catalogue.json` classes), so the sheet still refuses the other five classes with "unknown class" (existing behaviour, not changed here); their chore rows are loadable now.
