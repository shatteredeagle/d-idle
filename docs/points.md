# Points: stat and skill points are the player's (roadmap R3, agency P0)

> status: current (card agencyp0, 2026-11-19). Decision: ADR 0020 (path D, user decision): the player spends every stat point and skill point; the hero plays what it actually has; autopilot is a per-hero switch, default off; unsupported skills are allowed with a label and one confirmation for attack skills. Source: `docs/design/fable-review-2026-11-19/04-stat-skill-agency.md` section 7 (P0). Tags: V verified (how said), S secondary, U unverified. This card builds P0 only: no planner, no combat-engine change, nothing spends by itself beyond what the old plan table did when the switch is on.

## 1. What a player sees

- The Points page (`/watch/points`, link in the cockpit header): banked stat and skill points, one `+` per attribute with a one-line preview ("+1 Vitality: +3 life, +1 stamina", from the class table), what the worn gear asks of Strength and Dexterity (and what is lacking), the autopilot switch and the last ten actions with who did them (client class and user-agent family, never an address).
- The Skills page (`/watch/skills`) now has its `+` control when the backend can spend by verb. A skill the hero cannot use yet that is an attack asks once ("Fire Ball is an attack your hero cannot use yet: it will not cast it. Confirm to put the point there anyway"); passives, auras and non-attack skills are allowed and not asked about.
- Needs-you row and return screen: while points are banked and the hero's autopilot is off, "3 stat points and 2 skill points waiting for you" shows as a Needs-you row (kind `points`) and as the return screen card "Points waiting".
- Default: the bot BANKS points. A hero whose autopilot switch is on, a measured or drill run (`D2BOT_RUN` set, `D2BOT_GEAR=bot`), or `resource.pointsDecisions` = 1 spends from its plan table as before (now by verb, see 3).

## 2. API (answer listener behind the same gate as the quest API: origin list, token, JSON only, small body)

