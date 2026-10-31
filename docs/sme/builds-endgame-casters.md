# SME: endgame builds, caster classes (Sorceress, Necromancer, Druid), classic 1.14d softcore single player

Owner: SMEKB-B. Companions: `builds-endgame-melee.md`, `builds-endgame-amazon-farming.md`, `builds-gear.md`, `builds-leveling.md`. The earlier `docs/builds-endgame.md` is the bot's per-class build tables and breakpoint list; this file is the broader human view (alternatives, gear, farming).

Tags: **V** fetched this session (URL) or project data, **S** secondary, **U** from memory/unverified. D2R: marks D2R-only facts. Everything below targets 1.14d with the SP caveat: **ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Harmony, Faith, Edge, Ice, Brand, Death, Obedience, Dragon, Dream, Pride, Phoenix, Last Wish, Destruction, Rift, Oath, Lawbringer, Wrath, Winter, Widowmaker) are not craftable in vanilla offline 1.14d** (`builds-gear.md` section 0). Maxroll guides name them everywhere, so each item below carries an SP-offline substitute.

Sources fetched 2026-10-29: Maxroll D2R Season 14 build guides `https://maxroll.gg/d2/guides/{frozen-orb-sorceress,lightning-sorceress,fire-wall-sorceress-guide,hydra-sorceress,blizzard-sorceress,poison-nova-necromancer,bone-spear-necromancer,summoner-necromancer-guide,fissure-druid,summon-druid,tornado-hurricane-druid}` (V as read, D2R text); Maxroll `https://maxroll.gg/d2/resources/{breakpoints-animations,horadric-cube-recipes,important-quests}`; `https://maxroll.gg/d2/resources/immunities` (V by `docs/builds-endgame.md`). Breakpoints: asgard and Almar tables via `docs/builds-endgame.md` section 3 (S). Item numbers beyond what is cited are U.

## 1. Cross-class concepts used below

- Breakpoints (frames at 25 fps; a higher FCR/FHR/IAS than the next threshold changes nothing): see `docs/builds-endgame.md` section 3 for per-class tables (Sorc FCR 0/9/20/37/63/105/200; Necro FCR 0/9/18/30/48/75/125; Druid human FCR 0/4/10/19/30/46/68/99/163; Sorc FHR 0/5/9/14/20/30/42/60/86/142/280; all S).
- Resist caps: 75% each, plus -40 (Nightmare) / -100 (Hell) penalty before the cap (V project `docs/difficulty.md`). So Hell needs raw 175 to be maxed; 125 raw gets you +25 which is the project's minimum. Maxroll's guides assume 75 in Hell (V).
- Immunities: Hell packs are immune to something; you need a second damage source or a Lower Resist/Infinity/Conviction merc (Conviction and Lower Resist only at 1/5 effect against true immunes; D2R: Sunder Charms remove immunity, not in 1.14d) (V immunities page via docs/builds-endgame.md, S for 1.14d).
- Mercenary: Act 2 desert merc, Might in Nightmare/Hell (V M3, project merc.md). In SP offline, Infinity polearm is not available (ladder), so use a plain high-damage rare/unique polearm; Holy Freeze for crowd control (S).
- Teleport: Sorceress skill 1 point at clvl 18; everyone else Enigma (Jah Ith Ber, not ladder, V P1) or a Staff of Teleportation charge item from Ormus (V ML-DRU).

## 2. Sorceress

