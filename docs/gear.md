# Gear and ammo model (agent GEAR, package `internal/gear`)

Tags: **V** verified (MPQ tables via `cmd/d2tables`, or live per `docs/classes/smoke-amazon.md`, or a guide page fetched 2026-10-29), **S** secondary, **U** unverified.
Closes smoke-amazon findings 1, 2, 3 (weapon/ammo model) and 5 (builds). No live game was used.

## Data (V, extracted at setup, not committed)
Extracted by `cmd/d2tables` into the tables cache (`gear.json`, `armor.json`, `stamp.json` with the source file sizes and sha256) from Patch_D2 > d2exp > d2data: skills.txt `itypea/etypea`, itemtypes.txt, weapons/misc/armor.txt (weapons, ammo, shields), charstats.txt. The extractor is a Go port of the former gen.py (V: on the worker host `D2_DATA_DIR=/tmp/d2mpq D2_LEGACY_DIR=<old json from git history> go test -run TestMatchesLegacyJSON ./internal/gear` showed every value of the old data.json and armor.json equal, 2026-10-30). `TestRealMatches` (`D2MPQ=<dir> go test`, default `/tmp/d2mpq`) compares it to `internal/data`.

- Skill weapon needs (V skills.txt itypea): Magic/Fire/Cold/Ice/Guided/Exploding/Immolation/Freezing Arrow, Multiple Shot, Strafe, Pierce need `miss` (bow, crossbow, Amazon bow). Poison Javelin, Lightning Bolt, Plague Javelin, Lightning Fury need `jave`. Jab, Power Strike, Impale, Charged Strike, Fend, Lightning Strike need `spea`. Poison Dagger needs `knif`. Sorceress and Necromancer spells: no requirement.
- V itemtypes: `jave` has Equiv2 `spea`, so a javelin satisfies the spear melee skills (Jab, Charged Strike...) but a spear does not satisfy javelin skills. U whether the game really lets Charged Strike use javelins (the data says yes).
- V: bows `shoots` bowq, crossbows xboq; Arrows `aqv` (stack 100..350, cost field 256) and Bolts `cqv` (60..250) are `PermStoreItem`; quivers have BodyLoc rarm/larm. V live: the quiver sits in the LEFT hand, body slot 5, replacing the Buckler (smoke-amazon). Quantity is item stat 70 (V live).
- V charstats: Amazon starts with a Javelin (rarm) and Buckler (larm), Necromancer Wand, Sorceress Short Staff... no bow. Amazon start 20 str 25 dex 20 vit 15 ene.
- V weapons.txt: bows have no Str requirement but a Dex one (Short Bow 15, Hunter's Bow 28, Stag Bow 45 + clvl 14 + str 30).

## API (`gear`)
```go
e := gear.Equipped{Weapon: "hbw", WeaponQty: 0, Offhand: "aqv", OffhandQty: 339} // base codes, slot 4 / slot 5, stat 70
gear.WeaponProblem("Amazon", e)  // "" or a reason: refuse to attack
ok, why := gear.SkillUsable("Multiple Shot", e)
gear.UsableSkills(ladder, e)      // filter a skill ladder
gear.AmmoIsLow(e); gear.LowAmmo(e) // U policy: quiver 20% (arrows 70, bolts 50), thrown stack 33% (javelin 20)
gear.Restock(e) []Purchase        // arrows/bolts/javelins to buy, "" when fine
gear.StartPurchases("Amazon","bow") // Short Bow + full Arrows (javelin path: Javelin stack)
gear.PickLauncher(stockCodes, class, str, dex, clvl) // best affordable-by-stats bow, ranked by base cost (V monotone with damage)
gear.VendorForAct(1)              // Charsi 154, tab "misc" (V live); other acts U
gear.StartItems(class), gear.ClassStart(class)
gear.Builds(), BuildJSON(name), DefaultBuild(class), MainAttackAt(build, clvl)
```
WeaponProblem rules: class-restricted item on the wrong class, launcher without a fitting quiver/ammo, empty thrown stack, Amazon with an empty main hand (U). Unknown base codes never refuse. Sorceress/Necromancer may use anything or nothing.

## What other packages must do (not done here, FRIENDLY2 owns them)
- `d2`/`mem`: read the main-hand and left-hand base codes and stat 70 for `Equipped`.
- `combat`: before casting call `SkillUsable(skill, equipped)` (drops Multiple Shot with a javelin, finding 2); when `WeaponProblem != ""` do not attack, report `no-weapon`/`no-ammo` and trigger a town trip; add a `no-effect` ignore reason.
- `town`: chore "restock": `Restock(e)` then `VendorForAct`; Charsi Misc tab (V live: tab click (361,75), Arrows cell (1,6), Bolts (2,0), 81 gold per full Arrows stack). Equip the quiver into slot 5 (shield slot). First shopping for a bow Amazon: `StartPurchases`.
- `leveler`: allow `miss` weapons for the Amazon (finding 1), never put a shield over the quiver, `leveler.buildFiles` entries for the four builds below (copy `internal/gear/builds/*.json` into `internal/plandata` and docs/plan/data, or read `gear.BuildJSON`); `DefaultBuildName`: amazon -> `amazon_bow_normal30`, necromancer -> `necromancer_summoner_normal30` (`gear.DefaultBuild`).

## Builds (`internal/gear/builds/*.json`, same schema as `build-paladin-normal30.json`, plus `main_attack`)
`gen_builds.py` writes them and asserts prerequisites, skill reqlevels, 5 stat points and max 20 per row against skills.txt; `TestBuilds` repeats the checks in Go.
| file | source |
|---|---|
| `necromancer_leveling_normal30` | V Maxroll necromancer-leveling (Teeth 2-5/7-10/13, Clay Golem 6, Amplify 11, Iron Maiden 12, Bone Armor 14, Bone Wall 15-16, Corpse Explosion 17+, Bone Spear 18-20; stats 2-3 Energy, Vit, Str 12-13 and 17-21, Vit); respec at 27 recorded, disabled |
| `necromancer_summoner_normal30` | U own order (warriors, Clay Golem 6, mages 12+, Bone Armor 14, Corpse Explosion 15+); end state direction V Maxroll summoner guide (Raise Skeleton, Skeleton Mastery, Corpse Explosion maxed); default for the Necromancer |
| `amazon_bow_normal30` | V Maxroll amazon-leveling (Fire Arrow, Multiple Shot 11, Exploding Arrow 12+, stats), U adaptation: Poison Javelin points 6-10 become Fire Arrow because the bot has no weapon swap; default for the Amazon |
| `amazon_javelin_normal30` | U own design (Poison Javelin to 12, Lightning Bolt from 14) |
| `druid_fire_normal30` | V Maxroll druid-leveling (Firestorm, Molten Boulder, Fissure from 12); U Oak Sage 16, Arctic Blast 17, Cyclone Armor 18; default for the Druid |
| `assassin_fireblast_normal30` | V Maxroll assassin-leveling (Fire Blast, Claw Mastery, Burst of Speed 6, Wake of Fire 12+); default for the Assassin; no weapon needed |
Maxroll guide pages gave no Hell/clvl 31+ tables here; rows stop at clvl 30 like the Paladin build. Quest points (Den, Radament, Izual) are folded into rows 5, 15, 21 (Necromancer) because the planner pulls rows forward. Whether the start skill (Raise Skeleton for the Necromancer) already has a level is U.

## Dead ends / not verified live
- Quiver slot 5 and stat 70 are V from smoke-amazon, not re-checked by this agent. Vendors other than Charsi, javelin sale tab, javelin recovery from the ground, crossbow quiver fit in practice: U.
- Bow damage is not in `internal/data` (bows read dam 0-0), so `PickLauncher` uses cost.
- Live checklist: (1) Charsi stock: does Misc show Javelin stacks? (2) With a Javelin worn, cast Charged Strike on an adjacent monster: does it deal damage? (3) Equip a Crossbow and Arrows: does the game refuse the cast? (4) Pick up thrown javelins and see if the stack grows.

## GEARWIRE: what is wired (agent/gearwire, offline only)
- combat (`internal/combat/gearcheck.go`): `noteGear` runs each fight tick, reads base codes and stat 70 of slots 4/5, logs `weapon problem, not attacking` once per distinct reason; `attackSkills` drops skills with `gear.SkillUsable == false`, returns nothing on a `WeaponProblem`, and promotes `gear.MainAttackAt(Fighter.GearBuild, clvl)` for an Amazon. `Fighter.WeaponProblem()`, `AmmoLow()` for callers. Fake tests: `TestGearFilterAmazon`.
- town (`internal/town/ammo.go`): chore step `ammo` (`ChorePlan.Ammo`, set by cmd/d2bot), pure `AmmoPlan`, buys at `gear.VendorForAct` (Charsi class 154, V live per smoke-amazon), then wears bow (slot rArm) and quiver (lArm) with Take + slot click like leveler.doSwap. `NeedsTrip` reason "ammo low" only for a worn launcher. `SetAmmoPath("bow"|"javelin")` from the leveler build name in cmd/d2bot. Tests: `TestAmmoPlan` (pure); `TestAmmoChoreStartFlow` needs /tmp/d2mpq (not run on the head box, U).
- leveler/plandata: the 4 builds are copied to `internal/plandata/build-<name>.json` and `docs/plan/data/` (TestMatchesDocs), selectable by name, `DefaultBuildName` falls back to `gear.DefaultBuild` (amazon_bow_normal30, necromancer_summoner_normal30). `ChooseGear` never swaps away a worn bow or quiver (`Cand.Ranged`).
- Necromancer: combat already runs `Curse`, `KeepSummons`, Bone Armor via `necroTick`/`Buff`; nothing more wired. U live.

### Live checklist (U, d2-live Amazon, then Necromancer)
Amazon (fresh, Javelin + Buckler, >= 300 gold, config levelerBuild empty so the bow build is the default):
1. Start the bot in town: log "class-aware start ... levelerBuild amazon_bow_normal30". Chores run `ammo`: Charsi Misc tab, Short Bow then Arrows bought (about 100 + 81 gold), both worn: slot 4 = Short Bow, slot 5 = quiver (Buckler and Javelin go to the pack, cursor empty). If the slot click fails, log shows "did not reach slot".
2. Read slot 4/5 codes and stat 70 (tools/dumpitems or combat log). Fight 10 monsters: log must show Fire Arrow / Magic Arrow, never Jab; quiver count drops about 0.7 per kill.
3. Set the quiver low (spend arrows or cheat on Amazon throwaway only via d2cheat): "ammo low" trip reason, Charsi sells a full stack, quiver refilled and still worn.
4. Empty the quiver in the field: one "weapon problem, not attacking" log, no attack clicks, town trip.
5. Javelin path (levelerBuild amazon_javelin_normal30): Javelin stack kept, no bow bought; the open items 1 to 4 of the dead-end list above (javelin sale, recovery).
6. Leveler Equip with a bow and quiver worn and a better shield or sword in the pack: no swap planned.
Necromancer (levelerBuild empty = summoner): "levelerBuild necromancer_summoner_normal30"; skill points follow the table (Raise Skeleton, Clay Golem at 6, Bone Armor 14); in a fight Amplify Damage then Raise Skeleton from a corpse, Teeth as attack; confirm hotkeys for the new skills bind (combat Layout f1..f7 are fixed names, the build file's hotkeys are not checked).

### Necromancer GEARWIRE live results (necrolive, Necki i2, 2026-10-29)
V: levelerBuild `necromancer_summoner_normal30` chosen by default; skill points follow the table (13 points at clvl 14 spent as rows 2-14); hotkeys bind for learned skills from the combat layout (not the build file's F-keys); Amplify Damage cast at packs; Raise Skeleton / Skeletal Mage / Clay Golem raised and counted; Player.States shows Bone Armor state 14. Details and fixes: docs/summon.md "Live results". Teeth was never the attack in these runs (the army and golem kill first; U).
### Amazon live results (AMALIVE, i3 "Amazi", 2026-10-29)
- V check 1 (start): log "class-aware start ... levelerBuild amazon_bow_normal30". Ammo chore bought Short Bow (Charsi tab 2) and Arrows (Misc tab, 81 gold, quantity 350), bow ends in slot 4 (rarm), quiver in slot 5, Buckler and Javelin in the pack, cursor empty. Stat 70 of the quiver = 350 (V).
- V bugs found and fixed: (a) `Town.Stock` is read right after the trade window opens and `t.pages` could be incomplete, so the tab click was silently skipped: Short Bow purchase failed ("gold did not change") and Arrows bought a Large Axe off the remembered Weapons tab. Pages are now rebuilt from every `Stock` read; tab settle raised to 600 ms (`tabSettleMs`). (b) chore order: `shop` (TP tome 450 gold) ate the budget before `ammo`; `ammo` now runs before `shop`. Manual right-click at the right cell always bought (bow 93, arrows 81 gold at clvl 3).
- Vendor tabs V: the game reopens Charsi on the last tab used, not the first; Armor, Weapons, Weapons (bows, crossbows), Misc (arrows (1,3), bolts (1,6)); a right-click buy of a quiver auto-equips it into the free left-hand slot.
- V check 2: 19 kills in Blood Moor with the bow worn, quiver 350 -> 323 (about 1.4 arrows per kill), no "weapon problem" lines, leveler took Magic Arrow at clvl 2 and Fire Arrow at 3 (hotkeys f1/f2 bound).
- V check 6: with a Large Axe, Axe and Dagger in the pack across several leveler runs the bow and quiver stayed worn.
- U checks 3 (ammo low trip), 4 (empty quiver) and 5 (javelin path): not run. The quiver count cannot be lowered by cheat (item stat, d2cheat only edits player stats) and a raw left-click on the worn quiver in the inventory panel did not pick it up (hover only).
- Cheat note: a fresh character has no gold stat (d2cheat gold fails "stat not present") until it holds 1 gold; "Amazi" was added to `cheat.Allowed`.
- Dev note: `townlive chores ammo|javelin`, `townlive shop 154 tab=N rclick=X,Y` probes.

## Endgame tables (ENDBUILD)

Seven clvl 2..90 tables from `docs/builds-endgame.md` are registered (V: `go test ./internal/leveler ./internal/gear`): the sources are `docs/plan/data/builds-endgame/` (generator `gen_endgame.py`), copies in `internal/plandata/build-<name>.json` and `internal/gear/builds/<name>.json` (copy with `cp`, the generator is the source of truth). Names: `sorceress_blizzard_endgame90`, `paladin_hammerdin_endgame90`, `paladin_hammerdin_zeal_endgame90` (variant, below), `necromancer_summoner_endgame90`, `amazon_multishot_endgame90`, `druid_tornado_endgame90`, `assassin_sentry_endgame90`, `barbarian_warcry_endgame90`.

- Selection (W1/W2): opt-in only. Config `levelerBuild` / `leveler.Options.Build` takes any table name or the alias `endgame` (`leveler.ResolveBuild`, resolved per class in `leveler.buildFor` and `cmd/d2bot/class.go`). With no knob the defaults are exactly the old Normal-30 tables (`TestGearBuildsLoadAndDefault`, `TestEndgameTables`). New characters pick the endgame table from clvl 2 (no respec); there is no mid-game switch for an existing character (the lines differ, e.g. Charged Bolt vs Ice).
- Sorceress (W2): the data file for `sorceress_blizzard_endgame90` says `respec.enabled=false` and spends the Ice line from clvl 2 (V: file), so it needs no respec at 26; that makes it the endgame choice and the Lightning "fallback" table (stops at clvl 30, cannot be continued) is NOT. The combat build stays Blizzard (`planForClass`, only `fallback` selects Lightning). Hell Blizzard needs kiting (builds-endgame.md Q6, U), so a Sorceress that dies in Hell has no ready sibling table.
- Banked points: rows with no skills are saved points; the unlock row spends the bank (V `TestEndgamePaladinPlanSpendsBank`: Paladin row 18 spends 11 hammer points, rows 4-5 plan nothing).
- Name fix (V `gear.TestBuilds`): hotkeys must use skills.txt names, so Decoy is `Dopplezon` and Iron Golem is `IronGolem`.
- `gear.SkillUsable` (V gear.json: Smite itypea1=shld): a shield-only skill now checks the off hand instead of the weapon, so Smite is usable with a sword and a shield (it was rejected for every weapon before).
- Paladin clvl 2-17 (W4, V from the table): the plain table has exactly 1 Smite point (clvl 3) plus Might, Holy Bolt and Charge; the rest is banked until 18. The variant `paladin_hammerdin_zeal_endgame90` (U) takes Smite to 9 by clvl 11, Sacrifice 1 at 13 and Zeal 5 at 14-17 (Zeal needs Sacrifice and clvl 12, V skills.txt) and pays for it from the last fillers (Blessed Aim 17 instead of 20, Defiance 1), Blessed Hammer still starts at 18. Which of the two clears Normal Act 1-2 is U: use the variant if a live test shows the plain one cannot.
- Mana (W7, U): `docs/plan/data/potions-belt.json` has a `mana_heavy_builds` block (two mp columns, 35% drink threshold) for the War Cry, Blizzard, Tornado and Summoner builds. GATEWIRE wired it: `resource.PlanForBuild` / `Input.Build` (set from `leveler.ResolveBuild(levelerBuild, class)` in `bot.resInput`) use the mana-heavy columns, `ManaNeed` 2 (also for the Barbarian) and the 35% drink line in Nightmare/Hell only (V: `go test ./internal/resource`, the Go table is checked against the json). Not wired: `internal/town/belt` `PlanForClass` still calls `resource.PlanFor`, so the town shopping layout does not yet follow (owner of internal/town).
- Resist gate (W8): `leveler.ResistGate(target, resists)` (pure): Nightmare needs fire/cold/lightning >= +20 and Hell >= +25 after the target's penalty (-40/-100), poison >= 0, final value capped at 75. The input is the raw value without the difficulty penalty (whether the memory read contains it is U, difficulty.md live check 2). GATEWIRE wired it (V: `go test ./internal/bot ./internal/task`): `Plan.EntryAllowed(diff, level, area, res *d2.Resists)`, `Plan.DifficultyFor(level, margin, res)` and `task.ReadyForNext(next, clvl, res)`; nil resists skip the gate. `bot.wantedDifficulty` and `cmd/d2bot taskChain` pass `leveler.RawResists(player)`; the knob `leveler.ReadContainsPenalty` (default false, U) is the only place that assumes whether the read value has the penalty. The planner itself passes nil for the difficulty it already stands in (refusing there would only pause it). A refused hop = keep farming; in the goal chain a refusal with the level met queues a reach_level goal for one more level (`goals/chain.go`).
- Death telemetry (W10): every death (all difficulties) logs `death diff= area= clvl= mlvl= gap= killer= class= boss= affixes= dist= build=`; mlvl is the monstats level for the difficulty (area overrides U), affixes from the killer unit. `cmd/d2soak/deathlines.go` parses and summarises it (not yet called from d2soak main).
- Mana (W7, U): `docs/plan/data/potions-belt.json` has a `mana_heavy_builds` block (two mp columns, 35% drink threshold) for the War Cry, Blizzard, Tornado and Summoner builds. Nothing reads it yet (the combat tuning in `internal/env/tuning_combat.go` has the single set); wiring is for the owner of internal/env.
- Resist gate (W8): `leveler.ResistGate(target, resists)` (pure): Nightmare needs fire/cold/lightning >= +20 and Hell >= +25 after the target's penalty (-40/-100), poison >= 0, final value capped at 75. The input is the raw value without the difficulty penalty (whether the memory read contains it is U, difficulty.md live check 2). It is not wired: `Plan.EntryAllowed` (internal/bot) and `task.ReadyForNext` have no resists in their signatures, so a hook needs a signature change (see report).

## Armor table and Str/Dex gating (agent ARMORDATA)

- V `armor.json` (d2tables cache): all 202 armor.txt rows (body armor, helms, circlets, shields, belts, boots, gloves, class helms and shields): type, tier (normcode/ubercode/ultracode), min/max AC, req Str, level req, speed, block, durability, max sockets, size. armor.txt has no reqdex column (V header), so armor never needs Dex; claws do (weapons.txt reqdex = reqstr, V).
- Regenerate: `go run ./cmd/d2tables --data <MPQ dir>` (automatic on first use when `D2_DATA_DIR` is set). Tests use made-up rows (`fixture_test.go`); tests that assert real values skip without tables; `TestRealMatches` (the worker host) compares every row with `internal/data`.
- API: `gear.ArmorByCode`, `ArmorCodes`, `Requirement(code)`, `CanEquip(code, str, dex, clvl)`, `Milestones(build)`, `StatNeeds(build, clvl)`. The leveler already refuses unmet requirements in `leveler.Candidate` (via `internal/data`, same table); `TestCandidateRefusesUnmetStrDex` checks every armor row (the worker host).
- Key bases (V str / level req / sockets): Mage Plate 55/25/3, Dusk Shroud 77/49/4, Archon Plate 103/63/4, Sacred Armor 232/66/4, Sacred Targe 86/47/4, Monarch 156/54/4, Large Shield 34/0/3, Heraldic Shield 40/12/4, Quhab (claw) 57 Str and 57 Dex.
- `internal/gear/milestones.json` lists the planned gear per endgame build (U choice and deadlines); `gen_endgame.py` derives the Str/Dex stages from it and `TestMilestoneStats` checks the build rows reach them. Found: the generator spent surplus points on Vitality so Amazon Dex 110 by clvl 35 was unreachable; it now looks ahead (feasibility, not a plan change). Results: Sorc/Druid/Necro Str 55 (was 65/65/40), Assassin Str 57 and Dex 57 (was Str 60, Dex 0, claws unusable), Barb Str 77 (was 80), Paladin Str 43 by 25 then 86.
- U: `speed` and `str_bonus` semantics; the item level caps sockets below max_sockets; Maxroll unique names (Skin of the Vipermagi, Arachnid Mesh) not modelled.