| route | what |
|---|---|
| `GET /api/hero/points` | the sheet (`didle.points/1`): `enabled`, `hero`, `class`, `level`, `statPoints`, `skillPoints`, `banked`, `waiting`, `autopilot`, `verbs` (`stat`, `skill`, `via`, `why`), `stats[]` (`key`, `name`, `id`, `now`, `base`, `classBase`, `spent`, `effect`, `preview`, `needsGear`, `canAdd`, `why`), `gear[]` (`slot`, `item`, `str`, `dex`, `shortStr`, `shortDex`), `recent[]` (the audit trail), `note` |
| `POST /api/hero/points/stat` | `{"stat":"str\|dex\|vit\|ene"}` one point by verb, read back; 200 `ok` with `before`, `after`, `left` |
| `POST /api/hero/points/skill` | `{"skill":<id>,"confirm":false}`; 409 `rule` (the game's own rule, with the reason), `other_class`, `level_cap`, `confirm` (`needsConfirm`), 400 `unknown_skill`, 200 `ok` with `tier` |
| `POST /api/hero/points/autopilot` | `{"on":true\|false}`; a repeated press is `ok` with `already` |
| `POST /api/skills/allocate` | the older route; now routed to the same desk, one point, no confirmation (an unsupported attack is refused) |

Refusals carry `code` and `reason` in words. Other codes: `off` (resource.pointsApi 0), `no_hero`, `busy` (hostiles near: points are spent only at a calm moment, the leveler's rule), `no_points`, `no_verb` (501), `verb_failed`, `read_failed`, `not_taken` (502: the verb was sent and the read-back shows no change). Every POST is audited to `<state>/quests/<hero>.points.audit.jsonl` (ring of 200); the switch lives in `<hero>.points.json` next to it.

## 3. Spending by verb (no clicks on d2d)

- Verbs: `cmd stat <stat> [n]` and `cmd skillpt <index>` of the d2d control channel. They are stock upstream commands ("what a remote client would send", fork `docs/control_channel.md`), need no lab flag, and are NOT a new fork addition. V from the fork source (`~/d2d-lab` checkout of the devp13 line): stat ids 0 Strength, 1 Energy, 2 Dexterity, 3 Vitality (`apps/d2d/panels.hpp` `kStatButtons`; the dump's `stats.raw[0..3]` agree, `dump_head_late.json`); `skillpt` takes the skill's INDEX 0..29 within its class, not the skills.txt id (`components/game/world/commands.cpp` passes it to `rules::can_learn` as `skill_index`); the world applies it at its next tick. Go: `d2dctl.Client.CmdStat`, `CmdSkillPt` (existed, had no non-test caller), `BaseAttrs` (new, reads `dump stats` `stats.base`).
- Capability check: the verbs have no `caps` token. `cmd/d2bot/pointswire.go` treats the first "unknown" or "usage" answer as "no verb", records why, and from then on the sheet says so and the API answers 501 `no_verb`; the leveler falls back to clicks. U which text a build without the verb answers (stock d2d has the verbs, S). The exact fork wish list for devp14 is `docs/process/points-verbs-devp14.md`.
- Read-back: after the verb the desk polls the player (8 tries, 150 ms) until the unspent count fell, and for a skill until its hard points rose. A verb that changes nothing is `not_taken`, never "ok".
- The bot's own spending (autopilot): `leveler.Options.Verbs` tries the verb per point and falls back to the click path for the rest of the run at the first failure or when a read-back does not move. Game.exe/Wine keeps the leveler's clicks through `d2input`; there the API answers `no_verb` ("spends by clicking the panels") and the Points page still shows the sheet.

## 4. Legality

- The game's own rules (V: d2d `can_learn`, `components/rules/rules.hpp:1757`, which is the decompiled game.exe icon check FUN_004ac200): a point is free, the skill is under its `maxlvl`, character level >= `reqlevel`, every `reqskill` has a hard point. `skilltree.CanAdd` mirrors it; the desk adds the class gate (another class's skill is refused).
- The per-level cap (kolbot `AutoSkill.js:54-62`: points <= clvl - reqlevel + 1): d2d and the decompiled check do NOT enforce it (V, same function: no such test), so the bot does not enforce it (the knob `resource.skillCapPerLevel` was retired 2026-11-27). The brief's "lab check" is therefore half answered: d2d does not enforce it; whether Game.exe 1.14d does is U (checklist below). The shipped tables that would break it if it were enforced: the hammerdin endgame table banks and spends 11 hammer points at clvl 18 where the cap allows 1 (V `TestHammerdinTableBreaksThePerLevelCap`); the full sweep over every table with the real skills.txt is `TestPerLevelCapOnShippedTablesReal` (runs on the worker host, logs each first break). No table was changed: under d2d they are legal.
- Capability label (interim, until the generated capability table of P1): `supported` when a combat profile names the skill (`combat.Playable`) or it is a passive; otherwise `unsupported`. "Attack" is a skill with a kind in `internal/power/profiles.json` `skillKinds`. An unsupported attack needs one confirmation; everything else is allowed.
- Ladder-only runewords: five endgame tables carry `"ladder_only": [...]` (barbarian_warcry: Spirit, Insight, Fortitude; druid_tornado, necromancer_summoner, sorceress_blizzard: Spirit; paladin_hammerdin: Spirit, Fortitude). Tagged, not deleted; `leveler.Build.LadderOnly` exposes it; `TestLadderTagsOnEndgameTables` pins the list against the 23 documented ladder words (S: design review 03; V for Spirit and Insight: `docs/sme/builds-gear.md`). The check against `runes.txt` is `data.Runeword.Ladder` on real data (not run here).
- Missing skill kinds in the power profiles: 58 attack skills had a damage row but no `skillKinds` entry, among them Fire Bolt, Fire Ball, Frozen Orb, Meteor, Nova, Hydra, Whirlwind, Blade Fury, Immolation Arrow (V: `skilldamage.json` vs `profiles.json`; `TestSkillKindsCoverAttacks`). Added, each as melee, missile or spell by how the skill is delivered (S: veteran knowledge, not checked against the tables; borderline: Shock Wave and Blade Fury are U). A resolver run that failed with "no kind" now resolves; a name that resolved before resolves the same (`go test ./internal/power` unchanged). 21 damage-table names stay unkinded on purpose (auras, armors, buffs, monster skills; listed in the test).
- Resist order: already in game.exe order (V `d2.ResistRules.Effective`: penalty added, then capped at 75 + max bonus, then the -100 floor; used by `readiness`, `combat`, `gear.ResistValue`). The open item is U TASK-216: whether the panel value the d2d dump gives (`power/hero.go`) already contains the difficulty penalty. Not changed.
- Stat and resist assumptions that break under free allocation (listed, data defaults not changed): `leveler.classFloor` (only the Sorceress has a Strength floor of 12 to wear the starter armor; other classes U), `leveler.maxNeed` (40 Str/Dex cap on a reported gear requirement), the plan tables' stat targets by Spirit/Fortitude (above), `power/hero.go` attack rating `(Dex-7)*5` (S), `internal/resource/melee.go` Strength/Dexterity bonus as a percent (S), and every endgame table's resist plan assuming the table's own gear. None of them reads the player's allocation; they read the equipped gear and the class, so a free allocation changes the numbers, not the code paths.

## 5. Knobs (`docs/knobs.md`)

`resource.pointsApi` (1: the read-only sheet and tile), `resource.pointsDecisions` (0: the player; 1: the bot, set to 1 by `withGearPolicy` for measured and drill runs) (the former `resource.skillCapPerLevel` is retired). Group `resource` because the knob groups are a closed set (like `gearDecisions`). Per hero: the autopilot switch.

## 6. Tests and scenarios

- Unit: `go test -race -count=20 ./internal/points ./internal/watch -run 'Points|SkillsAllocate'`; leveler verb path `./internal/leveler -run Verb`; `./cmd/d2bot -run 'PointVerb|PointsAre|WornReq|PointsNeeds'`; `./internal/d2dsrc -run SheetMatchesDump` (sheet equals the recorded dump for two classes).
- Proving Ground (`tools/provingground/scenarios/points.json`, new assertion `stat_between` and observer reading `dump player,stats,skills`): `points.bank-by-default` (25 stat and 5 skill points stay banked) and `points.autopilot-spends-by-verb` (all spent) run on the builder (long lane), classes Paladin and Sorceress; `points.api-skill-legal-and-illegal` and `points.sheet-matches-dump` are `pending` because a scenario has no way to POST to the bot's listener yet (the format needs a `trigger.api`); their offline equivalents are the unit tests above.

## 7. Live checklist (nothing here ran against a game)

1. On a lab d2d, level 6 hero with banked points: `GET /api/hero/points`, then `POST .../stat {"stat":"vit"}`; expect `ok`, `after` = `before + 1`, `left` one less. U until run.
2. `POST .../skill` with a legal skill: `ok`; with a level-18 skill at level 6: 409 `rule` "needs character level 18".
3. Unsupported attack (Blessed Hammer at level 18 on a Paladin): 409 `confirm`, then with `confirm:true` 200 and the Skills page shows the point.
4. Autopilot on: the bot spends the rest by verb (log line `leveler: plan ...`, no panel clicks); off: points wait, the Needs row shows.
5. Game.exe (lab check, U): put 2 points into a level-18 skill at character level 18 by hand; does the game refuse the second? Only if the game refuses would a cap need to come back (as a rule, not a knob).
6. Akara respec on d2d (needed by P1, U).

## 8. Not done (by design of this card)

No planner, no capability table generation, no guide tracks, no respec flow, no auto-spend beyond the old plan table (P1 and later). The cockpit has a header link and a Needs-you row, not a drawn tile in the layout editor.
