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
