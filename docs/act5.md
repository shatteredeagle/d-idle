# Normal Act 5 plan (agent ACT3PLAN, docs only)

> status: OFFLINE PLAN: never run live; 460+ code commits since this was last touched (doc audit 2026-10-30). Treat every row as U. Current state: docs/plan/STATUS.md.

Companion of `docs/act3.md` (shared work list X1..X7) and `docs/act4.md`. Target classic LoD 1.14d (Act 5 needs the Expansion flag, V-02). Docs only.

Tags: **V** verified (our code at f5b7197, live results in other docs cited, REAL 1.14d tables on the worker host read 2026-10-29, or a page I fetched), **S** secondary (URL),
**U** unverified / memory. Sources: Arreat Summit `https://classic.battle.net/diablo2exp/quests/act5.shtml` and `.../rewards.shtml` (V text);
Kolbot `https://raw.githubusercontent.com/blizzhackers/kolbot/master/d2bs/kolbot/libs/bots/{Nihlathak,Questing,Baal,Rusher}.js`, `.../libs/common/Common.js` (Ancients, Baal), `.../tools/RushThread.js`, `.../libs/modules/sdk.js` (S);
repo port notes `docs/archive/prior-art/02-runs.md` (S).

## 1 What exists today (V, code at f5b7197)

| layer | Act 5 | note |
|---|---|---|
| `saves` words | `act5_intro` 32 (U), `siege_harrogath` 35, `rescue_arreat` 36, `prison_of_ice` 37, `betrayal_harrogath` 38, `rite_of_passage` 39, `eve_of_destruction` 40 | 37 and 40 V live (docs/saves.md), others S order; `act4_done` 28 gates the log tab |
| `quest.Hops` | `A4->A5 Tyrael` (class 367, "Travel to Harrogath", `Pre terrors_end`, U) | fakes only |
| `quest.Menus` | Tyrael 367 only | no Larzuk, Malah, Anya, Qual-Kehk, Nihlathak rows |
| `task` | nothing | |
| `boss` | no Shenk, Nihlathak, Ancients, Baal; `Script.NoTP`, `Constraints.NoTownPortal` exist as data | `NoTownPortal`/`NoLeaveArea` are NOT applied anywhere in the bot (grep: only the struct fields and the executor hook comment), so the Ancients constraint is untested end to end |
| `nav` | `wpLevels[4] = {109,111,112,113,115,123,117,118,129}` (waypoint numbers 30..38, V table order); tab x for act 5 = 344 extrapolated (U) | |
| `town` | `Acts[4]`: town 109, Heal/Shop Malah 513, Repair Larzuk 511, Gamble Anya 512, Merc Qual-Kehk 515, waypoint object 429 | classes V, roles U |
| difficulty hop | design only (docs/progression/difficulty.md); no code | |

## 2 Quest chain (text V Arreat act5.shtml / rewards.shtml)