### 2.1 Frozen Orb / Blizzard (cold) — standard "safe" Sorceress
- Skills (V Maxroll frozen-orb-sorceress, D2R): 1 each Static Field (to 10 points? the guide says Static Field 10 points), Teleport, Frozen Armor, Warmth; max Frozen Orb, Cold Mastery, Ice Bolt, Ice Blast, Blizzard (remainder). Blizzard line in 1.14d is the same (Frozen Orb exists in classic LoD; the Maxroll text says it has a cooldown, whether 1.14d has one is U).
- Classic Blizzard: Blizzard 20, Glacial Spike, Ice Blast, Cold Mastery 20, Frozen Orb and Static Field 1; Frozen Armor, Teleport, Telekinesis, Warmth 1. (V project `build-primary-blizzard.json` equivalent per docs/builds-endgame.md 7.1.)
- Stats: Vitality all after Str/Dex to wear gear; Energy 0 (S Maxroll "Strength for gear, rest Vitality").
- Breakpoints: FCR 63 (Normal/NM) then 105 (Hell; V Maxroll "63% or 105%"), FHR 42-60. Set variant FHR 86.
- Key gear offline-safe: Heart of the Oak (Ko Vex Pul Thul staff/mace, +3 skills, FCR 40, res 30-40), Enigma armor (Jah Ith Ber) for Teleport and +2 skills, Lore helm (Ort Sol) then Harlequin Crest/Shako, Rhyme or Ancient's Pledge shield (Spirit Monarch is Spirit = ladder, skip), Skin of the Vipermagi (+skills, resists), Death's Fathom orb (+cold skills, V Maxroll names it, unique orb, 1.14d S), Tal Rasha's set, Arachnid Mesh (belt, +1 skills, FCR 20, V), Frostburn gloves (U), Magefist gloves (FCR 20, V ML-DRU), Stone of Jordan x2, Nightwing's Veil, Trang-Oul's Claws. Weapon swap: Call to Arms (Amn Ral Mal Ist Ohm; Battle Orders etc., +skills, V Maxroll swap list) or Wand of Lower Resistance.
- Resist targets: raw 75 each in Normal/NM, 175 for a Hell cap; at minimum NM +20 post-penalty (raw 60), Hell raw 125 (+25) (V project).
- Immunities: cold immune packs (131 rows V difficulty.md): Static Field + Telekinesis stun on bosses, Lower Resist wand from Nightmare, or skip. Lightning Sorc is the answer to cold immune: see 2.2.
- Farming: Mephisto (moat), Andariel, Pindleskin (V Maxroll "destroy Andariel, Mephisto and Pindleskin"), Eldritch/Shenk (Act 5), Baal waves with Static Field; Cows with Blizzard crowd. Hell Blizzard needs kiting; bad vs fast melee (V Maxroll cons: "Very Squishy", "Specific Breakpoints Are Crucial", "Slow Against Cold Immunes").

### 2.2 Lightning (Lightning + Chain Lightning, Static Field) — top boss killer
- Skills (V Maxroll lightning-sorceress): 1 each Telekinesis, Teleport (if no Enigma), Static Field, Frozen Armor, Warmth, Thunder Storm; max Lightning, Lightning Mastery, Chain Lightning, Charged Bolt, Nova (remainder). "Huge damage range, only 1 damage type, limited early starter" (V cons). Classic note: Lightning/Chain Lightning have their own FCR table (7/15/23/35/52/78/117/194, S asgard).
- Gear: Heart of the Oak flail, Eschuta's Temper (unique staff, V Maxroll lists it), The Oculus (unique orb), Crescent Moon, Wizardspike, Tal Rasha's, Stormshield or Rhyme/AP shield, Metalgrid belt? (U). Lower Resist charges wand on swap for immunes (V Maxroll note 1). Treachery on a merc or swap to pre-buff Fade for Ubers (V).
- Farming: all Hell areas late, Mephisto, Baal, Ubers (needs the Uber keys; V Maxroll Ubers variant; D2R text, same mechanic in 1.14d, S).

### 2.3 Fire (Fire Wall / Hydra / Meteor) — cheap boss farmers
- Fire Wall (V Maxroll fire-wall-sorceress-guide): 1 each Static Field, Energy Shield, Teleport, Frozen Armor, Enchant; max Fire Wall, Fire Mastery, Warmth, Inferno, Telekinesis. "Great Mephisto farming, very squishy, prefers stationary targets, limited farming locations, low gear cost" (V). Breakpoints FCR 63/105, FHR 60.
- Hydra (V Maxroll hydra-sorceress): 1 each Static Field, Teleport, Frozen Armor, Frost Nova; max Hydra, Fire Ball, Fire Bolt, Fire Mastery, Telekinesis; 105% FCR (V). Starter: spec cold/lightning to handle immunes first.
- Classic note (U): Fire sorcs hit fire immune walls first (largest immune group, project 106 rows V), so most humans keep Fire Wall for Mephisto/Andariel/Baal only; a Cold Mastery-free fire build lacks immune breakers.

