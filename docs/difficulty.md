# Difficulty: Normal -> Nightmare -> Hell (levels, resists, immunities, affixes, detection work list)

Target: classic LoD 1.14d (expansion install), offline single player. Not D2R.
Companion to `docs/progression/difficulty.md` (selection screen, unlock, hop mechanics), which this file does not repeat.

Tags: **V** = verified, with the source named (our install data on the worker host, our code, or a Blizzard page I fetched);
**S** = secondary source or snippet; **U** = unverified / from memory. "D2R:" marks facts that differ in D2R.

Sources actually used:
- Install tables (V): `~/.cache/d2lod-cheat-data/{monstats,levels,difficultylevels,superuniques,strings}.tsv` on the worker host, extracted from the 1.14.3.71 MPQs by `internal/data` (docs/data.md). Quoted as "tables".
- Blizzard Arreat Summit (LoD edition, 1.1x-era text, fetched 2026-10-29, V as a Blizzard primary but written for 1.10-1.13; the numbers match our tables where I could cross-check):
  `https://classic.battle.net/diablo2exp/basics/difficulty.shtml`, `.../monsters/basics.shtml`, `.../monsters/bonus.shtml` ("Arreat Bonus" tables: champion/unique modifiers).
- Our code: `internal/combat/target.go`, `internal/d2/units.go`, `internal/resource/*`, `docs/d2-1.14d-memory.md`, `docs/combat.md`.
- Failed fetch: purediablo immunity guide (returned an image), so class workarounds below are U unless a project class doc says otherwise.

## 1. Numbers that apply to the whole run

| Item | Value | Tag |
|---|---|---|
| Player resist penalty (all difficulties of this install) | Normal 0, Nightmare -40, Hell -100 (applies to fire/cold/lightning/poison, and per Arreat also hirelings) | V tables `difficultylevels.tsv`; code `data.DifficultyInfo.ResistPenalty` |
| Death XP loss | Nightmare 5%, Hell 10% (of the span to the next level); corpse retrieval restores 75% of the loss; none in Normal | V tables (DeathExpPenalty 0/5/10) and Arreat difficulty page |
| Normal-only classic -20/-50 penalty | not applicable to this install | V tables show -40/-100; the -20/-50 in `docs/progression/difficulty.md` G4 is wrong for 1.14d LoD: use -40/-100 |
| Resist cap | 75 base, raised by Maximum resist items/mastery; a res stat can sit far below 0 (the stat list stores the raw value, display clamps) | S (Maxroll); `internal/d2/player.go` comment: "stored, before display clamping" |
| Immune | monster resist >= 100 for that element (stat ids 36 phys, 37 magic, 39 fire, 41 lightning, 43 cold, 45 poison) | V code `combat.Target.Immune`, docs/combat.md |
| Monster life regen on higher difficulties | monsters regenerate fast when not hit; stay on one target | V Arreat monsters/basics |
| Life/mana steal, freeze/chill, AI curses | Nightmare: freeze, chill and AI curses (Confuse, Attract, Dim Vision) cut 1/2; Hell: life/mana steal cut 2/3, freeze/chill/AI curses cut 3/4 (steal has no Nightmare cut listed) | V Arreat difficulty page |
| Static Field floor | cannot reduce life below 33% Nightmare, 50% Hell (Arreat); Normal 25% is only our tuning default | V Arreat (NM 33, Hell 50), V code `StaticFloorNorm/NM/Hell = 25/33/50` |
| Monster skill levels | +3 Nightmare, +7 Hell | V Arreat |
| Unique bonus abilities | Normal 1 extra, Nightmare 1 more, Hell 2 more (Arreat: "Unique Monsters gain one bonus ability" NM, "two" Hell) | V Arreat (wording is of the whole pack, see section 4 for the exact modifier rules) |
| Hireling / summon damage vs act bosses | Normal 50%, Nightmare 35%, Hell 25% | V Arreat difficulty page |
| Champion / Unique damage bonus | +90% Normal, +75% Nightmare, +66% Hell (Champion/Unique pack members; Extra Strong stacks on top, section 4) | V Arreat bonus.shtml |
| Champion/Unique/minion HP multiplier | Minions x2/x1.75/x1.5, Champion x3/x2.5/x2, Unique x4/x3/x2 (Normal/NM/Hell) | V Arreat bonus.shtml |
| Champion level +2, Unique and minions +3 | over the area monster level | V Arreat bonus.shtml |

## 2. Recommended character level per difficulty and act

Monster levels come straight from `levels.tsv` (Ex columns, expansion). These are the real area monster levels; uniques are +3, champions +2 (V Arreat). Recommended entry levels are a judgment: "do not fight an area whose monster level exceeds your clvl by more than the margin" plus the resist gate.

### 2.1 First-area monster levels (V tables, `levels.tsv` MonLvl1Ex..3Ex)

