# SME: gear, runewords, cube and vendors for a first playthrough (classic LoD 1.14d, offline SP)

Owner: SMEKB-B. Companion files: `builds-leveling.md`, `builds-endgame-*.md`. Project data (runes.txt of our install) lives in `docs/items.md`, `docs/gear.md`; this file is the human-knowledge layer on top.

Tags: **V** = fetched this session from a primary or reliable source (URL given) or read from project data; **S** = snippet or secondary; **U** = unverified / from memory. "D2R:" marks a D2R-only fact. All Maxroll pages below are D2R Season 14 text.

Sources fetched (2026-10-29, curl with browser UA, text-converted):
- A1 Arreat Summit runewords rules (live): https://classic.battle.net/diablo2exp/items/runewords.shtml (V, rules and item-type definitions; the per-word lists are on sub-pages I could not resolve).
- A2 Arreat Summit Horadric Cube (Wayback): https://web.archive.org/web/2012/http://classic.battle.net/diablo2exp/items/cube.shtml (V).
- A3 Arreat Summit Runes (Wayback): https://web.archive.org/web/2012/http://classic.battle.net/diablo2exp/items/runes.shtml (V).
- A4 Almar's Guides runeword lists (1.1x era data, shows ladder flag): https://www.almarsguides.com/Computer/Games/Diablo2/Crafting/Runewords/{Weapons,Armor,Helm,Shield}/ (V as read; secondary site).
- A5 Almar: ladder runewords in single player on 1.14d: https://almarsguides.com/Computer/Games/Diablo2/Misc/RuneWords/LadderOnlyInSinglePlayer/ (V as read).
- M1 Maxroll cube recipes https://maxroll.gg/d2/resources/horadric-cube-recipes ; M2 Maxroll important quests https://maxroll.gg/d2/resources/important-quests ; M3 Maxroll mercenary https://maxroll.gg/d2/resources/mercenary ; M4..M7 Maxroll leveling guides per class (`/d2/guides/<class>-leveling`).
- P1 project data: `docs/items.md` (runes.txt, cubemain.txt of this install, V per that doc), `docs/gear.md`.

## 0. The headline finding: ladder-only runewords do not work in vanilla 1.14d single player

- V A5: "Ladder Only Rune Words ... by default" are not available in Single Player on 1.14d/1.14c. Players edit/replace `Runes.txt` (launch `-direct -txt`, `Data\Global\Excel\Runes.txt`) to enable them. A4 also tags each word "Other Req: Ladder Only" (S).
- V P1 (`docs/items.md` 3.1): our runes.txt carries a server/ladder flag on 23 rows: Spirit, Insight, Fortitude, Infinity, Dragon, Dream, Edge, Faith, Harmony, Ice, Brand, Death, Destruction, Last Wish, Lawbringer, Oath, Obedience, Phoenix, Pride, Rift, Widowmaker, Winter, Wrath. A4 additionally marks Voice of Reason and Grief as ladder (S, differs from P1; trust P1 for this install, re-check live).
- So the old assumption "offline SP makes everything" (items.md Q1) is probably WRONG for runewords. Evidence order: A5 (V, says blocked), P1 flag (V), no source says SP ignores the flag. Still verify live once (insert Tal Thul Ort Amn into a 4-socket sword; if the item name does not turn gold the word is blocked). **U until live.**
- Rune upgrade recipes are a different matter: A2 (V): Hel and above "will only work for Single-Player, Open, or Ladder Characters", i.e. fine offline. The ladder flag in cubemain.txt (P1) does not contradict that because SP counts as allowed for recipes (S).
- Practical consequence for an unattended bot or a human playing offline without file edits: **Spirit, Insight, Fortitude, Infinity, Harmony, Faith, Edge, Ice, Brand, Death, Obedience, Dragon, Dream, Pride, Phoenix, Last Wish, Destruction, Rift, Oath, Lawbringer, Wrath, Winter, Widowmaker are NOT in the plan.** Use the "available in SP" list in section 2. Anything in Maxroll/D2R guides that names those words must be substituted (section 6).
- Not blocked and very strong: Enigma, Call to Arms, Heart of the Oak, Stealth, Lore, Ancient's Pledge, Rhyme, Smoke, Treachery, Bone, Duress, Lionheart, Chains of Honor, Beast, Fury, Doom, Eternity, Hand of Justice, Breath of the Dying, Kingslayer, Silence, Crescent Moon, Venom, Wind, Chaos, Famine, Black, White, Memory, Splendor, Sanctuary, Exile (S A4 "N/A" requirement lines; check each in-game by name).

