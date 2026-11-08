# wizardclass: the first-run wizard offers classes from the plan tables (TASK-145)

Decision F11 (docs/decisions.md; ADR 0018 class-generic amendment). Code: `cmd/d2dsetup/wizardclass.go`, `tables.go`.

## Rule
- The class list is data. `tables.json` is only the catalog (name and d2d UI index per class, V `docs/d2d-fidelity.md`); the wizard offers a catalog class only if `internal/plandata` has a leveling table for it. The `supported` key is no longer read from the file.
- A class qualifies with a **normal30** table: a `build-*.json` whose `build` contains `normal30`, class from its `class` key.
- **Interim rule until TASK-144 (sorcnormal30) lands**: a class whose only table is a `build-fallback*.json` is offered too, marked `table: fallback` with a note; the page shows "(fallback table)". Today that is the Sorceress only (the bot plays her on the leveler's default table `fallback`, Lightning; a table without `class` key is the Sorceress's, as in `leveler.ParseBuild`). When a Sorceress normal30 file is added the class switches to `normal30` and the note disappears, with no code change.
- A catalog class with no table at all is dropped from the list. Choosing any class not in the list, in `PUT /api/inputs` or in a saved `wizard.json` that is not done yet, is an error naming it (`unknown class "x"`).
- `default_class` in `tables.json` is `paladin` (preselected). It must be in the list or loading fails.

## Tables by class today (S: read from `internal/plandata`)
| Class | Table kind | Files |
|---|---|---|
| Paladin | normal30 | `paladin-normal30`, `-v2`, `zealot-lite` |
| Necromancer | normal30 | `necromancer_leveling_normal30`, `necromancer_summoner_normal30` |
| Amazon | normal30 | `amazon_bow_normal30`, `amazon_javelin_normal30` |
| Assassin, Barbarian, Druid | normal30 | one file each |
| Sorceress | fallback | `build-fallback-lightning*.json` |

## Class to container
`finishLocked` writes `D2D_CLASS='<d2d_index>'` into `wizard.env`; the entrypoint sources it and passes `$D2D_CLASS` as `--start-class` / `--d2d-class`. Paladin is 2 and the Sorceress 4 (d2d UI order). This is what `docs/paladinsmoke.md` seeded by hand.

## Built vs verified
- Built and tested offline (`go test -p 1 ./cmd/d2dsetup`): list follows plan data (a fake plan adds and removes classes), real plan classes all in the catalog, Paladin preselected, Sorceress fallback note, unknown class error naming it (API and saved file), golden of the class payload (`testdata/classes.golden.json`), `wizard.env` sourced by a real `sh` gives `D2D_CLASS` 2 (Paladin) and 4 (Sorceress).
- Page: `node --check page/app.js` passes (S: syntax only). Rendering in a browser was NOT run.
- Not verified live: no container run, no game. The first real pick of a non-Paladin class in a container is still to do.
- Other classes being listed means their normal30 table exists, not that the bot plays them well; only the Paladin has a smoke run.
