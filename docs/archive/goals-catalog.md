# Goal catalog: short-term objectives (research, docs only)

Layer above the five existing goal types (`reach_level`, `farm_area`, `kill_boss`, `collect_item`, `idle`, see docs/api.md). Each row is one
objective a user could pick. "Maps to" says whether an existing type already expresses it (E) or a new type is needed (N, see tracking.md section 4).
Companion files: execution.md (how), tracking.md (measuring), planner.md (composition).

Tags: (V) verified from a fetched primary/reliable source, (S) snippet or secondary source, (U) unverified / from memory or a guess. Effort is
wall-clock game time for a GOOD bot on the primary build (Blizzard Sorceress, docs/plan/leveling.md) once the capability exists; every effort is (U)
unless marked otherwise (we have no live data beyond 2,700 XP/h at clvl 1, docs/bot.md). Risk: L = death unlikely, M = occasional deaths,
H = real death risk or item/progress loss (softcore assumed). Tiers: N = Normal, NM = Nightmare, H = Hell. Area ids are levels.txt ids from
recollection (U) and must be checked against the user's MPQ at setup (docs/plan/data/runs.json says the same).

Sources: quest names, NPCs, items and triggers (V) https://classic.battle.net/diablo2exp/quests/act1.shtml (act2..act5 likewise);
rewards (S) https://www.purediablo.com/?p=3457 ; rune/boss run data in repo docs/plan/data/runs.json and docs/prior-art/02-runs.md (kolbot/SoloPlay port notes).

Naming: `family.verb:param`. Parameters in braces are free. "all" tiers = same objective in N, NM and H with different effort.

## 1. Leveling milestones (family `level`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `level.reach:5` | level=5 | N | none | 25 min (U) | L | E reach_level |
| `level.reach:10` | level=10 | N | Cold Plains reachable | 1 h (U) | L | E |
| `level.reach:18` | level=18 (Teleport for a sorc) | N | Act 2 area access helps | 3.5 h (U) | M | E |
| `level.reach:24` | level=24 (Normal Act 4 floor, docs/prior-art/01) | N | Act 3 | 7 h (U) | M | E |
| `level.reach:30` | level=30 | N | Normal Act 4+ | 10 h (U) | M | E |
| `level.reach:40` | level=40 (NM entry ~27-38, docs/plan/data/progression-normal.json) | N/NM | NM unlocked | 25 h (U) | M | E |
| `level.reach:60` | level=60 | NM/H | Hell unlocked helpful | 70 h (U) | M | E |
| `level.reach:75` | level=75 (Hell Mephisto floor, runs.json) | H | Hell | 150 h (U) | H | E |
| `level.reach:85` | level=85 | H | Hell | 300 h (U) | H | E |
| `level.reach:99` | level=99 | H | Hell | 1500 h (U) | H | E |
| `level.for_difficulty` | difficulty | N,NM | none | ends at the level floor of the next difficulty | M | N (meta, planner) |

## 2. Quests Act 1 (family `quest`, param `quest`, `difficulty`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `quest.den_of_evil` | Akara, area 8 | N (NM/H skip, see execution.md) | clvl >= 4 recommended | 15 min (U) | L | N quest (E via kill_boss misuse is wrong: all monsters) |
| `quest.sisters_burial` | Blood Raven, area 17 | all | den done not required | 15 min (U) | L | E kill_boss "Blood Raven" + turn-in N |
| `quest.hire_rogue` | Kashya hire | all | sisters done | 5 min (U) | L | N |
| `quest.search_for_cain` | Inifuss tree area 5, Akara, Stony Field stones area 4 | N (NM/H optional) | Blood Raven done | 25 min (U) | M | N |
| `quest.rescue_cain_tristram` | Tristram area 38, cage | N | stones order solved | 10 min (U) | M | N |
| `quest.forgotten_tower` | Countess, areas 20-25 | all | none | 20 min N (U) | M | E kill_boss "The Countess" |
| `quest.tools_of_the_trade` | Smith in Barracks area 28, Charsi imbue | all | clvl >= 8 (V) | 25 min (U) | M | E kill_boss "The Smith" + turn-in N |
| `quest.sisters_to_slaughter` | Andariel, area 37 | all | Normal: clvl ~22 (runs.json) | 20 min (U) | H (poison) | E kill_boss "Andariel" |
| `quest.act1_complete` | all required Act 1 steps + travel with Warriv | all | Andariel dead | 20 min after Andariel (U) | L | N act_change |

