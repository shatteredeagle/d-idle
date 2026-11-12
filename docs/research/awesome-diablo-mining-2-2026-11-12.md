# awesome-diablo mining, round 2 (2026-11-12)

> status: research, docs only (TASK-369, card mine2). Consumers: the leveling and campaign owners (plan tables, `docs/campaign.md`), the Skills page / tooltip owners (`docs/skilltrees.md`, `docs/inventory-tooltip.md`), the watch UI owner, and the "networked play is intended" decision. Nothing here is a decision; proposals are proposals.

## 0. Rules and tags

- Read only. Shallow clones into new scratch directories, nothing run, nothing built; deleted afterwards. Nothing copied into the repo: this page paraphrases facts and names files. Licences differ (GPL-3.0 in our repo, standard library only), so every row records the licence; "no licence: read-only reference" means do not reuse even small parts.
- Tags: **V** = I read it myself in a clone at the commit shown, or fetched the page (URL given). **S** = snippet, README claim or secondary source. **U** = unverified or from memory. V here means "the source says it", NOT "verified against our own oracle": no row is oracle-checked (we never ran their code or compared with game.exe), so no claim in this page is the strongest kind of verified.
- Target is classic 1.14d. D2R-only facts are flagged `[D2R]`. kolbot-SoloPlay runs on D2BS in real games (1.14d is a supported version, S README), so its behaviour is real play but of a bot with a different engine, merc and loot policy than ours.
- Clone commits: kolbot-SoloPlay `a5b6aa9` (2026-11-11), D2ItemToolkit `029069e`, d2itemreader `026bfad`, DiabloInterface `4757969`, MF_run_counter `f2281fd`, d2helper `8bfec5c`, diablo-soul `206d353`, d2gs `061d32f`, d2-dedicated-server `b661cac`, fabd/diablo2 `4511256`, BetweenWalls/portal `56eedf7`, d2-drop-simulator `83ffe36`, d2-recon (IgorKonovalov/diablo-2-recompilation) `83c8a5b`, d2-114d-codex `e4497e1d`.

## 1. Ranked source table (usefulness to us)

