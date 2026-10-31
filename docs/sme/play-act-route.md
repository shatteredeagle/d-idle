# Play patterns 1: a human route through all five acts

Target: classic Diablo II LoD 1.14d (offline single player, 8 or fewer players in the game, no Realms). Not D2R.
Written 2026-10-29 by SMEKB-C. Tags: **V** = primary or reliable source fetched this session (URL given), **S** = secondary source or search
snippet, **U** = from memory or inference, not verified. `[D2R]` marks a fact that is D2R-only or whose 1.14d status is unknown.

Sources used (all fetched 2026-10-29 unless marked):
- A1 = Arreat Summit mirror, quest rewards: https://classic.hiveworkshop.com/diablo2exp/quests/rewards.shtml (V, Blizzard text, LoD 1.1x era)
- A2 = Arreat Summit mirror, per-act quest pages: https://classic.hiveworkshop.com/diablo2exp/quests/act2.shtml (also act3, act4, act5) (V)
- A3 = Arreat Summit mirror, difficulty: https://classic.hiveworkshop.com/diablo2exp/basics/difficulty.shtml (V)
- A4 = Arreat Summit mirror, experience: https://classic.hiveworkshop.com/diablo2exp/basics/experience.shtml (V)
- MX = Maxroll walkthrough: https://maxroll.gg/d2/resources/full-story-progression (V text, but D2R-era wording)
- MQ = Maxroll important quests: https://maxroll.gg/d2/resources/important-quests (V, D2R)
- MS = Maxroll Sorceress leveling route: https://maxroll.gg/d2/guides/sorceress-leveling (V, D2R Season 14 numbers)
- AL = Almar's guide: https://almarsguides.com/Computer/Games/Diablo2/Leveling/ (V, old 1.10-era, twinked and untwinked routes)
- HC = PureDiablo hardcore sorceress guide p5: https://www.purediablo.com/strategy/diablo-2-hardcore-sorceress-guide/5 (V, 1.1x era)
- OUR = our own docs: docs/difficulty.md (monster levels from levels.tsv, V there), docs/plan/data/progression-*.json, internal/task/*_campaign.go.

## 1. The shape of a human run

1. Normal is a tutorial. Humans do not clear it; they do the quests that pay (skill points, stat points, +20 life, sockets) and the
   forced boss chain, and level on whichever area gives the best XP for their level (AL, HC). The campaign order in one line:
   Den of Evil, Blood Raven (merc), Tools of the Trade (optional), Countess runes, Andariel, Radament, Staff parts, Summoner/Duriel,
   Golden Bird/Lam Esen/Khalim's Will/Travincal/Mephisto, Izual/Hellforge/Diablo, Larzuk/Anya/Ancients/Baal. (S, MX, AL)
2. A boss kill gates the next act and the next difficulty. Required kills: Andariel (A1->A2), Duriel (A2->A3), Mephisto (A3->A4),
   Diablo (A4->A5), Baal (next difficulty) (V, https://classic.hiveworkshop.com/diablo2exp/monsters/bosses.shtml: "must be defeated the first time to progress").
   Everything else is optional for progression. Blood Raven is not on the critical path (U: Burial Grounds is not needed to reach the
   Catacombs; MX lists it as "required" only in the quest-log sense).
3. Difficulty gates (V, A3): Nightmare opens after Baal on Normal, Hell after Baal on Nightmare. Hardcore characters unlock after Normal.
4. Humans move on from an area as soon as its XP drops (monster level more than about 5 below clvl; table in section 6). Hence "level up
   with the story, not by grinding a fixed spot" (S, AL "Realistic route").

## 2. Quest rewards (what is worth doing)

Rewards repeat in each difficulty unless noted, so the totals are "x3" (V, MQ: +12 skill points, +15 stat points, +30 all-res, +60 life across three).
Anything marked [D2R] means the page is D2R; the 1.14d state is U unless the A1 table (classic text) says the same.

| Quest (act) | Reward | Notes for a bot | Source |
|---|---|---|---|
| Den of Evil (A1, Akara) | +1 skill point; one free respec from Akara (stats and skills) | Kill every monster in the Den (Blood Moor), return to Akara. Kill the Fallen Shaman first, they revive. Hell Den has Cold and Fire immune packs. Respec per difficulty: A1 says "1 free respecialization"; D2R says once per difficulty (U for 1.14d per-difficulty) | V A1, MQ, MX |
| Sisters' Burial Grounds (A1, Kashya) | Rogue mercenary to hire (also free hire at clvl 8 per A2 "available after the second quest or level 8") | Kill Blood Raven. Not on the critical path | V MX, hirelings page |
| Search for Cain (A1, Akara) | Free identify by Cain in that difficulty; Akara gives a Magic ring (Normal) or Rare ring (NM/Hell) | Needs Underground Passage 1, Dark Wood, Tree of Inifuss (Treehead Woodfist is required), Cairn Stones in Stony Field, Tristram. Skippable. Our bot skips it by default | V MX |
| Forgotten Tower (A1) | Countess (rune source). No quest reward in single player | Tower Cellar 1-5. She is the best early rune farm, humans return many times (MS, AL: "farm The Countess for runes") | V MX, MS, AL |
| Tools of the Trade (A1, Charsi) | Imbue: turns a base item into a Rare. Needs clvl 8 to take the Malus from the Barracks (Smith) | Item level = clvl + 4 (V MQ). Humans save it for a good base (circlet/belt/boots/exceptional). Highly random | V A1, MQ |
| Sisters to the Slaughter (A1) | Andariel kill opens Warriv's ride to Act 2 | Poison is the danger. Take antidotes or a TP and heal | V MX |
| Radament's Lair (A2, Atma) | +1 skill point; Act 2 vendor prices lower; drops a Book of Skill (use it for the point), chest has the Horadric Scroll | Sewers 3, he is often Cold-immune in NM, Physical-immune in Hell. Many humans skip him until clvl ~18 (MS) | V A1, MX, MQ |
| Horadric Staff (A2) | Needed for Duriel: Cube (Halls of the Dead 3), Shaft (Maggot Lair 3), Amulet (Claw Viper Temple 2, from the Tainted Sun altar) | Cube is also the single most useful item for a human (see economy and builds docs) | V MX |
| The Summoner (A2) | Kill opens the portal to Canyon of the Magi; true tomb is the symbol shown in the quest log | Arcane Sanctuary: four directions, three dead ends; take the WP on arrival | V MX |
| The Seven Tombs (A2): Duriel | Opens Tyrael door then Meshif to Act 3 | Duriel level 22/47/72, HP 2188/8938/22625, Cold 50/100/100 resist, melee 22-32 (Normal) (V, PlanetDiablo archive, see play-safety) | V MX, PD |
| The Golden Bird (A3) | +20 permanent life (Potion of Life from Alkor); also Jade Figurine drop is random from a unique/elite | Alkor sometimes needs two talks | V A1, MX |
| Blade of the Old Religion (A3) | Free Iron Wolf mercenary if you have none, plus a Rare ring from Ormus | Do it with Khalim's Will step 2 (Gidbinn near Flayer Dungeon) | V MX |
| Khalim's Will (A3) | The flail makes Travincal's Orb smashable (opens Durance). Brain/Eye/Heart/Flail drops. Flail also a strong early melee weapon | Longest A3 quest; Travincal council gives the flail (V MX) | V A2, MX |
| Lam Esen's Tome (A3) | +5 stat points | Tome sits in the Ruined Temple, Kurast Bazaar; quest triggers only after entering the Bazaar | V A1, MX |
| The Blackened Temple / Guardian (A3) | Council kill; Mephisto then red portal | Mephisto is the best set/unique farm in NM/Hell (MX) | V MX |
| The Fallen Angel (A4, Tyrael) | Izual: +2 skill points | Izual is never immune but tanky: 30% resists, 75 cold in Hell; Burning Souls hit hard in NM/Hell | V A1, MQ, OUR difficulty.md |
| Hellforge (A4, Cain) | Gems 1 regular + 2 Flawless + 1 Perfect and (expansion) a rune: Normal El..Sol (A1 text via fetch summary said El..Amn, sources conflict, U), NM Sol..Um, Hell Hel..Gul | Needs Mephisto's Soulstone (Cain can give another); Hephasto drops the hammer | V A1, MX |
| Terror's End (A4) | Diablo kill; red portal to Harrogath | 5 seals, 3 elite packs | V MX |
| Siege on Harrogath (A5, Larzuk) | Adds sockets to one item (max by base and ilvl; Magic 1-2, Rare/Set/Unique/Crafted 1); ilvl 40+ for max | Humans call it the best item quest. Plan the base item first (S, MQ) | V A1, MQ |
| Rescue on Mount Arreat (A5) | Tal, Ral, Ort runes (expansion) | Free 15 barbarians in Frigid Highlands; failed quests are possible (rejoin) | V MX, A2 |
| Prison of Ice (A5, Malah/Anya) | +10 all resistances (not magic or physical); Anya gives a class-specific Rare (ilvl = clvl) | Talk Malah, free Anya, talk Malah again. Order matters (V MX) | V A1, MX |
| Betrayal of Harrogath (A5) | Personalize an item (not worth it) | Nihlathak's Corpse Explosion can one-shot in Hell | V A1, A2 |
| Rite of Passage (A5) | XP boost to at most one level (1.4M / 20M / 40M cap) | Needs clvl 20 / 40 / 60 (V, A2 act5). Madawc, Korlic, Talic; do not leave the area or open a TP | V A1, A2 |
| Eve of Destruction (A5) | Baal | Waves 1-5 then Baal; Worldstone Keep 1-3 | V A2, MX |

Nightmare and Hell: every quest above is offered again per difficulty (U for the exact set offered; MQ says rewards add up across difficulties, V).
Our docs/nightmare-campaign.md says the same assumption is U in our bot. Human advice is to re-run Den of Evil on Nightmare immediately after
Normal (V, A3: "Re-running Den of Evil on Nightmare right after beating Normal is also recommended").

## 3. A concrete route and the levels a human holds at each step

Numbers marked "twinked" are for characters with bought or found gear; "untwinked" for a fresh character. Levels are character level (clvl).
These are human community numbers, not monster-level tables (those are in docs/difficulty.md and play-safety). Where two sources disagree both are shown.

### Normal

| Stage | What the human does | clvl at start of stage | Source |
|---|---|---|---|
| A1 start | Blood Moor, Den of Evil (skill point and respec) | 1-3 | V AL ("Levels 1-4: Act 1 Quest 1"), V MQ |
| A1 | Cold Plains, Burial Grounds (Blood Raven) or Cave 1, then Stony Field, Dark Wood, Black Marsh | 4-12 | S AL: Tristram runs for 4-15 (needs Cain done), MX |
| A1 | Forgotten Tower (Countess runes) | ~10-14 | S MS (Countess "once runes are collected"), AL |
| A1 | Tools of the Trade (needs clvl 8) | 8+ | V A1, MX |
| A1 | Andariel | 15-17 (AL: "Levels 15-17: Act 1 Quest 6"); HC advice: join a Kill Andariel game, Barracks and beyond are "generally not a safe place" for an untwinked sorceress | S AL, V HC |
| A2 | Radament (skill point) | MS: skip until clvl 18; AL: do Act 2 quests 1-5 at 17-25 | V MS, AL |
| A2 | Staff parts, Tainted Sun, Summoner | 17-25 | V AL |
| A2 | Duriel | HC: level 24 recommended (Energy Shield); AL: after Tomb of Tal Rasha farming to ~24; OUR doc: min 26 | S HC, AL |
| A3 | All quests before Mephisto: Golden Bird (+20 life), Lam Esen (+5 stats), Khalim's Will | 25-26 | V AL ("Levels 25-26 ... Cruise through Act 3 and do all its quests before killing Mephisto") |
| A3 | Flayer Jungle and Kurast give XP; HC says skip A3 for sorc (flayers fast, dangerous) | 25+ | V AL, HC |
| A4 | Izual (+2 skill points), Hellforge, Diablo | MS: enter A4 around clvl 26-30 after Meph; AL: Chaos Sanctuary farm to ~35 | S MS, AL |
| A5 | Larzuk sockets, Rescue, Anya, then Ancients at clvl 20+ (usually 30+), Baal | MS: "Level to 25 ... Throne waves for 26" before Nightmare | V MS |
| Normal end | Humans enter Nightmare around clvl 26-38 depending on source: MS enters at 26 (respec first), AL says 25-40 chaos/baal runs, Arreat says stay in Act 5 until ~40 for good XP | 26-40 | V MS, AL, A3 |

Stat and skill points: Radament +1, Den +1, Izual +2 (then +1 each in NM/Hell: a total of +12 over three difficulties, V MQ); Lam Esen +5 stat (per difficulty).

### Nightmare

| Stage | Human guidance | Source |
|---|---|---|
| Entering NM | HC sorc: enter at 33-35. MS: enter at 26 after Normal Baal, level Act 1 NM to 35. AL: 25-40 Chaos/Baal; NM 41-63 "skip most content, waypoint to waypoint" | V HC, MS, AL |
| Act 1 NM | Den of Evil immediately (skill point and respec) and Countess runs for runes | V A3, MS |
| Act 2 | HC: "Level 33-42 in Act 2 (skip Act 1)", stop at 46 in Lost City to avoid XP loss | V HC |
| Act 4 entry | HC: 42-46; Hellforge reward in NM = Sol..Um rune | V HC, A1 |
| NM end / Hell entry | HC: enter Hell at 50 (55 more conservative). MS (hardcore pre-entry): clvl 38-40 before NM, 65+ before Hell, cap fire/cold/lightning to 75 raw before each difficulty | V HC, MS |
| NM resists | Players and hirelings take -40 (V A3); a human treats 75 raw as the goal, not 75 net | V A3, MS |

### Hell

| Stage | Human guidance | Source |
|---|---|---|
| Hell entry | 50-55 (HC) or 65+ (MS for hardcore). Resists are -100 in Hell (V A3), so a capped character shows -25 net unless it adds items/Anya's +10/charms | V |
| Hell levels | HC: levels 55-70 in Act 2 then A4 at 70; "do not level in Act 3". AL: Countess in Hell is the best A1 XP at 63-73 | V HC, AL |
| Hell 75+ | Mausoleum, Pit, Andariel, Cow, Ancient Tunnels, Arcane, Summoner, Lower Kurast, Travincal, Mephisto, Pindleskin (MS "Farming at L75+", D2R wording) | V MS |
| Skipped by many | Den of Evil Hell until the build is strong (Cold/Fire immune packs); Cold-immune monsters; Act 3 for casters | V MS, HC |

## 4. Which bosses to skip early (and when to fight them)

| Boss | Normal | Why skip or fight | Source |
|---|---|---|---|
| Blood Raven | Fight at 6-10 with ranged or merc | Fire arrows, summons Hungry Dead; drops a rogue merc | S (rankedboost snippet), V MX |
| Countess | Return often for runes from ~10 | Not required for progress; best early rune source | V MS, AL |
| Andariel | clvl 15-18 (untwinked) or party; a sorc without a strong merc does not melee her | Poison is huge in NM/Hell; a Rogue merc or Act 2 merc can be the distraction | S, V HC |
| Radament | Skip until ~18 (MS) | Skill point is worth waiting for a stronger character | V MS |
| Summoner | 22-26 | Needed for the Arcane quest; he teleports | V MX |
| Duriel | 24-26, with potions, a stack of Thawing/full life, a merc to absorb hits | See play-safety; melee 22-32, cold aura slows | V HC, PD |
| Mephisto | 28-30 normal; best farm in NM/Hell | Ranged poison/lightning, hides behind the moat in Durance 3 (U) | V MX |
| Izual | 26-30 | Tanky, +2 skill points, repeat each difficulty | V MX |
| Diablo | 29-33 | Seals have packs; Lord de Seis can be cold immune; fight only after clearing a pentagram check (U details) | V MS, HC |
| Baal | 33-35 normal | Waves; a portal next to the throne lets a human stop and heal; skip waves is the Maxroll druid advice | V MS |

## 5. Waypoints a human activates (and where they go to save time)

Humans activate every waypoint as they go, then use them as teleports during runs. Nine in Acts 1, 2, 3 and 5; three in Act 4.
Counts are from memory (U); names confirmed by MX where in brackets.
- Act 1 (9): Rogue Encampment, Cold Plains, Stony Field [V], Dark Wood [V], Black Marsh [V], Outer Cloister [V], Jail 1, Inner Cloister, Catacombs 2 [V].
- Act 2 (9): Lut Gholein, Sewers 2, Dry Hills [V], Halls of the Dead 2, Far Oasis [V], Lost City, Palace Cellar 1, Arcane Sanctuary [V], Canyon of the Magi.
- Act 3 (9): Kurast Docks, Spider Forest [V], Great Marsh, Flayer Jungle [V], Lower Kurast, Kurast Bazaar [V], Upper Kurast, Travincal [V], Durance 2 [V].
- Act 4 (3): Pandemonium Fortress, City of the Damned, River of Flame [V].
- Act 5 (9): Harrogath, Frigid Highlands [V], Arreat Plateau, Crystalline Passage [V], Glacial Trail, Halls of Pain, Frozen Tundra, Ancients' Way, Worldstone Keep 2 [V].
Waypoint activation is per difficulty (U; the save keeps them per difficulty, our bot's memory reading should check).

## 6. XP efficiency: why humans move on

The XP the game gives depends on monster level minus your level (V, A4 table). Under clvl 25 the table is 100% for -5 to +5, 81% at -6, 62% at -7, 43% at -8, 24% at -9, 5% at -10 or more; above 25 the penalty starts at 6 levels (81, 62, 43, 24, 5). Over the clvl 69 line an extra penalty applies (95% at 70 falling to 0.59% at 98). Death costs 5% (NM) or 10% (Hell) of the XP gap to the next level, never a level; corpse retrieval returns 75% (V A4). Normal has no XP death penalty.
Consequences a human acts on:
- Bosses and uniques give 5x XP (minions too), champions 3x (V Arreat bonus.shtml, https://classic.hiveworkshop.com/diablo2exp/monsters/bonus.shtml, and AL). Fast runs only stop for champions, uniques and bosses (V AL).
- Normal Act 5 and Chaos Sanctuary are the preferred XP for clvl ~26-40 (AL, A3). Nightmare Act 1 is the XP floor for ~26-35 (MS).
- Party XP bonus is multiplayer only (V A4); irrelevant to a solo offline bot except that `/players X` raises monster HP and XP (Arreat monster basics page: HP x (players+1)/2).

## 7. Where our route differs from the human route (cross-check against OUR docs)

Our campaign data: Normal Act 1 entries: Den (clvl 1, 5 without kit), Burial Grounds, Tools (8), Tower (14), Andariel (22); Act 2: Radament 16, Cube/Viper/Shaft 16-18, Staff 20, Arcane/Summoner 24, Duriel 24 (internal/task/campaign.go, act2_campaign.go, docs/plan/data/progression-normal.json). Our floors: NM Blood Moor 36, Act 2 NM 40-44, Duriel NM 50, Hell Blood Moor 64.
| Topic | Human | Our bot | Judgement |
|---|---|---|---|
| Andariel level | 15-18 (twinked or with merc), often earlier with help | 22 | Ours is more conservative. Good for safety, costs XP time. U whether a bot fight below 22 survives; do not lower without a live test |
| Radament | 17-18 | 16 | Close |
| Duriel | 24-26 | 24 (campaign) to 26-27 (progression table) | Close; PureDiablo says 24 with energy shield (sorc specific) |
| Cain quest | Done by many for the free ring and Cain ID; AL does Tristram runs 4-15 | Off by default | Our choice saves a risky chain; humans do it mostly for Tristram XP |
| Quest points | Den +1, Radament +1, Izual +2 = 4 skill points in Normal | Bot claims after each; leveler pulls rows forward | OK |
| Entering Nightmare | clvl 26-35 depending on class and gear | 36 at Blood Moor (docs/difficulty.md) | Ours is 1-10 levels later; safer, slower |
| Entering Hell | 50-55 (HC), 65+ (hardcore MS) | 64 | Close to the hardcore view, behind the softcore view |
| Resist cap before NM/Hell | 75 raw target (MS) | Gate only needs net >= 0 post-penalty (docs/difficulty.md) | Ours is looser; a human aims to cap before entering |
| Waypoint use | Hop WP to WP, TP back for repairs | Planner uses WPs; waypointToTown is act 1 only | Gap (docs/deathspiral.md, stallfatal.md) |
| Re-run Den on NM after Normal | recommended | Nightmare template replays Den (U) | Same intent |
| Skip Act 3 for sorc | HC says skip | Bot walks Act 3 for quest chain (Golden Bird, Lam Esen, Khalim's Will) | Reasonable for a bot (needs the stat points and Meph) but A3 is where fast flayers kill |
| Rite of Passage requires 20/40/60 | Yes | Template nodes not verified (U) | Check levels |

## 8. Open questions and caveats

- 1.14d versus D2R: the Maxroll pages are D2R; the Blizzard Arreat pages (A1-A4) are the old LoD text and are our authority. Differences to watch: (a) Den of Evil respec per difficulty, (b) Cain ring quality, (c) Hellforge reward also gives a rune only in expansion (all LoD = expansion), (d) D2R has shared stash and extra inventory space (not 1.14d).
- No source gave an authoritative 1.14d "level to enter Act N" table. The numbers above are human heuristics; OUR docs/difficulty.md derives entry levels from monster levels (levels.tsv, V there): use both.
- Wowhead D2R "recommended levels" (Den 1-3, Staff 17-20, Hellforge 29-30) came only as a snippet (S): https://www.wowhead.com/diablo-2/guide/act-4-walkthrough-quests.
- U: whether the Blizzard-era "Akara respec" exists in 1.14d offline single player on each difficulty. Verify live in a throwaway character before the bot depends on it.
