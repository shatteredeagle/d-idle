# Resist floors per difficulty (classic LoD 1.14d)

> Gap 1 of the 2026-11-27 SME reviews. Tags: **V** = fetched this run (URL), **S** = secondary or community, **U** = unverified / heuristic. D2R-only facts are flagged. Data twin: `docs/sme/resist-floors.json`. Related: `mechanics-progression.md` section 7, `builds-gear.md` (resist rules of thumb), `play-safety.md` section 3.

## 1. Hard rules (V)

| Rule | Value | Source |
|---|---|---|
| Base resist by difficulty (LoD, fire/cold/lightning/poison) | Normal 0, Nightmare -40, Hell -100 | V Arreat https://classic.battle.net/diablo2exp/basics/resistances.shtml and https://classic.battle.net/diablo2exp/basics/difficulty.shtml ("Player and Hireling resistances are lowered by 40 in Nightmare / 100 in Hell") |
| Same penalty applies to the mercenary | yes | V same pages |
| Classic (non-expansion) penalty | -20 NM, -50 Hell (not our target) | V https://www.theamazonbasin.com/wiki/index.php/Resistance |
| Default cap / hard cap | 75 / 95 (only items/skills raising max-resist lift it) | V Arreat resistances page; V Basin Resistance page |
| Floor | -100 (resist never below) | V Arreat |
| Excess above the cap offsets penalties from curses/Conviction ("any excess can offset penalties") | yes | V Basin Resistance page |
| Penalty also hits poison length, mana damage, stamina damage (not magic/physical) | yes | V Basin Resistance page |
| Anya "Scroll of Resistance" (Prison of Ice) | +10 all four, per difficulty; the +10 is lost when you die in that game until you re-enter (screen still shows it) | V Basin, V Arreat |
| Pets and monsters have no cap/penalty | yes | V Basin |

Displayed resist = gear + bonuses - penalty. Gear needed (raw, before penalty) to display the 75 cap: **Normal 75, Nightmare 115, Hell 175** (V arithmetic from the penalties; the same 75/115/175 figures appear in a community comparison table seen in search snippets, S). Display 0 needs raw 0 / 40 / 100.

Disagreement: a Blizzard forum guide (D2R-era, S https://us.forums.blizzard.com/en/d2r/t/hardcore-guide-for-nightmarehell/159316) words it as "Nightmare -40 and Hell another -60"; that is the same total (-100) as Arreat, not a conflict. Gamer Guides (D2R) says players cannot exceed 95; matches the hard cap.

## 2. Safe floors by stage (DISPLAYED values after the penalty)

No primary source gives a "safe at clvl N" number: Arreat only says low resists make fights harder and that Hell is "difficult without a lot of resistances" (V). The table below is the community heuristic already used in `mechanics-progression.md` section 7 and `builds-gear.md`; treat every cell as **U** unless a tag says otherwise. A bot should use it as a default prior, not a verified threshold.

| Stage | Entry clvl (typical) | Fire / Cold / Lightning floor | Poison floor | Raw gear needed | Tag |
|---|---|---|---|---|---|
| Normal, Acts 1-5 | 1-~38 | 0 (no penalty); 25+ by Act 3 helps vs unique/enchanted | 0 | 0-25 | U |
| Nightmare Act 1-2 | ~40 | >= 0 (raw 40) | >= -40 (raw 0) | 40 | U |
| Nightmare Act 3-5 | ~45-60 | +20 (raw 60) per `early-gear-policy.md` target | 0 | 60 | S (repo policy) |
| Late Nightmare (Hell prep) | ~60-70 | 75 all three (raw 115) | 0..50 | 115 | S community |
| Hell Act 1-2 | ~70+ | >= 0 net (raw 100) to 25+ | -100 .. 0 | 100-125 | U |
| Hell Act 3-5 (Mephisto, Diablo, Baal, Ancients) | ~80+ | 50-75 (raw 150-175) | 0..75 | 150-175 | U |

Context that moves the floor: Conviction aura and Lower Resist curse drop resist or ignore the cap (V Basin: Conviction overwrite rules; S Blizzard forum guide above), so +5 over cap or max-res items matter late; a Fire/Lightning/Cold Enchanted pack adds +75 monster resist, not extra player damage (V `play-safety.md`). Bosses like Hell Nihlathak and Council members are called out as burst-lethal at low fire/lightning resist (S forum guide).

## 3. Per class (only where it differs)

No class has a different penalty or cap (V Arreat: character-wide rule). Differences are the supply of resist, all **U** for numbers:
- Paladin: shields (Rhyme, Spirit, AP) and the Salvation aura are the easy route; Paladin is the class that reaches 75 earliest (S, `builds-gear.md`).
- Barbarian: Natural Resistance passive adds resist (value by skill level not fetched, U); War Cry/Iron Skin do not.
- Sorceress / Necromancer / Druid / Amazon / Assassin: no class resist skill with meaningful scaling except Sorceress Frozen Armor (cold-only absorb, U); rely on gear, charms (Small Charm of Inertia style 5s) and Anya.
- The Act 2 Desert Guard aura Thorns/Holy Freeze/Defiance do not add resist; Prayer heals (see `merc-auras.md`).

## 4. Open items
- A source with act-by-act safe resist numbers for LoD 1.14d was not found (a 2026-11-27 web search returned none). If the user wants verification, the live check is: log per-death elemental source and the player's displayed resist at the time.
- Class skill resist values (Natural Resistance, Paladin Resist Fire/Cold/Lightning on Salvation, Sorceress Cold Armor) are not tabulated here.