## 3. Quests Act 2
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `quest.radament` | Atma, Sewers 3 area 49 | all | Act 2 reached | 20 min (U) | M | E kill_boss "Radament" + book + turn-in N |
| `quest.horadric_cube` | Halls of the Dead 3 chest (area 58) | N | Act 2 reached; do after Radament (scroll) | 20 min (U) | M | N |
| `quest.viper_amulet` | Claw Viper Temple 2 altar (area 61, U) | N | Lost City route | 25 min (U) | M | N |
| `quest.staff_of_kings` | Maggot Lair 3 chest (area 64, U) | N | Far Oasis | 25 min (U) | M | N |
| `quest.horadric_staff` | cube staff + amulet, use at orifice | N | the three above | 5 min (U) | L | N |
| `quest.arcane_sanctuary` | Palace Cellar 3 portal, Jerhyn/Drognan | N | clvl ~20 (runs.json Summoner 24) | 30 min (U) | M | N |
| `quest.summoner` | Summoner, area 74 | all | Arcane Sanctuary, Teleport strongly preferred | 25 min (U) | H | E kill_boss "The Summoner" |
| `quest.duriel` | Tal Rasha's tomb, Duriel area 73 | all | staff (N), clvl ~24 | 30 min (U) | H | E kill_boss "Duriel" |
| `quest.act2_complete` | Tyrael, Jerhyn, Meshif boat | all | Duriel dead | 10 min (U) | L | N act_change |

