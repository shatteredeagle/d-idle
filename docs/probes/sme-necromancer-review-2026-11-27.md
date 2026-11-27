# SME review: Necromancer data built by card necroepic (2026-11-27)

> status: current. Advice only, no code run, no game. Tags: SRC (read in this repo, path given), EXP (veteran experience, classic 1.14d, not looked up), UNSURE. The opt-in table `necromancer_summoner_ce1` is NOT in main at bbbf771e5 (grep finds no such table or file): the CE5 vs CE1 verdict below is made from the `necromancer_summoner_normal30` rows it would edit.

## Verdict per item

| Item | Verdict | Reason |
|---|---|---|
| `necromancer_summoner_normal30` | Sensible, with one trim | Skeleton + Mastery first, golem at 6, Amplify at 11, mages at 12 is the order a human uses (SRC docs/sme/class-necromancer.md section 0; EXP). Prereqs all hold: Teeth before Corpse Explosion, Raise Skeleton before Mastery and Mage (SRC skills table in the same file). CE has 5 points by clvl 30: too many (change 3). |
| `necromancer_leveling_normal30` (bonespear) | Depends, two defects | Bone Wall x3 (rows 15 to 16) and Iron Maiden x1 (row 12) are guide points that only pay off after the guide's clvl 27 respec, which the table disables (SRC the table's `respec.enabled:false`). Rows 22 to 30 pour 9 points into CE while `main_attack` is Bone Spear from clvl 18 and Bone Spear sits at 3 points. No army: only a Clay Golem tanks, 130 life at clvl 16 (SRC docs/plan/data/proposezone.json note). A human plays this build kiting; the profile has no stance. Keep it opt-in. |
| `necromancer_summoner_endgame90` | Sensible as the Maxroll end state | Max Raise Skeleton, Mastery, CE, Amplify, one point each of Iron Golem, Revive, Lower Resist and the curse chain (SRC the table tags). Rows 2 to 6 put 25 Strength before any Vitality: wrong if the table ever runs below clvl 31 (change 8). In Hell skeleton warriors die fast and boss damage from summons is cut to 25% (SRC class-necromancer.md line 197); this build leans on 1-point Revive and 6 mages, so expect it to be the weakest Hell clear of the three (EXP). |
| Profiles | Sensible | `summoner` stance `behindArmy` 4 subtiles, golem first, CE only when no summon wants the corpse (SRC docs/probes/necrocaps-2026-11-26.md). Gaps: no key for Weaken, Decrepify, Lower Resist, Fire Golem (SRC gap doc row 7). In Hell Lower Resist is the whole reason to buy the curse chain, and it cannot be cast. |
| Policy rows (stance knobs, mana band, lead 1) | Plausible, unproven | Copied Sorceress caster values (U). `kiteLifePct` 85 and `crowdAbandonN` 10 are caster-kite knobs: a summoner stands behind skeletons, it does not kite, and a 10-monster crowd is what the army is for. Mana band 30/60 only labels low/mid/high, harmless. |
| levelplan / act2plan rows | Sensible | Blood Moor 1 to 4, Cold Plains 5 to 8, Den at 5 to 6, caves, Dark Wood, free Rogue from Blood Raven, Cellars/Catacombs to 24, Act 2 Dry Hills at 15: the route a human takes. The 15 to 24 Cellars row comes from a Paladin log, not a Necromancer; an army caster is ahead of that XP rate (EXP). |
| Merc | Defiance is right; the Might note is wrong | Desert hires in Normal offer Prayer, Defiance, Blessed Aim; Might exists only in Nightmare hires and is absent in Hell (SRC docs/sme/mercenary.md lines 17, 22). So `docs/classes/necromancer.md` "wants Might from the desert merc" is a Nightmare-only option. Aura is shared with minions (SRC mercenary.md line 23): Defiance Normal and Hell, Holy Freeze or Might in Nightmare if gold allows. Act 1: take the free Blood Raven Rogue, not a paid one. |
| Advisor rune rows | Sensible | Spirit (Tal Thul Ort Amn: +2 skills, FCR), Lore, Heart of the Oak, Bone (Sol Um Um torso, necro skills) all fit (SRC internal/advisor/advisor.json). Spirit is `ladder:true`: it is a 1.14d ladder-only runeword, so offline single player cannot make it (SRC; EXP agrees). Check the advisor honours `ladder` for an offline hero. |

## Early game, clvl 1 to 12 (EXP)