| quest (word) | trigger | work | reward | tags |
|---|---|---|---|---|
| Siege on Harrogath (35) | Larzuk, or entering the siege area | Bloody Foothills (110, no waypoint: waypoint 111 then walk west): kill Shenk the Overseer (SU 42, class 479) in his siege camp; talk to Larzuk | **Larzuk adds sockets** to one chosen item; works in every difficulty (V rewards.shtml) | positions S: Kolbot Shenk (3846,5120)/(3883,5113), prior-art (3876,5130); the socket UI is U |
| Rescue on Mount Arreat (36) | Qual-Kehk after Shenk, or entering Frigid Highlands | Frigid Highlands (111): three cages (object 473, V "cagedwussie1"), destroy the Prison Door monsters (class 434, V) next to each, captives are class 534; talk to Qual-Kehk; **can fail** ("unable to complete", V text), then a new game is needed | barbarian mercs; runes (S, U which) | skippable; Kolbot Questing.barbs S; failure detection U |
| Prison of Ice (37) | Malah, or near Anya in the Frozen River | Crystalline Passage 113 -> Frozen River (114 Cellar of Pity, V Vis): touch frozen Anya (object 558 "fana", platform preset 460), town, Malah gives the potion (item `ice` "Keep it to thaw Anya", V), back, interact Anya, town, Malah gives the Scroll of Resistance `tr2` (V "Right Click to Cast"), use it | all resistances +10 (Scroll) and a rare class item from Anya (V rewards.shtml) | flow S Kolbot Questing.anya; opens Anya's red portal to Nihlathak's Temple (S) |
| Betrayal of Harrogath (38) | Anya | red portal in town -> Nihlathak's Temple 121 -> Halls of Anguish 122 -> Pain 123 (waypoint 35) -> Vaught 124; kill Nihlathak (SU 60, class 526) at the platform (object 462) | personalize one item (V) | Kolbot aborts the run if revived Tomb Vipers exist (S); his Corpse Explosion kills (V text) |
| Rite of Passage (39) | Qual-Kehk / reach the top | Ancients' Way 118 (waypoint 37) -> Arreat Summit 120: touch the altar (object 546), kill Talic, Madawc, Korlic (classes 540/541/542, SU 43..45) **without leaving the area and without a town portal** (V text); clvl **20 / 40 / 60** needed for credit (V text) | one level of experience, max 1.4M in Normal (V rewards.shtml) | altar spot (10047,12622), statues 474/475/476 (S Common.js) |
| Eve of Destruction (40) | the Ancients | WSK 128 -> 129 (waypoint 38) -> 130 -> Throne 131: clear 5 waves, enter the Worldstone Chamber 132, kill Baal | difficulty complete | **Req**; words V live |

Critical path: Prison of Ice (portal), Nihlathak, Ancients, Baal. Siege (Larzuk sockets) and Rescue are optional but Larzuk's sockets are valuable. Prison of Ice is only Required if the
Nihlathak portal matters; Betrayal (38) is not needed for the Ancients in the game itself (S Kolbot `Questing` order: shenk, barbs, anya, ancients, baal; the Rusher sequence skips Nihlathak).
The existing docs say the Ancients need Nihlathak first (catalog.md "Ancients ... Nihlathak dead"); that is a bot-side ordering choice, not a game rule, U.

## 3 Areas and waypoints (V levels.txt; data names, in-game names in brackets)