| # | Source | Licence | What we get | How to use | Risk |
|---|---|---|---|---|---|
| 1 | blizzhackers/kolbot-SoloPlay | GPL-3.0 (V LICENSE) | A working 1-99 leveling bot: area order by clvl, skill-point orders per build, attack skill per clvl band, potion/town/chicken settings (section 3). Closest thing to "what a bot should do" | Read as a benchmark; copy NO code (GPL-3.0 happens to match our licence, but we are Go stdlib-only and its structure is JS/D2BS); turn findings into data rows in our own tables | Different engine (D2BS has teleport/packet casting, pickit, merc rules); some choices are for speed of the whole 1-99 run, not clvl 1-12 xp/h |
| 2 | IgorKonovalov/diablo-2-recompilation (d2-recon) | no licence file: read-only reference | Reviewed 1.14d specs: monster AI catalogue (344 constants), monster level pool and spawn, to-hit, damage, reductions, stats and states; evidence-tagged. 21 AI findings docs, `docs/specs/monsters/ai/catalogue.md` (4720 lines) | Cross-check source for d2d deviations and for combat tuning; cite spec path + commit; never copy pseudocode | Documentation by an agent pipeline (their own tags C/I/L); not verified against us. No licence: ideas and facts only |
| 3 | Nugwa628/d2-114d-codex | no LICENSE file; LEGAL.md only: read-only reference | 1.14d function/struct maps and specs (sim: pathing, rng, tick, stat lists; monsters: AI bodies, umods; combat hit/damage/vitals) | Second opinion next to d2-recon and `docs/d2-1.14d-memory.md`; the pathing and RNG specs help the lockstep / nav owners | 226 MB clone (tsv, ghidra exports); same unverified status |
| 4 | ResurrectedTrader/D2ItemToolkit | MIT source; embeds Blizzard tables (not covered) | The 1.14d item tooltip engine reconstructed from disassembly, byte-identical goal. Shows what a full tooltip contains (section 4) | Spec of what our tooltips lack; reimplement the needed parts in Go from the user's own tables at runtime | Written by an AI agent "without human supervision" (README, V); its `docs/writers.md` is NOT in the repo (README names it, V: `git ls-files` shows no docs); embedded Blizzard tables must not be committed |
| 5 | jaenster/d2-dedicated-server | MIT | Headless 1.14d game server (wine and a native Linux build), realm daemon, tested with real 1.14d clients; multiplayer and "a full party of eight" work (V docs/STATUS.md). Section 6 | Reference for the networked-play decision | Needs the user's own game files; see section 6 for risks |
| 6 | espenmjos/d2gs | MIT | One-command PvPGN + D2GS closed realm (Docker, Wine 9). D2GS is 1.13d DLLs with the 1.14d `Patch_D2.mpq` (V README), so it is NOT a stock 1.14d server | Cheapest way to a LAN/closed realm for tests | Server engine differs from 1.14d (1.13d binaries): not a fidelity oracle |
| 7 | jaenster/d2-clientless | no licence: read-only reference | A clientless 1.14d client in Zig (BNCS, MCP, BNFTP, D2GS) (V README); readable protocol reference | If a networked or headless bot is ever wanted | No licence; Battle.net rule in AGENTS.md ("do not add anything that talks to Battle.net") applies to anything built from it |
| 8 | jaenster/libd2 | MIT (V LICENSE text; the API reports NOASSERTION) | Clean-room Zig 1.14d core: map gen (seed to world), item generation, D2GS wire protocol, saves, pathfinding; bindings including Go (`go get github.com/libd2/go`, S README) | A possible oracle for map seeds, item drops and paths (fidelity cards); check licence of each package before reuse | Not read in depth (U); Go binding is a third-party dependency, we are stdlib-only |
| 9 | fabd/diablo2 | MIT | Area level and experience table by difficulty (`wiki/experience.html`), generated by `code/levguide.pl` from 1.13 data (V) | Quick cross-check of area levels (section 3.3) | 1.13 data, 2021; ours already has `docs/sme/mechanics-progression.md` |
| 10 | squeek502/d2itemreader | public domain (Unlicense, V LICENSE) | C parser for .d2s/.d2x items and `docs/formats/{d2,d2s,atma,plugy}.md` format notes | Reference for the item bit layout (save bits, params) if we decode item blobs | We read memory/d2d text, not saves: low use today |
| 11 | DiabloRun/DiabloInterface | MIT | Streamer/speedrun overlay (memory reader for 1.13c to 1.14d) | UI ideas, section 5 | none (ideas only) |
| 12 | oskros/MF_run_counter | MIT | Run counter, drop log, holy-grail tracker, session timer | UI ideas, section 5 | none |
| 13 | dorianprill/d2helper | no LICENSE file: read-only reference | Passive packet-capture overlay (egui), listens on D2GS port 4000 | Idea: passive observer of the networked game | Rust; on the networked path only |
| 14 | sagosand/diablo-soul | no licence: read-only reference | A bot narrated as a web MUD (event diff to prose, WebSocket, watch from a phone) | UI ideas, section 5 | Diablo 1 / DevilutionX |
| 15 | pairofdocs/d2-drop-simulator | AGPL-3.0 | Boss/monster drop simulator from TreasureClassEx/ItemRatio, seedable, session log (V README) | Concept check of our TC/MF maths (`docs/research/treasure-trace.md`) | AGPL: ideas only; 1.13d data folder |
| 16 | BetweenWalls/portal | AGPL-3.0 | Character planner: skill info to clvl 60, synergy maths | Not mined (U): data layout is JS per skill | AGPL; skill data we already have from tables |
| 17 | ThePhrozenKeep/D2MOO, CE_Database | MIT (D2MOO V API), CE_Database none | D2 source-like reimplementation / code-edit database | Already used by other cards (`docs/d2-1.14d-memory.md`) | not re-read |
| 18 | Roger-Takeshita/Kolbot_Roger-That | no licence | Fork of kolbot with D2Bot.exe committed | Not useful: skip (cloned, listed, deleted) | Binary in repo |
| 19 | tlentz/d2modmaker | GPL-3.0 | Go program that edits game tables from a config (V: `cmd`, `internal`, `configs`) | A Go example of reading/writing Excel tables, nothing for gameplay | Mods tables; irrelevant to a fidelity bot |