## 1. Runeword rules (V A1 unless noted)

- Item must be socketed, normal or superior (white/grey), exactly as many sockets as runes, correct item type. Magic, rare, set, unique, crafted items cannot become runewords. Runes go in the right order, one at a time; a wrong order or wrong rune leaves a plain socketed item with only rune bonuses (the runes are lost). Runewords do not work in Wirt's Leg.
- Type groups (V A1): Body armor = chest only. Helms include barbarian helms, druid pelts, circlets. Shield includes necromancer heads and paladin shields. "Maces" = Mace, Morning Star, Flail and upgrades only; Club, Spiked Club are "clubs"; War Hammer, Maul, Great Maul are "hammers" (not maces). Staves are staves only (not wands). Melee = hand-to-hand, blunt, combo, polearm, spear, knife, sword, axe. Missile = bows and crossbows. Weapons = melee + missile + orbs.
- Item level requirement of a runeword is the highest rune plus the base item's own (S; Hel rune has requirement 0 in the table of P1; Hel reduces item requirements by 15/20%, V A3).
- Rune bonuses stay on top of the runeword (V A1). Socket the runes into plain socketed bases, do not "test" a recipe (loss of runes, V A1 warning).

## 2. Runewords available offline (non-ladder) worth knowing, by stage

Rune order is socket order. Levels are the A4 required clvl (S). "Use" is human practice (S/U). Effects: P1 V where cited in `docs/items.md` section 5; otherwise U.

### Normal (clvl 12-30), runes El..Amn, all cheap
| word | runes | base | notes |
|---|---|---|---|
| Stealth | Tal Eth | any body armor, 2 sockets | clvl 17; FCR 25, FRW 25, FHR 25, +6 dex, +15 stamina, mana regen (V P1 values; Maxroll uses it in every class, V M4..M7). Best first word: get it at clvl 17. |
| Ancient's Pledge | Ral Ort Tal | shield, 3 sockets | clvl 21; res all 13 (cold 30), +50% def, damage to mana (V P1). Quest runes: Act 5 "Rescue on Mount Arreat" gives Ral, Ort, Tal (V A3 text "quest reward in Act V", S for which quest and per difficulty); Maxroll builds AP at Act 5 Normal (V M5). |
| Lore | Ort Sol | helm, 2 sockets | clvl 27; +1 all skills, +10 energy, mana per kill, light radius (V P1 values partly). Sorc/Necro/Druid classic helm. |
| Nadir | Nef Tir | helm, 2 sockets | clvl 13 cheap helm, -light radius (V P1). Barbarian starter helm (V M7). |
| Steel | Tir El | sword, axe, mace, 2 | clvl 13; IAS 25, ED 20, open wounds 50 (V P1). Paladin/Barb/Assassin-claw not allowed. |
| Malice | Ith El Eth | melee, 3 | clvl 15; monster cannot heal, -100 def, but -5 life regen for you (V P1: skip for a bot). Barb Normal weapon (V M7). |
| Leaf | Tir Ral | staff, 2 | clvl 19; +3 fire skills, Firebolt/Warmth/Inferno +3, cold res (V P1). Fire sorc staff. |
| Zephyr | Ort Eth | bow/crossbow, 2 | clvl 21; Amazon bow (V P1/M4 optional). |
| Strength | Amn Tir | melee, 2 | clvl 25; ED 35, +20 str, +10 vit, crushing blow (V P1). |
| King's Grace | Amn Ral Thul | sword or scepter, 3 | clvl 25; Paladin option vs undead/demons (U). |
| Rhyme | Shael Eth | shield, 2 | clvl 29; res all 25, cannot be frozen, +25 MF, 20 block/FBR (V P1). Shael clvl 29. Maxroll builds Rhyme on a Bone Shield (V M4, Amazon). |
| Peace | Shael Thul Amn | body armor, 3 | clvl 29 (A4); Amazon skills +; Maxroll mentions it as a boss-damage base for Amazon (V M4, runeword name stripped by conversion: S). |
| Holy Thunder | Eth Ral Ort Tal | scepter, 4 | clvl 23, cheap lightning scepter for Paladin (U). |
| Honor | Amn El Ith Tir Sol | melee, 5 | clvl 27; +160% ED, +10 str, +1 skills, regen (V P1). Needs a 5-socket base (Larzuk). |
| Smoke | Nef Lum | body armor, 2 | clvl 37 (Lum); res all 50, +75% def, FHR 20 (V P1). Cheap Nightmare armor; Maxroll calls it "cheap Body Armor" (V). |