| Act / area | Normal | Nightmare | Hell |
|---|---|---|---|
| A1 Blood Moor (wilderness 1) | 1 | 36 | 67 |
| A1 Cave 1 | 1 | 36 | 79 |
| A1 Catacomb 4 | 12 | 43 | 73 |
| A1 Cave 5 | 7 | 39 | 85 |
| A2 Desert 1 / Sewer 1 | 14 / 13 | 43 / 43 | 75 / 74 |
| A2 Tal Rasha tombs, Duriel's Lair | 17 | 49 | 80 |
| A2 Arcane Sanctuary | 14 | 48 | 79 |
| A3 Jungle / Kurast / Travincal | 21-24 | 49-54 | 79-82 |
| A3 Durance of Hate 3 | 25 | 55 | 83 |
| A4 Plains of Despair (Mesa 1) | 26 | 56 | 82 |
| A4 Chaos Sanctuary (Diablo 1) | 28 | 58 | 85 |
| A5 Siege / Barricade | 24-26 | 58-60 | 80-81 |
| A5 Ice Caves 1-3 | 29 | 61-62 | 82-84 |
| A5 Temple | 32-36 | 63-64 | 83-84 |
| A5 Worldstone Keep 1 / Throne of Destruction | 39 / 43 | 60 / 66 | 81 / 85 |
| A5 Pandemonium 1-3 | 50 | 75 | 83 |

Surprises that matter for a gate table (V tables): Hell Act 1 is not "easy": caves 3/4/5 and crypts are 69-85 in Hell while Blood Moor is 67; Normal Moo Moo Farm is 28/64/81. The existing floor `hell Blood Moor 64` assumes the wilderness only.

### 2.2 Entry levels

Project floors today (`docs/plan/data/progression-*.json`, R): Nightmare 36 at Blood Moor (was 34, changed to the doc value by WIREUP), 45 Arcane, 50 Duriel's Lair, 52 Travincal, 56 Pandemonium, 60 Throne; Hell 64 Blood Moor, 72 Lut Gholein, 75 Travincal, 80 Pandemonium, 82 Harrogath/Throne, 85 Arreat Summit.

Rule of thumb that matches these (U, community-consistent): enter an area at clvl >= (area monster level - 2) in Normal, >= (area monster level - 5) in Nightmare when resists are met, >= (area monster level - 8 to -10) in Hell with an immunity-capable build and resists >= post-penalty 0. Concrete recommendation per act (R, derived from section 2.1; the planner may override with tests):

| Difficulty | Act 1 | Act 2 | Act 3 | Act 4 | Act 5 | Bosses (min clvl, R) |
|---|---|---|---|---|---|---|
| Normal | 1 | 14-18 | 22-24 | 26 | 24-30 | Andariel 18, Duriel 26 (existing doc says 26), Mephisto 30, Diablo 31-33, Baal 33-35 |
| Nightmare | 36 | 44 | 50 | 56 | 58 | Andariel 45, Duriel 50, Mephisto 55, Diablo 55-57, Baal 58-60 (Arreat: "Nightmare monsters start at level 36"; docs/plan/leveling.md) |
| Hell | 66 (wilderness only; caves 75+) | 72 | 75 | 80 | 82 | Andariel 65, Summoner 70, Mephisto 75, Diablo 80, Baal 82 (docs/plan/leveling.md, R) |

Resist gate (R): before leaving Normal: raw fire/cold/lightning >= 40 each is NOT required, but the post-penalty value must be >= 0 for the planner to hop (existing G4). To stay in Nightmare Act 3+, aim for post-penalty >= +20. For Hell aim for post-penalty >= +25 (raw >= 125, i.e. maxed 75 plus items/charms); poison >= 0 post penalty. These are U numbers; they come from the usual "cap everything before Hell" advice (S: Maxroll quoted in docs/plan/leveling.md).

Arreat's own advice: stay in Act V Normal until about level 40 before Nightmare (V, difficulty page). Our planner will normally be much further along.

## 3. Immunities in our install

All numbers from `monstats.tsv` (V, tables). Counts are enabled, killable, non-NPC monster rows with that immunity at that difficulty (an immunity to two elements is counted in the combined row).

| Difficulty | Single immunities (rows) | Notes |
|---|---|---|
| Normal | poison: Iron Golem, turrets, catapults, barricades, skeleton archers 6-11 (many are objects). Fire: Fire Golem. Nothing a leveling char meets in Acts 1-2 | almost none |
| Nightmare | lightning 14 (Black Soul/willowisp4-8, bighead5-10, Flying Scimitar...), fire 11 (fallenshaman5, doomknight1, skmage_fire3, goatman9, clawviper8), cold 16 (frozen horrors 1-4, succubuswitch5-9, goatman7/8/10, trapped souls), poison 43, phys 3 (Spirit of Barbs, Heart of Wolverine, Oak Sage: Druid pets, not hostiles) | resists start at NM Act 3+ |
| Hell | lightning 88, poison 102, cold 131, fire 106, magic 8, physical 14, plus combos below | "In Hell almost all monsters are immune to something" (V Arreat monsters/basics) holds only loosely: well over half of hostile rows have no 100+ resist, but the dangerous packs do |