Web (outside the lists, fetched): Blizzard's Arreat Summit experience page (archived), Maxroll Paladin leveling and general leveling, Almar's leveling guide. Rows in section 3.4.

## 2. Highest-value findings in one screen

1. kolbot-SoloPlay levels a non-Barbarian in Normal by Den (once, for the quest), then Blood Raven early (it clears the Cave first when below clvl 5), then repeats **Tristram until clvl 12** (V `Tools/SoloIndex.js`). Our hour1 run did Blood Moor 41 percent and Den 28 percent of the hour and reached clvl 6; our campaign puts Tristram at clvl 9 floor, 11 recommended and optional. Two independent human guides (Almar, Maxroll) also say Tristram runs for roughly clvl 4-5 to 15 (section 3.4).
2. Both kolbot and Maxroll put **Holy Fire from clvl 6** on the Paladin; our v2 table drops Holy Fire for Might until clvl 16 and attacks with plain Attack until Zeal (clvl 12). 
3. kolbot refuses to farm the Den after the quest ("xp is bad in den", V comment in `SoloIndex.js`); our hour1 second Den farm earned 223 xp/min against Cold Plains 558 xp/min (V `docs/probes/hour1-analysis-2026-11-12.md` section 2).
4. D2ItemToolkit's `Ranges` (roll spans), `ShowItemLevel` and per-viewer lines (block chance, Smite damage, attack speed) are the features our item tooltip lacks on the game.exe backend.
5. Server projects exist that serve real 1.14d clients, including a party of eight (V d2-dedicated-server `docs/STATUS.md`), which makes the networked-play decision a build/run question, not a research question.

## 3. Leveling routes and build data

### 3.1 What kolbot-SoloPlay does (V, commit `a5b6aa9`; paraphrased)

Script order for Act 1 (V `Tools/SoloIndex.js`, `scripts`): corpsefire, mausoleum, den, bishibosh, bloodraven, tristram, treehead, countess, smith, pits, jail, boneash, andariel, a1chests, cows. Run conditions in Normal:

| Script | Normal run condition (kolbot) | Notes |
|---|---|---|
| corpsefire | only when `me.den` and in Hell | not a Normal script |
| den | quest not done; skipped at clvl 9-11 (Sorceress 9-17) | comment: "xp is bad in den so only run it once we can blast through". So it is run for the quest, never farmed |
| bishibosh | clvl above 10 AND broke; never for Sorceress | not an xp route |
| bloodraven | quest not done, or broke, or Barbarian not yet done Tristram | below clvl 5 it first clears the Cave level (`clearLevelUntilLevel(5)`) then goes (V `Scripts/bloodraven.js`). Reward noted in file header: free merc |
| tristram | `!me.tristram` OR clvl below 12 (Barbarian: 8) OR broke; skipped if clvl below 6 and the game is over 15 minutes old ("underground passage can be tough") | the leveling area. Needs Cain's quest chain: scroll from the Dark Wood tree, Stony Field waypoint, stones, Rakanishu kill |
| countess | rune need (checks for Tal/Eth etc.), or broke; Barbarian clvl 9-11 | not an xp route; clvl below 12 it takes a "scenic route" through hawk nests |
| smith | clvl above 6 (until the quest item is done) | Charsi quest, xp small |
| boneash | skipped below clvl 10; runs while clvl below 12 or broke | Cathedral / Inner Cloister area |
| andariel | skipped below clvl 11; first kill when not killed; again when broke | |

