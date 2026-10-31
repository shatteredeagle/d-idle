# Play patterns 3: what kills characters, and what humans do about it

Target: classic Diablo II LoD 1.14d, offline single player. Not D2R. Written 2026-10-29 by SMEKB-C.
Tags: **V** = fetched this session (URL) or read in our code/logs, **S** = snippet or secondary, **U** = memory or inference.
`[D2R]` = D2R-only or unknown in 1.14d.

Sources:
- BON = Arreat Summit mirror, monster bonuses (champion and unique modifiers, numbers): https://classic.hiveworkshop.com/diablo2exp/monsters/bonus.shtml (V, primary Blizzard text, 1.13 notes included)
- MB = Arreat mirror, monster basics: https://classic.hiveworkshop.com/diablo2exp/monsters/basics.shtml (V)
- SUP = Arreat mirror, super uniques: https://classic.hiveworkshop.com/diablo2exp/monsters/super.shtml (V)
- BOS = Arreat mirror, bosses: https://classic.hiveworkshop.com/diablo2exp/monsters/bosses.shtml (V)
- DIF = Arreat mirror, difficulty: https://classic.hiveworkshop.com/diablo2exp/basics/difficulty.shtml (V)
- CHR = characters (death, corpse, regen): https://classic.hiveworkshop.com/diablo2exp/basics/characters.shtml (V)
- HCR = hardcore rules and tips: https://classic.hiveworkshop.com/diablo2exp/basics/hardcore.shtml (V)
- HIR = hirelings: https://classic.hiveworkshop.com/diablo2exp/basics/hirelings.shtml (V)
- MM = Maxroll mercenaries (D2R): https://maxroll.gg/d2/resources/mercenary (V)
- PD = PlanetDiablo archive, Beating Duriel: https://gamespy-archives.quaddicted.com/sites/www.planetdiablo.com/diablo2/strategy/general/duriel.shtml.html (V)
- HCS = PureDiablo hardcore sorceress guide p5: https://www.purediablo.com/strategy/diablo-2-hardcore-sorceress-guide/5 (V)
- FOR = Blizzard forum "Hardcore - How to survive ACT I - Normal" (written for Legacy D2): https://us.forums.blizzard.com/en/d2r/t/hardcore-how-to-survive-act-i-normal/5494 (V)
- FOR2 = Blizzard forum "Hardest monsters in the game?": https://us.forums.blizzard.com/en/d2r/t/hardest-monsters-in-the-game/45162 (V, opinion, mostly Hell)
- MX = Maxroll walkthrough: https://maxroll.gg/d2/resources/full-story-progression (V, D2R wording)
- KB = Kolbot default config: https://raw.githubusercontent.com/blizzhackers/kolbot/master/d2bs/kolbot/libs/config/Sorceress.js (V)
- OUR = docs/difficulty.md (section 4 modifiers, from the same Arreat page), docs/combat.md, docs/knobs.md, docs/deathspiral.md, docs/stallfatal.md.

## 1. The five things that actually kill (human ranking)

1. **Low life with no escape**: running out of potions and portal scrolls, then being caught out of town. Humans always keep a TP and ~50% life ("keep your life above 50% at all times", V FOR). This is the number one rule.
2. **Elemental, Extra Strong, Cursed and Lightning champions/uniques**: "Elemental champions are often what kills players" (V FOR). Mods and numbers in section 2.
3. **Immune packs (mostly Hell)**: "the hardest monsters ... are the monsters immune to you especially in packs" (V FOR2). Humans skip them, distract with a merc/summon, or switch skills (V FOR2).
4. **Ranged and poison**: Blood Raven's arrows, Andariel's poison and cloud, vipers in Halls of Vaught, Fetish darts, Rogue archer packs with Extra Strong/Multishot/Fanaticism (V FOR, FOR2).
5. **Big-hit bosses in confined rooms**: Duriel (melee 22-32 Normal, 63-95 NM, 114-172 Hell, Holy Freeze slow; V PD), Smith in the Barracks, Nihlathak's Corpse Explosion (V MX).