Notable Hell rows (V tables, Level(H) in brackets):
- Phys immune: Ghost/Wraith/Specter lines and swarms (wraith1-5 are 50 phys; wraith6-9 phys 50 and poison 100 are the Act 5 guests), `swarm1-5`, `succubus4`, `succubuswitch3` (Stygian Fury), `bloodlord3`, `deathmauler4`, `baboon6`, `vampire9` (phys + cold). Ghostly champions display Immune to Physical but are really 80% damage reduction (V Arreat bonus.shtml).
- Magic immune (resist 100): Horadrim Ancient/Guardian/Unraveler lines (`unraveler2-9`, [77-85]); `zombie5`, `brute5` (rare). `unraveler2` = Guardian is magic+fire, `unraveler3` = Unraveler is magic+cold; Dark Elder (superunique, zombie5) has magic.
- Fire immune (>=100): Fire Golem, `fallen*`/`fallenshaman*`, Fetishes 3+ (Soul Killer has lightning 110, Stygian Doll has fire 100/110, Flayer has cold 100 or fire 105), `doomknight1`/`2` (fire 110, lightning 100), `councilmember1` (fire 120, others 100 -> immune per member differs), `megademon1-3` Balrog/Pit Lord/Venom Lord, Siege Beasts, Fire Boars, Hell Spawn (fire 110-120), Slayer.
- Cold immune: Oblivion Knight `doomknight3` (cold 180), Frozen Horror line (cold 150-190), Vile Mother/Child (cold 140-160), Trapped Soul (cold 150 and poison 90), Frozen Creeper..., Ghoul/Night/Dark/Blood Lord vampire line (cold 110-150), Siren/Hell Witch (cold 125-160), Ice Boar/Spawn (130-160).
- Lightning immune: Black/Burning Soul / Gloam wisps (100-115), `fingermage*` (Doom Caster, Strangler, Storm Caster: lightning 100), scarabs, Hollow One, Skeleton mages (lightning variants), Baboons Hell (fire 100-110 instead), Council Member 2/3.
- Poison immune: Skeleton archers, Mummies, Sand Maggots/queens, Mosquitos, Bone Fetish, Regurgitators, Tentacles, Defilers (Putrid/Wretched/Fetid/Rancid/Rank poison 100-120 Hell, V), Griswold (poison 120).
- Multiple: `megademon6` Pit Lord row: ALL five resists 145 in Hell (hcIdx 712, [84]); enabled in monstats but I have not found where it spawns (U: probably an Act 5 guest/Uber-adjacent row). `unraveler2/3`, `flyingscimitar` (fire+lightning 100+), `lightningspire` (lightning+poison).

Bosses (V tables, Hell resist [phys/magic/fire/ltng/cold/poison]):
- Andariel [66,0,-50,66,66,66]; Duriel [50,33,75,75,95,75]; Mephisto [20,50,75,75,75,75]; Diablo [45,0,50,50,50,50]; Baal [50,0,50,50,50,50]; Summoner [0,0,75,75,75,0]; Izual [30,30,30,30,75,30]; Blood Raven [50,50,50,50,50,50]; Nihlathak [0,25,33,33,70,70]; Griswold [50,0,0,50,0,120]. **No act boss is immune to a single element in 1.14d Hell.** Hell Mephisto/Duriel are NOT immune either; do not build a skip-boss rule on class immunity, build it on damage output.
- Superunique Lord De Seis (class `doomknight3` = Oblivion Knight): Hell resists ResMa 25, ResFi 60, ResLi 60, ResCo 180, ResPo 75 (V tables), so he and every Oblivion Knight are cold immune in Hell. The project note "Lord de Seis may be cold immune" is correct (V).
- Kaa the Soulless (`unraveler3`): magic 100 and cold 100 in Hell (V tables). Achmel the Cursed (`unraveler5`): magic 100, others 66. The Ancients (barbarians) have no monstats immunity (unraveler is the Horadrim Ancient mummy class). Dark Elder (`zombie5`) is magic immune in Hell (V tables).