"broke" = gold below `500 + clvl*100*sqrt(clvl-1)` (capped), only from clvl 15 and once Act 2 is open (V `Core/Globals.js` `brokeAf`). Other global settings (V `Core/Globals.js`): leave for town below 35 percent life (softcore), chicken (leave game) at 10 percent, `UseHP` 80 percent, `UseMP` 65 percent, rejuvenation at 50 percent, merc potion at 75 percent, `ScanShrines` includes the experience, combat, skill, refilling, health, mana shrines. Default belt `hp,mp,mp,rv`.

### 3.2 Paladin: their data vs ours (normal difficulty)

kolbot Paladin "Start" build (V `BuildFiles/paladin/paladin.StartBuild.js`, active before the respec at clvl 19):

| Item | kolbot (V) | Ours (V: `internal/plandata/build-paladin-normal30-v2.json`, `data/campaign/normal.json`) | Difference |
|---|---|---|---|
| Skill points in order | Might (clvl 2), Sacrifice (3), Holy Fire first point (6), Resist Fire up to 4 (Den points), Holy Fire to 4 total (8), Smite (10), Zeal first point + Charge (12), Zeal to 4 (15), Holy Fire to 6 (17), Resist Fire to 16 after respec | Might 2, Sacrifice 3, Prayer 4, Might to 6 (5-9, 11), Smite 10, Zeal 12-15, Holy Fire 16-19, Charge 16 | They take Holy Fire at 6; we take it at 16. We take Prayer at 4; they never do. They take Resist Fire (4 pts) for Corpsefire/Den fire |
| Stats | vitality target 80, dexterity 27, strength 47 then all vitality | strength 2, vitality 3 per level (rows), +1 dex at 6 | Comparable (both mostly vitality) |
| Attack skill by clvl | 1: plain Attack; 2-5: boss skill Sacrifice if learned else Attack, with Might as aura; 6-8: Holy Fire as aura; 9-11: same, potion buffer shrinks; 12+: Zeal as attack with Holy Fire aura, Might fallback, Charge on | v2: plain Attack with Might until 12, then Zeal. Hotkeys planned: Might clvl 2, Holy Fire clvl 6, Zeal clvl 12 (same file) | The aura swap to Holy Fire at 6 is in OUR hotkey plan but the point is not bought |
| Low-mana skill | Attack | n/a | |
| Belt | 4 columns health at clvl 1-8 | not compared (`docs/belt-plan.md` has the plan options) | |
| Inventory potion buffer (`HPBuffer`) | 8 at clvl 2-8, then 2 (expansion) or 4 (classic) | not compared | spare potions kept to avoid town trips |
| Town trigger | life below 35 percent (softcore) | our policy sheet | compare in `docs/sme/potion-policy.md` (U) |
| Chicken | 10 percent | ours: chicken line in resource policy | not compared |
| Final build | respec at clvl 19 to Blessed Hammer + Concentration (their `LevelingBuild`) | respec disabled in our table (`respec.enabled=false`) | known; Maxroll agrees (respec at 19, V section 3.4) |
| Merc | wants Holy Freeze merc (Act 2 Normal later) | free Rogue from Blood Raven | |

Area progression by clvl (Paladin, kolbot vs ours):

| clvl | kolbot (V) | Ours (V campaign) |
|---|---|---|
| 1-4 | Blood Moor / Cold Plains waypoint on the way ("scenic route": it clears shrines and chests), Den (quest), Cave to reach clvl 5 | Blood Moor until 2, Cold Plains WP at 3, Den at 4 (floor 3) |
| 5-8 | Blood Raven (Burial Grounds) for the Rogue | Blood Raven 9 recommended (floor 8) |
| 6-11 | Tristram runs | Cain: floor 9, recommended 11, optional |
| 10-11 | boneash from 10, Charsi (smith) from 7 | Charsi's hammer 13 (floor 10) |
| 11-14 | Andariel from 11; Countess when runes needed | Countess recommended 14 (13 with merc); Andariel by per-class table |

### 3.3 Sorceress: their data vs ours

