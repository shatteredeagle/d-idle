# SME: leveling builds for all 7 classes, clvl 1-40 (classic LoD 1.14d, offline SP, softcore)

Owner: SMEKB-B. Feeds an unattended bot, so each class ends with a "no-respec bot line" (U, my adaptation). Companion: `builds-gear.md` (runewords, cube, vendors), `builds-endgame-*.md`.

Tags: **V** fetched this session (URL), **S** secondary or snippet, **U** unverified or from memory. D2R: marks D2R-only facts. Target is 1.14d.

Sources (all fetched 2026-10-29, V as read): Maxroll D2R Season 14 leveling guides, text-converted: `https://maxroll.gg/d2/guides/{amazon-leveling-guide,sorceress-leveling,necromancer-leveling,paladin-leveling,barbarian-leveling,druid-leveling,assassin-leveling}` (ML-AMA, ML-SOR, ML-NEC, ML-PAL, ML-BAR, ML-DRU, ML-ASN); `https://maxroll.gg/d2/resources/general-leveling` (ML-GEN); quests `https://maxroll.gg/d2/resources/important-quests` (MQ); cube `https://maxroll.gg/d2/resources/horadric-cube-recipes` (MC). Project: `docs/gear.md` builds table, `docs/leveler.md`, `docs/difficulty.md`. Skill mechanics in these pages match 1.14d except where flagged. Maxroll guides stop being 1.14d-safe wherever they name D2R items (Hustle, Sunder Charms, Warlock) or ladder-only runewords (Spirit); see `builds-gear.md` section 0 for the ladder finding.

## 1. Rules that hold for every class

- Skill points: 1 per clvl from 2 (so clvl 30 = 29 points) plus quest points: Den of Evil +1, Radament +1, Izual +2, per difficulty (V MQ), 12 total in three difficulties. A skill's level requirement gates it (skills.txt reqlevel; see `the d2tables cache gear.json`, V per project docs).
- Stat points: 5 per clvl. Human default: Vitality, with Strength/Dexterity only to wear gear or hit block, Energy only for pure casters in the first ~13 levels (ML-SOR, ML-NEC, ML-AMA, ML-ASN all do "Vit" or "Energy early then Vit", V). Quests: Lam Esen +5 stat points, Golden Bird +20 life, Prison of Ice +10 all res, per difficulty (V MQ).
- Elite packs (champions, uniques) give far more XP than loose monsters; Maxroll says to focus them (V ML-*). Do the Den of Evil first (clvl 1-5), then the Countess (Black Marsh, Forgotten Tower) for runes (V ML-*).
- Equipment steps repeated in every Maxroll guide (V): buy Tome of Town Portal and Tome of Identify, a Belt from Fara (Act 2), shop a 2-socket body armor and 2-socket helms at Charsi (for Stealth and Lore, must be non-magic), get Stealth at clvl 17, a Gemmed (Ral-socketed) helm for fire resist at 19, a Staff of Teleportation from Ormus (Act 3) at clvl 20-23 (wearable at 24; check only red-background staves), Larzuk socket, then be about clvl 19.8 at the end of Act 3 and 24-26 before Nightmare. Maxroll also notes Quest runes and respec tokens.
- Respec tokens: Akara after Den of Evil in each difficulty (V MQ "Akara ... Reset your Stats and Skill Points, once per difficulty"). The Maxroll plans respec once at ~clvl 19-38 (Amazon 38, Necro 26, Sorc 26, Paladin 19, Barb 31 via Cow, Druid 38, Assassin 38). **The bot cannot respec yet** (`docs/leveler.md`, `internal/leveler/build.go`), so section 3 gives a no-respec line per class.
- Over-level before each difficulty: Maxroll targets clvl ~38-40 for Nightmare and 65+ for Hell (V ML-DRU/ASN header), resists capped (75) is their wish; project gates use post-penalty +20 NM / +25 Hell (V `docs/builds-endgame.md` section 3).
- Early life/mana: Akara (Act 1) sells Minor/Light Healing and Mana potions; Lysander Act 2; Malah Act 5. A belt (12 slots from Belt/Heavy Belt in Act 2, V docs/items.md section 6) matters more than any armor at clvl 12. Strangling/Choking/Rancid gas potions in the weapon swap are the human AoE for poison-free classes (V ML-SOR, ML-NEC, ML-PAL, ML-BAR, ML-ASN): D2R shop item; in 1.14d they are found as drops only, S.
- Mercenary at the start: Rogue Scout from Kashya at Act 1 Quest 2 or clvl 8 (V M3 D2R, 1.14d: hireable after Blood Raven or clvl 8, S): bow, Fire Arrow or Cold Arrow variant. Act 2 Desert Merc (Greiz): Prayer/Defiance/Blessed Aim in Normal, Thorns/Holy Freeze/Might in Nightmare and Hell (V M3, project merc.md). Common human choice: Act 2 Defiance or Prayer in Normal, switch to Might in Nightmare. Rogue dies often, 1.14d revive price depends on level (V M3: up to 50,000 gold).

