# SME: endgame Amazon builds and farming runs/drop knowledge, classic 1.14d softcore single player

Owner: SMEKB-B. Companions: `builds-endgame-casters.md`, `builds-endgame-melee.md`, `builds-gear.md`, `builds-leveling.md`; bot tables `docs/builds-endgame.md` (7.7 Amazon, section 5 merc).

Tags: **V** fetched this session or project data (URL named), **S** secondary, **U** memory/unverified. D2R: marks D2R-only facts. 1.14d offline. **Ladder-only runewords (Faith, Harmony, Edge, Ice, Brand, Wrath, Infinity, Insight, Spirit, Fortitude, Grief, Last Wish, Phoenix, Dream...) are not craftable in vanilla offline 1.14d** (`builds-gear.md` section 0). The Amazon is the class that suffers most, because her top runewords (Faith, Ice, Brand, Harmony) are ladder-flagged.

Sources fetched 2026-10-29 (Maxroll D2R Season 14, V as read): `https://maxroll.gg/d2/guides/{lightning-fury-amazon-guide,strafe-amazon,freezing-arrow-amazon-guide,poison-javelin-amazon,multiple-shot-amazon}`; `https://maxroll.gg/d2/resources/{gold-magic-find,horadric-cube-recipes,important-quests,general-leveling,mercenary}`; project `docs/gear.md`, `docs/builds-endgame.md`, `docs/difficulty.md`.

## 1. Amazon facts

- Weapon type decides skills: bow/crossbow for arrow skills, javelin for Lightning Fury/Poison/Plague Javelin, spear/javelin for Charged Strike/Jab/Power Strike/Impale/Lightning Strike (V project docs/gear.md from skills.txt itypea). A quiver sits in the left hand (no shield) for bows (V docs/gear.md live).
- Breakpoints (S docs/builds-endgame.md): FCR 0/7/14/22/32/48/68/99/152; FHR 0/6/13/20/32/52/86/174/600; IAS per weapon type (V Maxroll names 52% IAS for Lightning Fury, 30% minimum; 69% pierce for strafe).
- Passives: Dodge/Avoid/Evade 1 point each give a chance to evade (V Maxroll), Pierce for projectiles (82-100% for Fury, 69% for Strafe).
- Valkyrie (needs Decoy -> Slow Missiles -> Inner Sight and Evade) as meat shield (V gear/skills.txt).
- Amazon is the weakest class to bot without good gear (V project class ranking 16).

## 2. Builds

### 2.1 Lightning Fury / Charged Strike (Javazon) (V Maxroll lightning-fury-amazon-guide)
- Skills: 1 each Jab, Pierce, Dodge, Avoid, Evade, Slow Missiles, Decoy, Valkyrie; max Lightning Fury, Charged Strike, Lightning Strike, Power Strike, Lightning Bolt. Pierce enough for 82% (or 100% with Razortail belt, S D2R). Stats: Str/Dex for gear, rest Vit; block 50-75% with Rhyme Bone Shield (V).
- Breakpoints: 52% IAS for Fury (30% minimum), 68% FCR for Teleport (V). Javelins: stacks of Javelins of Swiftness for IAS (V ML-AMA), keep extra stacks to reduce repair trips.
- Weakness: lightning immune is common in Hell; fix with Infinity merc (ladder, not possible), Lower Resist wand charge, Crack of the Heavens (D2R-only), Jab/physical (V Maxroll). In SP offline substitute: Might merc, Lower Resist wand, Charged Strike with Jab fallback, or switch builds.
- Gear offline-safe: Enigma or Staff of Teleportation for Teleport, Chains of Honor/Lionheart/Smoke armor, Rhyme or Ancient's Pledge shield, Gemmed helm, Andariel's Visage, Harlequin, Stormshield (shield, V Maxroll names it). Javelins: Thunderstroke (unique javelin, V Maxroll names it), Javelins of Swiftness from Malah/Charsi (V ML-AMA).
- Farming: Cows (best, V "The Secret Cow Level"), Countess, Pit, Arcane Sanctuary, Mephisto, Baal (V Maxroll tables); needs Lightning-immune handling.
- 1.14d: the classic Javazon (Charged Strike + Lightning Fury + Jab) is the same shape; early Lightning Fury needs 24+ levels (Lightning Fury reqlevel 30, Lightning Strike 24, Charged Strike 12, V project data).

### 2.2 Strafe / Multiple Shot bowazon (V Maxroll strafe-amazon; project default `amazon_multishot_endgame90`)
- Skills (V strafe): 1 each Valkyrie, Pierce (to 69%); max Strafe, Guided Arrow, Multiple Shot, Critical Strike, Penetrate (remaining). Stats: Str for gear, majority of rest into Dex for damage, remainder Vit.
- Weapons (V Maxroll): Faith Grand Matron Bow (ladder, skip), Windforce (unique), Buriza-Do Kyanon (unique crossbow, 100% pierce, V), Eaglehorn, Hustle (D2R-only), M'avina's Caster, Mist (ladder). SP-offline: Windforce, Buriza-Do Kyanon, Eaglehorn, rare/crafted bows, Zephyr, Melody. Helm: Andariel's Visage, Guillaume's Face, Jeweler's diadem. Armor: Chains of Honor, Treachery, Enigma, Lionheart. Quiver: arrows/bolts stack of 350 at the vendor (V docs/gear.md).
- Use: "epitome of an auto-turret playstyle", 100% pierce, 95% to-hit; high IAS breakpoints (V).
- Cons (V project class ranking, Maxroll "gear dependent", "cannot rely on mercenary DPS").
- Immunities: physical immune packs: use Magic Arrow (part magic) or Guided Arrow? (V project). Cold/Fire arrows for Freezing/Exploding builds.