kolbot "Start" Sorceress (V `sorceress.StartBuild.js`): Charged Bolt to 3 (clvl 4), Ice Bolt (5), Frozen Armor (6), Telekinesis (7), Frost Nova (8), Static Field 4 (10), Nova to 7 (17), Teleport (18), Static Field 6 (20), Ice Blast (22), respec at 26 to Blizzard. Stats: energy to 40/45/50/60 interleaved with vitality (and strength 35); our rows: 5 vitality per level from clvl 2 to 24 (V). Attack: clvl 1 Fire Bolt; clvl 2-11 Charged Bolt then Ice Bolt/Ice Blast as the second skill (Fire Bolt if neither); clvl 12+ Nova with Charged Bolt, Frost Nova and Ice Blast alternates; Blizzard from 24. HP/MP buffers 4 and 10; belt `hp hp hp hp` at clvl 1 then `hp hp mp mp`.

| Item | kolbot (V) | Ours (V `build-sorceress_normal30.json`) | Difference |
|---|---|---|---|
| Early skills | Fire Bolt (free), Charged Bolt 3, Ice Bolt 1 (clvl 5), Frozen Armor (6) | Warmth 2, Charged Bolt 3-5, Frozen Armor 5, Frost Nova 6, Charged Bolt 7-10, Telekinesis 11, Ice Bolt 12 | We never use Fire Bolt or Ice Bolt before 12 and put a point in Warmth (mana regen) and Frost Nova |
| Energy | early energy to 40-60 | none (vitality only) | Their casters are mana-bound too (comment in file: Static Field/Nova for bosses); our measured Sorceress is mana-bound (clvl 3-4 in 49 min, TASK-138, in `docs/research/class-early-game.md`) |
| Den | any clvl 1-8 (skipped 9-17) | clvl 6 recommended (5 floor) | |
| Blood Raven | as Paladin, but Sorceress clears the Cave first below clvl 5 | clvl 10 | |
| Bishibosh | never | not a chapter | agrees |
| Teleport | clvl 18, Nova bridge 12-23 | Teleport 18, Ice Blast 13-23 (cold line, because Lightning/Chain Lightning are known broken on d2d, `docs/capabilities.md`) | We chose the cold line by necessity; Nova (7 points by 17) is their single-target-free pack killer |
| Final build | Blizzard / Blova | Blizzard 30 | agrees |

### 3.4 Other classes (kolbot "Start" builds; attack skill thresholds only, V)

| Class | Start build (desc) | Attack by clvl (V) |
|---|---|---|
| Amazon | Charged Strike build, respec at its respecOne | 1: Attack; 2: Jab; 6-7: Power Strike; 18: Charged Strike |
| Assassin | fire trap (Fire Blast, Wake of Fire) | table templates at 1, 2, 12 (values not extracted, U) |
| Barbarian | Battle Orders, Frenzy, Double Swing, Sword Mastery | 2: Bash/Attack; 6: Bash; 12: Double Swing; 24: Frenzy; respec at 30 |
| Druid | Firestorm + Fissure | 1: Attack, then Firestorm; templates at 12/13 (values not extracted, U) |
| Necromancer | Teeth + Bone Spear | 2: Teeth; 18: Teeth then Bone Spear; respec at 26 |

These agree in shape with `docs/research/class-early-game.md` (EXP): plain attack for 1-5, one cheap skill from 2-6, the build skill later. The Necromancer here is a Teeth/Bone caster, not our summoner.

### 3.5 Human guides, fetched (V, 2026-11-12)