### Nightmare (clvl 30-60), runes Sol..Pul
| word | runes | base | notes |
|---|---|---|---|
| Lionheart | Hel Lum Fal | body armor, 3 | clvl 41; +15 str +20 vit +15 dex +50 life, res all 30, +20% ED (V P1). |
| Treachery | Shael Thul Lem | body armor, 3 | clvl 43; IAS 45, Fade on hit, Venom on strike, +assassin skills (V P1 table says `ass`); Assassin and anyone for Fade (Amazon Ubers variant uses it, V). |
| Bone | Sol Um Um | body armor, 3 | clvl 47; Necromancer +2 skills, Bone Armor/Spear on attack, mana (V P1). |
| Duress | Shael Um Thul | body armor, 3 | clvl 47; +def 150-200%, open wounds, cold dmg (V P1). |
| Stone | Shael Um Pul Lum | body armor, 4 | clvl 47; +60% def, +16 str/dex, FHR, big def vs missile/melee (S A4). |
| Prudence | Mal Tir | body armor, 2 | clvl 49; damage reduction + res (S). |
| Black | Thul Io Nef | club/hammer/mace, 4 | clvl 35; crushing blow, +ED, knockback (S). |
| White | Dol Io | wand, 2 | clvl 35; Necromancer Bone skills +; Maxroll Necro gear (V M6). |
| Memory | Lum Io Sol Eth | staff, 4 | clvl 37; caster staff (S, U effects). |
| Splendor | Eth Lum | shield, 2 | clvl 37; +1 skills, FCR 10, light (U). |
| Delirium | Lem Ist Io | helm, 3 | clvl 51 (U effects; random skill procs). |
| Radiance | Nef Sol Ith | helm, 3 | clvl 27 (S). |
| Melody | Shael Ko Nef | missile, 3 | clvl 39; Amazon bow skills (S/U). |
| Voice of Reason | Lem Ko El Eld | mace/sword, 4 | A4 says ladder only (S); skip. |
| Crescent Moon | Shael Um Tir | axe/polearm/sword, 3 | clvl 47 (S). |
| Venom | Tal Dol Mal | weapon, 3 | clvl 49 (S). |
| Kingslayer | Mal Um Gul Fal | sword/axe, 4 | clvl 53 (S). |
| Heart of the Oak | Ko Vex Pul Thul | staff or mace, 4 | clvl 55; +3 all skills, FCR 40, res 30-40, Oak Sage and Raven charges (V P1). Vex clvl 55; the caster endgame staff. |
| Silence | Dol Eld Hel Ist Tir Vex | weapon, 6 | clvl 55 (S). |
| Call to Arms | Amn Ral Mal Ist Ohm | any weapon, 5 | clvl 57; Battle Orders/Battle Command/Battle Cry + Mastery; the swap weapon of every build with a weapon swap (V M-class pages mention CtA prebuff). |
| Chaos | Fal Ohm Um | claw, 3 | clvl 57; Assassin (V P1). |
| Wind | Sur El | melee, 2 | clvl 61 (S). |

### Hell (clvl 60-85), runes Mal..Zod
| word | runes | base | notes |
|---|---|---|---|
| Enigma | Jah Ith Ber | body armor, 3 | clvl 65 (Jah 65, Ber 63); oskill Teleport (the item casts Teleport), +2 all skills, +45% FRW, strength per level (V P1). End goal armor for casters, merc-less teleporting for everyone. Not ladder. |
| Chains of Honor | Dol Um Ber Ist | body armor, 4 | clvl 63; +2 skills, res all 65, DR 8%, +20 str? (S/U). Great Paladin/melee armor. |
| Beast | Ber Tir Um Mal Lum | axe/hammer/scepter, 5 | clvl 63; Barbarian Fanaticism (S; Barbarian Werewolf). |
| Fury | Jah Gul Eth | melee, 3 | clvl 65; Barbarian (V P1). |
| Famine | Fal Ohm Ort Jah | axe/hammer, 4 | clvl 65 (S). |
| Doom | Hel Ohm Um Lo Cham | axe/polearm/hammer, 5 | clvl 67; freeze, Holy Freeze aura (S). |
| Eternity | Amn Ber Ist Sol Sur | melee, 5 | clvl 63 (S). |
| Hand of Justice | Sur Cham Amn Lo | any weapon, 4 | clvl 67; Holy Fire (S). |
| Breath of the Dying | Vex Hel El Eld Zod Eth | weapon, 6 | clvl 69 (S). |
| Sanctuary | Ko Ko Mal | shield, 3 | clvl 49 (S). |
| Exile | Vex Ohm Ist Dol | paladin shield, 4 | clvl 57 (S, Paladin shields only). |
| Gloom | Fal Um Pul | body armor, 3 | clvl 47 (S). |
| Rain | Ort Mal Ith | body armor, 3 | clvl 49 (S). |
| Principle | Ral Gul Eld | body armor, 3 | clvl 55; Paladin (S). |
| Bramble | Ral Ohm Sur Eth | body armor, 4 | clvl 61; Thorns aura (S). |