## 2. Per-class Maxroll plans (V) and what they teach

Notation: "Lv a-b: skill" = put the points gained at those clvls into the skill (V). "Den/Radament/Izual: x" = quest points.

### 2.1 Amazon (ML-AMA)
Plan: Fire Arrow first, then Poison Javelin (Normal) and Exploding Arrow (bow) until the respec at 38 into Lightning Fury/Charged Strike.
- Normal Act 1: Lv2-5 Fire Arrow, Den: Magic Arrow, Lv6-10 Poison Javelin, Lv11 Multiple Shot, Lv12-13 Exploding Arrow. Stats: Lv2-11 all Vitality, Lv12 3 Dex 2 Str, Lv13 Str.
- Normal Act 2-5: Lv14 Exploding Arrow, Radament Fire Arrow, Lv15-20 Exploding Arrow, Izual Fire Arrow, Lv21-26 Exploding Arrow. Stats: Lv14-16 Energy, Lv17-26 Vitality.
- Nightmare Act 1-3: Lv27 Exploding Arrow, Den Fire Arrow, Lv28-31 Exploding Arrow, Lv32-38 Fire Arrow, Radament Fire Arrow. Stats Vitality.
- Respec at clvl 38 (Act 4 NM): Pierce 5, Dodge/Avoid/Evade 1 each, Lightning Fury 9, Charged Strike 19, then maximize Lightning Fury, Charged Strike, Lightning Bolt, Power Strike. Dex to 50% block.
- Gear steps: Hunter's Bow (attack speed) with 2 Chipped Rubies (+fire damage for Exploding Arrow); keep Poison Javelin on swap; Stealth 17; Lore cap; Bone Shield for Rhyme; Heavy Gloves of Readiness (IAS), Mammoth belt, Haste boots in NM; Javelins of Swiftness for the 50% IAS Charged Strike breakpoint; Wand of Lower Resistance (clvl 30-35) for bosses. Fighting notes: Poison Javelin laid in a line, lure monsters through the cloud; Exploding Arrow is mana hungry (V).
- What it teaches: points go into ONE attack skill from clvl 12 on (Exploding Arrow to 20ish); Maxroll never mixes bow and javelin synergies beyond Fire Arrow. Exploding Arrow stalls at Fire Immune (NM Act 4), which is why the respec exists.
- 1.14d/classic notes (U, from memory): the classic leveling standard for Amazon is javelin (Jab -> Power Strike/Charged Strike with Lightning Bolt/Fury at 24) or bow (Magic Arrow -> Fire Arrow/Multiple Shot -> Strafe/Guided). Penetrate/Pierce are not needed early.

### 2.2 Sorceress (ML-SOR)
Plan: Charged Bolt -> Static Field/Frost Nova -> Nova from 13 -> respec at 26 into Blizzard line.
- Lv2-4 Charged Bolt, Den Charged Bolt, Lv5 saved, Lv6 Frost Nova + Static Field, Lv7-9 Static Field, Lv10 Telekinesis, Lv11 Frozen Armor, Lv12-13 Nova. Stats: Lv2-10 Vit, Lv11-13 Str.
- Lv14-17 Nova, Radament saved, Lv18 Nova + Teleport, Lv19-20 Nova, Izual Static Field, Lv21-26 Nova. Stats Lv14-16 Vit, Lv17-21 22 Str 3 Vit, Lv22-26 Vit.
- Respec at 26: Static Field 5, Teleport 1, Frozen Armor 1, Ice Blast 7, Glacial Spike 9, Blizzard 3 (save 1 so at Lv30 Cold Mastery + Blizzard), then Blizzard. Stats: Strength to wear gear, rest Vitality.
- Gear: Short Staff with +2/+3 Charged Bolt from Akara (about 3.4k/5.5k gold); Faster Cast Rate jewelry; Wand of the Apprentice; Gemmed Cap; Stealth 17; Staff of Teleportation at 24 in swap; Spirit sword ladder-only so not available offline (see gear file); Wand of Lower Resist at 30-35.
- Notes: Charged Bolt shotguns at close range; Static Field short radius at level 1-3; beetles do not trigger their bolts if hit by Static Field; Nova is mana-expensive below 13; do not use Nova at 12 (V).
- Classic (U): the old standard was Fire Bolt -> Fire Ball/Fire Wall or Frozen Orb/Blizzard; cold from clvl 2 is workable (Ice Bolt/Ice Blast, Frozen Armor) but slow before Blizzard at 24.

