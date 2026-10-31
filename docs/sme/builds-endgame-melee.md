# SME: endgame builds, melee classes (Paladin, Barbarian, Assassin), classic 1.14d softcore single player

Owner: SMEKB-B. Companions: `builds-endgame-casters.md`, `builds-endgame-amazon-farming.md`, `builds-gear.md`, `builds-leveling.md`; bot tables in `docs/builds-endgame.md`.

Tags: **V** fetched this session or project data (URL named), **S** secondary, **U** memory/unverified. D2R: marks D2R-only facts. Target 1.14d offline. **Ladder-only runewords (Spirit, Insight, Fortitude, Infinity, Grief, Death, Last Wish, Obedience, Pride, Oath, Dragon, Phoenix, Dream, Edge, Faith, Harmony, Ice, Brand, Wrath, Destruction, Rift, Lawbringer, Widowmaker, Winter, Voice of Reason) are not craftable in vanilla offline 1.14d** (`builds-gear.md` section 0); Maxroll's melee guides are full of them (Grief Phase Blades, Last Wish, Fortitude, Obedience, Oath, Infinity). Each build below lists the SP-offline substitutes.

Sources fetched 2026-10-29 (D2R Season 14 text, V as read): `https://maxroll.gg/d2/guides/{blessed-hammer-paladin,zeal-paladin,smite-paladin,whirlwind-barbarian-guide,frenzy-barbarian,double-throw-barbarian-guide,war-cry-barbarian,lightning-sentry-assassin,whirlwind-assassin,wake-of-fire-assassin}`; `https://maxroll.gg/d2/resources/{breakpoints-animations,mercenary,important-quests}`; project `docs/builds-endgame.md` (breakpoint tables, S).

## 1. Shared facts for melee