Honest limits: weapon words depend on the class doing melee. Casters need Enigma + Heart of the Oak + Lore/Rhyme/Smoke; they are the SP-available answer to "Spirit".

## 3. Rune effects when socketed (V A3)

Armor/helm | shield column | weapon column. Use for filling plain sockets when no runeword exists yet.
- Tal: weapon +75 poison/5s | armor/helm poison res 30, shield 35. Ral: weapon +5-30 fire | fire res 30 / 35. Ort: weapon +1-50 lightning | lightning res 30 / 35. Thul: weapon +3-14 cold | cold res 30 / 35. These four are the cheap resist runes for helms and shields (sockets in a 2-socket Cap give 60 of one or two types).
- El +15 def, +1 light | Eld slower stamina drain / +7% block (shield) | Tir +2 mana per kill (all) | Nef +30 def vs missile | Eth regen mana 15% (armor) / -25% target def (weapon) | Ith +9 max dmg (weapon) / 15% damage to mana (armor) | Amn 7% life steal (weapon) / attacker takes 14 | Sol +9 min dmg / damage reduced 7 | Shael +20% IAS / +20% FHR (+20% FBR shield) | Dol flee / life regen 7 | Io +10 vit | Lum +10 energy | Ko +10 dex | Fal +10 str | Lem gold find | Pul +75% damage to demons / +30% ED def | Um open wounds 25% / all res 15 (armor, helm), 22 (shield) | Mal prevent monster heal / magic damage reduced 7 | Ist MF 30/25 | Gul +20% AR / +5% max poison res | Vex mana steal 7% / +5% max fire res | Ohm +50% ED / +5% max cold res | Lo 20% deadly strike / +5% max lightning res | Sur blind / max mana 5% (+50 shield) | Ber 20% crushing blow / damage reduced 8% | Jah ignore target defense / +5% life (+50 shield) | Cham freeze / cannot be frozen | Zod indestructible.
- Min clvl: El 11, Eld 11, Tir 13, Nef 13, Eth 15, Ith 15, Tal 17, Ral 19, Ort 21, Thul 23, Amn 25, Sol 27, Shael 29, Dol 31, Hel (-), Io 35, Lum 37, Ko 39, Fal 41, Lem 43, Pul 45, Um 47, Mal 49, Ist 51, Gul 53, Vex 55, Ohm 57, Lo 59, Sur 61, Ber 63, Jah 65, Cham 67, Zod 69 (V A3, agrees with P1).
- Common vs rare (V A3): El..Thul plus Amn, Shael, Thul "pretty common"; Sol..Um semi-rare; Mal..Zod extremely rare. Arreat's advice: throw away low runes, keep Sol and up; for a bot, keep everything (cube value).

## 4. Horadric Cube recipes that matter (V A2/M1; 1.14d same unless flagged; P1 V matches cubemain)

