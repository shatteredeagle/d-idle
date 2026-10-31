# d2d fidelity checklist (agent D2DFIDELITY, 2026-10-30)

Question from the d2-sme: how far can the bot's tests trust d2d (the native D2 reimplementation, our private lockstep fork,
docs/d2d-fork.md) as a stand-in for the original Game.exe 1.14d? Each row below is one measurement on d2d, the expected value
and its source, and a verdict. Gap rule (docs/d2d-backend.md section 7): **more than 20 % = fidelity bug**.

Tags: **V-d2d** = measured on d2d in this session (script and raw output named); **V-live** = number from our live logs;
**S** = secondary source (docs/sme/*, Arreat, Amazon Basin as quoted there); **U** = unverified; **D2D-src** = d2d's own research
notes (`docs/research/re/*.md` in the fork clone: they say "traced in game.exe", we did not re-check them).
Verdict words: *matches* (within 20 % or exact), *differs by X %*, *cannot test* (no original-game number to compare, or d2d is
all we have).

## 0. Setup and method

- All runs on the worker host, one d2d at a time, `nice -n 10`, private SDL dummy driver, private socket and XDG dirs under
  `~/d2d-lab/fidelity/` (data copied read-only from `~/d2d-lab/data`), throwaway Sorceress/Barbarian/Paladin created with
  `--start-class` (UI order: 0 Barbarian, 1 Necromancer, 2 Paladin, 3 Amazon, 4 Sorceress, 5 Druid, 6 Assassin, V-d2d `dump player`).
  Binary: the fork's `--lockstep` build (`step N` = N world ticks of 40 ms). No game instance, Wine prefix or save of ours was touched.
- Safety: `fid.py` pauses (kills d2d) while the 1-minute load average is above 45 and stops at less than 15 GB free on `/`. Load
  went to 115 once (someone else's job); the runs waited and resumed, nothing was lost.
- Our scripts (repo: `docs/d2d-probes/`, run on the worker host next to `~/d2d-lab/fidelity/`, raw output in `~/d2d-lab/fidelity/out/`, not in the repo):

| Script | Check | Output file |
|---|---|---|
| `fid.py` | library: start/stop one lockstep d2d, `step`, `monsters`, safety | |
| `fid_regen.py OUTDIR` | 1 regen | `out/regen.json` |
| `fid_potion.py OUT CLASS [VIT]` | 1 potions | `out/potion_sorc*.json` |
| `fid_stam.py` | 1 stamina | `out/stam.log` |
| `fid_aggro.py OUT SEED LEVEL...`, `fid_aggro_report.py` | 2 wake distance | `out/aggro.jsonl` |
| `fid_hit.py OUT SEED` | 2 hit wake, rally, leash | `out/hit7.jsonl` |
| `fid_sweep.py OUT SEED CLVL CLASS LEVEL...`, `fid_drops_batch.sh`, `fid_drops_report.py`, `fid_xp_report.py` | 3 drops, 4 XP | `out/drops*.jsonl`, `out/lowxp.jsonl` |
| `fid_vend3.py NPC CMD...` (real time; `fid_vendor.py`, `fid_vend2.py` were the exploration steps) | 5 prices | stdout, `out/*.png` |
| `fid_resist.py OUT` | 6 resists | `out/resist.json` |
| `fid_timing.py OUT` | 7 frame counts | `out/timing*.json` |
| `fid_death.py OUT` | 8 death rules | `out/death.json` |
| `fid_misc.py` | checks that `debug stat 3` (Vit) takes effect | `out/misc.log` |
| `fid_queue*.sh` | the sequential queues that ran all of the above (a queue script must not be appended to while it runs) | |

- Speed: a full first-visit sweep of Blood Moor takes 17 to 22 s wall, Cold Plains 28 to 35 s, Stony Field 23 s, Dark Wood 40 s,
  Den 6 to 10 s (lockstep, loaded box). 120 kills of statistics cost about 10 s; the whole drop/XP data set below (7885 kills,
  5 areas, 8 clvls, 63 seeds) took about 40 minutes wall including process restarts.
- Known lockstep/probe quirks: `debug stat 12 N` sets the level but **not** the life/mana maxima (set stats 6..9 as 8.8 fixed point,
  value x 256), `debug points` applies on the next tick (step before `debug skill`), the first `cmd move` after a warp needs a priming
  click, dead monsters stay in `monsters` (mode DD/DT), levels are built once per process (new seed or process = new monsters).

## 1. Summary table

| # | Check | N | d2d result | Expected (source) | Verdict |
|---|---|---|---|---|---|
| 1a | Mana regen | 8 runs x 60 s game time, maxmana 35..250 | exactly `floor(256*max/3000)*25/256` per s (0.195/s at 35, 2.05/s at 250), town = field | same formula (S mechanics-combat 7.2, D2D-src) | **matches** (exact) |
| 1b | Life regen | 8 x 60 s | 0 per s, town and field | no natural life regen (S) | **matches** |
| 1c | Stamina regen standing | 10 s | 7.2/s at max 74 = max/256 per tick | max/256 per tick (S 7.2) | **matches** |
| 1d | Minor Healing, Sorceress | 28 drinks | +30.16 life over 192 ticks (7.68 s), same every time | 30 over 7.68 s (S mechanics-combat 7.3) | **matches** |
| 1e | Potion double fill | 24 drinks at Vit 150 | 0 doubles (expected about 9: 37 %) | (Vit-2)/4 % (S 7.3) | **differs** (feature missing or `debug stat` does not feed it, U which) |
| 2a | Wake distance | 4 levels, 1053 monsters first seen | wake only inside about 7 to 8 cells (35 subtiles); 0 of 771 beyond 9 cells in 1 s | aidist 35 subtiles (D2D-src); original: not measured | **cannot test** on original, matches d2d source |
| 2b | Hit wakes? / rally / leash | see section 3 | see section 3 | D2D-src: hit does not wake, no call for help | see section 3 |
| 3a | Items per kill | 1962 BM, 2196 CP, 1178 SF, 1148 DW, 1401 Den kills | 0.21 BM, 0.29 CP, 0.35 SF, 0.29 DW, 0.34 Den | live: loot lines 0.40 "seen" per kill incl. gold piles (V-live) vs d2d 0.39 incl. piles | **matches** (weak, see 3) |
| 3b | Gold piles per kill | same | 0.13 to 0.14 | live not separable | cannot test |
| 3c | Gold per pile | same | BM 2.9 (max 5), CP 5.3, Den 3.8, SF 13.7, DW 18 | live: "gold 0 -> 3 in 167 kills Blood Moor" (g1-log) | consistent, no formula source |
| 3d | Quality of equipment-like drops | 893 items | normal 38 %, low 35 %, magic 15 %, superior 7.5 %, rare 4.1 %, 1 set, no unique | doc 2.3 table is for 100 % MF qlvl 1 | **cannot test** (rare rate higher than the doc, bosses in the mix) |
| 4a | Level-difference penalty | 5 clvls x 5 areas | table applied (Carver mlvl 5: 42.0, 34.0, 26.0, 18.0 XP at clvl 10..13 = 100/81/62/43 %) | S mechanics-progression 2.1 | **matches** |
| 4b | XP per kill at clvl 4 to 6 vs live | BM 598, CP 1251, Den 456 kills | BM 21.2-22.0, CP 26.5-31.7, Den 28.8-31.1 | live (soak-analysis 2): BM 18.3, CP 26-31, Den 16.8-19.4 | CP **matches**, BM **+20 %**, Den **+60 %** |
| 4c | XP per kill at clvl 12 | see section 5 | BM 0.45, CP 1.9, Den 2.4, SF 15, DW 35 | live 22 avg at clvl 11-12 (g1-log: 11.2k XP/h, 499 kills/h) | matches only for SF/DW-type areas |
| 5 | Vendor prices | 10 consumables, 14 gear items | consumables equal our tables; sell = 50 % of buy; Tome of TP 250 | live: Tome of TP 450 (V-live), sell lore 25 % (U) | consumables **match**; Tome **-44 %**; sell **differs by 2x vs lore** (U); gear prices unverified |
| 6 | Resist penalty | 2 classes x 3 difficulties | 0 / -40 / -100 on all four, cap 75 | 0/-40/-100 (S) | **matches** |
| 7a | Sorceress cast period | 8 spells, 8 casts each | 14 ticks for every spell | S table: 13 (0 FCR), Lightning and Chain Lightning 19 | **differs by +8 %** (Fire/Charged/Ice Bolt, Fire Ball) and **-26 %** (Lightning, Chain Lightning) |
| 7b | Melee swing | Barbarian, Paladin x 5 weapons x 6 swings | 13 to 21 ticks, WSM scaling exact | no base table in our docs (U) | WSM formula **matches**, base U |
| 8 | Death gold, XP | 120 deaths (3 difficulties x 4 clvl x 10 purses) | 120/120 identical to the rule, XP loss 0 / 5 / 10 % of span | rule + S | **matches** the documented rule |

## 2. Regeneration, potions (check 1)

Method: `fid_regen.py` sets clvl 12, maxmana M (stat 9), mana 0, life 10 of 100, steps 1500 ticks (60 s), reads `dump stats` raw
(8.8 fixed point). Result V-d2d (`out/regen.json`): mana gain per second 0.195 (M 35), 0.488 (60), 0.781 (100), 1.172 (150), 2.051 (250), identical in
the Rogue Encampment and in Blood Moor (monsters dead). That is `floor(256*M/3000) * 25 / 256` to the digit, i.e. the
documented formula including the truncation (35 mana take 179 s to fill, not 120). Life gain 0 in all runs. Stamina (`fid_stam.py`): 0 to 72.6 of 74 in 10 s
standing (7.22/s = 74/256 per tick x 25).
Potions (`fid_potion.py`, fresh Sorceresses, 4 starting Minor Healing Potions each; 4 drinks at Vit 10 and 24 at Vit 150): every one of the 28 drinks +30.156 life,
first change on the tick of the drink, last 192 ticks later (7.68 s). Matches the 30 / 7.68 s of docs/sme/mechanics-combat.md 7.3.
Overlapping potions (chain drinking) were not run; mana potions were not run (a Sorceress starts with healing potions only; they
drop, but buying was not reachable, see section 6).
**Double fill**: with `debug stat 3 150` (Vit 150, confirmed by `dump stats`) 24 of 24 drinks gave the single amount. The documented rule gives about 37 %.
Either d2d does not implement it for this path or the roll does not read the written stat: U. If it is missing, high-Vit characters
get less out of potions in d2d than in the original (only matters for Barbarian and Paladin with Vit).

## 3. Aggro, wake, leash (check 2)

`fid_aggro.py`: on a 10-cell grid over every level, warp the player to a stop, record every monster seen for the first time (its
distance d0), step 25 ticks (1 s), call a monster awake if it left mode NU or moved more than 0.3 cell, kill everything near the stop and go on.
Bucket table (woke / n, boss kinds excluded), Blood Moor, Cold Plains, Stony Field, Den:

| d0 (cells) | BM | CP | SF | Den |
|---|---|---|---|---|
| 0 to 5 | 6 of 8 | 26 of 36 | 13 of 14 | 8 of 10 |
| 5 to 7 | 5 of 8 | 23 of 27 | 8 of 9 | 6 of 18 |
| 7 to 9 | 7 of 31 | 19 of 64 | 5 of 29 | 0 of 28 |
| 9 and more | **0 of 150** | **0 of 302** | **0 of 210** | **0 of 109** |

(Dark Stalkers do not occur in these Normal areas; Dark Hunters and Spearwomen behave the same. 1053 non-boss monsters in two seeds per area; 771 of them first seen at 9 cells or more, none woke; the farthest monster that woke was 8.0 cells away, the others 7.9, 7.8.)
So the wake edge sits at about 7 cells (35 subtiles); within a second not every monster inside has noticed yet (think cadence). This is
exactly D2D-src's "aidist 35 subtiles, no sight needed outdoors" and the Den shows no cave "area alert" within the 1 s window (D2D-src says the
area flag needs a monster that saw the player first). **Verdict: matches d2d's own source; cannot test against the original** (our bot has never measured
a wake radius live; d2d also updates monsters 30 cells out, more than Game.exe's active rooms, D2D-src deviations).

Hit test and pack behaviour (`fid_hit.py`, `out/hit7.jsonl`, `out/hit8.jsonl`; 8 trials A, 2 runs of B and C, small N):
- **A, hit from outside the wake radius** (Fire Bolt level 1 at a sleeping monster 6.3 to 9.6 cells away; 75 ticks later): 9 trials, 7 hit (hp dropped), 2 missed. The six hits from outside 7 cells (Zombie, Fallen Shaman x2, Moon Clan x2, Carver)
  ended in mode WL or NU but **moved at most 0.31 cell in 3 s** (a stagger, not a chase). In 5 of those 6 trials **no neighbour within 7 cells woke** (4 to 8 neighbours each, all NU, moved 0). In the sixth (Stony Field, Moon Clan hit at 9.0 cells) three neighbours, two of them
  Dark Rangers (archers), were awake and moving, one attacking: not separable from their own notice radius (ranged monsters, **U**). The one clear chaser was a Dark Hunter hit at 6.3 cells, inside the radius (mode RN, moved 2.9 cells). So a hit does not wake a monster outside its notice radius and
  does not alert the pack: matches D2D-src ("being hit does not wake a monster, there is no call for help"). Small N (6 hits).
- **B, rally**: player placed 6.5 cells from a sleeping pack leader (Carver pack of 8, members 0 to 7 cells from the leader): 1 of 8 (run 1: 2 to 3 of 7) awake at any time over 200 ticks, the others stayed asleep although pack mates were awake and within a few cells.
  No pack-wide wake. (Only a Carver and a Dark Hunter pack were tested: the Fallen "leader within 15 / Shaman sets the group charging" rule of D2D-src was not triggered and is **untested**.)
- **C, leash**: after B the player is warped 30 cells away; 50 ticks later all pack members are idle (NU) and stay so for 300 ticks (mean distance 27 to 33 cells): monsters give up at once when the target leaves their range, they do not path across the map. (A real player running away at speed, which drags a chaser to the edge of its leash, was not simulated: **U**.)
Verdict: matches d2d's source; the original's behaviour is **cannot test** here and must be settled live (plan-v2 already lists aggro ranges as a live item).

## 4. Drops (check 3)

`fid_sweep.py`: fresh d2d per seed (clvl 10 to 14 cycle, Sorceress with 600 life), every area swept on a 10-cell warp grid, `debug kill` at every stop
(counts as real kills with XP and drops, V-d2d spike), every new ground item picked up and read with `dump items`, inventory cleared. Gold piles read from the `ground` label.
Seeds: 20 (BM), 11 (CP), 8 (SF), 5 (DW), 19 (Den). Kills per area are unique unit ids. `fid_drops_report.py` output (all data, rounded):

| Area | seeds | kills | items/kill | gold piles/kill | gold per pile mean (median, max) | gold/kill |
|---|---|---|---|---|---|---|
| Blood Moor | 20 | 1962 | 0.213 | 0.137 | 2.9 (3, 5) | 0.40 |
| Cold Plains | 11 | 2196 | 0.294 | 0.134 | 5.3 (5, 65) | 0.72 |
| Stony Field | 8 | 1178 | 0.346 | 0.123 | 12.3 (8, 155) | 1.51 |
| Dark Wood | 5 | 1148 | 0.294 | 0.141 | 18.0 (15.5, 145) | 2.54 |
| Den of Evil | 19 | 1401 | 0.335 | 0.125 | 3.8 (3, 11) | 0.48 |

(7885 kills in total; clvl 4 to 6 and 10 to 14 mixed, gold per pile does not depend on clvl: Blood Moor 2.9 to 3.1 across clvl 10 to 14. 0 pickups failed.)
Quality of the 893 equipment-like items: Blood Moor low 48 % / normal 30 % / superior 12 % / magic 8 % / rare 2 %; Cold Plains 37 / 39 / 6 / 14 / 4; Stony Field 25 / 42 / 10 / 19 / 3 (+1 set); Dark Wood 26 / 48 / 4 / 18 / 4; Den 37 / 33 / 6 / 17 / 7.

Reading:
- **Items per kill**: 0.21 to 0.35. Our live loot logs (32 soak logs in `~/d2lod-live/soak24/runs`, 893 `loot:` lines, 5076 `combat kill` lines): 2056 items+piles "seen" =
  0.40 per kill (an upper bound: lines are written per fight, not per kill). d2d: items 0.29 + piles 0.13 = **0.42** (all areas, 7885 kills). **Matches** (weak: the live
  number mixes areas and is not per kill). Live gold taken: 2762 gold in 893 lines (small piles are skipped by policy), so gold/kill is not comparable.
- **Gold**: piles are tiny where we farm early (BM 3, CP 5, Den 4 gold, max 5 / 65 / 11), 14 to 18 in Stony Field and Dark Wood. Independent of clvl in the 10..14 range (BM 2.9 to 3.1 over five clvls). The original formula is not in our docs (U), so there is no expected value; the bot must not count on pile gold before Stony Field.
- **Item mix**: potions dominate (Minor Healing is the most common drop in every area: 30 to 48 per area), then arrows/bolts, mana potions, keys, scrolls (TP/ID),
  junk armor and weapons. Dropped TP scrolls: 13 (BM) to 12 (Den) per 1000 kills.
- **Quality** (893 equipment-like items, potions/scrolls/keys/ammo excluded): normal 38.1 %, low 34.9 %, magic 15.2 %, superior 7.5 %, rare 4.1 %, one set item (0.1 %), **no unique**. The doc's
  table (mechanics-items-drops 2.3) is for 100 % MF and qlvl-1 items; rare 4.1 % is about 3x its normal-monster 1.25 %, but the mix contains superuniques (Corpsefire, Bishibosh), champions
  and uniques (about 2 % of monsters, see the area lists), which roll rares much more often. **Cannot test**: per-source quality needs the source of each item, which `ground` does not give. D2D-src (`drops.md`) says its
  roll code was diffed 100000/100000 against game.exe in an emulator; that is the best evidence we have.
- Not measured: unique/set rates (1 set and no unique in 893 equipment-like drops, expected < 2 % for normal monsters), rune drops, chests.

## 5. XP (check 4)

`fid_xp_report.py` (same sweeps, clvl and exp reset before every kill so the clvl stays fixed; XP per stop, then least squares per monster type):

Average XP per kill by area and clvl (all monsters of the sweep, kills = unique units, 7885 in total), and kills to the next level:

| Area | clvl 4 | 5 | 6 | 10 | 11 | 12 | 13 | 14 |
|---|---|---|---|---|---|---|---|---|
| Blood Moor | 21.5 | 22.0 | 21.2 | 4.8 | 0.41 | 0.45 | 0.39 | 0.38 |
| Cold Plains | 26.8 | 26.5 | 31.7 | 15.3 | 8.3 | 1.9 | 2.0 | 1.3 |
| Stony Field | - | - | - | 26.7 | 24.3 | 15.1 | 10.8 | 5.9 |
| Dark Wood | - | - | - | 52.6 | 41.6 | 34.6 | 29.0 | 16.8 |
| Den of Evil | 30.1 | 31.1 | 28.8 | 11.2 | 4.4 | 2.4 | 1.9 | 0.68 |

Kills to the next level at clvl 12: Dark Wood 652, Stony Field 1489, Den 9365, Cold Plains 11,989, Blood Moor 50,597. Monster levels seen (Normal): Blood Moor 1, Cold Plains 1 to 2 (superunique Bishibosh 5), Den 1 to 2 (Corpsefire 4), Stony Field 1 to 5 (Rakanishu 8), Dark Wood 5 to 6 (Treehead 8).
At 500 kills/h (our live rate) that is 1.3 h for Dark Wood and 3 h for Stony Field at clvl 12.

- The level-difference penalty of docs/sme/mechanics-progression.md 2.1 is applied (every row of the implied-base column is constant within rounding for the
  single-type rows: Carver 5 gives 42.0 / 34.0 / 26.0 / 18.0 at clvl 10 / 11 / 12 / 13, i.e. 100 / 81 / 62 / 43 % of 42). Fallen (mlvl 1, base about 0.5 XP) round to 0 below clvl 11:
  integer XP per kill, as in the original.
- Minions and uniques pay about 5x, champions about 3x (e.g. Champion Skeleton 80 vs Skeleton 14 at clvl 10), as documented.
- **Comparison with live (V-live, docs/soak-analysis.md section 2)**: at clvl 4 to 6 d2d pays Cold Plains 26.5 to 31.7 XP/kill (live 26 to 31 at clvl 3 to 5) = matches; Blood Moor 21.2 to 22.0
  (live 18.3 at clvl 5) = +20 %; Den of Evil 28.8 to 31.1 (live 16.8 to 19.4 at clvl 4 to 6) = **+60 %**. The Den gap may be the mix (d2d counts every monster including Corpsefire 19 XP, minion packs
  at 5x; the bot may not clear all of them) rather than the formula; live per-kill XP by monster is not logged, so it cannot be split.
- **clvl 12 reality check**: d2d pays 0.45 XP/kill in Blood Moor, 1.9 in Cold Plains, 2.4 in the Den, 15 in Stony Field, 35 in Dark Wood. Live "11,239 XP/h at 499 kills/h at clvl 11 to 12" is 22.5 XP/kill:
  consistent with Stony Field (24.3 at clvl 11) or a Stony Field/Dark Wood mix, **not** with Cold Plains or the Den. A bot test that farms Cold Plains or the Den at clvl 12 on d2d will see about 1k XP/h, and that is probably also what
  the original does (monster levels are the same table: Normal uses the monster's own level, D2D-src). The pace tables (docs/sme/pace-tables.md) that give 10 to 25k XP/h for Cold Plains at clvl 5 are right for clvl 4 to 6 only.

## 6. Vendor prices (check 5)

The trade window is reachable, but only in **real time** with a particular click sequence: walk next to the NPC (`cmd move` beside it, then `cmd interact <index>` until `menu` lists the lines and the player is within 3 cells), `move` onto the `trade` row,
wait 10 ticks, `click`, wait 10 ticks, `click` again (`store=<vendor index>` in `state` then, Akara 0, Charsi 2). In lockstep, or with a single click without hover, the click is ignored (V-d2d, 6 tries; cause U). Then: a **right click** on a store cell buys the item into the first free inventory cell
(left click does nothing), the gold delta is the price; a left click on an inventory item and a left click on an empty store cell sells it (gold delta = sell price). Script `fid_vend3.py NPC CMD...` (real time, throwaway Sorceress, clvl 12, Normal, 20000 gold), raw output in this session's transcript only.

| Item | d2d buy | d2d sell | Expected buy (source) | Verdict |
|---|---|---|---|---|
| Scroll of Town Portal | 100 | 50 | 100 (S economy-data 2, live 100 V-live per the brief) | **matches** |
| Scroll of Identify | 80 | 40 | 80 (S) | matches |
| Tome of Town Portal (5 scrolls) | **250** | 125 | **450 live at clvl 3 (V-live, economy-data)**, base cost 250 (S) | **differs by -44 % from live** |
| Tome of Identify (5) | 200 | 100 | 200 (S) | matches table, live U |
| Minor Healing Potion | 30 | - (Minor Mana sold for 30 = 50 %) | 30 (live V) | matches |
| Minor Mana Potion | 60 | 30 | 60 (S) | matches |
| Stamina 25, Antidote 40, Thawing 25, Key 45 | as listed | - | 25 / 40 / 25 / 45 (S) | match |
| Quilted Armor | not seen as a plain item: Superior (socketed 1) 131, Strong of Health (magic) 131 | - | 140 live | cannot compare, but see below |
| Cap (socketed 2) | 60 | - | 64 | |
| Leather Gloves (Superior) | 75 | - | 80 | |
| Buckler of Strength (magic) | 63 | - | 68 | |
| Short Sword of Icebolt (magic) | 67 | - | 72 (plain) | |
| Sturdy Leather Armor of the Jackal (magic) | 450 | - | 481 (plain) | |
| Repair, Short Staff 10 of 20 durability | **10** | - | formula U (S economy-data: unknown) | cannot test |
| Identify at Cain | not reachable (Cain is not in the camp before the Tristram rescue) | | 100 per item without the quest (S) | cannot test |

- **Sell = exactly 50 % of the buy price** for every item tried (scrolls, tomes, potions). Community lore in our docs (economy-data 4) says about a quarter (U); if the lore is right, bot gold-per-hour estimates built on d2d sells are 2x too high. **Must be checked on the original**.
- **Tome of Town Portal 250 vs live 450**: the live number is a measured price (leveler.md, clvl 3) and the brief lists it too; d2d charges the table cost. Either d2d misses a charge/stack surcharge or the live price has one. Affects the bot's early gold plan (a tome is the first purchase).
- Magic and superior items at Charsi cost about the same as the plain base (6 % under, not 2 to 8 times over as in the original: live Wand 205 plain vs 1669 magic, economy-data 5): d2d's vendor pricing of affixed items looks wrong or is based on the base only. 7 weapon and armor samples (Hand Axe of Health 159,
  Large Axe 331, Two-Handed Sword of Health 603, Spear (ethereal) 281, Axe (socketed) 377, Scimitar 256) show no consistent relation to the table costs either, so the price formula of d2d for weapons is **unverified** and the bot must not plan purchases of non-potion gear from d2d prices.
- The potion/scroll/tome/key rows equal our price tables to the gold: the cheap-consumables loop (buy TP/potions, sell junk) can be tested on d2d for flow, not for the sell-side income.


## 7. Resists (check 6)

`fid_resist.py`: `debug difficulty 0/1/2/0` on a Sorceress and a Paladin (clvl 12), `debug attack` and `dump stats` after each. Result V-d2d: res=0/0/0/0, -40/-40/-40/-40, -100/-100/-100/-100
(fire/lightning/cold/poison), cap 75 each, back to 0 on returning to Normal. Matches the original penalties (S). Note: `debug difficulty` also moves the character to that difficulty's act-1 town and levels are rebuilt.

## 8. Combat timings (check 7)

`fid_timing.py`. Sorceress (clvl 12, skills set by `debug points`): the cast command is re-issued every tick and the period between mana drops is the
cast duration. Every spell gave **14 ticks**, 8 casts each: Skill ids 36, 38, 39, 45, 49, 53, 47 = Fire Bolt (mana cost 2), Charged Bolt (4), Ice Bolt (3), Ice Blast (7), Lightning (9), Chain Lightning (11), Fire Ball (6) at 3 levels each (the names are from memory of Skills.txt, U; the distinct mana costs prove the right skill fired)
(the single-cast `SC` mode also lasts 14 ticks, the bolt appears on tick 7). Expected (S, docs/sme/mechanics-breakpoints.md 2): 13 frames at 0 % FCR for Sorceress spells, **19** for Lightning and Chain Lightning.
Verdicts: bolts and Fire Ball +1 frame (+8 %, matches under the 20 % rule, but systematic; the documented formula carries a `-1` that d2d's tick counting may not), **Lightning and Chain Lightning 26 % faster than the table** (the
animation difference is not modelled) = **differs**. Impact: a Lightning/Chain Lightning build tested on d2d casts a quarter faster than in the original; the bot's default Charged Bolt line is within 8 %.
Melee (`debug equip` weapon in the right hand, one `cmd skill 0` next to a monster, count ticks in mode A1, 6 swings each, deterministic):

| Weapon (WSM, S Arreat) | Barbarian | Paladin | formula check (base B: ceil(256*B/floor(256*(100-WSM)/100))) |
|---|---|---|---|
| Scimitar (-20) | 14 | 13 | B 16 / 15: 14 / 13 |
| Long Sword (-10) | 15 | 14 | 15 / 14 |
| Short Sword (0) | 16 | 15 | base |
| Short Spear? (+10) | 18 | 19 (spear class animation) | 18 |
| War Hammer (+20) | 21 | 19 | 21 / 19 |

The Barbarian base is 16 ticks and the Paladin 15 for a 0-WSM sword; WSM scaling follows the documented formula exactly (without the `-1`). Our docs do not hold base attack frames per class and weapon (U), so the
level cannot be compared; the +1 of the cast (14 vs 13) hints the same offset may apply. `debug attack` shows IAS/FCR/FHR fields (`wsm`, `ias`...) but no item with IAS was available to test the IAS equation. **Monster attack rates were not measured** (needs MonStats data: cannot test).
Walking speed (Sorceress, light, town) 1.87 cells per second (9.3 subtiles/s) was incidentally measured (U against the original).

## 9. Death rules (check 8)

`fid_death.py`: clvl 10, 12, 14, 20 x 10 (purse, stash) pairs (0..50000 purse, 0..20000 stash) x Normal/Nightmare/Hell, `debug die`, dismiss, read purse, ground pile and exp.
Rule tested (docs/sme/mechanics-items-drops.md 9, d2d-probes.md 3): toll = min(clvl, 20) % of (purse + stash), never taking the total below clvl x 500, taken from the purse; the rest of the purse drops as one pile; XP loss 0 % Normal, 5 % NM,
10 % Hell of the current level's span. Result V-d2d: all 120 cases give pile = purse - toll exactly (e.g. clvl 12: purse 6000 / stash 5000 -> pile 4680; clvl 20: 50000 / 20000 -> 36000; clvl 10 purse 6000 -> 5400 because of the 5000 floor); the pile is picked up again
as soon as the respawned player stands within half a cell of it (d2d's gold autoloot, a deviation from the original, deviations.md improvement 5), which made the first readings of each block look like zero tolls until the purse after respawn (or the pile delta) was used; all 120 cases verified that way. XP loss: Normal 0 at every clvl;
Nightmare 721 / 1127 / 1761 / 6718 at clvl 10 / 12 / 14 / 20 and Hell exactly double: 5 % and 10 % of the span (e.g. clvl 10: span 14,429, 5 % = 721). The earlier probe's "toll 0 below clvl x 500" holds (clvl 12, purse 1100: pile 1100, purse 0).
Also seen: after 17 deaths the corpse list stayed at 17 (probable cap, the original keeps one body; U). **Verdict: matches the rule as documented (see the 2026-10-30 autoloot correction below); the rule itself (and the level x 500 floor) is still the d2d/doc rule, to be confirmed on the original** (G1 case: 1125 gold at clvl 11, all left the purse, V-live, consistent).

### Correction 2026-10-30: re-run with `--toggle autoloot=off` (agent D2DTOGGLE)

Every probe above ran with d2d's default QoL deviation `autoloot` ON (our start scripts passed no `--toggle`). Re-run on the private fork,
throwaway Sorceress, `--no-save`, lockstep, `autoloot=off,autoparty=off`, script `~/d2d-lab/fidelity/fid_death2.py` (18 cases: Normal/NM/Hell x clvl 10 and 20 x
purse/stash 6000/5000, 50000/20000, 1100/0; same die/dismiss sequence as `fid_death.py`). Result V-d2d:
- The pile is **not** picked up on respawn any more: `purse_after` = 0 in 18/18 cases (before, the purse after respawn already included the pile).
  Piles accumulate on the ground within one game, so the per-death pile is the delta of the ground total.
- The pile of each death equals purse - toll **exactly in 18/18** (clvl 10, 6000/5000 -> 4900; clvl 10, 50000/20000 -> 43000; clvl 20, 50000/20000 -> 36000;
  1100/0 -> 1100 at both clvls: the clvl x 500 floor). XP loss unchanged: 0 Normal, 721 / 6718 Nightmare and 1442 / 13437 Hell at clvl 10 / 20.
- So the earlier numbers **still hold**: death gold = min(clvl,20) % of (purse + stash) (floor clvl x 500 on the total), taken from the purse, the rest of the purse dropped as one pile. Only the
  first-reading artefact (zero toll) is gone, and the old "all 120 verified via purse after respawn" method was valid only because autoloot re-added the pile.
- Autoloot off, walking: the respawned player stood exactly on a 5940 pile (`cam` = pile x,y) for 200 ticks: purse stayed 0 and the pile stayed in `ground`; `cmd pickup <unit>`
  then walked/picked and gold became 5940, pile gone (V-d2d). First case: `cmd pickup 1` in the 18-case run likewise 0 -> 4900.
- Other probes that may have been touched by autoloot: gold per kill piles (section 5, read from `ground` right after a kill, so piles were probably still there, but a
  character walking into a pile first would have lost it from the count: treat gold/kill as a lower bound, U) and the 8 h soak model. Re-run before relying on them.

## 10. Ranked deviations that matter for the bot

Ranked by what a wrong d2d number would cost the bot's tests (1 = worst):

1. **Combat danger is not validated at all (cannot test).** Monster damage, attack rate, AR, HP scaling and the original's wake/leash ranges were not measured; the aggro radius d2d uses (7 cells) is its own source's number, d2d updates monsters 30 cells out (more than Game.exe's active rooms, D2D-src), levels are built once and never respawn, and there is no deterministic original to compare. A death-rate comparison (check 9 criterion) is therefore the least trustworthy metric; use d2d for "does the logic hold together", not for "how often does the bot die".
2. **Lightning and Chain Lightning cast in 14 ticks instead of 19** (-26 %). Any Sorceress plan that relies on them (our default build is Charged Bolt, so today it is only a future risk) is tested 36 % too fast. Frost/Fire/Charged Bolt are +1 frame (+8 %) over the table.
3. **Vendor economy**: sell = 50 % of buy (lore says 25 %, U), Tome of TP 250 vs live 450, magic/superior gear priced at about the base cost, repair 10 gold for a 10 point Short Staff (formula unknown). Gold plans (when can we afford a tome, how fast does selling loot pay) cannot be calibrated on d2d; consumable buy prices are right.
4. **XP per kill in the Den +60 % and Blood Moor +20 % vs live at clvl 4 to 6** (Cold Plains matches). Our live per-kill XP by monster is not logged, so the composition cannot be separated from the formula. At clvl 12 the low areas pay almost nothing on d2d (Cold Plains 1.9, Den 2.4, Blood Moor 0.45 XP/kill): a farming plan that stays there will stall; the penalty table itself matches.
5. **Potion double fill missing (0 of 24 at Vit 150)**: potions on d2d are weaker than documented for high-Vit/Energy characters; harmless for a Sorceress at clvl 12 (2 % anyway).
6. **Gold autoloot** (now OFF in our standard launch, see docs/d2d-fork.md section 10; d2d by default walks into gold within half a cell; the original needs a click): the bot's pile-pickup behaviour and its "small gold pile skipped" policy cannot be tested for real; death recovery picks the pile up by accident.
7. **Corpse list holds many bodies and any can be claimed** (earlier probe) and stops at 17 in our run; the original keeps one. Item/XP recovery logic is easier on d2d than in the original.
8. **Drop quality mix** includes superuniques; rare 4.1 % of equipment-like drops is about 3x the doc figure for normal monsters. Per-kill item counts (0.29 + 0.13 piles) agree with live. Unique/set rates untested.
9. Lockstep/probe artefacts, not fidelity: levels built once per process (use restarts or new seeds for new monsters), RAM +100 MB per level, trade window only reachable in real time.

**Must still be verified on the original game** (each one is a live probe, nothing here can replace it): (a) wake distance and give-up/leash distance of Fallen, Zombies, Quill Rats and Dark Stalkers, plus whether a hit wakes a monster outside its range and whether Fallen packs rally; (b) cast and attack frame counts for our Charged Bolt Sorceress (13 or 14?) and Lightning (19?) at 0 FCR, and one IAS/FCR breakpoint; (c) monster damage per hit and attack interval at Normal Act 1 (death rate depends on it);
(d) vendor sell ratio (25 % or 50 %), Tome of TP price (250 or 450, and what changes it), repair and identify costs, magic item price markup; (e) item quality rates for normal monsters at 0 MF (a few thousand kills, loot logging with quality) and unique/set rates; (f) the death toll rule below clvl x 500 and the single-body corpse rule; (g) XP per kill by monster at clvl 4 to 6 in the Den and Blood Moor (log the monster name with each XP gain);
(h) potion double fill and chain-drinking; (i) gold autoloot absence (our live loot logic handles piles by clicking).

## 11. Replicated soak on d2d (check 9)

Goal: run the **same character plan** for a simulated 8 h on d2d in minutes and compare three numbers with our live runs (docs/soak24.md, soak-analysis.md): deaths/h, XP/h, gold/h.

**What runs where.** d2d with `--lockstep --seed S --start-screen ingame` (menus need real time, so a run starts a character with `--start-screen ingame` or loads one with lockstep off, then `lockstep on`). The bot drives d2d through `internal/d2dctl` and `internal/d2dsrc` / `d2dinput` (docs/d2d-backend.md phases 1 to 2b). Every decision point calls `step n` (n = 5 ticks = 0.2 s of game time is a human-like
reaction granularity); the bot's `Clock` must be the d2d `game_ms` and nothing in the loop may sleep or time out on wall time (a wall-clock timeout would fire in 1/100 of the game time). Replies of `step` are deferred, so the call itself is the wait.

**Cost.** 8 h = 720,000 ticks. Measured lockstep speed: 3000 ticks/s in town, 9400/s in an empty field, 2200 to 9000/s with monsters awake (docs/d2d-fork.md section 4); a bot that steps 5 ticks per decision spends 144,000 decisions, at about 0.5 ms per
request round trip about 1.5 to 2 minutes of RPC on top of 2 to 5 minutes of simulation: **about 5 to 10 minutes of wall time per simulated 8 h on one core** (U for the bot loop's own cost: our Go decision code is not measured under this rate). 10 seeds in parallel (one d2d each, 10 of the worker host's 16 cores, `nice 10`, load permitting) give 80 simulated hours in under 15 minutes.
Memory grows by about 100 MB per first-visited level (docs/d2d-probes.md): restart the process (save, quit, load) every ~10 level builds or every simulated hour, as the live bot does with its fresh game every ~4 minutes (which also re-rolls the levels, as d2d's restart does).

**What the harness needs beyond what exists today** (ranked): (1) a `Clock` backed by `clock`/`step`; (2) the character loop's lifecycle verbs on d2d (new game with levels re-rolled: `save` + restart wrapper `restart.sh NAME`, difficulty switch `debug difficulty`); (3) death handling: d2d's death screen needs a click + Escape and the body is claimed with `cmd interact -3000 - i`; (4) `dump items/stats/quests` instead of memory reads (the fork's `dump`); (5) a vendor flow that works in lockstep (click hover rules above are real-time only: either switch `lockstep off` for the town trip, or ask upstream for the fix, or buy with `cmd` verbs if one exists: U); (6) pack/gold/loot policy via `ground` + `cmd pickup` (gold needs no pickup on d2d: policy must tolerate that).

**Pass criterion** (adopts the 20 % rule of docs/d2d-backend.md section 7): for each metric M in {deaths/h, XP/h, gold/h}, with live value L (>= 24 h of live soak at the same plan, areas and clvl band) and d2d value D (pooled over >= 10 seeds x 8 simulated hours, bootstrap 95 % interval over runs):
`|D - L| / L < 0.20` **and** L inside D's interval. Deaths/h needs the most data: at the live 0.5 deaths/h (g1-log attempt 14) a 20 % resolution needs about 25+ deaths per side, i.e. >= 50 simulated hours (the 10 x 8 h batch gives 40 hours: 20 deaths, interval +- 40 %: widen the pass to "L inside the interval" for deaths until a second batch). Compare only the same clvl band, area list and gear tier;
report the table per area (XP/kill by area is the finest check: Den +60 % and BM +20 % already break it today, section 5). Failing metrics point to the check above that explains them (vendor sell ratio for gold/h, area XP table for XP/h, check 2 for deaths/h).

**Expected today (before any fix)**: XP/h matches in Cold Plains and Dark Wood-like areas and is 20 to 60 % high in the Den/Blood Moor at clvl 4 to 6; gold/h from pile pickup is plausible but from selling it is ~2x live if the lore of 25 % is right; death rate unknown (item 1 of the ranking). So the replicated soak is a **regression and logic-coverage rig**, not yet a calibrated predictor; the first useful calibration step is the live probes listed under "must still be verified".