## 2. Monster modifiers, in numbers (V BON; difficulty order Normal/Nightmare/Hell)

- Champion (blue): +90/75/66% damage, +67/56/49% attack rating, +20 movement, HP x3/2.5/2, monster level +2, XP x3. Unique (gold): level +3, HP x4/3/2, XP x5; a Unique has 3-6 minions (their AI can be changed, the Unique's cannot). Minions HP x2/1.75/1.5. A random pack is Unique 80% of the time, Champion 20% (V BON).
- Champions come in 2-4; all but one are plain Champions, the rest are Champion, Berserker, Fanatic, Ghostly or Possessed with equal chance (V BON).
  - **Fanatic**: +90/75/66% damage and attack rating bonus, **+100 movement speed**, -70% defense. Fast and fragile: runs you down, dies easily (V BON). Catacombs "tainted and spider packs with fanaticism or super-fast mods" were called out as the Catacombs danger (V FOR).
  - **Berserker**: +270/225/198% damage, but 1/4 of Champion life (V BON).
  - **Ghostly**: Immune to Physical displayed, actually 80% damage reduction, +33-50% cold damage (V BON).
  - **Possessed**: 2x Champion life, +20 movement, cannot be cursed (V BON).
- Unique bonuses (the dangerous ones), "All permitted bonuses have an equal chance, applied in displayed order" (V BON):
  - **Fire Enchanted**: fire damage +66/+100%, fire resist +75, explodes on death for fire+physical damage. The explosion shrinks with difficulty: Normal 11.25-18.75% of a normal monster's max life in ~2.7 yards; NM 7-11.7% in 3.3 yards; Hell 0.75-1.25% in 4 yards (V BON). Dangerous for weak characters in Normal; in Hell the resist/damage is what matters, not the explosion.
  - **Lightning Enchanted**: lightning +66/+100%, lightning resist +75; when hit or in hit recovery it releases 8 bolts of level mlvl/2 doing (2 x lvl) - 1 each. Leapers cannot spawn it (V BON). Many hits = many bolts: "run away" from LEB with Extra Fast (V HCS, "Rakanishu, run away"). Rakanishu is a super unique with it (V FOR).
  - **Cold Enchanted**: cold damage +66/+100%, cold resist +75, cold length 4 + 0.2 x mlvl s, level mlvl/2 cold nova on death (V BON). Slows melee classes; Coldcrow in Cold Plains has it (V FOR).
  - **Extra Strong**: unique +135/112/99% damage, +90/75/66% attack rating; minions +67/56/49% damage (V BON). Treewood Fisthead (Extra Fast + Extra Strong) is "run away" (V HCS; the page says "Treewood Fisthead").
  - **Extra Fast**: +100 movement speed for unique and minions (V BON). Cannot be outrun; fight in a corridor.
  - **Mana Burn**: mana damage +66/+100%, magic resist +20; "mana damage is 40% higher in Nightmare and 100% higher in Hell"; "due to a bug, mana damage is multiplied by 256 when applied by a melee attack" (V BON). A caster with a small mana pool is drained to zero in a few hits; humans never melee it, keep mana potions, or flee. Do not treat the bug as a safe-fight tool.
  - **Stone Skin**: +50% physical damage resist and doubled base defense (V BON). Physical builds lose half their damage; casters ignore.
  - **Cursed**: 75% chance to cast Amplify Damage level (mlvl/5)+1 when the target is hit by the unique's attack or missile (V BON). Griswold's curse "doubles all damage done to you" (V FOR). Fix: leave the area, kill the curser first, or use the town healer to remove it (V FOR: "town healer removes curses and poison").
  - **Aura Enchanted**: one of Might (monlvl/6), Holy Fire (/6), Blessed Aim (/5), Holy Freeze (/7), Holy Shock (/8, needs mlvl 20+), Conviction (/8), Fanaticism (/8) (V BON; the "Thorns" in other lists is U). Conviction lowers player resists, Might/Fanaticism raise damage; "avoid Might and Fanaticism auras" (V MS D2R).
  - **Magic Resistant**: +40 to fire/cold/lightning resists, not selectable in Normal (V BON).
  - **Spectral Hit**, **Multishot** (triples missiles), **Teleportation**, **Cold/Fire/Lightning Enchanted** minions get +33-50% element damage in NM/Hell (V BON).
- Bosses and super uniques: 1 extra random bonus in Nightmare, 2 in Hell on top of their own (V SUP). Bosses can only be chilled, not frozen; stun length rarely applies; crushing blow halved; "Prevent Monster Heal" and life steal are weaker; hirelings and summons do only 50/35/25% damage to act bosses (V BOS, DIF).
- Life steal and mana steal are halved in NM and cut by 2/3 in Hell; freeze/cold length and Confuse/Attract/Dim Vision are halved/quartered in NM/Hell (V DIF).
- Static Field minimum monster HP left: 33% NM, 50% Hell (V DIF). Our `combat.staticFloorNM` 33 and `staticFloorHell` 50 (V OUR knobs.md) match.
- Monster HP regenerates quickly in Hell: finish targets, do not switch (V MB).

## 3. Resists, curses and statuses

- Player and hireling resistances are lowered by 40 in Nightmare and by 100 in Hell (V DIF). A human caps fire/cold/lightning at 75 raw before entering NM and tries to be >= 0 net in Hell; Anya's +10 and Charm resists make the gap (V MX, MS).
- Fire resist matters most to level ~18, plus lightning resist against Lightning Enchanted bosses (V HCS).
- Life does not regenerate naturally (V CHR): only potions, healers, Replenish Life, wells. This is why our "life low, cannot regenerate" trigger is correct (V OUR stallfatal.md section 6).
- Poison: Antidote cures instantly and gives +50 poison resist (and +10 max) for 30 s; Thawing +50 cold resist for 30 s; Stamina potion gives unlimited stamina 30 s (V https://classic.hiveworkshop.com/diablo2exp/items/potions.shtml). Humans drink Antidote before Andariel and Thawing before Duriel (V FOR, MX). Our code has `AntidoteNeeded` (V docs/difficulty.md).
- The town healer clears curses and poison (V FOR). A human uses it when cursed rather than fighting on.
- Monsters can crit 5% for 2x damage (V FOR).

## 4. Known killers by area (Normal and early Nightmare)

| Area | Danger | Source |
|---|---|---|
| Act 1 Blood Moor/Den | Fallen Shaman revive; first elite packs at clvl 1-3 | V MX, OUR g1-log (naked clvl 1-3 died in Den) |
| Cold Plains Cave | Coldcrow: Cold Enchanted, archer pack, cold nova on death | V FOR |
| Burial Grounds | Blood Raven: arrows, zombies; clear first, dodge arrows; not on the critical path | V FOR, MX |
| Stone/Dark Wood | Treehead Woodfist: minions are the threat; Dark Wood tree quest | V FOR, MX |
| Forgotten Tower | Mana-drain ghosts; goatmen, dark rangers; rogue archers with Extra Strong/Multishot | V FOR, FOR2 |
| Barracks, Jail, Cathedral | Surround risk; Smith hits hard; Pitspawn Fouldog (Cold Enchanted + Cursed) in Jail 2; Bone Ash | V FOR |
| Catacombs | Tainted/spider packs with Fanatic/Extra Fast; Andariel poison; "haul butt and port to town" | V FOR |
| Act 2 sewers/oasis/ tombs | Frost Nova/Static Field fine; sewers need a chilled approach; do not cast static field near beetles (their Charged Bolts) | V HCS, MS |
| Act 2 Claw Vipers | Fast, ice spikes immobilize/slow | S gamerant snippet |
| Act 3 | "Skip it": flayers fast and deal serious damage (HC sorc advice); Might Enchanted flayers one-shot in NM/Hell | V HCS |
| Act 4 | Avoid City of the Damned and River of Flame untwinked: Abyss Knights and Damned resist cold/static, Urdars regenerate; Burning Souls hit hard; Izual melee | V HCS, MX |
| Act 5 | Nihlathak Corpse Explosion; Ancients (Madawc lightning-resistant, Korlic cold, Talic fire; full immune in Hell); wave bosses at Baal | V MX |
| Hell anywhere | immune packs: Unravelers (magic), Gloams, Halls of Vaught vipers (kill in under a second at the cloud) | V FOR2 |

Level guideline from HCS for a hardcore sorceress (V, 1.1x era; use as a conservative cap): Duriel at 24; enter NM at 33-35, Hell at 50-55; Act 2 for 33-42 (NM), stop in Lost City at 46 to avoid XP loss; do not level in Act 3 or River of Flame.

## 5. Human safety habits (hardcore style, which is also what a bot should do)

From HCR, HCS, FOR, MX (all V):
1. Keep Healing and Rejuvenation on the belt; Rejuvenation is instant and better in tight spots (V HCR).
2. Pull enemies through chokepoints (doors, bridges) instead of entering rooms full of them (V HCR); "move a few steps at a time so you don't attract more than 5 monsters" untwinked in Act 1 (V HCS).
3. Never fight below ~50% life; retreat and port when a dangerous pack appears in Catacombs (V FOR).
4. Drink Antidote before poison fights, Thawing before cold fights (V FOR).
5. Kill support first: shamans that revive, curse casters; then minions, then the unique (V MB: "clear the minions first, then the unique"; "Always attack monsters that revive other monsters first").
6. Cold shatters corpses so they cannot be raised (V MB).
7. When in trouble run to a waypoint, stairs, dungeon door or town portal (V MB).
8. Keep audio on; footsteps and sound cues are a warning (V FOR; irrelevant to a bot, but it must read monster mode/mods instead).
9. A mercenary, always; and a Tome of Town Portal (V FOR, MX).
10. "Don't leave the game after dying before your body is looted" (hardcore) and "Save and Exit" recovery in softcore gives 0 XP back (V HCR, CHR).

## 6. Death, corpses and retrieval

V CHR:
- After death press Esc and restart in town; the corpse stays where you died, shown as a purple X on the automap when near.
- Left-click the corpse to pick up items; unequipped items stay in your inventory; items that do not fit remain on the corpse; gold you dropped must be re-collected.
- Up to 16 corpses; at 16 items drop to the ground. Open (offline) characters keep only the OLDEST corpse between games (V CHR: "Open characters keep only their oldest corpse"). **This matters for our bot: our deathspiral doc assumes older bodies may survive later deaths (U); CHR says only the oldest corpse persists for an open character.** Realm characters keep the corpse with the most valuable equipment.
- If you exit without retrieving it, the corpse appears in town at the start of your next game, but you lose dropped gold and the chance to regain XP (75% of lost XP back only when retrieved at the place of death, NM/Hell).
- XP penalty NM 5%, Hell 10% of the gap to the next level, never a level loss; Normal none (V DIF, A4).
- Gold: share equal to clvl % (cap 20%) of carried gold; single player exemptions on the page (V CHR). Strength-restricted gear may not re-equip in the wrong pickup order (V CHR).
Human retrieval rules (S/U): return only after healing and refilling potions; leave a TP at the town side; carry a Rejuv; clear the pack at the corpse first or lure it away; do not go back in Hell if the thing that killed you is still there and you cannot beat it; keep a spare weapon/armor in the stash to recover with.

## 7. Mercenaries

V HIR and MM:
- **Act 1 Rogue (Kashya)**: ranged, Cold Arrow/Freezing Arrow or Fire Arrow/Exploding Arrow plus Inner Sight, bows only; available after Blood Raven, or from level 8-9 (conflict between pages, U which); a free hire for Quest 2 per MX. Fragile; good first merc.
- **Act 2 Desert Mercenary (Greiz)**: melee Jab with an aura (Normal: Prayer / Defiance / Blessed Aim per HIR; Nightmare: Thorns / Holy Freeze / Might per MM; Hell per HIR Prayer/Defiance/Blessed Aim and per MM Thorns/Holy Freeze/Might, so the sources conflict: U, check `hireling.txt`); spears, polearms, javelins. The most used merc by humans because of the aura (V MM, "most frequently used"). Cheap, it soaks hits at Duriel (V PD: "weak but cheap and can draw hits away").
- **Act 3 Iron Wolf (Asheara)**: caster (Cold/Fire/Lightning), cannot block; Fire most discussed, Cold "a terrible mercenary" (V MM).
- **Act 5 Barbarian (Qual-Kehk)**: Bash/Stun, or two-sword Frenzy; free hire from Quest 2; strong melee (V MM).
- Act 4 has no hireling NPC; Tyrael only revives (V HIR, MM). Any special NPC can revive any merc type (V MM; our docs/progression/merc.md says the same, U live).
- Death: if you die, the merc dies too (V HIR); revive costs min(int(hlvl^2/2) x 15, 50,000): hlvl 20 = 3,000; 40 = 12,000; 57 = 24,360; capped from hlvl 82 (V MM, the formula is D2R page, U in 1.14d). A merc cannot exceed your level; a higher-level merc needs a fresh hire (V HIR).
- Items: unequip before hiring a replacement or they are lost forever (V MM, MT). Merc gear: resist, high defense, damage, life-replenish, Ethereal; Mana/Energy/Vitality useless (V HIR).
- Mercs heal from potions, wells, healers; shift-click a potion to give it (V POT). Poison can kill an unhealed merc (V HIR).
- Boss damage from mercs is reduced to 50/35/25% (V DIF, BOS).
- Humans revive the merc between fights and before the next boss (V KB `MercWatch`, instant revive during battle; our prior-art 05-town.md). Merc potion threshold: KB `UseMercHP` 75 (V KB).

## 8. Chicken, retreat and potion thresholds humans use (numbers)

- Kolbot (V KB, default for any class): drink healing at 75% life, Rejuv at 40%, mana potion at 30%, merc heal at 75%; chicken (leave game) at 30% life; TownHP/TownMP 0 (off) by default; HealHP 50 (go to healer in town); RepairPercent 40; MinColumn [3,3,3,0] belt refill; BeltColumn hp, hp, mp, rv; MinGameTime 60 s.
- Human hardcore rule: retreat below 50%; chicken is the last resort (V FOR, HCR).
- Retreat order of preference: (1) drink, (2) back off/kite or close a door, (3) portal to town, (4) waypoint, (5) leave the game ("chicken"), (6) never melee on to 0.

## 9. How our bot differs (checked against OUR docs)

| Topic | Human | Our bot | Gap or note |
|---|---|---|---|
| Drink threshold | 75% heal, 40% rejuv (Kolbot) | `drinkLifePct` 55, `drinkRejuvPct` 35 (V knobs) | Ours drinks later; with a 1 s potion gap a late drink is the likelier death (U). Raising to ~65 is cheap |
| Chicken | 30% | `chickenPct` 25 (N, NM), 30 Hell | Lower in Normal/NM; fine, but chicken is also not the main safety valve |
| Retreat before chicken | Port to town/waypoint/door at 50% | `resource.fleeLifePct` 50: with no healing potion and no portal the bot breaks away instead of trading blows; `resource.retreatBelt` default 0 = off (leave when life is under the drink line and few potions left); `curseRetreatPct` 60 for curses; `resource.minEngageLifePct` 40 (V knobs.md). stallfatal.md section 5 recorded a 60 s standing-still life drop before `fleeLifePct` existed (V) | Mostly covered now; `retreatBelt` is off by default and was only switched on by hand in G1 (V g1-log.md). A human always has this behaviour on |
| Elite handling | Skip or avoid Extra Strong/Cursed/Mana Burn/LEB/Fire Ench in Normal when weak | `combat.PlanForTarget` mentions Skip/KeepRange (V docs/difficulty.md); bot catalog has no affixes yet ("mlvl 0 and affixes none") | **Gap**: affixes not yet read, so the human elite rules cannot fire in Normal |
| Corpse | Retrieve at once after healing and shopping; XP back 75% | `gearCorpse`: retrieves the newest worthwhile body, max 2 h old, falls back to older bodies (V deathspiral.md) | CHR says only the OLDEST corpse persists for an open character; our "older body" logic may chase corpses that no longer exist (U, verify with a live death test) |
| Merc | Always hired and kept alive | A2 merc quest node exists; `town.Revive` gated behind unverified flag (V merc.md) | Bot can play without; a merc is the biggest survivability item for a caster in Normal (V FOR, MX) |
| Antidote/Thawing | Drunk before Andariel/Duriel | `AntidoteNeeded` exists; no Thawing for Duriel (U, check) | Add a pre-boss buff list |
| Skip bosses | Skip Radament, Andariel until stronger, skip Act 3 for casters | Gate levels: Andariel 22, Duriel 24, Mephisto 30 (V campaign.go) | Ours is conservative; fine |
| Town portal first | Always keep a TP; never leave town without | `buyFirstTP`, tome at 450 gold; G1 attempt 6 left with 3779 gold and no purchase (V g1-log.md) | **Bug**: shopping kit not buying when quest step owns town time |
| Hell packs | Skip immune packs, switch skills, use merc | `PickByResist`, `ImmuneWorkaround` logged but skills not cast (V difficulty.md) | Gap in Hell only |
| Gate on gear | Never fight naked | `GearGate` from clvl 5 (V deathspiral.md) | Same intent |

## 10. Open questions

1. Does an offline character keep only its oldest corpse (CHR) or the newest (our assumption)? Test: die twice on a throwaway and read the corpse list in memory.
2. 1.14d Desert Mercenary auras by difficulty (HIR and MM disagree).
3. Whether `lowLifeGate` should also fire inside a fight (stand-still death at 51 s, V stallfatal.md).
4. Real 1.14d crit chance and Fire Enchanted explosion damage (BON numbers are from the 1.13 era page; 1.14d is the same engine, S).

## ROUTEGUARD implementation notes (2026-10-30)
Implemented the SME decision EXP after the Soaky Monastery-road deaths: R1 optional Imbue skip (`campaign.imbueMinLevel` 20, `campaign.imbueNeedsMerc` 1, non-melee classes never), R2 route guard (`route.monasteryMinLevel` 20; areas 7 and 26-37; open with a living merc or clvl >= 20 plus kit of 4 healing, a portal, mana for casters; the Andariel and Act-1-done entries share it), R3 level picker excludes the guarded areas and gained Cave 2, Crypt and Mausoleum steps in progression-normal.json, R4 escalating repeat-site bans, PANIC crowd rule, autopsy facts. Details: docs/plan/plan-v2.md 11.2, docs/combat.md "Crowd rule", docs/deathspiral.md. All U live: no live run was made.
Known gaps: portal charges are not readable (`World.TPKnown` false), so "2 portal scrolls or a tome" is checked as "a portal item exists"; the merc backend is not wired, so the guard effectively means clvl 20 until `task.MercAlive` or the unit-table merc (`HasMerc`) reports a living merc.