Cube is 4 wide x 3 high (V M1). Never press Transmute with items you want to keep (V M1).
- Rune ladder: 3 of El..Ort -> next rune up, no gem (V A2): El->Eld->Tir->Nef->Eth->Ith->Tal->Ral->Ort->Thul. Thul and up need a gem: 3 Thul + Chipped Topaz = Amn; 3 Amn + Chipped Amethyst = Sol; 3 Sol + Chipped Sapphire = Shael; 3 Shael + Chipped Ruby = Dol; 3 Dol + Chipped Emerald = Hel; 3 Hel + Chipped Diamond = Io; 3 Io + Flawed Topaz = Lum; 3 Lum + Flawed Amethyst = Ko; 3 Ko + Flawed Sapphire = Fal; 3 Fal + Flawed Ruby = Lem; 3 Lem + Flawed Emerald = Pul; **2** Pul + Flawed Diamond = Um; 2 Um + Topaz = Mal; 2 Mal + Amethyst = Ist; 2 Ist + Sapphire = Gul; 2 Gul + Ruby = Vex; 2 Vex + Emerald = Ohm; 2 Ohm + Diamond = Lo; 2 Lo + Flawless Topaz = Sur; 2 Sur + Flawless Amethyst = Ber; 2 Ber + Flawless Sapphire = Jah; 2 Jah + Flawless Ruby = Cham; 2 Cham + Flawless Emerald = Zod (V A2). A2: the Dol->Hel step and above work for "Single-Player, Open, or Ladder" only (V).
- Gem ladder: 3 gems of one type and grade -> next grade up to perfect (V A2). Gem grade: chipped, flawed, normal ("standard"), flawless, perfect. Skulls likewise.
- Socketing white items: Tal + Thul + Perfect Topaz + normal body armor; Ral + Amn + Perfect Amethyst + normal weapon; Ral + Thul + Perfect Sapphire + normal helm; Tal + Amn + Perfect Ruby + normal shield (V A2/M1). Results are random 1..max. M1 (D2R) gives per-max-socket odds, e.g. max-2 base: 83% two sockets, 17% one; max-3: 67% three; max-4: 50% four (U for 1.14d, same table on D2R). Low-quality and superior bases refuse (V M1). Not for bases the bot cares about: use Larzuk instead.
- Remove sockets: Hel + Tome/Scroll of Town Portal + socketed item destroys the contents (V M1/P1). Never.
- Repair: Ort + weapon, Ral + armor (V P1). Recharge: charge items + chipped gem + Ort (V M4 mentions it for Teleport/Lower Resist charges).
- Rejuvenation: 3 health + 3 mana + chipped gem = Rejuvenation; with a normal gem = Full Rejuvenation (V A2). Spare potions only.
- Unique upgrade: Normal unique weapon + Ral + Sol + Perfect Emerald -> Exceptional version; armor + Tal + Shael + Perfect Diamond; Exceptional -> Elite: weapon Lum + Pul + Perfect Emerald, armor Ko + Lem + Perfect Diamond (V A2/M1). Ladder-flagged in P1 data for the elite step (S: A2 lists it "Single Player, Ladder Only"), exceptional step is fine. Sets and rares have similar recipes (M1).
- Quest recipes: Staff of Kings + Amulet of the Viper = Horadric Staff; Khalim's four parts = Khalim's Will; Wirt's Leg + Tome of Town Portal = Cow Level portal (V M1; needs Baal quest done in that difficulty).
- Magic reroll: 3 perfect gems + magic item = new magic item same type; 3 rings -> amulet, 3 amulets -> ring (junk). Magic weapon reroll with socket: 3 chipped/flawless gems + magic weapon = socketed magic weapon (ilvl 25/30); Maxroll leveling uses this to reroll magic wands/scepters for +skills (V M6, M5: "1 Magic Wand + 3 Chipped Gems", buy wands cheap from Drognan/Akara). Good human trick, high value for Necromancer (+Teeth/Bone Spear wand) and Paladin (+skills scepter).
- Crafted gear (U for exact rune pairs, V for one example): magic item of a listed base + Jewel + a rune + a Perfect gem. Four groups by gem: Hit Power (sapphire), Blood (ruby), Caster (amethyst), Safety (emerald) (S P1 section 4.3). V M7: Heavy Gloves + Perfect Ruby + Nef Rune + Jewel gives crafted (Blood-type) gloves. Mid-game crafted gloves/belts/rings/amulets are a normal human upgrade path.
- Prismatic Amulet: 6 perfect gems (1 of each type) + amulet = amulet with all res 16-20 (V A2/M1). Rings: Ring + Perfect <gem> + potion = Jade/Garnet/Coral/Cobalt ring (poison/fire/lightning/cold 21-30%), recipes from M1 (D2R page; also in 1.14d U).

## 5. Larzuk and quest rewards that matter for gear (V M2 D2R; 1.14d mostly identical, tag S)