| id | name | wp | MonLvl N/NM/H | Vis |
|---|---|---|---|---|
| 109 | Harrogath | 30 (object 429) | town | none |
| 110 | Bloody Foothills (Shenk) | none | 24/58/80 | none |
| 111 | Rigid Highlands [Frigid Highlands] | 31 | 25/59/81 | none |
| 112 | Arreat Plateau | 32 | 26/60/81 | 113 |
| 113 | Crystalized Cavern 1 [Crystalline Passage] | 33 | 29/61/82 | 112, 115, 114 |
| 114 | Cellar of Pity [Frozen River, Anya] | none | 29/61/83 | 113 |
| 115 | Crystalized Cavern 2 [Glacial Trail] | 34 | 29/61/83 | 113, 117, 116 |
| 116 | Echo Chamber [Drifter Cavern] | none | 29/61/84 | 115 |
| 117 | Tundra Wastelands [Frozen Tundra] | 36 | 27/60/81 | 115, 118 |
| 118 | Glacial Caves 1 [Ancients' Way] | 37 | 29/62/82 | 117, 120, 119 |
| 119 | Glacial Caves 2 [Icy Cellar] | none | 29/62/83 | 118 |
| 120 | Rocky Summit [Arreat Summit] | none | 37/68/87 | 118, 128 |
| 121 | Nihlathak's Temple | none | 32/63/83 | 122 |
| 122 | Halls of Anguish | none | 33/63/83 | 121, 123 |
| 123 | Halls of Death's Calling [Halls of Pain] | 35 | 34/64/84 | 122, 124 |
| 124 | Halls of Vaught | none | 36/64/84 | 123 |
| 125..127 | Hell1..3 [Abaddon, Pit of Acheron, Infernal Pit] | none | 39/60..62/81..83 | not used |
| 128 | Worldstone Keep Level 1 | none | 39/65/85 | 120, 129 |
| 129 | Worldstone Keep Level 2 | 38 | 40/65/85 | 128, 130 |
| 130 | Worldstone Keep Level 3 | none | 42/66/85 | 129, 131 |
| 131 | Throne of Destruction | none | 43/66/85 | 130, 132 |
| 132 | The Worldstone Chamber | none | 43/66/85 | 131 |

**Correction (V from levels.txt) to older docs**: `docs/progression/bosses.md` and `docs/goals/execution.md` give Halls of Pain 122 and Halls of Vaught 123. The table says Halls of Anguish 122, Halls of Pain 123 (waypoint 35), Halls of Vaught 124. `nav.wpLevels` (123) is already right; `runs.json` Pindleskin stops `[109,121]` is right. Nihlathak's final area is 124.
Not in Vis (level crossings, U): Harrogath -> 110 -> 111 -> 112 -> 113 chain edges, 115/117 to the Tundra, 120 -> 128 after the Ancients. `act5Hints` follows the waypoint order (Harrogath 109, 111, 112, 113, 115, 117, 118), S order from the map, confirmed by first ExitTo. Rows 7 and 8 of the Act 5 panel (Ancients' Way, WSK2) may need the icon-column click like Act 2 rows 7/8 (U).
Anya's portal to 121 and the red portal in Harrogath: object classes are not in my dump (the Act 5 town portal object is U); Kolbot `Pindleskin.js`/RushThread use the town red portal near (5130,5120) (S, prior-art).

## 4 NPCs, objects, items

NPCs (V monstats): Larzuk 511, Anya `drehya` 512 (frozen form `drehyaiced` 527, V), Malah 513, `nihlathak` 514 (the town NPC), Qual-Kehk 515, Cain `cain6` 520, Tyrael `tyrael3` 521 (class that appears after Diablo U).
Objects (V names, roles S Kolbot sdk.js): 429 Act 5 waypoint, 460 "dummy" (FrozenAnyasPlatform), 558 "fana" (FrozenAnya), 462 "dummy" (NihlathaksPlatform), 473 cagedwussie1, 546 ancientsaltar, 474/475/476 "Ancient Statue 3/1/2", 547 "To The Worldstone Keep Level 1", 563 "The Worldstone Chamber" (Baal portal), 569 "Throne of Destruction", 449 "gate".
Items (V): `ice` Malah's potion ("Keep it to thaw Anya"), `tr2` Scroll of Malah - Boost Resistances ("Right Click to Cast"). Larzuk's sockets and Anya's personalize have no item.
Mercenary: Qual-Kehk 515 hires the Barbarian (Bash, Stun rows, 1-hand), Normal rows from clvl 28, price factor 9000 (V hireling.txt): usable but expensive; Act 3's Iron Wolf is cheaper (docs/act3.md 4).

## 5 Cube

No quest recipe in Act 5 (V: cubemain has no Act 5 quest row; Wirt's leg / cow portal row exists, out of scope).

## 6 Bosses and fights (V table data unless S)

| boss | where | prerequisite | data | fight facts |
|---|---|---|---|---|
| Shenk the Overseer | Bloody Foothills 110 | none; floor clvl 25 (runs.json R) | SU 42, class 479 `overseer1`; TC "Act 5 Super A" | stand around (3846..3883,5113..5130) (S, three sources disagree by up to 40 subtiles, so use the monster list, not coordinates); camp of minions; Larzuk after |
| Nihlathak | Halls of Vaught 124 | Anya's portal or waypoint 35 | SU 60, class 526 `nihlathakboss`; Normal cold 70, poison 70, Hell fire/light 33, cold/poison 70; level 36 area | platform preset 462; Corpse Explosion on raised corpses is lethal (V Arreat text): kill adds' corpses or rush; abort on revived Tomb Vipers (S); Mana Burn risk in Hell (danger-table S) |
| Talic, Madawc, Korlic | Arreat Summit 120 | clvl 20/40/60 (V text), altar touched | classes 540/541/542; level 37/68/87; Normal: Talic phys/fire/poison 70, Madawc lightning/cold 70, Korlic phys/fire/poison 70; Hell: Talic fire 110, Madawc lightning 100, Korlic cold 150 (immune) | **No town portal and never leave the area** (V text); an empty summit resets them (V text: "if no one is alive on the summit"), so a dead bot restarts the fight. Pick targets by resist: Talic and Korlic take lightning/cold/magic, Madawc takes fire/physical/poison/magic (V Normal table). All three attack together after the altar |
| Baal waves | Throne 131 | WSK3 reached; floor clvl 32 (runs.json R) | waves S (Common.js): 1 Warped Fallen/Shaman, 2 Baal Subject Mummy / Cold Mage, 3 Council member (class 557) plus a hydra check, 4 Venom Lord, 5 Lister the Tormentor (SU 65, class 571) and minions 571/572/573; super unique names SU 61..65: Colenzo, Achmel, Bartuc, Ventar, Lister (V) | throne rectangle x 15072..15118, y < 5080 (S), wait spots (15095,5029) and (15116,5026) for casters, 8 clear nodes every 20 s (S prior-art); souls and dolls on entry (S Kolbot quit option); Baal at (15090,5014) is not attackable until the waves end |
| Baal | Worldstone Chamber 132 via portal object 563 | all waves dead, the throne Baal (class 543) gone (wait up to 3 minutes, S) | class 544 `baalcrab` level 60/75/99, resist Normal fire/light/cold 33, poison 50; Hell phys 50, elements 50 (V); duplicate class 570 `baalclone` (V exists) | stand (15134,5923) (S); Arreat text V: "Baal can spawn in duplicates ... do not waste time on the fake"; Static Field recommended by Arreat (V text) |

Dangers per class (U unless V above): Sorceress - Normal Ancients: Blizzard/cold hits Talic and Korlic (0 cold, V table), lightning for Talic/Korlic, fire for Madawc; casting range matters because all three charge. Paladin hammer builds do magic damage which none resists in Normal (V: magic 0 on all three Ancients). Amazon/Assassin with poison or physical skills face 70 percent walls on two of three (V). Barbarian physical: Madawc only. Necromancer: poison/bone vs Talic/Korlic poor (70 percent) but Madawc ok.
Baal: any class needs a plan for wave 3 hydras, wave 5 and the Chamber's tentacle/appendage adds (classes 562..566 V exist); casters stand at the S coordinates, melee in the Chamber.
The Normal levels in the tables (24 to 43 for Act 5 areas) mean a character at the Act 4 floors (clvl 30) is under-levelled: `progression-normal.json` has Arreat Summit enter 36 and Throne enter 38 (R) while Rite of Passage only needs clvl 20 (V). Gate in the planner by level, not only by quest.

## 7 Support matrix and gaps (V code)

| capability | have | gap |
|---|---|---|
| Hop A4->A5 | `Hops[3]` | Tyrael menu row or portal 566, Expansion check |
| Act 5 data | none | `quest/act5.go`, menus (Larzuk, Malah, Anya, Qual-Kehk, Nihlathak, Cain A5) |
| Prison of Ice | `StepTalk`, `StepInteract` primitives | multi-trip talk chain with state: Anya touch, Malah potion, interact Anya, Malah scroll, `UseItem` (exists in `Act2Env`) |
| Larzuk sockets | none | menu row plus item selection UI (U; likely needs a screenshot-driven grid pick) |
| Shenk, Nihlathak scripts | framework | scripts with `PresetNPC`/object anchors, Corpse Explosion hazard (no producer), Tomb Viper abort check (needs monster class list in `boss.View`, V it has `Mon.Class`) |
| Ancients | `NoTP`, `Constraints.NoTownPortal` data | enforcement in the bot (nothing applies the constraint); altar interaction; three `Match` targets chosen by resist; death handling that restarts the fight |
| Baal waves | none | a wave loop (new mechanism like the seal executor): classify wave by monster classes in the throne rectangle, clear, idle clear every 20 s, souls/dolls abort, Baal portal, chamber fight |
| Difficulty hop | design (docs/progression/difficulty.md) | implement after Baal; out of scope for Normal Act 5 |

## 8 Work list (A5-n; shared X1..X7 in docs/act3.md)

- **A5-1 quest data (d2-worker)**: `internal/quest/act5.go`, `act5_menus.go`: areas V (table above), NPC classes V, objects 429/460/462/473/546/558/563 and items `ice`/`tr2`; `TestAct5ConstantsAgainstRealTables`; also fix `docs/progression/bosses.md` and `execution.md` area ids (123/124).
- **A5-2 task nodes (d2-worker)**: `internal/task/act5.go`: `siege` (Shenk + Larzuk), `rescue` (optional), `prison_of_ice`, `nihlathak`, `ancients` (`Constraints{NoTownPortal, NoLeaveArea}`, `Ready` requires clvl 20 and full potions), `baal`, `act5_complete`; `NormalAct5` and `NormalActs12345`; floors (Shenk 25, Ancients 20/28 by the R table, Baal 32).
- **A5-3 boss scripts (d2-worker)**: `internal/boss/act5.go`: `Shenk` (Match class 479 + `WithUniqueNo`), `Nihlathak` (Match 526, anchor preset 462, hazard: Tomb Viper abort, corpse explosion), `Ancients` (three scripts or a `MultiMatch`, `NoTP`, `Lure` off), no walking away from the altar spot.
- **A5-4 constraint enforcement (d2-worker)**: apply `Constraints.NoTownPortal/NoLeaveArea` in the bot's `Hooks.OnStart` (`internal/bot/task.go`): suppress the combat `TownPortal` action and the "town trip due" cause while the step runs; test that an Ancients plan never emits a portal.
- **A5-5 Baal wave executor (d2-worker)**: `internal/boss/waves.go`: pure data (wave classes, throne rectangle, wait spots, clear nodes) plus a loop over `boss.World`; souls/dolls entry check; portal to the chamber; Baal script (class 544, stand (15134,5923)). Offline fake-world tests with scripted waves.
- **A5-6 Prison of Ice and Larzuk dialogs (d2-worker offline, d2-live to flip tags)**: `dialogAdapter` rows for Malah/Anya/Larzuk/Qual-Kehk, item selection for the socket reward, `UseItem` of `tr2`.
- **A5-7 nav (d2-worker, then d2-live)**: `act5Hints`, `knownWaypointPos[109]`, tab x for act 5, icon rows; live: edge crossings, Frozen River entrance, Arreat Summit -> WSK stairs after the Ancients (they only open then).
- **A5-8 live passes (d2-live)**: Shenk and sockets; Anya/Malah chain; Nihlathak; Ancients on a cheated clvl 30+ character with gear; Baal throne (waves) then Chamber. Each is a long run: use `tools/live.sh --bg`.
- **A5-9 end of difficulty (d2-worker later)**: after word 40, `diff.complete` hand-off to the Nightmare hop in docs/progression/difficulty.md.

## 9 Open questions

1. Object classes of Harrogath's red portal(s) and Anya's portal to Nihlathak's Temple (not in my dump; S: Pindleskin portal near (5130,5120)).
2. Larzuk's socket UI flow and which item the bot should choose (rules in docs/gear.md if any, else a weapon slot of the equipped build, U).
3. Rescue on Mount Arreat: failure signal and reward (U; Arreat rewards page does not list it).
4. Whether the quest needs Nihlathak dead before Qual-Kehk opens the Ancients: the game does not require it per Kolbot (S), existing docs do (U).
5. Wave classes 1-2 (Warped Fallen/Shaman and Baal Subject Mummy/Cold Mage) are Kolbot constants (S); read their class ids from monstats in a throwaway test before coding the executor.
