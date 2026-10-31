# ACTBOT and ACT45BOT: Acts 2-5 wiring in cmd/d2bot (offline, nothing run live)

Tags: V verified (code or tables), S secondary, U unverified live.

## What is wired
- **X2** `cmd/d2bot/boss.go`: `newBossRunner` returns `*actRunner` over `boss.AllActs()` (Acts 1 to 3). Super unique numbers are
  resolved by `superUniqueScripts` (only the Countess today, V). Council members, Mephisto, Radament, Summoner and Duriel
  match by monster class (V), no number needed. `KillBoss("High Council")` runs `boss.RunCouncil` (member 1 required,
  2 and 3 best effort). A name without script (Fangskin, the optional Claw Viper guard) returns `task.KindNotFound`, so the
  optional step skips. When Shenk/Nihlathak/Hephasto/Ancients scripts exist, add their names to `superUniqueScripts`.
- **X3** `cmd/d2bot/acts.go`, `objects.go`, `task.go`, `internal/bot/task.go`:
  - `taskQuestIDs` gained radament, horadric_staff, arcane_sanctuary, summoner, duriel (word `seven_tombs`), act2_complete
    (`act2_done`), lam_esen, khalims_will, blade_old_religion, golden_bird, blackened_temple, guardian. Fetch legs and the relic
    legs have no word and run on item proxies (V that the ids exist: `TestTaskQuestIDsExistInSaves`).
  - `npcByAct` / `actTalkPaths`: Atma, Drognan, Jerhyn, Meshif (A2 210, A3 264), Warriv (A1/A2), Tyrael, Greiz, Fara,
    Lysander, Cain (A1/A2/A3), Hratli, Alkor, Ormus, Asheara. Classes V; every row except Atma "Talk" is U.
  - `objectAdapter` (bot.Objects) resolves the object names the nodes use (`objSpecs`, classes V objects.txt; relic chest roles
    405/406/407 and red portal 342 are S). Kinds: click, leave (portal/door must change the area), orb (click until mode
    changes), place (staff into the orifice, port of act2live `PlaceItem`), `item:<name>` right-click.
  - `TaskDeps.Cube` (town.Transmute), `TaskDeps.HireMerc` (town.Hire, row from `mercRows`), and `taskActs` implements
    `task.Transmuter`, `MercHirer`, `OrbSmasher` (UseWeapon + orb click), with the pseudo items `Horadric Staff placed`,
    `Book of Skill read`, `Compelling Orb smashed` (per game) and `Mercenary` (from `HasMerc`) in `World.Items`.
  - `World` reports kills of Radament, Fangskin, Summoner, Duriel, Council, Mephisto, and counts worn quest weapons as held.
  - `bot.PickOpts.Names` carries the item name from `task.Actions.PickUp` to the looter (quest items taken whatever the pickit says).
- Templates: `normal_act3` (8 entries, merc on) and `normal_all` (Acts 1-3) in `task.Template`, `goals.actQuests(3)`, `campaignActs`.

## Live checklist (d2-live, one dated line each into docs/progression/results.md)
1. Open each Act 2/3 NPC menu (Drognan, Jerhyn, Cain A2/A3, Tyrael, Alkor, Hratli, Ormus, Asheara, Meshif A3): fix
   `quest/act2_menus.go` / `act3_menus.go` rows and `actTalkPaths`.
2. Hire window of Greiz and Asheara: read which row carries which aura, set `mercRows[act]` (empty = the step is a prereq fault).
3. Which of 405/406/407 is Heart/Brain/Eye; chest names in `objSpecs`; red portal class (342 vs 60); journal click opens a portal.
4. Orifice: `placeStaff` accepting the Staff (never reached, docs/act2.md P4). Compelling Orb: `Interact` loop changes mode 404.
5. Cube with 4 inputs (Flail, Heart, Eye, Brain), and that the Will is worn on pickup/transmute result (items count as held when worn).
6. Item display names for Act 3 quest items (`task.Item*`, U) match `Catalog.Item(...).Name`.
7. `RunCouncil`: do all three members spawn; the Flail drop; Mephisto sidestep mode.
Act 1 gaps closed offline (ACT1GAPS, see "Act 1 objects" below); live checks pending.

## Acts 4 and 5 (ACT45BOT, offline)
- **Templates**: `task.Template` has `normal_act4` (Izual, Diablo, hop = 3), `normal_act5` (Siege, Ancients, Baal = 3) and `normal_all` =
  `NormalActs12345`; `nightmare_all` / `hell_all` replay the same five acts (`difficultyAll`, floors from `combat.EntryGate` plus
  `bossFloors`, new rows Diablo 55/80 and Baal 58/82 from docs/difficulty.md, R). Defaults live in `defaultOptions`: no Hellforge,
  Rescue, Prison, Betrayal or Barbarian (all U or can fail or costly). `goals.actQuests(4)=3`, `(5)=3` (kept equal by
  `TestGoalsTemplateCounts`), `campaignActs` labels 4, 5 and "Acts 1-5".
- **Floors**: Act 5 Ancients 36 and Baal 38 (progression-normal.json summit and throne enter levels, docs/act5.md; the Ancients'
  own credit rule of clvl 20 is not the limit, the area level 36+ is). Shenk 25, Nihlathak 30 unchanged.