- Siege on Harrogath (Act 5 Q1, kill Shenk): Larzuk sockets one item of your choice; every base gets its maximum sockets limited by item level (ilvl 1-25 / 26-40 / 41+): e.g. Crystal Sword 4 at ilvl 26-40 and 6 above; Monarch always 4; Large Shield 3; Heraldic Shield and Targe 4; most body armors 2-4; wands/daggers 1 (V M2). The ilvl-based cap exists in 1.10+ (S). It is once per difficulty, so three sockets per character total. Cows drop Crystal/Broad Swords whose ilvl lands in the 4-socket band, hence the Spirit sword plan (V M2, M5), which in our SP offline setup is dead (Spirit is ladder-only) - use the socket for a Honor/Strength/CtA base, a Heart of the Oak staff, Lore helm base or the best shield (U).
- Imbue (Charsi, Act 1 Tools of the Trade, needs Horadric Malus): one normal item becomes a rare, ilvl = clvl+4 (V M2). Humans imbue Diadems, Circlets, Coronets, druid pelts, primal helms, belts, boots (V M2).
- Hell Forge (Act 4 Q): Hephasto drops hammer; destroying Mephisto's Soulstone gives 1 regular + 2 flawless + 1 perfect gem and a rune: El..Sol in Normal, Sol..Um in Nightmare, Hel..Gul in Hell (V M2 D2R; 1.14d S).
- Den of Evil +1 skill and respec token (V M2: "Akara resets stats and skills once per difficulty"; in 1.14d the respec is also given by Akara for Den, S; Maxroll says 3 respecs total). Radament +1 skill, Izual +2 skill points (V M2). Golden Bird +20 life, Lam Esen +5 stat points, Prison of Ice (Anya) +10 all res: 30 total res and 15 stat points if done in all three difficulties (V M2). Do all three: the Anya scroll is worth three resist-gear slots.
- Rune quests: Act 5 "Rescue on Mount Arreat" Ral/Ort/Tal (V A3 S, which quest U). The Countess drops runes in the Forgotten Tower: Maxroll uses her Normal/NM runs for Tal, Eth, Ral, Tir, Ort, Sol, Shael (V M4..M7); the Forgotten Tower is in the Black Marsh.

## 6. Substitutions when a guide names a ladder-only runeword (for a SP-offline player or bot)

| guide says | use instead | why |
|---|---|---|
| Spirit sword/shield (caster, paladin) | Lore helm, Rhyme or AP shield, Smoke/Stealth armor; weapon: plain staff +skills, Leaf (fire), Heart of the Oak from Nightmare, White wand (necro), Honor/Steel/Strength (melee) | Spirit blocked (A5, P1) |
| Insight / Infinity / Pride / Obedience merc polearm | a good rare or unique polearm/spear for the Act 2 merc (his weapons are spears, polearms, javelins only, V M3); Might aura from Nightmare hire | ladder-only (P1); no SP equivalent. Beast/Doom/Famine are axes/hammers, not usable by him |
| Fortitude armor | Chains of Honor (4os), Lionheart (3os), Smoke, Duress, Stone, Enigma, Treachery, Bone | Fortitude ladder |
| Infinity / Conviction | none in SP; use Lower Resist (staff/wand charges), Might merc | ladder |
| Harmony, Faith, Edge, Ice, Brand, Wrath (Amazon bows) | Zephyr early, Melody, uniques: Buriza-Do Kyanon, Windforce, Eaglehorn, Stormstrike-type bows (names from M-pages, S) | these bow words are ladder-only (A4) |
| Dream, Phoenix, Dragon | Sanctuary / Exile / Splendor shields, Vampire Gaze/Shaftstop for merc (uniques) | ladder |

D2R-only (U from memory; check by name against `internal/data` Runewords, P1 V): Hustle, Mosaic, Metamorphosis, Sunder Charms, Crack of the Heavens and the Warlock class in the Maxroll pages. Grief and Death exist in 1.14d but are ladder-flagged.

## 7. What to buy and keep, by stage (S/U; human consensus from M4..M7 unless tagged V)

Vendor map (S, standard 1.14d knowledge): Act 1 Akara: potions, scrolls, staves, wands, scepters for casters? (V M4..M7 shop +skill wands/staves at Akara in Normal and NM); Charsi: armor/weapons (check for 2-socket armor/helms/bows each run, V M4); Gheed: gamble (rings, amulets, charms? no). Act 2: Fara (armor/blunt, belts), Drognan (staves, wands, orbs, V M6 "bone wand from Drognan"), Elzix gamble, Lysander (potions), Greiz (merc). Act 3: Ormus (staves incl. Staff of Teleportation chance, V M4), Hratli (belts, boots? javelins), Alkor. Act 4: Halbu, Jamella (gamble). Act 5: Larzuk, Malah (potions), Anya gamble.

