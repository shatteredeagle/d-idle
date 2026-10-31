# 02 Boss and area runs

Common run skeleton (KB:scripts/*.js, `Loader`):
```
run(script):
  skip if bossid already killed this game (Attack._killed set) or quest-complete rule says skip
  go to town first unless startArea == current area (skipTown list)
  travel: waypoint to startArea (or journeyTo if no WP), precast buffs (force=true)
  moveToExit chain / moveToPreset(...) to boss position
  kill boss (Attack.kill / Attack.clear with boss id)
  pickit; optional chest opening; return true
on any non-script error: log, screenshot, continue with next script (never abort the game loop)
```
Between scripts the Loader does town chores only when the next script is not starting in the current area.
Scripts end when: boss dead (monster mode dead / unit invalid and recorded in killed-set), attack count cap (300 per kill, `MaxAttackCount`; SP uses 1000), or a hard error.
Mapping: `internal/bot/runs/*.go` each a struct with Name, StartArea, BossID, Steps; needs from `internal/d2map`: exit list per area (target area id, position, type walk/stairs), preset units (object/monster by txt id with room coords), WP objects; from `internal/d2`: monster list with classid/name/mode/spec flags, object list with mode.

## Quick table (world coordinates; KB unless noted)
| run | WP / start | route | stand / approach | detect and end |
|---|---|---|---|---|
| Andariel | Catacombs 2 WP | exit to Catacombs 3 then 4 (moveToExit chain) | move to (22549,9520); KO uses (22561,9553); SP walkers path (22572,9635),(22554,9618),(22542,9600),(22572,9582),(22554,9566) clearing Dark Shamans, teleporters go (22571,9590). Classic sorc dance: when boss within 10 step to x=22535 or 22560 at y=9520 alternating | kill classid Andariel; wait 2 s for minions; pickup. Poison res < 75 : buy antidotes first (SP) |
| Duriel | Canyon of the Magic WP | exit to correct Tal Rasha tomb (`getRoom().correcttomb`; the one with the Orifice object); move to HoradricStaffHolder preset offset (-11,+3) until Duriel's Lair portal object visible within 20; use portal (telekinesis if available, else walk/click) | classic sorc: while boss <=10 away step to (22638, 15722 or 15693) depending on side | Lair area; kill classid Duriel; if portal fails: clear 10, retry |
| Mephisto | Durance of Hate 2 WP | exit to Durance 3 | stand (17566,8069) (KO safe spot (17568,8069)); moat trick for teleport sorc: path (17563,8072) -> (17575,8086)->(17584,8091)->(17600,8095)->(17610,8094) with 350..2500 ms waits, lure until Mephisto >27 away, then cast from the moat; melee/other: when Mephisto mode == Attacking2 (his ranged cast) sidestep to a spot at radius 18 around him at angle +-30/60/90 deg | kill Mephisto; optional red portal at (17590,8068) then (17601,8070); chests at (17572,8011),(17572,8125),(17515,8061); council killing at (17600,8125),(17600,8015),(17643,8068) |
| Diablo | River of Flame WP | exit to Chaos Sanctuary; entrance (7790,5544) [KO (7796,5561)]; star at (7791,5293) [KO spawn (7800,5286)] | seals via precomputed node paths (below); wait <= 30 s at star after last seal (8 s first, then pre-cast); Diablo "lights" packet 0x89 0x0C = spawned (SP/KB event; for us, poll monster list for classid Diablo) | kill Diablo; throw if not found after 30 s then recheck seals |
| Baal | Worldstone Keep 2 WP | exit to WSK3 then Throne; move to (15095,5029) | clear throne; waves: see below | after waves, wait for Baal throne monster gone, take portal to Worldstone Chamber, move (15134,5923), kill |
| Countess | Black Marsh WP | exit chain Forgotten Tower -> Tower Cellar 1..5 (levels 1-5) | find SuperChest preset in level 5: x = roomx*5+x. If 12565 stand (12578,11043); if 12526 stand (12548,11083) | `Attack.clear(20, boss=Countess)`; her chests excluded from opening (unit.x 12526 or 12565) |
| Summoner | Arcane Sanctuary WP | move to Journal preset offset (-3,-3) | clear 15 around boss | red portal appears after kill; else click Journal up to 5 times then portal to Canyon |
| Radament | Sewers 2 WP | exit to Sewers 3, then preset HoradricScrollChest | kill classid Radament | open chests radius 20 |
| Pindleskin | town A5, red portal (5130,5120 region) | via Anya's portal or WP Halls of Pain + exits | stand (10058,13234..13236) | kill super unique; `Attack.haveKilled` skips repeat |
| Nihlathak | Halls of Pain WP | exit Halls of Vaught, move to NihlathaksPlatform preset (+10,+10 optional) | KO: stop 40 away from `NihlathakWildernessStartPosition` object then choose the corner at +-13 with the largest sum of distances to enemies | abort if Tomb Vipers (revived) present (ViperQuit) |
| Eldritch/Shenk | Frigid Highlands WP | Eldritch stand (3745,5084); Shenk (3876,5130) in Bloody Foothills; Dac Farren (4478,5108) | clear boss | chest (large sparkly) after Eldritch optional |
| Cows | Rogue Enc stash | needs Wirt's Leg (Tristram portal from Stony Field, Wirt corpse at (25048,5177)) + TP book in Cube -> Cow portal near stash | Cow level: see below | king preset id; `Attack.clearLevel` if kill king else room sweep avoiding "bad rooms" |
| Tristram | Stony Field | Rakanishu preset, kill, wait for Tristram portal (<=1 min retries), red portal in Tristram, stand at portal y+6, then clearLevel | needs Cain quest (Inifuss tree -> scroll) | |
| Izual | City of the Damned WP | moveToPresetMonster Izual in Plains of Despair | kill | |
| Hephasto | River of Flame WP | HellForge preset | kill, optional clear river | |
| Travincal | Travincal WP | start (orgX,orgY)= WP position; councils at (+101,-56); leapers use (+60,-53),(+64,-72),(+78,-72),(+74,-88) | clear 3 council classids | |
| Bishibosh/Corpsefire/Rakanishu/etc. | cold plains WP etc. | moveToPresetMonster id | `Attack.clear(15, boss)` | |
| Mausoleum, Pit, Crypt, Tombs, Maggot Lair | clearLevel scripts | exits chain | clear everything with ClearType | |
| ChestMania | per act list | journey to each area, open all chests in area | | |
| Smith/Cube/Staff/Amulet | see 01 | | | |
(KO equivalents: `KO:internal/run/*.go`; Baal throne position (15095,5042), pre-portal safe (15116,5071).)

## Diablo / Chaos Sanctuary in detail (KB:core/Common/Diablo.js)
Layout: each of 3 seals has 2 layouts (A or B), detected by comparing the seal preset coordinate to a constant: Vizier seal preset y == 5275 -> layout 1 else 2; Seis x == 7773 -> 1 else 2; Infector x == 7893 -> 1 else 2.
Waypoint paths (nodes in order, each node: move, clear radius 30 with a custom sort, record as "cleared"):
- entrance -> star: (7794,5517),(7791,5491),(7768,5459),(7775,5424),(7817,5458),(7777,5408),(7769,5379),(7777,5357),(7809,5359),(7805,5330),(7780,5317),(7791,5293).
- star -> Vizier A: (7759,5295),(7734,5295),(7716,5295),(7718,5276),(7697,5292),(7678,5293),(7665,5276),(7662,5314). B: (7759,5295),(7734,5295),(7716,5295),(7701,5315),(7666,5313),(7653,5284).
- star -> Seis A: (7781,5259),(7805,5258),(7802,5237),(7776,5228),(7775,5205),(7804,5193),(7814,5169),(7788,5153). B: (7781,5259),(7805,5258),(7802,5237),(7776,5228),(7811,5218),(7807,5194),(7779,5193),(7774,5160),(7803,5154).
- star -> Infector A: (7809,5268),(7834,5306),(7852,5280),(7852,5310),(7869,5294),(7895,5295),(7919,5290). B: (7809,5268),(7834,5306),(7852,5280),(7852,5310),(7869,5294),(7895,5274),(7927,5275),(7932,5297),(7923,5313).
- Boss fight spots: Vizier A (7691,5292) B (7695,5316); Seis A (7798,5194) B (7796,5155); Infector A (7876,5296; casters only) B (7928,5295). Pre-attack cast targets (area spells): Viz A (7676,5295) B (7684,5318); Seis A (7778,5216) B (7775,5208); Inf A (7913,5292) B (7915,5280).
- Seal-open handling: up to 5 tries; with telekinesis stand 15-18 away; otherwise walk to the seal (+2,0; Seis +5,+5) and interact; wait 500 ms+ping (Seis 1000+ping); if >1 monster nearby on attempt >1 or infector seal, clear 15 first. A seal is open when its object mode != 0. Infector has two seals (open #1, kill Infector, open #2 unless "fast").
- Boss detection: after opening a seal the unique spawns; poll monster list by locale name up to 16 x 250 ms; if not found maybe wait for "seal glow" object (3 min cap). Seal defender classes: Oblivion Knight, Venom Lord, Storm Caster (KO uses these three as "seal elites").
- Order: Vizier, Seis, Infector (SealOrder config). KO opens seals in object order 4,5,3,2,1 with clear radius 15 and a 5 s wait for the elite pack, then returns to town for potions if belt low.
- `diaSpawnWatcher` thread: every 250 ms checks "Diablo spawned" flag and aborts the seal logic with a ScriptError so the fight starts immediately. For us: monster list scan each tick.
- Dangerous: fire-spewing "fire" objects: `getMonsterCount` adds +100 per fire object within 4 of a candidate spot to avoid standing in flames.
- Wait position for casters before Diablo appears: moveNear (7791,5293) within 35 (sorc), 25 (other casters), or (7788,5292) exact for melee.

## Baal waves (KB:core/Common/Baal.js)
Throne rectangle: x 15072..15118, y y<5080 means "inside throne" (bottom entrance archway at (15097,5099)). Corners: BL (15072,5073) BR (15118,5073) TL (15072,5002) TR (15118,5002); Baal at (15090,5014); wave 5 position (15094,5029); sorc positions: (15116,5026) default, (15106,5040) with Frozen Orb before wave 4.
Wave identification by classid of any attackable monster inside throne: 1 = Warped Fallen/Shaman, 2 = Baal Subject Mummy/Cold Mage (Mummies), 3 = Council member (Council4) plus Hydra check, 4 = Venom Lord (VenomLord2), 5 = Lister the Tormenter + minions.
Loop (`clearWaves`): while throne Baal ("ThroneBaal" classid) exists: identify wave, `clearClassids` for that wave; idle: cleansing if poisoned (paladin), pre-attack at (15094+-2, 5024+-2) with sorc area spell, every 20 s `clearThrone` (8 nodes (15097,5054),(15079,5014),(15085,5053),(15085,5040),(15098,5040),(15099,5022),(15086,5024),(15079,5014) skipping nodes with 0 monsters within 30). Hydras (Council wave) not owned by player -> move to the nearest corner and wait until the hydra expires. Total timeout 30 min.
After waves: precast, wait <= 3 min for throne Baal gone, `clearThrone` if hit; step to (15090,5008); portal object WorldstonePortal; in chamber move (15134,5923) and kill Baal (`hurt` variant stops at a life percent for "leech" use).
Quit conditions: Souls (Burning Soul) or Dolls (Soul Killer / "doll" explosions) present on entry -> leave game (config DollQuit/SoulQuit) for chars not immune.

## Cow level (KB Common/Cows.js)
`buildCowRooms`: take preset Cow King position, mark the room and rooms adjacent (two rings) as "bad" (king pack); all other rooms become sweep targets sorted by distance with "seen" rooms last. Teleporting chars hop room-center to room-center (nearest walkable within 10); walkers check that the A* path nodes do not enter bad rooms; if so re-queue that room once. Each stop `Attack.clear(30)`. Failure if clear returns false.
Prereqs (all Normal difficulty cleared, Tristram quest done, never killed King this difficulty): Wirt's leg from Wirt body; buy second TP tome if only one; put leg + tome in Cube -> portal near stash within 10 x 200 ms polls.

## Chest and shrine runs
- Chest list (all container object names): `chest, loose rock, hidden stash, loose boulder, corpseonstick, casket, armorstand, weaponrack, barrel, holeanim, tomb2/3, roguecorpse, ratnest, corpse, goo pile, largeurn, urn, chest3, jug, skeleton, guardcorpse, sarcophagus, object2, cocoon, basket, stash, hollow log, hungskeleton, pillar, skullpile, jar1-3, bonechest, woodchestl/r, barrel wilderness, burialchestr/l, explodingchest, chestl/r, groundtomb, icecavejar1-4, deadperson(2), evilurn, tomb1l, tomb3l, groundtombl` (KB:core/Misc.js:12-24). Defaults to open: chest, chest3, armorstand, weaponrack, range 15 while pathing (Config.OpenChests).
- Open chest: skip opened (mode != 0); locked chest and no key (non-assassin) skip; up to 7 tries: telekinesis for first 3 if > 5 away, else walk to (x+1,y+2) within 3 then interact packet (first 3) / click (later and for quest chests); success = unit mode flips within 1 s; on failure stop walking. Rebuild the list after every 5 chests when moved > 5 from start.
- Shrines (`Misc.shriner`): wanted rules: Health if hp<80 (or <=50 and far for walkers: walk distance > 10); Mana same with mp; Refilling < 85/50; Experience always; Skill if no active shrine buff; Resist shrines when resist low; Armor/Combat/Monster when within 15 walk; Gem shrine only if a gem in inventory (prepareForGemShrine); Poison/Exploding only when gold < LowGold; never overwrite an Experience shrine; keep the last shrine state until <30 s remain. Walk-distance cutoffs (non-teleporters): 10 (heal), 15 (most), 30 (stamina if low). Wells: use when hp<75 mp<75 stamina<50 (autoshrine) or status effects Frozen, Poison, Amplify, Decrepify.
- Shrine detection needs the object's `objtype` (shrine class) - for us: object txt id + shrine state field in the object data (memory doc 2.5 objects). UNVERIFIED which field.

## What kills runs (failure modes the code defends against)
1. Lost target reference: bosses re-spawn as new units when Necro raises them; code re-finds by gid then by walking to last location (`Attack.kill`).
2. Standing in boss ranged attacks (Mephisto "Attacking2" mode -> sidestep; Andariel poison cloud: antidotes + stepping 10; Duriel hits at close range: sorc steps to fixed spots).
3. Wrong layout: Chaos seals A/B, Countess chest x, Duriel tomb; handled with preset lookups instead of fixed coordinates.
4. Boss not found (late spawn): polling loops 16 x 250 ms; Diablo 30 s; Baal waves 30 min; then error and next script.
5. Pack blocking the exit/portal: clear 10-15 radius then retry; if still failing leave game.
6. Seal elites killing a weak char: wait for the elite pack at the seal, clear in a radius 15..40 with priority sort; leave game if HP chicken.
7. Revived Tomb Vipers (Nihlathak, Pindleskin) and Dolls/Souls (Baal): abort run.
8. Running out of potions or merc dead mid-run: return to town in-run (KO InRunReturnTownRoutine, KB TownChicken) then take own portal back.
9. Throne mobs not cleared -> wave detection stuck: idle clear every 20 s.
10. Portal reuse failing (shared portal): try own portal first, keep the oldest `lastPortalTick` guard of 2.5 s (KB Pather.usePortal).

## Mapping
| step | package | data needed |
|---|---|---|
| WP travel | `bot/runs` + `nav` | WP object in town/area (txt ids from `internal/data`), which WPs are active (waypoint bitmask in memory UNVERIFIED location; Kolbot caches via first WP UI open) |
| moveToExit chain | `nav` | `d2map` exits per area: target level id + position |
| moveToPreset(...) | `nav` | `d2map` preset units: type (monster/object/stairs), txt id, room x/y*5 + local x/y |
| boss lookup | `bot` | monster list by classid and by unique name (super-uniques have name index in monstats; compare `uniqueno`/SuperUnique id) |
| seals, doors, chests | `bot` | object list (txt id, mode, selectable) |
| wave detection | `bot/combat` | monster classids inside a rectangle |