- **Boss runner** (`cmd/d2bot/boss.go`): `actRunner` embeds `boss.Act45Runner` with `Ops = objectAdapter` (Diablo seals, the Ancients'
  altar, Baal's portal); "The Ancients" has no script of its own (composite), so it is exempt from the missing-script check.
  `superUniqueScripts` also resolves Hephasto, Shenk and Nihlathak from SuperUniques.txt (harmless when the name does not match; the
  script constants 41, 42, 60 are V from the tables).
- **ObjectOps** on `objectAdapter`: `UseObject` clicks the object of the class nearest to `at`, `ObjectMode` reads its mode (error when
  not streamed in).
- **Objects** (`objSpecs`): Hellforge place/smash (`objPlace` with the item code, `objSmash` = 4 clicks), `Frozen Anya` (558),
  `Anya's portal` (class 60, `objLeave`). `placeStaff` became `placeItem(class, code)`. `taskActs.UseForge` = place Soulstone, swap in
  the Hammer (`UseWeapon`), smash, set `Hellforge used`. `UseItem` = `Interact("item:"+name)`, sets `Scroll of Resistance read`.
  `Interact("Frozen Anya")` sets `Anya touched` or, with Malah's Potion in the pack, `Anya thawed`.
- **NPCs**: Tyrael per act (A2 251, A4 367, A5 521), Cain A4/A5, Jamella, Halbu, Larzuk, Malah, Anya, Qual-Kehk, Nihlathak. Talk paths are
  the plain "Talk" row (Tyrael "harrogath" = main row "Travel to Harrogath"); all U. `mercRows[5]` stays empty (Bash row unknown).
- **Red portal per act**: `quest.ObjRedPortal = 60` (generic portal: Arcane portal V live, Anya's portal S Kolbot) and
  `quest.ObjRedPortalA3 = 342` (Durance to Fortress, S Kolbot); the Act 3 hop uses 342.
- Quest ids: `taskQuestIDs` gained izual, hellforge, terrors_end, siege_harrogath, rescue_arreat, prison_of_ice, betrayal_harrogath,
  rite_of_passage, eve_of_destruction. `act4_hop` and `merc_act5` have no word. World reports kills of the seven new bosses and the items
  Soulstone, Hammer, Malah's Potion, Scroll of Resistance (display names U).
- Not done: Rescue's `cage` object (the nearest-object lookup would click the same cage three times), Larzuk's and Anya's item pick
  (needs the item-choice window), Act 1 objects.

### Live checklist, Acts 4 and 5 (d2-live)
1. Tyrael A4 menu: rows after Izual and Diablo, "Travel to Harrogath" appears; or the hop is a portal object (566 "Harrogath"); fix `actTalkPaths`.
2. Class of Cain in the Fortress (246 or 265), Jamella 405, Halbu 257 respond.
3. Seals: `UseObject` on classes 392..396, mode changes after the click; `RunDiablo` end to end (layout probe, Vizier/Seis/Infector, Diablo spawns).
4. Hellforge: which of place-on-forge by cursor, then Hammer equipped plus click, works; quest word `hellforge` (26) changes; the item display names
   `Mephisto's Soulstone`, `Hell Forge Hammer` match `Catalog.Item(...).Name`.
5. Ancients: altar object 546 mode before and after, `RunAncients` with no town portal. Baal: wave loop, portal 563 into area 132.
6. Act 5 NPC rows: Larzuk (sockets window), Malah potion and scroll, Anya personalize, Qual-Kehk; Barbarian hire row for `mercRows[5]`.
7. Anya: frozen object 558 touch/thaw click, Malah's Potion and Scroll of Resistance names, scroll right-click effect; red portal class 60 into area 121.
8. Rescue cages (473) and Prison Doors (434), if the optional is enabled.

## Act 1 objects (ACT1GAPS, offline; every click U live)

`TaskDeps.Objects` was already set in `newTaskDeps`; what was missing were the names. Now in `cmd/d2bot/objects.go` and `cain.go`:
- `Tree of Inifuss` (class 30, U): click, then `internal/bot` Interact calls PickUp for the scroll (the task node has none).
- `cairn-stones`: classes 17..21 (S Kolbot), backtracking search (`solveCairn`): a stone that stays lit (mode differs from its
  start mode, U) is kept, a wrong one resets (U) and the prefix is re-clicked; max 30 clicks, then a Timeout fault. If the
  Tristram portal (class 60, S) already exists it only enters. It ends by walking through the portal ("Tristram portal",
  objLeave) and sets the `Tristram portal` pseudo item.
- `Cain's Cage` (class 26, U) and `cage` (Frigid Highlands, 473 V): objOnce, the nearest object not yet clicked in this
  area (`onceSet`), so the Rescue step's three calls click three different cages.
- Talk rows: Akara `bark` and Charsi `imbue:best` / `imbue` are the plain `Talk` (the scroll and Malus are taken by the Talk
  with the item in the pack, S). The Imbue item choice itself is NOT done.
- Stash and waypoint: no Act 1 step Interacts with them; waypoints go through `TaskDeps.Waypoint`, the stash through chores.

Live checks: (1) tree class 30 and that the scroll lands in the pack via PickUp; (2) Akara Talk with the scroll returns the
key; (3) stone classes and whether a lit stone shows as a mode change, and a wrong click resets; (4) portal class 60 appears
and the click changes area to 38; (5) cage 26 click frees Cain once; (6) Frigid Highlands: three `cage` calls open three
cages (473) and Prison Door monsters do not block; (7) Charsi Talk with the Malus: does an Imbue row appear, item picker.