### 3.1 Class workarounds (usual, U unless a class doc says otherwise)
Project class docs live in `docs/classes/*.md`; read them first. Generic workarounds (U, from memory, not re-verified here):
- Sorceress: cold immune -> Lightning/Fire; fire immune -> Lightning/Cold; lightning immune -> Fire/Cold; if two of three: Static Field + merc/Telekinesis not applicable, skip pack or Teleport past (our code: `fallbackSkill` Static Field, `PickImmune`, `IgnoreImmune`). Physical/magic immune monsters are not a problem for pure elementals.
- Necromancer: poison immune -> Bone Spear/Bone Spirit are magic damage (blocked by magic immune: unraveler lines/Dark Elder), Teeth is magic; Corpse Explosion is physical+fire (blocked by phys/fire); Lower Resist curse + Bone Spear is the usual mix. Many Hell Necro builds rely on a merc.
- Paladin: Fanaticism/Zeal-Smite physical melee is blocked by phys immune (wraiths, swarms, bloodlord3...) -> Holy Fire/Holy Freeze/Holy Shock auras or Vengeance (the physical-immune workaround in Arreat basics: Paladin Vengeance, Barbarian Berserk, elemental charms/weapons; V Arreat monsters/basics).
- Barbarian: Berserk (magic damage) for phys immune; magic immune is solved by elemental/war cry+hit.
- Amazon/Druid/Assassin: elemental damage choices; Druid Fire/Cold Hell builds hit the same single-element immunities as Sorceress.
- Merc: hirelings: Act 2 Desert mercenary with auras (Might/Prayer/Defiance) do physical only; Act 3 mercs elemental. A merc can finish an immune pack (project plan: "Static Field + merc is the plan for cold immunes", `combat/skills.go` comment).

D2R: Conviction/Lower Resist can break an immunity partially (pierce changes); I did not verify the 1.14d interaction with 100+ resists: keep U, do not rely on it.

## 4. Monster modifiers (champion / unique / minion)

Source: Arreat `monsters/bonus.shtml` (V, Blizzard primary, 1.13-era). monumod ids are in our code (`combat/target.go`: 5 strong, 6 fast, 7 curse, 8 resist (magic-resistant), 9 fire, 17 lightning, 18 cold, 25 manahit, 26 teleport, 27 spectralhit, 28 stoneskin, 29 multishot, 30 aura; the superuniques table above uses the same ids, V tables `superuniques.tsv` Mod1-3) and that the client Enchants byte array holds these same ids is U (D2BS).

| Modifier | Effect (V Arreat bonus.shtml) | Danger for the bot |
|---|---|---|
| Extra Strong (5) | Unique: +135% / +112% / +99% damage (Normal/NM/Hell), +90/75/66% attack rating. Minions +67/56/49% damage | high physical burst; melee builds die |
| Extra Fast (6) | +100 movement speed for Unique and minions | cannot be kited; closes at once |
| Cursed (7) | 75% chance to cast Amplify Damage level (mlvl/5)+1 when the unique hits you or its missile does | Amplify = +100% physical damage taken for the curse time; kill first or leave |
| Magic Resistant (8) | +40% fire, +40% cold, +40% lightning (each only if not already a third immunity) ("cannot be randomly selected in Normal") | creates cold/fire/lightning immunes from 60+ base resists |
| Fire Enchanted (9) | min/max fire damage +66% / +100%, **fire resist +75**, minions +33-50% fire damage in NM/Hell. Explodes on death: Normal 11.25-18.75% of base life in 2.67 yd, NM 7-11.66% in 3.33 yd, **Hell 0.75-1.25% in 4 yd (negligible)** | fire-immune in Hell for most bases; the old "never melee Fire Enchanted" rule only matters in Normal/NM |
| Lightning Enchanted (17) | lightning damage +66/+100%, **lightning resist +75**; when it is hit or put in hit recovery: 8 lightning bolts, level mlvl/2, damage (2*lvl)-1 each; leapers cannot spawn it | many hits -> many bolts: ranged builds trigger less when one-shotting; low LR dies |
| Cold Enchanted (18) | cold damage +66/+100%, cold resist +75, cold length 4+(mlvl*0.2) s; on death a level mlvl/2 cold nova | freezing slow; cold-immune for most Hell bases |
| Mana Burn (25) | mana damage +66/+100%, magic resist +20; difficulty resist penalty applies to mana damage (x1.4 NM, x2 Hell); **melee mana burn is multiplied by 256 due to a bug** | drains casters instantly; for Sorc/Pala/Necro: ranged kill only, never melee-tank |
| Teleportation (26) | teleports (the Teleport chance row was cut in the page) | closes at once, ignore kiting |
| Spectral Hit (27) | random Magic/Fire/Lightning/Cold/Poison hit +66/+100%, applies +40 to that resist when base <75; cold/poison length 1.6 s | damage type unknown ahead of time |
| Stone Skin (28) | +50% damage resist, base defense x2 | physical builds slow |
| Multiple Shots (29) | triples missiles (ranged uniques only) | dodge line of fire; kill first |
| Aura Enchanted (30) | one of Might (monlvl/6), Holy Fire (monlvl/6), Blessed Aim (monlvl/5), Holy Freeze (monlvl/7), Holy Shock (monlvl/8), Conviction (monlvl/8), Fanaticism (monlvl/8), all min level 1; Holy Shock needs mlvl >= 20; Fanaticism gives only the unique party +% damage | Conviction can drop player resists by tens of percent; Thorns is not in the list (Arreat names Might/HF/BA/HFreeze/HShock/Conviction/Fanatic only): the project danger-table JSON "Thorns" row is U |

