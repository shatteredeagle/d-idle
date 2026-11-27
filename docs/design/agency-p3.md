# Agency P3: share and import build files

> status: current (TASK-985, 2026-11-27). Tags: V = ran in a test of `internal/buildfile`, U = unverified.

## Share
A build file (`didle.build/1`, `internal/buildfile/README.md`) is shared as its canonical JSON (`Marshal`/`SaveFile`) or the one-line chat form (`Encode`/`Decode`, for files without phases, respec, quest points or `after`). `ReadShared(raw, rules)` parses strictly and returns every validation problem at once; nothing expands or runs an unvalidated file. `ImportPlan` writes the plan table the leveler reads; combat line and hotkeys stay engine data.

## Import a kolbot list
`ImportKolbotBuild(name, class, skillsJSON, statsJSON, rules)`: AutoSkill/AutoStat lists with skill NAMES (ids are an error), class as full name or three-letter code, `satisfy` maps to `hold`, `"all"` to `rest`, `"block"` is an error. The build carries `source: kolbot-list` and its validation problems (wrong-class skill, count over cap, unreachable skill); the caller decides. The class is a parameter and the skill data comes from `Rules`, so no class is hard-coded.

## Round trip (V)
- `TestExportImportIdentity`: for every shipped plan table (`PlanNames`, 23 of them), export, marshal, parse is the identity on the file; export, import, export is a fixed point (skills, stat goals, quest points).
- `TestPlansRoundTripThroughFiles` (existing): same cumulative allocation at every row.
- `TestImportKolbotBuildTwoClasses`: Paladin and Sorceress lists validate and import to plan tables; wrong class, unknown class, malformed entry and numeric id are refused.
Under the permissive rules (U for real skills.txt: only the capability-table rules cover Paladin and Sorceress; see `docs/agency.md` section 5).

## Not done
NIP importer (item filters, a different format; needs its own card); `POST /api/hero/track` (leveler wiring); a CLI or Points-page upload.