### 2.3 Freezing / Exploding Arrow bowazon (V Maxroll freezing-arrow-amazon-guide)
- Skills: 1 each Dodge, Evade, Avoid, Valkyrie, Strafe (to proc Hustle: D2R item); max Pierce (to 69%), Freezing Arrow, Cold Arrow, Immolation Arrow, Exploding Arrow, Fire Arrow. Needs +x Mana after kill (V). The Exploding Arrow variant is the Normal/NM leveling in `builds-leveling.md` 2.1.
- Cold/fire damage: cold/fire immune packs; Cold Mastery-free so no -enemy cold res besides gear.

### 2.4 Poison Javelin / Plague Javelin (V Maxroll poison-javelin-amazon)
- Skills: 1 each Dodge, Avoid, Evade, Impale; max Plague Javelin, Poison Javelin, Pierce (100% with gear), Penetrate; Lightning Fury/Bolt for the rest. Stats: Str/Dex for gear, rest Vit. Not a bot candidate (kite and clouds).

### 2.5 Other (S/U): Jab/Fend (spear melee), Lightning Strike, Poison Javelin classic (Poison Javelin 20, Plague, Lightning Fury 1).

## 3. Drop mechanics (V Maxroll gold-magic-find, D2R text; classic formulas are the same, S)

- MF raises the chance of Magic/Rare/Set/Unique from a base drop with diminishing returns: Rare = MF*600/(MF+600), Set = MF*500/(MF+500), Unique = MF*250/(MF+250) (V). It does nothing for runes, base tier, item count (V "Myths"): more players (/players X) increases drop count (V).
- Runes come from monster/boss treasure classes only; Hell act bosses and the Countess/Cows are the usual sources (S).
- Gold Find multiplies gold piles; used for gambling (Barbarian GF build, V).
- In D2R the merc's MF is added when the merc kills (V text); in 1.14d the player's MF does not apply to merc kills (U).
- "Failed" uniques/sets drop as high-durability rare/magic (V).

## 4. Farming runs (human practice; V where the Maxroll table is cited, else U)

Maxroll's farming spot table (V, same on every build page): Act 1: Countess (runes, Key of Terror), Pit (all items, max area level), Secret Cow Level (runes, bases, charms/gems/jewels); Act 2: Arcane Sanctuary (runes, charms, jewelry), Summoner (Key of Hate); Act 3: Ruined Temple, Mephisto (uniques, sets); Act 4: Chaos Sanctuary, Diablo; Act 5: Eldritch, Shenk, Pindleskin (gearing, gold, leveling), Thresh Socket, Drifter Cavern (all items), Halls of Anguish/Pain, Nihlathak (Key of Destruction), Worldstone Keep/Throne (all items, max area level), Baal (uniques, sets).

| run | where | what it gives | notes |
|---|---|---|---|
| Mephisto | Durance of Hate 3 (WP at Durance 2) | high uniques/sets, many runes | classic standard; quick, even for low-clear builds; Teleport helps; in Hell dangerous without lightning/cold/poison res and Mana Burn avoidance (U) |
| Andariel | Catacombs 4 (WP Catacombs 2) | uniques, some runes | easiest boss; Act 1 first target in each difficulty (V Maxroll ML) |
| Pindleskin | Nihlathak's Temple entrance, Red Portal from Harrogath after Prison of Ice | gear, gold (V Maxroll) | short, each run resets; champion packs guard (U) |
| Eldritch | Frigid Highlands area from the WP | gear, gold | fast; "Eldritch the Rectifier" |
| Shenk | Bloody Foothills outside Harrogath | gear, gold; also Siege on Harrogath quest | fast |
| Cows | Secret Cow Level (Wirt's Leg + TP tome in the cube; Baal quest done in that difficulty) | runes, bases (Crystal/Broad Swords), charms | crowd build only; Hell cows are dangerous for squishy builds (V MC) |
| Baal | Throne of Destruction waves + Baal | uniques, sets, experience | needs Teleport or walking from Worldstone Keep WP; waves 1-5 are hard (V ML-BAR) |
| Chaos Sanctuary | Hell Act 4 | max area level, seal bosses | demons; good XP |
| Countess | Forgotten Tower in Black Marsh | runes (Tal, Eth, Ral, Tir, Ort, Sol; V ML-*), Key of Terror | the early rune farm in Normal/NM; Hell Countess drops to high runes per cube (S) |
| Arcane Sanctuary / Summoner | Act 2 Hell | charms/gems/jewels, Key of Hate | requires Teleport |
| Pit | Tamoe Highland | all item types | many elites |
| Ubers | Hell Act 5 portals (Key of Terror/Hate/Destruction -> mini ubers -> Uber Tristram) | Hellfire Torch, Annihilus (separate) | needs strong build (V Maxroll Ubers variant) |

- Boss kills provide uniques; high runes come from Hell bosses and the Countess; Cows give runes and bases. Early SP loop (S): Countess Normal for El..Ort, NM Countess for Sol..Dol, Hell Andariel/Mephisto for the Hel+ band.
- Resists: Hell bosses need life leech and decent resists on all four elements (U); use the project `docs/difficulty.md` tables for per-monster immunities.
- Experience: Baal waves and Chaos in Hell give the best XP (V ML-GEN). In solo SP use /players X to raise XP and drops (V ML-GEN).

## 5. Open questions
1. Merc MF in 1.14d vs D2R sum-rule above.
2. Whether Windforce/Buriza/Eaglehorn drops are reachable in solo SP before Hell Act 5 (S: they are uniques, drop in Hell from any monster).
3. Pindleskin portal timing and Anya quest requirement.