- Arreat Summit (archived, `https://classic.battle.net/diablo2exp/basics/experience.shtml`, V): below clvl 25 a monster at most 5 levels above you gives 100 percent, 6 above gives 88 percent, 7 above 68, 8 above 36, 9 above 15, 10 or more 2; below you, 100 percent down to 5 levels, then 81/62/43/24/5. Party: +35 percent total in the same area, split by level. Players setting: experience times `(n+1)/2`, monster life times the same (V). This agrees with `docs/sme/pace-and-formulas.md`, no change.
- Maxroll Paladin leveling (`https://maxroll.gg/d2/guides/paladin-leveling`, V, D2R Season 14 page, skill mechanics match 1.14d): Might at 2, Sacrifice 3, Resist Fire at Den and 4-5, Holy Fire 6-9 and 13, Smite 10, Charge + Zeal 12; main attack "Holy Fire aura with Zeal"; Countess for runes (Tal, Eth), then Andariel; Radament in Act 2 at 14-19; respec to Blessed Hammer at 19. No xp/h figures, no Blood Raven.
- Maxroll general leveling (`https://maxroll.gg/d2/resources/general-leveling`, V): Den to clvl 3-5 (full party), then Tristram runs or Countess runs to 13-15, Tomb runs to 20, cows or Travincal to 25. Solo: same areas; `/players` raises xp but also life and damage.
- Almar (`https://almarsguides.com/Computer/Games/Diablo2/Leveling/`, V, old): beginner route Den + Cain at 1-4, Tristram repeated 4-15, Andariel 15-17, Radament and Tomb of Tal Rasha 17-25; fastest route Tristram 1-15, Tomb 15-24.
- Expected xp/hour: none of the sources gives a number (also `docs/sme/pace-tables.md` honest summary). kolbot has no tracker of xp/h either (Tools/Tracker.js tracks run data, S: not read). So the SME estimate (clvl 13-15 in an hour) stays U.

### 3.6 Area levels (fabd, V, 1.13 data)

Normal: Blood Moor 1, Cold Plains 2, Stony Field 4, Dark Wood 5, Black Marsh 6, Den 1, Cave 1 and 2 at 2, Burial Grounds 3, Tristram 6, Tower Cellar 1-5 at 7, Barracks 9, Jail 10, Cathedral 11, Catacombs 11-12 (`wiki/experience.html`). U: these are 1.13 numbers; our `mechanics-progression.md` has the 1.14d-oriented ones. If area levels match, then at clvl 6-11 (Tristram mlvl 6) the level-difference penalty is zero (monsters up to 5 above, up to 5 below are 100 percent), so the area is not what limits xp.

### 3.7 Comparison summary and gaps

| Topic | Verdict |
|---|---|
| Den once, not farmed | They match the SME's "Corpsefire at the Den"; we over-farm it (hour1) |
| Early Blood Raven | They go at clvl 5 (after Cave); we say 9. Merc earlier = faster kills; risk is Blood Raven's fire arrows at clvl 5 (U for our bot) |
| Tristram | Their main leveling area to 12; our optional quest at 11 |
| Aura | Holy Fire at 6; ours at 16 |
| Bishibosh | not an xp route in either |
| Countess | rune errand in both; ours is gated by level 14 |
| Potions / town | Their buffer (inventory potions) is explicit; our hour1 had 13.8 percent of the hour in town |
| Shrines | They scan for experience, combat, skill shrines in the field; ours: `docs/fixobj.md` mentions shrines; no xp-shrine policy found (U) |

## 4. Tooltips: what we may be missing

Our tooltip (`docs/inventory-tooltip.md`, V read) relies on d2d's `items` verb text; the game.exe backend shows base facts only ("Magic properties are not read on this backend", V). D2ItemToolkit (V, README + file list + `StatDescriptor` fields) shows what a full 1.14d tooltip is made of:

1. Property lines come from `ItemStatCost` description columns: priority, function, value, positive/negative string, second string, and a group version of the same (V field names `DescPriority`, `DescFunc`, `DescVal`, `DescStrPos/Neg/Str2`, `DescGrp*`), plus value shift and operator/param/base for derived stats. Sorting by priority, grouping (e.g. all-resist, min-max damage) and the 1023-character budget are applied in that order (V README "Do not join line.Text yourself").
2. Magic and rare names are built from prefix/suffix ids that index into a concatenated table (suffixes, prefixes, automagic); in D2R the prefix ids shift by 38 (V README). In 1.14d the order is suffix, prefix, automagic rows.
3. Lines that depend on the viewing player: required strength/dex/level (including class gating), block chance, Smite damage, attack speed, ranged oskill caps (the +3 cap on own-class oskills is 1.14d-only, V README), "usage" lines for the cube and keys.
4. Beyond-the-game options worth copying as ideas: **ranges** (the roll span of each stat, in grey), **item level** appended to the name, **sockets separated** (filler mods in their own block), **breakdown** (where each modifier came from).
5. Set items have their own writer; the shop price line is a documented gap there.