- Block needs Dex: Maxroll targets 75% block with a block shield (Holy Shield active for Paladin) (V zeal/smite guides). Big stats: Strength to wear (Fortitude Dusk Shroud 77 is ladder, so Chains of Honor/Lionheart/Smoke/Treachery needs lower Str; project: Mage Plate 55, Dusk Shroud 77, Archon Plate 103, Sacred Armor 232, V gear.md armor data).
- Breakpoints (S docs/builds-endgame.md): Paladin FCR 75/125, FHR 0/7/15/27/48/86/200 (swords) and for spears/staves 0/3/7/13/20/32/48/75/129/280; Barbarian FCR 105, FHR 27-48; Assassin FCR 65 (102 with Wake of Fire/Lightning Sentry swaps), FHR 48.
- Weapon speed and IAS: each weapon base has its own IAS breakpoints (V M-class breakpoint page for D2R; same in 1.14d, S). A dual-wield Barb or Assassin (claws) uses the same table with offset.
- Life leech and physical immunity: melee builds need Life Tap/Leech and a way to handle physical immune packs (ghost/wraith swarms): Bone Break and Sunder are D2R-only; in 1.14d use Decrepify (Reaper's Toll merc), Lower Resist/Might merc, a second damage skill, or skip the pack (S/U).
- Mana Burn and Lightning Enchanted unique monsters (Hell) are the melee killers (V project avoid table in docs/builds-endgame.md 7.1).

## 2. Paladin

### 2.1 Blessed Hammer (Hammerdin) — the standard Hell farmer, magic damage
- Skills (V Maxroll blessed-hammer-paladin via project): 1 each Holy Shield, Cleansing, Redemption, Fanaticism; max Blessed Hammer, Vigor, Concentration, Blessed Aim, Resist Lightning. Concentration is the damage aura, Vigor speed/stamina.
- Stats: Str to wear shield/armor (43 for Spirit sword is out), Dex ~70 for block with Holy Shield, rest Vitality; no Energy. Attack aura Concentration lifts hammer damage.
- Breakpoints: FCR 75 then 125 (V). FHR 48-86.
- Gear offline-safe: Enigma armor (Teleport), Chains of Honor/Lionheart/Smoke early, Heart of the Oak in a mace (V P1 base types staff or mace; one-handed mace plus shield is possible, U), Andariel's Visage, Harlequin Crest, Spirit Monarch -> Rhyme/AP/Splendor/Sanctuary instead; Exile (Vex Ohm Ist Dol, paladin shield, not ladder-listed in A4 so allowed offline, S; effects U), Maxroll names Hellwarden's Will etc. (D2R-ish, S); Call to Arms swap weapon.
- Resists: Lightning is the big threat: Resist Lightning 20 points, 75 cap target; mana via Meditation not available (Insight). Cannot be Frozen from Cham/Rhyme; a Ruby jewel of fervor on the merc (V Maxroll).
- Farming: Hammerdin is the Hell workhorse: Mephisto, Andariel, Pindleskin, Eldritch/Shenk, Cows, Baal waves, Chaos Sanctuary, Ancient Tunnels. Hammers need open ground, so it does badly in corridors (V project note). Immunities: only magic immune packs (V "almost no immunes"): unravelers, a few uniques.
- D2R note: guide variants may rely on Sunder Charms (D2R only); the classic Hammerdin is Blessed Hammer + Concentration + Vigor + Holy Shield + Enigma (U).

### 2.2 Zeal (V Maxroll zeal-paladin)
- Skills: 1 each Smite, Charge, Cleansing, Vigor; max Zeal, Fanaticism, Sacrifice, Holy Shield; remainder Defiance/Resist Lightning. Stats: Str to wear, Dex to 75% block, rest Vit.
- Gear (V Maxroll names): Grief Phase Blade (ladder, skip). SP substitutes (S, A4 lists them as non-ladder; effects U): Beast (Ber Tir Um Mal Lum, axe/hammer/scepter), King's Grace, Honor, Hand of Justice (Holy Fire), Doom (Holy Freeze), Eternity, Breath of the Dying, Kingslayer, Crescent Moon, Fury. Helm: Andariel's Visage, Guillaume's Face. Armor: Chains of Honor, Enigma. Shield: Ancient's Pledge, Sanctuary, Exile. Merc: Might (V).
- Pros/cons: top melee clear for Cows; vulnerable to physical immune packs (Bone Break D2R only), dies to Mana Burn.
- 1.14d note (U): classic Zeal+Fanaticism leveling builds are standard; Fanaticism increases attack speed and Zeal hits 5x, so IAS and AR matter.

### 2.3 Smite (V Maxroll smite-paladin)
- Skills: 1 each Cleansing, Vigor, Salvation; max Holy Shield, Fanaticism, Smite, Resist Lightning, Zeal. Uses a shield for Smite, Grief for damage (ladder). Stats: Str/Dex for block, rest Vit. Strong boss killer, weak vs. crowds; Mercenary Might or Reaper's Toll (V).
- SP: replace Grief with Doom/Breath of the Dying/Fury if you can make them, or a high-damage unique (Death Cleaver? U).

### 2.4 Other (U): Fist of the Heavens (Holy Shield, Fist of Heavens + Conviction), Holy Fire aura Paladin (Hand of Justice), Dream (ladder).

## 3. Barbarian

### 3.1 Whirlwind (V Maxroll whirlwind-barbarian-guide)
- Skills: 1 each Find Item, Grim Ward, Battle Command, Battle Cry, Increased Speed, Natural Resistance, Leap, Leap Attack; max Whirlwind, Blade Mastery (or Polearm Mastery by weapon), Battle Orders, Find Potion, Berserk. Stats: Str/Dex to wield dual weapons or a big weapon, rest Vit.
- Weapons (V Maxroll): dual Grief Phase Blades (ladder) or Beast Berserker Axe + Berserker Axe or Breath of the Dying/Last Wish/Obedience (mostly ladder). **SP substitutes (S): Beast, Breath of the Dying, Doom (Holy Freeze), Fury (Jah Gul Eth, V P1), Eternity, Famine, Kingslayer.** Armor: Enigma/Chains of Honor/Lionheart; Fortitude is ladder.
- Breakpoints: FCR 105 (V), FHR 27-48, IAS per weapon.
- Danger: physical immune packs, Mana Burn; WW is cast like "move over targets" so the bot struggles; project picked War Cry for the bot instead (V docs/builds-endgame.md).

### 3.2 Frenzy (V Maxroll frenzy-barbarian)
- Skills: 1 each Battle Cry, Find Item, Grim Ward, Shout, Battle Command, Increased Speed, Natural Resistance, Leap, Leap Attack; max Frenzy, Double Swing, Blade Mastery, Battle Orders, Berserk. Frenzy gives IAS and speed stacks. Dual-wield Grief ladder; SP: Fury/Beast/Doom/Kingslayer/Honor.

### 3.3 Double Throw (V Maxroll double-throw-barbarian-guide)
- Skills: 1 each Increased Speed, Iron Skin, Natural Resistance, Howl, Shout, Battle Command, Grim Ward, Frenzy; 10 points each Find Item and Battle Orders; max Throwing Mastery, Double Swing, Double Throw, Find Potion. Stats: Dex 200-300 for damage (V), Str for gear, rest Vit. Weapon: throwing weapons (stacks); high Dex; needs repair of throwing stacks (S). Very good boss killer; item-dependent.

### 3.4 War Cry (the project's chosen build) — see `docs/builds-endgame.md` 7.6. Skills: War Cry, Battle Cry, Taunt, Battle Orders, Howl; stat Str 77 for Dusk Shroud (Fortitude is ladder; use Chains of Honor or other armor). Needs massive mana (Insight is ladder), Cannot farm act bosses efficiently (V Maxroll cons).

### 3.5 Leveling/farming runs: Eldritch + Shenk + Pindleskin for Barbarians (V ML-BAR), Cows for Crystal Swords, Baal waves; Barb buffs (Battle Orders, Battle Command, Shout) let him support parties; solo he uses Find Item (Find Potion) for potions.

## 4. Assassin

### 4.1 Lightning Sentry (Trapsin) — project's pick (V docs/builds-endgame.md 7.5; Maxroll lightning-sentry-assassin)
- Skills: 1 each Burst of Speed, Fade, Cloak of Shadows, Mind Blast, Weapon Block; max Lightning Sentry, Shock Web, Charged Bolt Sentry, Death Sentry, Fire Blast. 5 sentries stacked in a spot, then Death Sentry on corpses.
- Stats: Str 57/Dex 57 for claws (Quhab, Chaos claws V gear data), rest Vit. FCR 65-102 depending on swap, FHR 48.
- Gear offline-safe: Treachery (Shael Thul Lem; IAS 45, Fade, +assassin skills; V P1), Chaos (Fal Ohm Um claw, V P1), Enigma, Lore, Rhyme/AP, Griffon's Eye (-lightning res, V assassin.md), Magefist, Arachnid Mesh; Infinity is ladder (skip); Wand of Lower Resistance for lightning immunes (V Maxroll). Merc: Holy Freeze + a rare polearm.
- Farming: Mephisto, Andariel, Pindleskin, Cows with trap clusters; Chaos Sanctuary boss kills (Death Sentry with corpses if immune, V ML-ASN). Lightning immune is the hard wall (V).

### 4.2 Wake of Fire (V Maxroll wake-of-fire-assassin)
- Skills: 1 each Death Sentry, Cloak of Shadows, Mind Blast, Shadow Master, Burst of Speed, Fade; max Wake of Fire, Fire Blast, Wake of Inferno, Death Sentry, Lightning Sentry. FCR 102 with Spirit Monarch (ladder, not craftable), Magefist, Arachnid Mesh, caster crafted amulet, 2 rare rings (V). Substitute Spirit with Lore + Rhyme etc. Claws: Cunning Greater Talons +skills (a crafted claw), Phoenix is ladder.
- Fire immune packs are the wall (the largest immune group, V project).

### 4.3 Whirlwind Assassin (V Maxroll whirlwind-assassin)
- Skills: 1 each Cloak of Shadows, Mind Blast; max Claw Mastery, Weapon Block, Fade, Death Sentry, Fire Blast only to 18, rest Venom. Claw runewords: Chaos (Fal Ohm Um) is the claw-only word; melee words such as Fury (Jah Gul Eth) also fit claws (V A1 'melee weapons' includes hand-to-hand, S). Enigma for Teleport. Great demon killer (S).

### 4.4 Other (S/U): Dragon Talon (kicks, with Dragon Claw), Phoenix Strike, Blades of Ice/Phoenix variants (D2R meta); Dragon Talon works with any boots and is a leveling favorite (S).

## 5. Run list for melee (S/U unless sourced)

- Same list as casters (`builds-endgame-casters.md` section 5): Mephisto, Andariel, Pindleskin, Eldritch/Shenk, Cows, Baal waves, Chaos, Pit. Melee builds without Teleport walk to Mephisto and Andariel: acceptable, Enigma makes it fast.
- Hammerdin: Chaos Sanctuary and Baal waves are the best leveling/drop runs (V Maxroll Hammer pages).
- Barbarians: Frenzy/WW melee Cows and Baal waves; throw barb (Double Throw) bosses.
- Assassin traps: boss runs and Cows if trap layout allows.
- Magic find: sockets Ist runes/jewels in Gemmed items, Ist in swap crystal swords (V Maxroll summon druid mention: "Gemmed Crystal Sword (6x Ist Runes)"). MF gear is for later; cap diminishing for unique chance ~250% (S).

## 6. Open questions
1. Exile (Vex Ohm Ist Dol, paladin shield) and Doom (Hel Ohm Um Lo Cham) availability offline: shown non-ladder by A4 (S), verify in-game.
2. The Act 2 merc cannot wield swords/axes (V M3), so Grief/Beast are for the player only.
3. Dex/Str requirements of high Hell armors vs the +/- of the bot's stat plan (V gear armor table).