## 4. Quests Act 3
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `quest.golden_bird` | Jade Figurine, Meshif, Alkor potion | N | Act 3 | 30 min (U) | M | N |
| `quest.blade_old_religion` | Hratli, Gidbinn in Flayer Jungle (area 78) | N | Act 3 | 25 min (U) | M | N |
| `quest.lam_esen` | Alkor, tome in Ruined Temple (area 94) | N | Kurast Bazaar | 20 min (U) | M | N |
| `quest.khalims_relics` | Eye (Spider Cavern 85), Heart (Sewers 93), Brain (Flayer Dungeon 3, 91) | N (NM/H optional) | Act 3 | 60 min (U) | M | N collect (E collect_item partly) |
| `quest.khalims_will` | Flail from Council, cube, smash Compelling Orb | all | relics + Travincal clear | 40 min (U) | H | N |
| `quest.blackened_temple` | kill the High Council, Travincal area 83 | all | none | 20 min (U) | H | E farm_area Travincal |
| `quest.mephisto` | Durance 3 area 102 | all | orb smashed (Khalim's Will) | 30 min (U) | H | E kill_boss "Mephisto" |
| `quest.act3_complete` | portal to the Fortress | all | Mephisto dead | 5 min (U) | L | N act_change |

## 5. Quests Act 4 and 5
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `quest.izual` | Plains of Despair 105 (Izual), Tyrael: 2 skill points (V reward text, S) | all | Act 4 | 20 min (U) | H | E kill_boss "Izual" |
| `quest.hellforge` | Hephasto (River of Flame 107), Soulstone from Mephisto, smash | all | Mephisto's Soulstone picked up | 40 min (U) | H | N |
| `quest.diablo` | Chaos Sanctuary 108, seals | all | River of Flame WP | 40 min (U) | H | E kill_boss "Diablo" |
| `quest.act4_complete` | Tyrael red portal to Harrogath | N,NM,H | Diablo dead | 5 min (U) | L | N act_change |
| `quest.siege_shenk` | Shenk, Bloody Foothills 110; Larzuk socket reward | all | Act 5 | 25 min (U) | M | E kill_boss "Shenk" + turn-in N |
| `quest.rescue_barbarians` | Frigid Highlands 111, doors, Qual-Kehk | all (can fail, V) | Shenk dead | 25 min (U) | M | N |
| `quest.prison_of_ice` | Frozen River 114, Anya, Malah; +10 res scroll | all | Act 5 | 30 min (U) | M | N |
| `quest.nihlathak` | Halls of Vaught 123 | all | Anya portal | 30 min (U) | H | E kill_boss "Nihlathak" |
| `quest.ancients` | Arreat Summit 120, three Ancients | all (clvl 20/40/60 V) | Nihlathak dead | 25 min (U) | H | N |
| `quest.baal` | Throne 131, Worldstone Chamber 132 | all | Ancients done | 50 min (U) | H | E kill_boss "Baal" (waves N) |
| `quest.act5_complete` | finish difficulty, new game next difficulty | all | Baal dead | 10 min (U) | L | N difficulty_change |
| `quest.cow_level` | Wirt's leg + Tome of TP in Cube, Moo Moo Farm 39 | once per diff (U) | Tristram open, Diablo/Baal done in N (U) | 60 min (U) | H | E farm_area "run:cows" + setup N |

## 6. Bosses and super uniques (farmable, family `boss`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `boss.blood_raven` | area 17 | all | none | 5 min/run (U) | L | E kill_boss |
| `boss.countess` | runes farm | all | The NAS access; min clvl 14/42/67 (runs.json) | 8 min/run (U) | M | E run:countess |
| `boss.smith` | Barracks | all | clvl 6+ | 6 min/run (U) | L | E |
| `boss.andariel` | min clvl 22/45/65 | all | Act 1 WP at Catacombs 2 | 6 min/run (U) | H | E run:andariel |
| `boss.summoner` | min clvl 24/50/70 | all | Arcane WP | 7 min/run (U) | H | E |
| `boss.mephisto` | min clvl 28/55/75 | all | Durance 2 WP | 6 min/run (U) | H | E run:mephisto |
| `boss.pindleskin` | Anya portal; min clvl from runs.json | all | Prison of Ice done | 3 min/run (U) | M | E run:pindleskin |
| `boss.diablo` | Chaos Sanctuary | all | River WP | 12 min/run (U) | H | E run:diablo |
| `boss.baal` | Throne waves | all | Ancients done | 18 min/run (U) | H | E run:baal |
| `boss.act5_trio` | Eldritch + Shenk + Pindleskin | all | Act 5 WP | 10 min/run (U) | M | E run:act5-bosses |
| `boss.kill_count` | boss, count | all | boss prerequisites | count x run | by boss | E kill_boss count |

## 7. Areas: clearing and exploration (family `area`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `area.clear:blood_moor` | area 2 | N | none | 8 min (U) | L | E farm_area untilKills |
| `area.clear:cold_plains` | area 3 | N | clvl 5 (progression-normal.json) | 8 min (U) | L | E |
| `area.clear:den_of_evil` | area 8 | N | clvl 2 | 12 min (U) | L | E |
| `area.clear:burial_grounds` | area 17 | N | clvl 6 | 10 min (U) | L | E |
| `area.clear:stony_field` | area 4 | N | Cold Plains | 10 min (U) | L | E |
| `area.clear:catacombs` | areas 34-37 | N | Andariel reachable | 30 min (U) | M | E |
| `area.clear:lost_city` | area 44 | N | Act 2 | 15 min (U) | M | E |
| `area.explore:act1_map` | visit every Act 1 outdoor area | N | none | 30 min (U) | L | N |
| `area.farm:ancient_tunnels` | area 65 xp farm | N,NM | Teleport; clvl 21-22 cap on Normal (SoloIndex) | 60 min (U) | M | E farm_area |
| `area.farm:chaos_sanctuary` | trash only | NM,H | Diablo seals | 20 min/lap (U) | H | E |
| `area.farm:cow_level` | Moo Moo Farm | all | cow portal | 30 min (U) | H | E run:cows |
| `area.farm:worldstone_keep` | areas 128-130 | NM,H | Act 5 WP | 30 min (U) | H | E |
| `area.farm:xp_best` | auto: bot picks the best area for level | all | planner | endless | M | E reach_level plan |

## 8. Waypoints (family `wp`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `wp.activate:act1` | 9 waypoints incl. town (U, recollection) | N | reach Catacombs 2 | 40 min (U) | M | N |
| `wp.activate:act2` | 9 waypoints | N | Act 2 | 40 min (U) | M | N |
| `wp.activate:act3` | 9 waypoints | N | Act 3 | 50 min (U) | M | N |
| `wp.activate:act4` | 3 waypoints (Fortress, City of the Damned, River of Flame) | N | Act 4 | 15 min (U) | M | N |
| `wp.activate:act5` | 9 waypoints | N | Act 5 | 45 min (U) | M | N |
| `wp.activate:all` | all 39 | N,NM,H | none | 3 h N (U) | M | N (Nightmare/Hell have separate flags, U) |
| `wp.activate:one` | area | all | route | 5 min (U) | L | N |

## 9. Resources and gold (family `res`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `res.gold:10k` | gold >= 10000 (pack + stash) | N | none | 30 min (U) | L | N |
| `res.gold:100k` | gold >= 100000 | N,NM | Act 2+ | 3 h (U) | M | N |
| `res.gold:1m` | gold >= 1,000,000 | NM,H | farm runs | 15 h (U) | M | N |
| `res.potions:belt_healing` | 10 healing potions in belt | all | gold | 5 min (U) | L | N |
| `res.potions:belt_full` | 4 columns configured (docs/plan/data/potions-belt.json) | all | gold | 5 min (U) | L | N |
| `res.potions:mana_rejuv` | n mana / rejuvenation | all | gold or drops | 10 min (U) | L | N |
| `res.scrolls:tp_tome` | Tome of Town Portal + 20 scrolls | N | 450 gold (U) | 5 min (U) | L | N |
| `res.keys:stash_free` | 50 percent stash free (sell/drop junk) | all | none | 10 min (U) | L | N |
| `res.repair:all` | all gear durability > 50 percent | all | gold | 3 min (U) | L | N (durability read UNVERIFIED, docs/bot.md) |
| `res.merc:hire` | Act 2 desert mercenary, aura by difficulty | N | gold | 10 min (U) | L | N |
| `res.merc:revive` | keep merc alive | all | gold | continuous | L | N |

## 10. Gear and items (family `item`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `item.have:type_quality` | {type, quality, count} e.g. "3 rare rings" | all | farm | varies | by area | E collect_item |
| `item.have:unique_any` | 1 unique, any | all | farm | 1-3 h N (U) | M | E collect_item quality=unique |
| `item.have:set_piece` | set, count | all | farm | 3 h (U) | M | E collect_item quality=set |
| `item.equip:upgrade_slot` | slot | all | tier score (prior-art 01 section 6) | opportunistic | L | N |
| `item.equip:resist_cap` | each resist >= 75 | NM,H | gear | 5 h (U) | L | N |
| `item.socket:larzuk` | Larzuk socket quest reward on a chosen item | once per diff | Shenk dead | 5 min (U) | L | N |
| `item.imbue:charsi` | Charsi imbue reward on a weapon/armor | once per diff | Malus (Smith) | 5 min (U) | L | N |
| `item.identify:all` | all unidentified items | all | gold or Cain | 5 min (U) | L | N |
| `item.keep:count_limit` | stash fill < n items | all | none | continuous | L | N |

## 11. Build milestones (family `build`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `build.skill:spend_all` | no unspent skill points | all | build table | per level up | L | N |
| `build.skill:key` | skill id, level (e.g. Teleport 1 at clvl 18) | all | prereq skills | at level | L | N |
| `build.skill:blizzard_20` | Blizzard 20 (build-primary-blizzard.json) | all | clvl 30+ | ~clvl 50 (U) | L | N |
| `build.stat:spend_all` | no unspent stat points | all | build table | per level up | L | N |
| `build.stat:requirement` | str/dex for equipped item | all | stats | at level | L | N |
| `build.respec:akara` | respec token flow | N | Den done (S) | 5 min (U) | M | N |
| `build.hotkeys:bound` | skills on F1..F8 | all | none | 2 min | L | N (done, combat.EnsureHotkeys) |
| `build.charm:life_skiller` | n charms of type | all | pickit | varies | L | N |

## 12. Runes, runewords, charms (family `rune`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `rune.have:tal_eth` | Tal + Eth (Stealth, docs/plan/data/runewords.json) | N | Countess runs | 2 h (U) | M | E collect_item name |
| `rune.have:any` | n runes of rank >= r | all | farm | varies | M | E collect_item type=rune |
| `rune.farm:countess` | until rune X | N | The NAS | 1-6 h (U) | M | E farm_area untilDrops |
| `rune.cube:upgrade` | 3 low runes -> 1 higher in Cube | all | Horadric Cube | 5 min (U) | L | N |
| `rune.runeword:make` | runeword name + base | all | runes + sockets + base | 15 min (U) | L | N |
| `rune.gem:flawless` | upgrade gems by Cube | all | cube | 10 min (U) | L | N |
| `rune.charm:small_life` | grand/large/small charm with life | all | farm | varies | L | E collect_item type=charm |

## 13. Survival and safety (family `safe`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `safe.deaths:max` | N deaths per hour, then pause (bot.pauseDeathsPerHour, exists) | all | none | continuous | L | E settings.maxDeaths |
| `safe.resist:cap` | resists >= x before entering NM/H (prior-art 01) | NM,H | gear | n/a | L | N (gate) |
| `safe.merc:alive` | revive within 1 town trip | all | gold | continuous | L | N |
| `safe.corpse:retrieve` | policy knob, exists (bot.corpseValueMin) | all | softcore | n/a | M | E bot policy |
| `safe.no_boss_below_clvl` | floor from runs.json min_clvl | all | none | n/a | L | E bot rule |
| `safe.stash_gold` | bank gold above level-based floor | all | stash | continuous | L | E town (exists) |

## 14. Difficulty progression (family `diff`)
| id | params | tiers | prerequisites | effort | risk | maps to |
|---|---|---|---|---|---|---|
| `diff.complete:normal` | Baal dead in Normal (expansion) | N | all 5 acts | 20-30 h (U) | M | N (meta of quests above) |
| `diff.complete:nightmare` | Baal dead in NM | NM | Normal done, clvl >= ~40, resists | 40-60 h (U) | H | N |
| `diff.complete:hell` | Baal dead in Hell | H | NM done | 100-200 h (U) | H | N |
| `diff.enter:nightmare` | new game on NM with the same character | NM | `diff.complete:normal`, level floor, res > 0 after -40 penalty (U, prior-art 01) | 10 min (U) | M | N |
| `diff.enter:hell` | same, Hell | H | `diff.complete:nightmare`, penalty -100 (U) | 10 min | H | N |
| `diff.act_pass:<n>` | beat the act boss of act n in the current difficulty | all | previous act | per act | by act | N |

## Count and notes
Concrete examples are the rows above (each `id` is one objective; parameterised ids such as `level.reach:N` stand for every value). Row count is
printed in README.md. Ordering inside a family follows play order, which the planner (planner.md) turns into a dependency graph.
Existing types cover only what the manager can count from kill/drop/level events. Everything marked N needs state from memory (quests, waypoints,
gold, inventory, skills) or a multi-step procedure; see tracking.md for what must be added to goals.Goal.