What may be missing in ours (U until compared line by line): range spans, item level, socket filler breakdown, per-viewer lines when running on game.exe. How: implement the description functions in Go from `ItemStatCost.txt`/`Properties.txt` that the user's tables already provide at runtime (`internal/data`), as a new package; use D2ItemToolkit only as a spec. Squeek's d2itemreader does not build strings (V README: it returns the parsed property lists), but its `docs/formats/d2.md` documents the item bit layout and is public domain.

## 5. Streamer / dashboard ideas (ideas only)

- DiabloInterface (V README): a configurable overlay of facts not always visible in game: gold, base stats, FCR/FHR/FRW/IAS, resistances, quests completed in percent, magic find, attacker-takes-damage, extra monster gold, `/players` value, softcore/hardcore, expansion/classic, plus counters tracked while the game runs and an autosplitter. For us: a "stats the viewer cannot see" strip on the watch page (breakpoints reached, MF, `/players`, quest percent).
- MF_run_counter (V README): run timer with automatic start/stop from memory, drop log with name autocomplete, **holy grail tracker** (found-item checklist and grail progress), per-character/per-run-type profiles, session timer that pauses when the game closes, archive of past sessions exportable as text. For us: a grail page for the idle viewer (items ever found), a per-run archive table, session timer separate from game time.
- d2helper (V README): transparent overlay with an automap-style view, entity markers (monsters, objects, ground items, missiles), a bottom status strip (difficulty, act, area, seed, mode), a character list with level/HP/MP. For us: the live map and entity markers idea for the watch map; the status strip layout.
- diablo-soul (V README): events diffed from the game state become short prose lines (a MUD narrator) streamed over a WebSocket, so a viewer follows the bot from a phone. For us: a text "story" feed from the existing event log, which fits the idle-game page and works with the phone view.
- MF counter and DiabloInterface both treat "per-run" as the unit; our unit is the game or chapter. A per-chapter card with kills, xp, time, drops and deaths is the same idea.

## 6. Servers and the "networked play is intended" decision