Pack rules (V Arreat bonus.shtml): random champion/unique pack: 20% champion, 80% unique; unique spawns with 3-6 minions; champions in groups of 2-4 (all but one are Champion, last is Champion/Berserker/Fanatic/Ghostly/Possessed). Unique gets 1 extra mod in NM, 2 in Hell (V Arreat difficulty page; the superuniques.tsv Mod1-3 are fixed per superunique). Champion subtypes: Berserker (+270/225/198% damage, 1/4 champion life), Ghostly (+33-50% cold damage, phys immune display but 80% DR), Fanatic (+100 movement, -70% defense), Possessed (2x champion life, cannot be cursed).
Resist bonuses are only applied when they do not create a third immunity or raise an existing immunity further (V Arreat) -> a pack member can have at most two immunities from mods.

Named bosses/uniques that need special treatment:
- **Mind Flayer**: the string `Mind Flayer` does not exist in our install (V: grep of `strings.tsv` finds Flayer, Flayer Shaman, Flayer Dungeon, not "Mind Flayer"). The Act 5 Hell mage line is Finger Mage `fingermage1-5` named Doom Caster / Strangler / Storm Caster (Hell level 82-85, lightning immune 100, V tables). Where community pages say "Mind Flayer" they most likely mean the Flayer (`fetish3`) or Finger Mage line: U. Key the rule on the class ids and AI name, not the display string.
- **Oblivion Knight** `doomknight3` (hcIdx 312, Hell level 85, cold 180, fire/lightning 60, poison 75, V tables). Skill list not available in my tables dump (U: casts curse/lower-resist class spells; ranged pack with Doom Knight and Abyss Knight).
- **Stygian Doll** `fetish5` (hcIdx 145, Hell level 83, fire 100, V tables) plus variants `fetish7`/`fetishblow5`/`fetishblow7` (fire 100-110), shamans `fetishshaman5`/`7` (fire 125). Fetish dolls explode on death for fire damage: U from memory (also reason the project code has `explodeNames = doll/soul`). Soul Killers: lightning 110 (`fetish4`) / fire 115 (`fetish8`).
- Other dangerous classes that appear in the project's danger-table (kept there): Fallen/Carver shaman (resurrect), Unraveler/Horadrim Ancient (raise skeletons), Blood Raven/Bone Ash/Baal hydras, Thrashers etc. (existing JSON).

## 5. What the bot can detect today (compare with our code)

| Signal | Source | Status in our code |
|---|---|---|
| Monster class id -> base resists per difficulty, level, boss flag, AI string, hp/damage tables | `data.Monster` (monstats row, 1.14d-tables Level/ResFi.. per N/NM/H) | V (the table above was produced from the same tables) |
| Live monster resists incl. curses and difficulty | StatList at unit+0x5C, stat ids 36/37/39/41/43/45; `combat.refineResists` (cache 1.5 s); falls back to monstats | V (docs/combat.md: "Immunity: stat list resists when present, else monstats") |
| Immune per element | `Target.Immune` (>=100), `ImmuneAll`; skill pick `PickImmune`; `fallbackSkill` Static Field; ignore reason `immune`/`no-skill` | V (unit tests; no live immune pack met yet) |
| Champion / unique / minion flag | `MonsterData+0x16` bits (2 normal, 4 champ, 8 boss, 16 minion): `d2.MonsterInfo.IsChampion/IsUnique/IsMinion` | **U**: town NPCs only (all zero); no live champion read yet (docs/combat.md "no champion, unique or minion was met in 12 minutes") |
| Which modifiers a unique has | `MonsterData+0x1C` Enchants[9] -> `combat.Affix` via `affixByMod` (5,6,7,8,9,17,18,25,26,27,28,29,30) | **U** (ids from monumod.txt, V in the real file; that the client holds them is from D2BS) |
| Unique name | `MonsterData+0x2C` wide-string pointer | U |
| Which aura an Aura Enchanted unique has | not stored in the enchants list (comment in `target.go`) | not detectable that way; the Aura state is a child state on the monster stat list (like `ReadStates`), U which ids |
| Monster states (curses on the player/monster, minion mark) | `d2.MonsterInfo.States` / `ReadStates`; `StateMinion = 105` | V for the minion mark |
| Player resists, level, mana/life | `d2.Player.Resist`, `Level`, `Difficulty` (byte at RVA 0x3A060C) | stat ids and Hell/NM byte values U (Normal only seen); `docs/progression/difficulty.md` |
| Mana burn rate, curse on the player (Amplify Damage), poison/cold state | player state list / mana deltas | not built (U) |
| Missiles / ground effects (Andariel poison, fire walls, lightning bolts, hydras) | missile table | not built: `boss.Danger*` kinds exist as U, "new work" |
| Monster level for danger scaling | `Monster.Level[diff]` (area override via `levels.tsv` MonLvl) | data present; not used for a "mlvl - clvl" gap rule yet |
| PackDanger rating | `combat/resource.go`: boss or `Danger >= 1` -> 2, elite -> 1, else 0; never 3 | partial: affix-specific score missing; `resource/types.go` comment mentions "3 deadly (Hell uniques)" but nothing produces 3 |
| Hell avoid table | `avoidHell = Cursed | ManaBurn | LightningEnchanted | SpectralHit` when `ClassLvl < 75` and not a named boss | V code, V tests only |
| Kill-first | `killFirstAffixes = Cursed | ManaBurn | MultiShot | Aura`; names `shaman`, `oblivion`, `unraveler`, `bloodraven`; explode names `doll`, `soul` | V code |