### 2.3 Necromancer (ML-NEC)
Plan: Teeth (spell) in Normal with Clay Golem/Iron Maiden against bosses -> Corpse Explosion/Bone Spear -> respec at 26 into Bone Spear/Bone Prison with Corpse Explosion.
- Lv2-5 Teeth, Den Teeth, Lv6 Clay Golem, Lv7-10 Teeth, Lv11 Amplify Damage (or Teeth if you own a +Teeth wand), Lv12 Iron Maiden (or Teeth), Lv13 Teeth. Stats: Lv2-3 Energy, Lv4-11 Vit, Lv12-13 Str.
- Lv14 Bone Armor, Radament Bone Wall, Lv15-16 Bone Wall, Lv17 Corpse Explosion, Lv18-20 Bone Spear, Izual Corpse Explosion, Lv21-26 Corpse Explosion. Stats: Lv14-16 Vit, Lv17-21 22 Str 3 Vit, Lv22-26 Vit.
- Respec at 26: Corpse Explosion 10, Bone Spear 9, Bone Prison 3 (optional Clay Golem), then Bone Spear, Bone Prison, Bone Wall, Teeth, Amplify.
- Gear: wand with +2/+3 Teeth from Akara (3.4k/5.5k), rerolled via cube (1 magic wand + 3 chipped gems, buy wands at <=750 gold, V MC), Bone Wand with +Bone Spear (must be a Bone Wand; White later), Necromancer shield with +skills, Stealth, Gemmed helm, Staff of Teleportation, rings/amulets with 10% FCR (75% breakpoint with all three slots).
- Notes: Teeth does not shotgun; use gas potions and Clay Golem as meat shield; Iron Maiden plus golem on bosses reflects damage (V); Wave 2 of Throne of Destruction (Unravelers magic immune) is the build's wall (V).
- 1.14d/classic (U): Summoner is the safest no-gear leveler: Raise Skeleton + Skeleton Mastery + Clay/Blood Golem + Amplify Damage + Corpse Explosion; minions tank and kill; mana low (Skeletons cost mana to raise only). Project default is exactly this (V `docs/gear.md`).

### 2.4 Paladin (ML-PAL)
Plan: Might/Sacrifice -> Holy Fire (Normal Act 1-3) with Charge/Zeal -> respec at 19 into Blessed Hammer.
- Lv2 Might, Lv3 Sacrifice, Den Resist Fire, Lv4-5 Resist Fire, Lv6-9 Holy Fire, Lv10 Smite, Lv11 saved, Lv12 Charge + Zeal, Lv13 Holy Fire. Stats: Lv2-5 Vit, Lv6 1 Dex, Lv7-11 Vit, Lv12-13 Energy.
- Lv14-19 Holy Fire and Radament Resist Fire. Respec at 19: Blessed Hammer 2, Charge 1, Concentration 1, Blessed Aim 8, Vigor 2, then Lv20-36 Blessed Hammer, Izual Vigor + Blessed Aim.
- Notes: Holy Fire fails vs fire resistance after Act 3; Steel scimitar (Tir El) is an optional clvl 13 upgrade; Charge costs lots of mana; scepters rerolled in the cube for +skills; keep Paladin shields with sockets/resists (V ML-PAL).
- Classic (U): Zeal Paladin (Sacrifice -> Zeal at 12, Fanaticism at 24) is the classic no-respec lineage; the 1.14d plain starting aura is Might/Holy Fire.

### 2.5 Barbarian (ML-BAR)
Plan: Bash -> Double Swing (dual scepters) -> Frenzy -> War Cry at 30 after a Cow-level respec at 31.
- Lv2-5.. Bash (to ~15), Double Swing from 6 (dual-wield scepters/flails), Howl at Radament, Taunt Lv16, Leap Attack + Battle Cry Lv18, Mace Mastery Lv19, Increased Stamina Lv20, Shout 21, Double Throw 22, Frenzy + Battle Orders Lv24-25. Stats: Lv2-7 Vit, Lv8-10 Dex, Lv11-12 Str, then Vitality.
- Gear: Malice flail at clvl 15, Stealth 17, Nadir/Gemmed helm, Khalim's Will used as a weapon while leveling (do not smash the orb yet), Act 1 Fire Rogue with Gemmed Hunter's Bow; Wyrm/Sounding Javelin from Malah; Staff of Teleportation at 29.
- Respec 31: Leap, Leap Attack, Concentrate, Whirlwind 1 each; Mace Mastery, Increased Speed, Natural Resistance 1; Battle Cry 9, Battle Orders 8, Battle Command 1, War Cry 2; Str 13 Dex 40 rest Vit; farm Eldritch/Shenk/Pindleskin and Cows for Crystal Swords.
- Barbarian weaknesses: no AoE early (gas potions are the human answer, V ML-BAR), mana is low, melee in Normal Act 2-3 is fragile (flayers).

