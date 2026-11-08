# Skill unlocks by quest or find (TASK-133)

> status: engine and Sorceress table built and tested offline (fixtures and the real tables on the worker host); nothing here has run in a live game. Not wired into the runner yet (see "Not built"). Design: ADR 0018 item 6 and F6, `docs/research/fun-ideas.md` section 12; levels: `docs/research/skill-req-check.md` (V, from the tables).

## What it is

A skill listed in `internal/plandata/unlocks-<class>.json` cannot take a point until its event has fired; a skill not listed is free (its Skills.txt level requirement still applies). Skills unlock by quest or find, never by currency.

- Schema: `{"class", "doc", "mustFree": [skills], "unlocks": [{"skill", "event", "codex", "tag", "note"}]}`. `event` is `kind:id` with kind `quest`, `waypoint`, `boss`, `item` or `clvl`; alternatives are joined with `|` (any one opens it). `codex` is the id of the Codex first (`internal/decide/firsts_act1.json`) that lights the icon. `tag` is S (veteran schedule, placement not measured), U (unsure) or V.
- Loader and rule: `internal/leveler/unlocks.go`. `LoadUnlocks(class, dir)` (unknown class is an error naming it), `Validate(db)` (every skill must exist in the tables and belong to the class; the error names the skill), `Unlocks.WrapRule` (the `unlocked(name)` refusal on top of the skills.txt rule), `Unlocks.Plan`.
- Points are NOT banked. A refused row flows to the next legal row, the same as a prerequisite refusal, so an unlock reorders the build, never saves a point. `TestLateTeleportUnlockReordersWithoutBanking` proves it.
- Progress: `Progress` holds the fired events and the free-choice point per hero. `Progress.Fire("boss:bishibosh")` is the one call the runner makes from its quest, waypoint and boss events. Measured runs use `AllMetProgress()` (every event met, ADR 0016 item 18) or leave `Options.Unlocks` nil; both leave the plan identical to the plain build (`TestBuildV3WithUnlocks`).
- Wiring: `Options.Unlocks` and `Options.Progress` on the Leveler; `PlanFor` uses them when set.

## The free-choice point (ADR 0018 F6)

The player may spend ONE point on a Codex-unlocked skill that the build table skips. `Unlocks.FreeChoices` lists the candidates (listed in the table, unlocked, legal by Skills.txt, not in the build table) each with the skill it displaces: the text from `FreeChoice.Consequence()`, for example "Static Field takes the point the build gives Charged Bolt; the build moves one point later". `Unlocks.ChooseFree` records the choice once; `Unlocks.Plan` then puts the next unspent point on it first.

## The Sorceress table

Events are the veteran schedule (`docs/research/class-early-game.md` section 2) with these adjustments:

| Skill | Event | Note |
|---|---|---|
| Frost Nova | quest:den_of_evil | Skills.txt level 6 (V). A skipped Den moves the point to Charged Bolt |
| Static Field | boss:bishibosh or waypoint:cold_plains | level 6 (V); not in v3, so free-choice only |
| Telekinesis, Warmth | boss:blood_raven | Telekinesis level 6, Warmth level 1 (V); v3 spends Telekinesis at 11 |
| Ice Blast, Fire Wall, Blaze | boss:countess | levels 6, 18, 12 (V) |
| Inferno, Fire Ball | quest:rescue_cain | levels 6 and 12 (V); the quest id is U |
| Teleport | waypoint:catacombs_2 | NOT Andariel: level 18 (V) and v3 spends it at 18, Catacombs run at clvl 14 to 17 (S) |
| Glacial Spike, Nova | boss:andariel | level 18 and 12 (V) |

Never locked (mustFree): Fire Bolt, Charged Bolt, Ice Bolt, Frozen Armor, Lightning, Chain Lightning, Lightning Mastery. Nothing the bot needs to survive locks later than Bishibosh: the only locks before Bishibosh are Frost Nova (not survival-critical) and, by Bishibosh, Static Field.

Checked against v3 (buildv3 replay, `TestBuildV3WithUnlocks`): with every event met the spent order equals the plain replay; with the events at their expected clvl (Den 5, Bishibosh 8, Blood Raven 9, Catacombs waypoint 17) no row is refused. Those clvl are S estimates, not measured.

## Verified vs not

- V (offline): the loader, the rule, reorder-without-banking, the free choice, a Paladin and a Sorceress case, `Validate` against the real tables on the worker host (`go test -run TestUnlocksValidate ./internal/leveler`, skips without the install).
- U: every event placement (veteran schedule); whether the runner can emit `waypoint:`, `boss:` and `quest:` events (waypoint-lit is U today); the quest id `rescue_cain`.
- Not built: the runner call to `Progress.Fire` and to `Options.Unlocks`; Codex skill icons lighting up and the free-choice picker in the watch UI (`internal/watch` is not owned by this card); persisting Progress with the hero. A follow-up card should do these.

## The other six classes (TASK-143)

Files: `internal/plandata/unlocks-<amazon|assassin|barbarian|druid|necromancer|paladin>.json` (copies under `docs/plan/data/`), from section 2 of `docs/research/class-early-game.md`. All events are the veteran's (tag S, a veteran estimate, not measured); tag U marks an entry that is unsure or that this table changed. Skill names are the Skills.txt keys, which differ from the display name for a few skills (Quickness is Burst of Speed, Eruption is Fissure, Fire Trauma is Fire Blast, Wake of Fire Sentry is Wake of Fire, Plague Poppy is Poison Creeper, Dopplezon is Decoy); every name was checked against the real tables (V, `TestUnlockSkillsExistInTheTables` on the worker host, skips without them). Levels are not stored in the files; they come from Skills.txt at run time, and several of the veteran's per-class level claims are wrong (for example Zeal is level 12, not 6, and Lightning Bolt level 12 for the Amazon; same pattern as `docs/research/skill-req-check.md`).

Deviations from the veteran schedule, all because a skill is a row of the class's normal30 build table and the rule is that nothing the bot needs locks later than Bishibosh:

- Amazon: Exploding Arrow and Multiple Shot (veteran: Andariel) are free; the bow table spends them at clvl 7 and 12.
- Assassin: Wake of Fire (veteran: Blood Raven) is free; it is the main attack from clvl 12.
- Barbarian: Battle Orders (veteran: Blood Raven), Frenzy (Andariel) and Howl are free; the table spends Howl at 7, Battle Orders at 26, Frenzy at 24, and Battle Orders is the class's defensive buff. Shout stays free as the veteran says.
- Necromancer: Skeleton Mastery (veteran: Den) is free (summoner table row clvl 3); Raise Skeletal Mage (Andariel) is free (row 12); Bone Spear moves from Andariel to the Cain quest (U, leveling table row 18).
- Paladin: Resist Fire stays free (Corpsefire is fire; the legacy table spends it at clvl 4), only Resist Lightning locks at the Den; Prayer stays free as the veteran says, only Cleansing locks.
- Amazon "Slow Missiles or Critical Strike" is ambiguous in the schedule: both are locked until the Den (U, open choice).

Tests (`internal/plandata/unlocks_class_test.go`): all seven files load; each class keeps its main attack, a defensive buff and an immunity skill free; at the expected event levels no row of the class's build tables is refused (`TestNoBuildRowIsRefusedByAnUnlock`; event levels are S estimates kept in the test); unknown class is an error naming it. Not verified live; the expected event levels are estimates.
