# Mercenary skills and aura levels per act and difficulty (classic LoD 1.14d)

> Gap 2 of the 2026-11-27 SME reviews. Tags: V fetched (URL), S secondary, U unverified. **D2R** = D2R-only. Complements `mercenary.md` (hire levels, prices, mechanics). Aura NAMES are V for LoD; aura LEVELS are only available from a D2R data file (S, section 3).

## 1. Which skill per act and difficulty (LoD 1.14d)

| Act | Merc | Normal | Nightmare | Hell | Source / tag |
|---|---|---|---|---|---|
| 1 | Rogue Scout | Fire: Fire Arrow + Inner Sight; Cold: Cold Arrow + Inner Sight | same | same | V Basin https://www.theamazonbasin.com/wiki/index.php/Rogue_Scout ("Fire Arrow, Inner Sight (LoD)"; D2R adds Exploding/Freezing Arrow) |
| 2 | Desert Mercenary | Combat Prayer; Defensive Defiance; Offensive Blessed Aim | Combat **Thorns**; Defensive **Holy Freeze**; Offensive **Might** | Combat Prayer; Defensive Defiance; Offensive Blessed Aim (**no Might in Hell**) | V Arreat hirelings (Wayback) https://web.archive.org/web/2010/http://classic.battle.net/diablo2exp/basics/hirelings.shtml; V Basin https://www.theamazonbasin.com/wiki/index.php/Desert_Mercenary ("Attributes (LoD)") |
| 3 | Iron Wolf (Eastern Sorceror) | Fire: Inferno + Fire Ball; Cold: Ice Blast, Glacial Spike, Frozen Armor; Lightning: Charged Bolt, Lightning | same | same | V Basin https://www.theamazonbasin.com/wiki/index.php/Iron_Wolf (D2R: Fire Bolt/Fire Ball/Enchant, Chilling Armor, Static Field) |
| 5 | Barbarian | Bash, Stun (and Battle Cry in D2R data); no aura | same | same | S: our MPQ-extract note in `mercenary.md` section 1; D2R file differs (section 3) |

The aura is always the merc's one passive aura and is shared with the party and minions (V Arreat: "most party friendly"). The three Desert aura slots are chosen at hire (the hire window lists the type), so a bot hiring a "Might merc" must do it in Nightmare.

**Disagreement (resolved by version):** Basin page text says "In D2R ... mercenaries in Nightmare and Hell can be hired with all 6 aura types"; Maxroll (D2R) says the same (cited in `mercenary.md`). Arreat and Basin "Attributes (LoD)" agree there is no Hell Might in LoD. Treat D2R-only: Might / Holy Freeze / Thorns in Hell, any aura in Normal NM mix.

## 2. Aura levels

Aura levels scale with merc level, not player skills (S, `mercenary.md`). Lowest hire level per block is 9 (Normal), 43 (NM), 75 (Hell) in the expansion hire table (S `mercenary.md`).

## 3. Level columns from the D2R data file (S, D2R, version caveat)

Source: blizzhackers/d2data `json/hireling.json`, "Updated for the 2.5 patch" (D2R) https://raw.githubusercontent.com/blizzhackers/d2data/master/json/hireling.json (S: this is D2R 2.5 data, not 1.14d). Columns are `Level` (skill level at the lowest hire level) and `LvlPerLvl` (growth per merc level; unit not verified, U). The Normal/Nightmare Desert rows match the Arreat aura names, so I believe the Normal and NM levels are the LoD ones (U).

| Desert aura | Hire window | Skill level at base (`Level`) | LvlPerLvl | Notes |
|---|---|---|---|---|
| Prayer | Normal L9 / NM L43 / Hell L75 | 3 / 11 / 18 | 7 / 7 / 0 | Normal+NM |
| Defiance | same | 3 / 11 / 18 | 7 / 7 / 0 | |
| Blessed Aim | same | 3 / 11 / 18 | 7 / 7 / 0 | |
| Thorns | NM L43 / L75 | 11 / 21 | 10 / 10 | LoD NM only; D2R also Hell |
| Holy Freeze | NM L43 / L75 | 6 / 16 | 10 / 0 | LoD NM only |
| Might | NM L43 / L75 | 7 / 15 | 8 / 8 | LoD NM only (D2R also Hell) |

A second block in the same file (`Version 0`, hire levels 9 / 31 / 55) gives aura levels 3 / 10 / 18 (Prayer, Defiance, Blessed Aim) and Thorns 5 / 13, Holy Freeze 6 / 14, Might 7 / 13 at 31 / 55; it matches the "first block" noted in `mercenary.md`. Which block 1.14d's hire window uses is U (`mercenary.md` section 2: hire level follows the player in 1.14d).

Other mercs, same file (D2R, S): Rogue Inner Sight L1 (hire L3), Fire/Cold Arrow L1; Iron Wolf Fire Ball L4 / Lightning L3 / Glacial Spike L1 at hire L15 (D2R skill set, not 1.14d), Barbarian Bash 8 / Stun 6 at hire L28. Treat as indicative only.

## 4. Gaps
- 1.14d aura levels per merc level were not extractable from a primary 1.14d source (Arreat does not publish them; our MPQ extract from `mercenary.md` is not present in this checkout). Read them live from the hire window or the extract before using any number above in code.
- Barbarian skill list for LoD (Bash/Stun, "Frenzy and Iron Skin" 1hs variants appear in the D2R file) is not confirmed for 1.14d.