Gaps vs section 4: Extra Fast, Extra Strong, Teleportation, Stone Skin, Fire/Cold Enchanted and Magic Resistant are parsed but get no rule. The Fire Enchanted explosion is negligible in Hell (V Arreat), so no avoid rule is needed there; Mana Burn melee is the nasty one.

## 6. Prioritised work list for d2-worker agents

Each item: what, the exact numbers, tag of the numbers. P1 first. All are offline-testable with `mem.Fake` and the real tables (`internal/data` skips when /tmp/d2mpq is absent).

### P1 Difficulty-aware entry gate (planner)
1. Make the resist gate use the install's real penalty: read `db.Difficulties[d].ResistPenalty` (0/-40/-100, V) instead of the -20/-50 note in `docs/progression/difficulty.md` G4. Gate: for the next difficulty, `min(fire, cold, lightning) + penalty(next) - penalty(current) > 0` (existing G4) and, for Hell Act 2+, `>= +25`; poison `>= 0` after penalty.
2. clvl floors: use section 2.2 (Nightmare 36 first area / 44 Act 2 / 50 Act 3 / 56 Act 4 / 58 Act 5; Hell 66 Act 1 wilderness, 75 Act 1 caves/crypts, 72 Act 2, 75 Act 3, 80 Act 4, 82 Act 5).
3. Replace the single "enter_clvl" per area by `monster level (levels.tsv Ex, V)` minus margin: Normal 2, Nightmare 5, Hell 8; keep Hell Cave 5/Crypt 3 at >= 75.

### P1 Immunity decision ("fight, kill with fallback, flee, skip")
4. Compute `immuneSet(t)` from `t.Res >= 100` (V). If the player's best attack skill and the fallback skill cannot damage the target (all damage types blocked), mark ignore reason `immune` and do not path through the pack. Keep the existing fallbacks (Static Field, merc).
5. Pack-level: if a pack member is immune to the bot's only element and `count(pack) >= 3` or it is a unique/champion, skip the pack (do not attack, do not chase). Never engage a pack with `ImmuneAll(...)` for all of the class's skills.
6. Boss rule: act bosses are not immune to any single element in Hell (V tables); only Lord De Seis (cold 180) and Kaa/Achmel (magic 100) are. Add class-specific boss exceptions rather than skipping bosses: cold-only builds skip Lord De Seis, magic-only builds skip Kaa the Soulless.
7. Mercenary/summon reliance: when the bot is blocked on a pack, `Fighter` should stand back and let the merc/pets finish (no self-pause; see nopause branch note).

### P1 Affix rules with exact numbers (once Enchants are verified live)
8. Run a one-off live census (`combatlive farm` prints `AFFIX ...`) in Nightmare/Hell to verify flag bits, enchants ids and unique name offset; until verified, keep affix rules gated behind `MonsterInfo.Enchants != 0`.
9. `Mana Burn`: in Hell never melee-tank; if mana loss > 20% of max mana in under 3 s, flee 8+ subtiles and drink mana (V melee bug x256 means one melee hit can empty the pool).
10. `Cursed`: Amplify Damage chance 75% on hit (V). If the player carries the Amplify state (state id U), treat all enemies as +100% damage: retreat or drink at life < 60% (U) and kill the Cursed unit first from range.
11. `Fire/Cold/Lightning Enchanted` raise the element's resist by 75: if base + 75 >= 100 for the bot's only element treat the unit as immune for that element (the live stat list already shows it, V code path) and prefer another element; Fire explosion in Hell is negligible, only handle it in Normal/NM (explosion radius 2.67 yd Normal, 3.33 yd NM).
12. `Lightning Enchanted`: 8 bolts of level mlvl/2 per hit; avoid melee hit chains in Hell unless LR >= 75 post-penalty; avoid when `clvl < 75` (current code, keep).
13. `Extra Fast` / `Teleportation` / `Extra Strong`: PackDanger 2 in NM/Hell for casters; stay near the portal; Extra Strong unique damage is +99% in Hell, +112% NM, +135% Normal (V): melee gate: only if `LifePct >= 70` and the merc is alive.
14. `Aura Enchanted`: the aura id is not in the enchants list; when a unit is `Aura` the dangerous ones are Conviction/Fanaticism/Might: treat as `KillFirst` (already) and PackDanger 2.
15. Produce PackDanger 3 (Hell uniques) in `combat.resourceInput`: any unit with `Cursed | ManaBurn | LightningEnchanted | SpectralHit | (Extra Strong & Extra Fast)` and `Hell` and `clvl < 75`, or any minion-of-unique pack with >= 5 live minions. Today the max is 2.

