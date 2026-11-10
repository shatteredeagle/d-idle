# Sorceress normal30 table (TASK-144, card sorcnormal30)

> status: since TASK-267 (sorcready2) the table is no longer v3-identical: Warmth at clvl 2, a cold line from clvl 12 (profile `sorceress/cold`), and it is the default of a fresh Sorceress. Current rows and reasons: [sorceress-early.md](sorceress-early.md) section 7. The text below describes the TASK-144 state.

Status: BUILT, offline-tested. NOT verified live.

## What it is
`internal/plandata/build-sorceress_normal30.json` (copy in `docs/plan/data/`, kept equal by `TestMatchesDocs`): clvl 2..30, `class` sorceress, `combat` lightning, `selector` `sorceress_normal30` (registered by `leveler.registerDataTables`, no Go change). Every class now has a normal30 table, so the first-run wizard (`cmd/d2dsetup`, TASK-145) lists the Sorceress as `normal30` without a fallback note.

## Rows: identical to v3 (V, `TestSorceressNormal30EqualsV3`)
The rows for clvl 2..30 are a verbatim copy of `build-fallback-lightning-v3.json` (the approved BUILDV3 table, `docs/buildv3.md`): Charged Bolt 2-5, Frozen Armor at 5 (Den point), Frost Nova 6, Charged Bolt 7-10, Telekinesis 11, Lightning 12, Charged Bolt 13-17, Teleport 18, Chain Lightning 19-30; stats 5 per level into Vitality with Strength at 25, 26, 30. No difference from v3, so no SME reasons to document. Skill ReqLevels are V from the real tables (`docs/research/skill-req-check.md`); the test against the install tables runs when `/tmp/d2mpq` exists and is skipped otherwise (not run in this branch). `TestNoBuildRowIsRefusedByAnUnlock` (now also for this table) shows the unlock schedule (`unlocks-sorceress.json`) refuses no row at the expected event levels.

## Selection (nothing changes by default)
- `SelectBuild`/`DefaultBuildName` still give a fresh Sorceress `fallback-lightning-v2` (test `TestDefaultSorceressTableUnchanged`); gates keep their pinned tables.
- Opt in with `levelerBuild sorceress_normal30` (`planForClass` then takes combat `lightning` from the table's `combat` key). A tester profile that wants it names that selector; no tester profile code exists in this branch.

## Open choice (U, needs the user)
`docs/research/class-early-game.md` section 1 (EXP, UNSURE) sketches a Fire Bolt / Ice Bolt / Fire Ball line. The brief and the approved v3 decisions (Den-first, Lightning line) win: building a fire or cold normal30 variant is a separate decision and a separate file. Acceptance criterion 1 is met by the data table; its fire-path wording is not followed.

## Verified vs not
Offline only: loader, equality with v3, unlock refusal, wizard list and golden. Not verified live: no soak with this selector.