| Question | Finding |
|---|---|
| Can a stock 1.14d client play on a self-hosted server? | Yes: espenmjos/d2gs (V README: PvPGN 1.99.8 + D2GS 1.13d DLLs with 1.14d `Patch_D2.mpq`, "Tested end to end: login, character creation, games, saving") and d2-dedicated-server (V `docs/STATUS.md`: real client logs in, creates and joins games, spawns in world, two clients see each other, a full party of eight, 450 games in 25 rounds clean) |
| Is it a 1.14d engine? | d2gs: no, 1.13d binaries. d2-dedicated-server: 1.14d via wine or a native macOS-build port; its status page lists the rough edges (read it before relying on it: S for the exact limits) |
| Needs the user's own game files? | Yes (d2gs README: about 1 GB of MPQs; d2-dedicated-server LEGAL.md V: it builds a stripped install from the user's copy) |
| Does a party change xp? | Yes: +35 percent per kill to the total and a share per level (Arreat Summit, V). Players setting scales xp and monster life the same way |
| Where does the bot sit? | Our bot reads/drives a real game.exe or a d2d process. With a realm the bot would drive a game.exe client in a networked game; d2-clientless would remove the client but is no-licence and talks the Battle.net-style protocol (AGENTS.md: "Do not add anything that talks to Battle.net"; this is a private realm, but the rule text needs the user's clarification) |
| Risks | Server-side AI, loot and timing differ from d2d's single-player game, so measured pace and gates need re-baselining; second player means non-deterministic arrivals (lockstep); licences: d2gs ships Wine + DLLs |

Passive packet capture (d2helper's approach) would be a read-only fidelity oracle for the networked path (U).

## 7. Beyond the lists

- Arreat Summit archive (V): experience page above. The same archive hosts the monster, difficulty and item pages; not fetched (`https://classic.battle.net/diablo2exp/monsters/basics.shtml`, S, search result).
- PureDiablo "Level Up Guide v1.10" and Almar are already in `docs/sme/pace-tables.md`.
- d2-recon's `docs/findings/ai/0014-community-cross-check.md` and `0020-owner-played-duriel-countess.md` (names only, U): worth a look for the Countess and Duriel behaviour.
- No Phrozen Keep monster-AI or hit-recovery article surfaced in the search (S: the search returned only generic pages), so the best AI source found is d2-recon's catalogue (row 2).

## 8. Proposals (ranked by expected xp gain; proposals only)

1. **Tristram as the clvl 6-12 leveling area.** Move `find_cain` earlier (floor 6, recommended 7-8 for the Paladin, with the first-time quest chain: Dark Wood scroll, Stony Field waypoint, Rakanishu, stones) and make the post-quest area picker prefer Tristram while clvl is below 12. Evidence: kolbot (V), Almar and Maxroll (V). Expected: the largest gain, because it replaces the Blood Moor / Den loops that took 69 percent of the hour. Risk: Cain's stone puzzle and Tristram's elite groups at clvl 6; a death-spiral guard (`docs/deathspiral.md`) should apply. Size: plan-table change plus the existing campaign chapter, not new code if the chapter and picker already support it.
2. **Buy Holy Fire at clvl 6 and run it as the aura (Paladin).** Evidence: kolbot, Maxroll (V); our own hotkey plan already has it from clvl 6. Change: rows 5-9 of the v2 table (Might 6 versus Holy Fire is the open U in the v2 note). Expected: faster pack kills at clvl 6-11; magnitude U. Test it as two tables in an A/B run.
3. **Stop farming the Den after the quest; use Cave and Cold Plains / Tristram instead.** Evidence: kolbot comment (V), hour1 section 2 (Den second farm 223 xp/min versus Cold Plains 558). Change: mark the Den non-farm after `den_of_evil` done in `progression-normal.json`. Cheap.
4. **Blood Raven earlier (clvl 5-6 after the Cave) for the free Rogue.** Evidence: kolbot order (V) and `bloodraven.js` (V). Our SME says 8-10; this is a disagreement, not a fact: ask the d2-sme before any change. Expected: faster kills for the rest of the early game (merc damage); risk: Blood Raven's fire arrows at clvl 5 and whether our bot revives/keeps the merc alive (`docs/deathspiral.md`, merc revive cap).
5. **Shrine pickup and an inventory potion buffer.** Evidence: kolbot `ScanShrines` list includes experience, combat and skill shrines and `HPBuffer` 8 at clvl 2-9 (V). Change: a policy row (experience and combat shrines on) and a spare-potion count that delays the town trip; hour1 had 13.8 percent of the hour in town. Expected: small per hour, but cheap; experience shrine strength in 1.14d is U (not looked up).

Not ranked: Sorceress energy points or an Ice Bolt point at clvl 6 (kolbot does both): the mana bound is real in our data, but the Sorceress table was a user decision of 2026-11-10; list for the d2-sme only. `/players` is already carried by TASK-170 (pactnative); the Arreat figure (xp and life both times `(n+1)/2`) means xp per monster HP is roughly constant, so the gain is only fewer walks per xp.

## 9. Open questions

- Does the d2-sme agree that Tristram at clvl 6-7 is safe for the bot Paladin (proposal 1) and that Blood Raven at 5-6 is not (proposal 4)?
- Are kolbot's clvl thresholds for Den/Tristram tuned for a teleport-less bot, or for any bot? The code does not say (U).
- Real xp/hour for kolbot Normal clvl 1-12 is not published in the repo; if it matters, run it ourselves on a throwaway (not in scope, no game).
- The d2-dedicated-server STATUS "Rough edges / next" list was not read in full (S).