## 3. Necromancer

### 3.1 Summoner (Skeleton + Mage + Revive + Corpse Explosion) — safest, slowest
- Skills (V project docs/builds-endgame.md 7.3, V Maxroll summoner-necromancer-guide): 1 each Iron Golem, Summon Resist, Decrepify, Dim Vision, Lower Resist, Revive; max Raise Skeleton, Skeleton Mastery, Corpse Explosion, Amplify Damage; Mage 6-ish.
- Stats: Str for gear (~55 with Enigma Mage Plate), Vitality rest.
- Breakpoints: FCR 75 then 125, FHR 39-56 (V docs/builds-endgame.md section 3).
- Gear: Bone armor (Sol Um Um, +2 necro skills) in NM/Hell, Enigma later; Heart of the Oak (staff) or White (Dol Io wand, necro bone skills, S/U) + a necro shield (Preserved Head etc.) with +skills; Spirit is out. Helm: Lore or Harlequin. Merc: Might + a good polearm.
- Weak spots: needs corpses (Corpse Explosion), minions die to uniques (V Maxroll cons), Wave 2 Unravelers magic immune only matters for Bone Spear builds (V ML-NEC).

### 3.2 Bone Spear / Bone Prison (V Maxroll bone-spear-necromancer)
- Skills: 1 each Clay Golem, Decrepify, Life Tap; max Bone Spear, Bone Prison, Bone Wall, Bone Spirit, Teeth. 125% FCR in best gear; Amplify not needed because magic damage is clean vs most except magic immunes (Unravelers, V). FCR 75 in mid gear.
- Gear: White wand, necro shield with +Bone Spear, Heart of the Oak, Enigma. All offline-safe.

### 3.3 Poison Nova (V Maxroll poison-nova-necromancer)
- Skills: 1 each Amplify Damage, Lower Resist, Bone Armor, Skeleton Mastery, Revive, Golem Mastery, Summon Resist, Iron Golem; max Poison Nova, Poison Explosion, Poison Dagger, Corpse Explosion; leftover into Lower Resist/Bone Wall/Golem Mastery (V). Poison Nova is the strong clear skill but poison-immune packs (Poison Dagger synergy, no immunity workaround except -res) exist (S).
- Gear (V Maxroll): Death's Web (unique wand, +2 poison/bone), Plague, Heart of the Oak flail, White base +2-3 Poison Nova, Trang-Oul's set pieces, Rhyme/AP shield. Rainbow Facet jewels are rare drops (S).
- Not great in SP offline early because the build needs Death's Web/big skill gear; Poison Nova clears but leaves a long solo game.

## 4. Druid

### 4.1 Fire (Fissure/Volcano) — leveling and starter
- Skills (V Maxroll fissure-druid): 1 each Oak Sage (max last), Raven, Summon Spirit Wolf, Dire Wolf, Grizzly; max Fissure, Firestorm, Volcano, Molten Boulder, Armageddon, Oak Sage. Stats: Strength to gear, rest Vit.
- Gear: Heart of the Oak (staff), Enigma, Lore pelt, Rhyme/AP, Leaf in Normal; fire immune packs the big wall; use Tornado/Hurricane (below) or summons.
- Fire druids carry the Normal/NM, then hit fire immunities in Hell (V project: fire immune 106 rows).

### 4.2 Tornado / Hurricane (V Maxroll tornado-hurricane-druid; project default)
- Skills: 1 each Oak Sage, Summon Spirit Wolf, Dire Wolf, Grizzly; max Tornado, Hurricane, Cyclone Armor, Twister, Oak Sage. FCR 99 (V). Physical + cold damage means fewer immune packs; hard to aim (V cons).
- Gear: Heart of the Oak staff, Enigma, Lore/Wolf pelt (+skills), Rhyme/AP shield, Smoke early; merc Might.
- Farming: Mephisto, Andariel, Pindleskin, Baal, Cows; "Reliant on Enigma", "Slower clear speed" (V cons).