### 2.6 Druid (ML-DRU)
Plan: Firestorm -> Molten Boulder -> Fissure (no respec until Mephisto in NM) -> Tornado/Hurricane at 38.
- Lv1-5 Firestorm, Lv6-7 Molten Boulder, Lv8-11 Firestorm, Lv12-13 Fissure, Lv14-26 Fissure, Lv27 Firestorm, Lv32-37 Firestorm, Lv38 Volcano. Stats: Lv2-11 Vit, Lv12-13 Str, 22 Str at Lv22-25.
- Respec before Mephisto (clvl 38): Tornado 15, Hurricane 9, Cyclone Armor 14, Summon Grizzly 1 (plus prerequisites) (V).
- Notes: Firestorm hits big monsters with all 3 projectiles up close; Molten Boulder explodes on large monsters; Fissure good in groups; pelts with +skills, Leaf staff (Tir Ral) in Normal.
- The fire line is the classic druid leveling (V ML-DRU); the bot project mirrors it (V docs/gear.md).

### 2.7 Assassin (ML-ASN)
Plan: Fire Blast + Burst of Speed -> Wake of Fire (12+, NM Act 3) -> respec 38 into Lightning Sentry/Death Sentry.
- Lv2-5 Fire Blast, Lv6-7 Burst of Speed, Lv8 Fire Blast, Lv9-11 Burst, Lv12-13 Wake of Fire, then Wake of Fire to 31, Fire Blast 32-38. Stats: Lv2-3 Energy, Lv4-10 Vit, Lv11 Str, then Vit.
- Respec 38: Cloak of Shadows 1, Mind Blast 1, Burst 3, Lightning Sentry 15, Death Sentry 9, Charged Bolt Sentry 10; lay 5 Lightning Sentries on one spot to focus fire (V).
- Gear: dagger from Charsi, a Leaf staff in NM? (swap), Death's Hand/Death's Guard and Twitchthroe (V, sets/uniques), Wand of Lower Resist.
- Notes: Wake of Fire is "set and forget" at 12; traps need open spots; place 2 traps per pack (V).

## 3. No-respec bot lines (U; my adaptation, follows project conventions in docs/gear.md and docs/builds-endgame.md)

The bot has no respec, so pick a build whose attack skill is unlocked early and never abandoned. Existing project choices (V `docs/gear.md`): Necromancer summoner; Amazon bow (Magic/Fire Arrow, Multiple Shot 11, Exploding Arrow 12); Druid fire (Firestorm, Molten Boulder, Fissure at 12); Assassin Fire Blast -> Wake of Fire; Paladin plain table; Sorc via `sorceress_blizzard_endgame90` (Ice line from clvl 2).

| class | stats | clvl 1-12 | clvl 13-30 | skip early |
|---|---|---|---|---|
| Amazon (bow) | Dex for bow needs, Vit, Str 20ish by 13 | Magic Arrow 1, Fire Arrow at 2-5, Multiple Shot 6-11, Exploding Arrow from 12 | Exploding Arrow to ~16, Guided Arrow, Valkyrie later; arrows stack 350 | Pierce/Penetrate/Critical Strike before 30; Strafe needs Dex |
| Necromancer | Energy 2-3 pts then Vit; Str for gear | Raise Skeleton + Skeleton Mastery, Clay Golem 6, Amplify 11, Iron Maiden 12 | Corpse Explosion from 15-17, Bone Armor 14, Revive after 24 | Bone Spear (needs Teeth+mana), Poison Dagger, Poison Nova until Nightmare |
| Sorceress | Vit; Energy 0; Str 43 by 24 | Ice Bolt, Frozen Armor, Telekinesis skip, Teleport 18 | Glacial Spike, Blizzard 24, Cold Mastery 30 | Nova (mana), Fire Wall before Fire Mastery; Energy Shield for a bot costs mana |
| Paladin | Str 43, Vit; Dex for block | Might, Smite, Holy Bolt, Charge | Blessed Hammer 18+, Holy Shield 24, Concentration | Sacrifice (self-damage), Zeal below 12, Redemption/Meditation early |
| Barbarian | Str for gear, Vit | Bash, Double Swing at 6 (needs two one-handers), Howl/Taunt | Frenzy 24, War Cry 30, Battle Orders | Whirlwind (needs Dex 40+), Throwing line, Berserk early |
| Druid | Vit, Str for staff | Firestorm 1-5, Molten Boulder 6-7, Raven 1 | Fissure 12-26, Cyclone Armor, Volcano 38 | Hunger, Rabies, Shock Wave, werebear line before 30 (stat heavy) |
| Assassin | Energy 2 pts then Vit | Fire Blast 2-5, Burst of Speed 6, Claw Mastery 1 | Wake of Fire 12-31, Shock Web/Lightning Sentry later | Dragon Claw/Talon (claws needed, Dex), Cloak of Shadows spam |