- Normal Act 1 is easy for a necro. Raise Skeleton 1 = 1 skeleton, 4 = 3 skeletons (formula slvl<4: slvl, else 2+floor(slvl/3), SRC class-necromancer.md). The danger is clvl 1 to 5 before the army exists: life ~45 to 75, Teeth with one point does 2 to 4 magic for 3 mana (SRC table). Stay in Blood Moor / Cold Plains until Skeleton 4 and Clay Golem (clvl 6), leave the Den until clvl 7 to 8 (the table buys the Den point at clvl 5 which is fine; the Den fight itself should wait).
- Corpses: Raise Skeleton and Revive both need a corpse in the bot's current model (SRC necrocaps doc; V used corpse vanishes). Whether classic 1.14d Raise Skeleton needs a corpse at all: UNSURE, EXP says classic LoD skeletons can be raised only from corpses but a human rarely lacks one. Do not let CE burn the corpses the army needs: the code already ranks summon before CE (SRC), keep it.
- Mana: Raise Skeleton costs 5 + slvl, Mage 8 at level 1 to 27 at 20, Golem 12 + 3 per level (SRC table). Rebuilding 8 units at clvl 20 is about 100 to 120 mana: roughly the whole pool. Hence mana potions matter more than Energy points; the table is right to put points in Vitality, not Energy.

## Belt and kit

- `potions-belt.json` gives hp, hp, mp, hp for clvl 1 to 30 (SRC). For an army rebuild that is one mana column: too few. Humans carry 2 healing + 2 mana from the Normal Act 2 onward (EXP). Resists: Normal needs none; start collecting from Nightmare. Charms: Small Charms of Vitality/resist, one Gheed's only if found. Gear: Bone Wand / Teeth wand before clvl 18, Lore helm (Ort Sol), +skills shield, Spirit unavailable offline.

## CE5 vs CE1 (TASK-1016)

- Corpse Explosion damage is a fraction of the corpse's life, so level 1 already scales with monster HP; extra levels mostly add radius and mana cost (15 at L1, 24 at L10, SRC table). UNSURE on exact percent per level (not fetched).
- Minion points compound: skeleton count rises every third level, damage and life per Mastery level; mages add ranged damage (SRC formulas). Four extra CE points at clvl 21 to 29 are worth more as Mage/Skeleton/Mastery.
- CE competes with the army for corpses, and the bot casts it as its own main attack from clvl 15. CE1 (row 15) is enough for Normal; add CE points back after clvl 50 per the Maxroll end state. Verdict: CE1 is sensible for the Normal table, CE max stays in endgame90.
- Needs the scenario pair first (the card says so): pack of 8 in Cold Plains with CE1 vs CE5, compare kill time and mana per pack.

## Ranked data changes (max 10)

| # | File | Field | Current | Proposed | Why | Conf |
|---|---|---|---|---|---|---|
| 1 | internal/plandata/build-necromancer_leveling_normal30.json (and docs/plan/data copy) | rows 15, 15 radament, 16 | Bone Wall x3 | Teeth x3 | Bone Wall is unused (no key, no Bone Prison without the disabled respec); Teeth gives +7%/lvl to Bone Spear (SRC table) | high |
| 2 | same | row 12 | Iron Maiden x1 | Teeth x1 (or Bone Armor) | Iron Maiden is a respec prereq only; nothing casts it in the `bonespear` profile | high |
| 3 | internal/plandata/build-necromancer_summoner_normal30.json | rows 21, 24, 27, 29 | Corpse Explosion x4 (CE5) | Raise Skeletal Mage x2, Raise Skeleton x1, Skeleton Mastery x1 | CE1 enough in Normal; minions scale harder (see section above) | medium |
| 4 | same file's `mana_heavy` and docs/plan/data/potions-belt.json mana_heavy_builds | flag | absent for normal30 and bonespear | `"mana_heavy": true` and list both | army rebuild empties one mana column | medium |
| 5 | build-necromancer_leveling_normal30.json | rows 22 to 30 | CE x9, Bone Spear stays 3 | Bone Spear x6 to 9, CE x3 to 4 | `main_attack` is Bone Spear from clvl 18; points should follow the attack, no respec | medium |
| 6 | docs/classes/necromancer.md section 5 (doc) | Might from the desert merc | Might | Defiance Normal and Hell; Might or Holy Freeze Nightmare only | Might not offered in Normal or Hell (SRC mercenary.md) | high |
| 7 | internal/combat/classdata/profiles.json summoner layout (task necrokeys) | layout | no key for Lower Resist, Decrepify, Weaken, Fire Golem | give Lower Resist and Decrepify keys before more CE | the curse chain is bought for nothing otherwise | medium |
| 8 | build-necromancer_summoner_endgame90.json | rows 2 to 6 | Strength 5 each | Vitality 5 each, Strength from clvl 12 | 15 Vit at clvl 6 is a one-hit death; matches normal30 | medium (only if run below 31) |
| 9 | internal/decide/policy_chores.json necromancer | combat.kiteLifePct, crowdAbandonN | 85 / 10 (cautious), 55 / 20 | kiting off or lower, crowdAbandonN raised, for the summoner profile | the army fights crowds; kite retreat strips the hero off its minions | low (U) |
| 10 | internal/advisor/advisor.json Spirit row | `ladder` | true | confirm the advisor skips it for offline heroes | Spirit unmakeable offline | medium |

## Gaps in docs/sme/ for the main agent to queue

- Exact CE damage per level and radius (skills.txt calc) and whether classic Raise Skeleton needs a corpse.
- A Necromancer-specific belt row and resist order for Nightmare.
- Act 2 Normal XP route for an army caster (only a Paladin log exists).