### 4.3 Summon (Wolves/Grizzly/Raven; V Maxroll summon-druid)
- Skills: 1 each Oak Sage, Werebear, Shock Wave, Carrion Vine; max Raven, Dire Wolf, Grizzly, Spirit Wolf, Heart of Wolverine; overflow into Lycanthropy. Passive strength is Fanaticism from Beast (not craftable? Beast is non-ladder and Berserker Axe works) (V Maxroll lists Beast).
- Not a no-gear bot build (summons die in Hell) but it is the human safe pick for Hell Cows (S).

## 5. Run list for casters (S/U unless sourced)

Maxroll's farming table (V `lightning-fury-amazon-guide` farming spots; same table in each build page): Act 1 The Countess (runes, Key of Terror), The Pit, Secret Cow Level (runes, runeword bases, charms/gems/jewels); Act 2 Arcane Sanctuary (runes, charms, jewelry), Summoner (Key of Hate); Act 3 Ruined Temple, Mephisto (uniques, sets), Act 4 Chaos Sanctuary and Diablo; Act 5 Eldritch the Rectifier, Shenk the Overseer, Pindleskin (all "gearing, gold, leveling"), Thresh Socket, Drifter Cavern, Halls of Anguish/Pain, Nihlathak (Key of Destruction), Worldstone Keep/Throne of Destruction, Baal. Ubers: Lilith -> Diablo's Horn, Duriel -> Baal's Eye, Izual -> Mephisto's Brain, then Tristram Mephisto/Diablo/Baal -> Hellfire Torch (V Maxroll, mechanic exists in 1.14d, keys from Countess/Summoner/Nihlathak per above).
- Mephisto run (Durance of Hate 3): Teleport from the WP (Durance 2) to Mephisto, kill him with Blizzard/Orb/Lightning, take the portal; drops are uniques/sets, with Hell Mephisto the standard rune/unique source for solo players (S).
- Andariel (Catacombs 4): short boss run; poison and cold immune risk depends on difficulty (U, check docs/difficulty.md V tables). Runes and uniques.
- Pindleskin: Red portal from Harrogath after the Anya quest, Nihlathak's Temple entrance; quick kill, often champion/unique minions, good for MF and gold (V Maxroll "Gearing, Gold, Leveling").
- Eldritch/Shenk: both near Harrogath; easy early pair for gear and levels (V Maxroll). Shenk is the Siege on Harrogath boss (Larzuk socket quest).
- Cows (Hell): Wirt's Leg + Tome of Town Portal in the cube; Baal quest must be finished in that difficulty (V MC). Crystal/Broad swords with Larzuk 4 sockets (V MQ). Dense crowds, great with Blizzard/Fissure/Nova; dangerous for squishy builds.
- Baal waves and Baal (Throne): the main experience and drops source; waves in Hell; teleporting casters do it fast (V ML-GEN).
- Chaos Sanctuary (Hell): max area level, seal bosses (Vizier, De Seis, Infector), then Diablo (V ML-GEN).
- Drops/MF: bosses are boss treasure classes; running Hell bosses yields the most high runes in solo play (S). Counters: avoid Mana Burn and Cursed or Lightning Enchanted uniques below clvl 75 (V docs/builds-endgame.md 7.1).

## 6. Pace targets (U)

Normal ends around clvl 30-33, Nightmare around 55-60, Hell at 75+ for Act 1-3; the project uses Normal 30 / NM 36-58 / Hell 64-85+ (V docs/builds-endgame.md section 2). Casters should have Enigma + Heart of the Oak + Rhyme/AP + FCR breakpoint before Hell Act 4.

## 7. Open questions
1. Does Frozen Orb have any cooldown in 1.14d (D2R has none? D2R guide says it does): U.
2. Whether 1.14d Static Field floors are 1/33/50 (same as D2R text; S from Arreat/Maxroll, V partial).
3. Do the bot's casters have enough gear for Hell Act 5 without Teleport (Enigma needs Jah+Ber, many Hell runs; U).