## 4. Early mana / life / potion problems and how humans solve them

- Mana: casters buy mana potions in bulk (V ML-SOR: "lots of Mana"), wear a Gemmed (Tir sockets give +2 mana per kill, Eth regen) helm; Lore, Spirit later. Amazon Exploding Arrow and Barb leaps eat mana; Frenzy/gas potions as alternates. Mana recovery in town is free, use the well (U).
- Life: Vitality first, life potions (Akara), Healing potions to stack on belt; Rhyme/Smoke later; mercenary as a tank; Clay Golem/skeletons as distraction (V ML-NEC).
- Potions: belt first (12-16 slots), then three or four columns: 2 health + 1 mana for casters, 3 health + 1 mana for melee (U, project potions-belt.json V). Buy a Tome of Town Portal early (scroll economy). Keep rejuvenation from the cube (3 health + 3 mana + chipped gem, V A2).
- Stamina: buy Stamina potions early (Akara) because melee classes run out walking (V ML-BAR "stamina potions").
- Resists before Nightmare/Hell: V ML-DRU/ASN: "capped (75) before entering"; realistic target +20/+25 after penalty (project).

## 5. Skills that are traps early (S/U)

Skills that scale with points into something you do not have yet: Amazon Strafe, Penetrate, Critical Strike; Sorc Energy Shield, Thunder Storm, Hydra before Fire Mastery, Meteor; Necro Poison Nova, Bone Prison early, Dim Vision, Weaken; Paladin Redemption, Salvation, Meditation, Conversion; Barbarian Whirlwind with <40 Dex, Throw skills, Berserk before 30; Druid Armageddon, Summon Vine, Heart of Wolverine early, Lycanthropy before Werewolf; Assassin Dragon Talon without claws, Shadow Master, Mind Blast spam. D2R: all the same except new skills do not exist in 1.14d (Warlock class).

## 6. Hybrid vs pure

- Human consensus (S, ML-* plus general knowledge): pure builds scale best in Hell; hybrids are good for leveling because 1-2 skills must carry you across three difficulties. Maxroll's own leveling is a pure-ish "one attack skill per phase". Warrior-type skill splits (Amazon bow+jav; Paladin aura+attack; Necro CE+bone) cost 20-30% damage but give immunity coverage (V `docs/builds-endgame.md` section 4).
- For a bot: pick the build that never needs a second damage type until Hell; add a physical or magic secondary (Fire Blast, Amplify Damage, Magic Arrow, merc Might) only for immunities (see builds-endgame).

## 7. What to tell a bot (U)

1. Spend points in this order of preference: attack skill > synergy for attack > buff aura > mobility > everything else (the project tables already do this, V docs/gear.md).
2. Keep a staff of teleport for the swap at clvl 24 when the class does not own Teleport (V ML-*); the bot has no weapon swap, so it uses Enigma later (`docs/builds-endgame.md`).
3. Don't take skills with prerequisites not needed by the main line (banked points, project convention V).
4. Leveling spots: Den of Evil, Countess, Cows (Normal) until ~25, Tristram, then Baal waves from 25 (V ML-GEN, MP only for waves in groups); Chaos/Baal Hell (V ML-GEN). Solo: /players X in SP raises monster life and XP (V ML-GEN).

## 8. Open questions

1. Respec: Akara's token is given after Den of Evil per difficulty (V MQ D2R); in 1.14d the same, S. The bot's plan to avoid it is OK but costs 10-20% power for the Paladin and Sorceress.
2. Gas potions in 1.14d are loot, not vendor items (S/U): the bot should not plan on them.
3. Which of the Maxroll first-act skill orders hold up without Teleport and potions? Unknown without a live test (project live results in docs/g1-log.md).