| stage | buy | keep | sell |
|---|---|---|---|
| Normal Act 1 (clvl 1-13) | TP tome, ID tome, belts later, stamina and life potions, mana potions for casters; caster: short staff/wand with +2/+3 to the attack skill if you have ~3500/5500 gold (V M5 sorc, M6 necro); barb: scepter x2; amazon: javelins stacks | chipped gems (for reroll), every Ruby, Tal/Eth/Ral/Tir from Countess | all magic gear, jewelry only if nothing to ID, white junk |
| Normal Act 2-3 | Belt at Fara (more potion slots), 2-socket armor/helm at Charsi, Bone Shield (Rhyme base) at Drognan, +skill wand reroll (cube) | Staff of Teleportation from Ormus around clvl 20-23 (V M4: Teleport charges staff needs clvl 24) | rares unless clearly good |
| Normal Act 4-5 | Large Shield (AP base, 3 sockets, V M5), Kite Shield necro | Q rewards, Larzuk socket for a key base, runes | all |
| Nightmare | Heavy Gloves/Boots/Belt with resist: Maxroll names "Coral Belt of the Mammoth", "Heavy Gloves of Readiness", "Russet Heavy Boots of Haste", "Garnet Belt of the Colossus" (V M4..M7: prefer fire+lightning resist, FRW, life); Wand of Lower Resistance from Akara/Drognan at clvl 30-35 (V M4 charges) | Lore, Smoke/Stealth, AP | duplicates |
| Hell | potions only in practice; rare ring/amulet from gambles | all runes Sol+, perfect gems, jewels, uniques of the build | everything below the build's needs |

Resist rules of thumb: Normal 0 penalty, Nightmare -40, Hell -100 (V P1 difficulty tables); cap 75 (S). Targets: Nightmare >= +20 after penalty (so ~60 raw), Hell +25..75 after penalty (raw 125..175). Sources: AP (13 + 30 cold), Rhyme 25, Smoke 50, Lionheart 30, Fortitude ladder n/a, Enigma none, Um in sockets 15/22, Ral/Ort/Thul/Tal sockets 30-35 single resist, Prismatic Amulet 16-20 all, Jade/Garnet/Coral/Cobalt rings 21-30 single, Anya scroll 10 all, Pooled "of the Colossus/Mammoth" belts and boots (V M4).

## 8. Common-sense item priorities for a first playthrough (S/U)

1. Before Normal Act 3: belt, 2-socket Stealth base, Tal+Eth (Countess), mana and life potions, +skill weapon for casters (see leveling file).
2. Normal end: AP shield, Lore helm if Ort+Sol, Teleport staff swap from Ormus for casters, Anya resist scroll done.
3. Nightmare: swap to Smoke when Nef+Lum exists, Rhyme shield, resist boots/gloves/belt, charge wand. Make Lore with Ort Sol. Mercenary: Act 2 Nightmare Might merc with a good rare polearm (Insight not available).
4. Hell: Enigma (Jah+Ith+Ber), Heart of the Oak (for staff users), Chains of Honor/Lionheart for non-Enigma, Treachery/Bone/Duress by class, Call to Arms swap weapon, uniques named in builds-endgame.
5. Always: stash every rune (cube-stackable) and every gem; keep Ral/Ort for repair only if no Repair All is possible.

## 9. Open questions

1. Is Spirit (and the 22 other flagged words) really blocked in the install used by the bot? Test live; P1 Q1. If blocked, builds-endgame must use section 6 substitutes.
2. Which Act 5 quest gives Ral/Ort/Tal in 1.14d and does it repeat each difficulty (A3 says quest reward in Act V only).
3. Which Charsi/Akara shop stock is present in SP vs realm (S only).
4. Exact 1.14d effects of Chains of Honor, Stone, Memory, Delirium, Prudence (not fetched; U).

Update (SMEKB-B2): Q1 (Ral/Ort/Tal quest) and the Grief/Voice of Reason ladder status are resolved in `builds-open-questions.md` (Rescue on Mount Arreat; both ladder-only, V). The class-by-class files are `class-<class>.md`.