### P2 Specific monsters (class-id keyed, V tables)
16. `doomknight3` (Oblivion Knight, id 312): kill-first, ranged pack, expect a curse caster (U); cold immune: Sorc/Necro cold builds skip.
17. `fetish5`/`fetish7`/`fetishblow5`/`fetishblow7` Stygian Doll and `fetishshaman5/7`: keep distance >= 4 subtiles when they die, fire explosion U; fire immune.
18. Fingermages `fingermage1-5`: lightning immune, do not melee for U reasons.
19. Ghost/Wraith/Specter and `swarm1-5`: physical immune; melee/physical builds skip unless an elemental/magic skill is ready.
20. Horadrim Ancient/Unraveler (`unraveler2-9`, Hell 77-85): magic immune and they raise skeletons: treat as `KillFast` (already) and flag magic-only builds as blocked.
21. Pit Lord `megademon6` (all five resists 145): hard skip (V tables) if seen.
22. Poison clouds: Andariel, Defilers, Griswold are poison immune in Hell: Necro poison builds do not use Poison Nova there; boss script must not rely on poison.

### P2 Flee rules (resource layer, numbers R)
23. Hell, mlvl - clvl >= 10: do not leave town for the area at all (planner), `clvl >= mlvl - 8` otherwise (section 2.2).
24. Chicken: life < 33% (existing tuning), never below 25% in Hell; mana-burn chicken at mana < 15% twice (danger-table JSON).
25. Poison/cold-length: in Hell with poison < 25 post-penalty drink antidote on poison state (boss/hazard.go `PoisonStatus`), keep 2+ antidotes.

### P3 Data/telemetry
26. `d2doctor` check: read the difficulty byte and print the resist penalty, expected resists and the area monster level.
27. Death log: tag every death with `difficulty`, `mlvl - clvl`, and the unit's affixes (feeds `DeathWeights`).
28. Verify live (Hell/NM character needed): difficulty byte values, immunity stat shown on live Hell monster, champion/unique flags and Enchants, Amplify Damage state id, Aura child state ids.

## 7. Open questions

1. Are `MonsterData` flags/enchants in 1.14d exactly the D2BS layout (flags bit 2/3/4, Enchants at +0x1C)? Only town NPCs were available (U).
2. Which skills do Oblivion Knight, Doom Knight, Stygian Doll and Finger Mages use in 1.14d (not in my table dump; `skills`/`monstats` skill columns were not extracted into the cache)? U.
3. Does Conviction/Lower Resist bring a 100+ resist monster below immunity in 1.14d? Not verified.
4. Where `megademon6` (all-145 Pit Lord, level 84) spawns, and whether `ubermephisto` etc. matter to us (Uber rows are level 110, Pandemonium only).
5. Is the purediablo/Maxroll per-class immunity text worth fetching with the worker host headless browser for the class workaround section?

## Wiring (WIREUP)
`combat.EntryGate` is called from `Plan.EntryAllowed` (internal/bot/plan.go): `DifficultyFor` no longer offers Nightmare/Hell below the act 1 floor, and the planner (`target0`) excludes a picked step the gate refuses and re-picks. Area monster level is not passed (0): only the act floor and the Hell act 1 dungeon floor apply. Act 1 "dungeon" = area id > 7 and != 17 (S, levels.tsv ids from memory). The progression-nightmare.json Blood Moor enter level is now 36 (both copies).
Merc: `Runner.mercChore` (resource.go) hires in Lut Gholein when no merc was ever seen and gold >= `Bot.MercHireMinGold`; revives only with `Bot.MercReviveUnverified` (default off, U dialog). Needs a `bot.MercTown` on the Townsman (`cmd/d2bot/merc_adapter.go`). Hire still needs a RowPicker (U).

## Wiring (MISSILES)
New, not yet called by the fight loop (fight.go is not in this agent's directories):
- `combat.SetTables(db)` once at startup: resolves Amplify/Decrepify/Lower Resist/Weaken/Iron Maiden state ids by name from states.txt (S defaults 9/56/61/19/55 otherwise) and missile kinds by missiles.txt name (U substrings firewall/hydra/chaos/lightning).
- `resInput`: `in.PackDanger = CurseDanger(p, PackDanger(...))`; fight loop: `CurseRetreat(p, t.CurseRetreatPct)` -> same path as the Safety retreat/drink.
- `engage` next to the existing generic dodge: `moved, err := f.sidestepMissiles(ctx, snap, &dodgeCount)` (knob `combat.missileDodge`, default 0, max 6 per fight, 800 ms gap).
- `fight()`: `var ft FightTelemetry`, `ft.Sample(p)` beside `f.sampleLife`, and `f.info(ft.Finish(res))` before return (line `fight area= diff= danger= kills= damage= minlife= potions= dodges= sec=`).
- `d2.Reader.MissileUnits()` returns (nil,false) on a failed read; missile table itself is U live.

### Wired (FIGHTWIRE)
All four points above are wired: `combat.New` calls `SetTables` when `e.Data` is set; `resInput` uses `CurseDanger`; `Fighter.Safety` drinks (or opens a portal when the belt is empty) under `CurseRetreat` in a fight; `engage` calls `sidestepMissiles` after the generic dodge (counter `fightState.mdodges`); `fight()` logs `combat fight area=... sec=` once per fight via a defer (also on death/chicken). Offline tested only; knob `missileDodge` stays 0 (U live). Parser: `cmd/d2soak/fightlines.go` (`ParseFightLines`, `FightText`), not yet called from d2soak main (SOAKTOOLS).

## Wiring (DIFFP2)
New pure code, not yet called by the fight loop (fight.go/resource.go glue are not in this agent's directories). All numbers R (reasoned) unless noted; behaviour of the monsters is U until a live Hell run.
- `combat.PlanForTarget(t, class, diff, clvl) Plan` (rules.go): Kite false for Extra Fast/Teleportation; Mana Burn: kill first, keep 6 subtiles, keep 40% mana, NoMelee in NM/Hell (V: melee x256 bug) for classes with ManaNeed > 0; Cursed/Aura/MultiShot kill first; Lightning Enchanted: keep 8 subtiles, NoMelee in Hell below clvl 75; class-id rules `doomknight3` (kill first, range 5, U), `fetish5/7`, `fetishblow5/7`, `fetishshaman5/7` (range 4, U), `fingermage*` (NoMelee, U), `megademon6` (skip, V), `unraveler*` (kill first). Loop use: `KeepRange` -> evade when closer, `Kite` gates the kiting code, `Skip` -> Ignored reason, `ManaReserve` -> resource.Input economy line.
- `combat.PickByResist` (same signature as `PickAttack`): switches element when the preferred one does < 60% of the best alternative's damage factor ((100-res)/100, Stone Skin halves physical). `combat.ImmuneWorkaround(class, target)` lists class skills (names U) that avoid the blocked elements; Merc/Skip when none. `combat.ResistPenalty`, `PlayerResists.Effective`, `combat.ResistDanger(danger, targets, res, diff)` raises pack danger when an elemental affix meets an effective resist < 0.
- `resource.HellTuning`, `PlanForDiff` (Hell level 45+ uses the two-rejuv belt, `PlanBuys` now uses it via `Input.Diff`), `PotionPolicyHell(in, t, HellCtx)` (antidote verdict as Reason "antidote", earlier mana potion vs Mana Burn), `AntidoteNeeded`, `AntidotesToCarry`, `ManaBurnTracker.Observe(now, mana, maxMana, spent)` -> ManaFlee (> 20% of the pool in 3 s) / ManaChicken (< 15% twice).
- Telemetry: `DeathInfo.Line()`, `.LevelGap()`, `.Weights(monKey)`, `combat.AffixTags(ts)`, `d2.DoctorLine(p, areaMLvl)` (for d2doctor), `combat.LevelGapOK(diff, clvl, mlvl)` (P2 item 23).
Live checks still needed: (1) the Enchants ids really appear on a NM/Hell unique (item 8) and which of Stone Skin's resist is in the stat list; (2) `PlayerResists` source: does the stat list read include the difficulty penalty; (3) Oblivion Knight / Stygian Doll / Finger Mage behaviour and explosion radius; (4) Mana Burn drain rate on a live caster to calibrate 20%/3 s; (5) skill names in `classOptions` match skills.txt; (6) Amplify/antidote state ids via `SetTables`.

## Wiring status (WIREFARM)
Wired into the fight loop (internal/combat/hard.go, fight.go), all gated on Difficulty != Normal: `PlanForTarget` per engaged target (Skip, NoMelee -> ignore reason avoid, KeepRange -> back off, Kite gates the kiting block), `PickByResist` instead of `PickAttack`, `ImmuneWorkaround` logged on an immune target (the skills are not cast: they are not in the profile order), `ResistDanger` in `packDanger` (fight telemetry and resource Input.PackDanger), `ManaBurnTracker` per fight (flee / chicken, mana spent by our casts is subtracted), antidote drink when `AntidoteNeeded`, `PotionPolicyHell` for the mana potion in Safety. `Input.Diff` was already filled by resInput; `PlanBuys` already uses PlanForDiff. The bot logs `DeathInfo.Line()` at a Nightmare/Hell death (mlvl 0 and affixes none: the bot catalog has neither). Not wired: `ManaReserve` into the economy line, `AntidotesToCarry` into the town shopping list, `d2.DoctorLine` (d2doctor does not read a live character), `DeathInfo.Weights`/`LevelGapOK`.
