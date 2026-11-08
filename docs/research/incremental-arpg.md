# Incremental and idle ARPGs: a deep pass on what to borrow

> status: research note of 2026-11-08, not a decision. `docs/decisions.md`, the ADRs and `GLOSSARY.md` are untouched. It builds on [fun-ideas.md](fun-ideas.md) section 2 (genre math, Pecorella, Guan, Machinations) and does not repeat it; read that first. Direction is fixed by ADR 0016 to 0019 (fun-first, Expedition Ledger, policy sheet, Codex, unlock ladder, Pacts, Hall). Target is classic 1.14d; nothing here changes vanilla XP or loot.

Tags: V = page read this session (URL given), S = search snippet or second-hand, U = from memory, unchecked. Fandom, Fextralife and the Godville wiki sit behind Cloudflare and could not be read (the headless browser on the worker box was refused too); those games are S or U, and each says so.

## 0. Answer in twelve lines

1. Our product is closest to **Godville** (a hero who acts alone, the player nudges), **Progress Quest** (the satirical ancestor), **Loop Hero** (the player shapes the world, the hero fights alone, expedition and camp) and the **Kolbot SoloPlay** community (a bot that levels a class from 1 to 99). Nobody ships all of these with a real ARPG underneath; that is our gap and our risk.
2. The strongest transferable idea is a **small, regenerating nudge budget** (Godville's godpower): the player influences, never commands, and the influence is rationed so it stays a decision. Our question queue plus the policy sheet are already this; the budget idea would tell us what a "free" order costs. (S)
3. The strongest proof that **the policy sheet is right** is the loot-filter ecosystem: Path of Exile and Last Epoch both ship a rule list (conditions, show/hide/recolor, top-to-bottom priority) and a community strictness ladder on top (V). Players use the filter as the game's real inventory UI.
4. The best **return-screen** precedents are Idle Champions style "what happened while you were gone", Egg, Inc. silos, AFK Arena idle rewards and Melvor's summary; none shows a story. Godville's diary is the one that does (S).
5. **Prestige** works when the reset is short and legible (Cookie Clicker's chips are a cube-root of lifetime; Clicker Heroes' ascension; Antimatter's layers) but players resent walls (Push Square on Clicker Heroes: "too many brickwalls", V). Our Hall retire is voluntary, which sidesteps that.
6. The loudest burnout evidence is **late-game pacing plus monetised shortcuts** (Idleon, Diablo Immortal, Idle Champions patron variants). We have no store, so the lesson is "never let the grind be the only content".
7. Bullet-heaven/ARPG-lite (Vampire Survivors, Halls of Torment, Brotato) proves **three-choice level-ups** and **weapon evolution by combination** are the cheapest high-feeling decision. Our analogue is the free-choice skill point and Codex unlocks, not a new system.
8. Spectator culture (Twitch Plays Pokemon, Dwarf Fortress, Kolbot) says watchers love **emergent stories and nicknames**, not optimal play. This backs "the first watch moment is a close call".

## 1. Comparison table (22 games, by archetype)

Columns: loop; player vs game; offline or return; prestige or meta; loot handling; decisions asked; one thing to borrow. Source tag at the end of each row.

### A. Zero-player and "hero acts alone" (our closest kin)

| Game | Loop | Player vs game | Offline / return | Meta | Loot | Decisions asked | Borrow |
|---|---|---|---|---|---|---|---|
| Godville | Hero wanders, fights, quests, sells, levels; every event is a diary line | God shapes with encourage, punish, miracle, voice, prayer; hero may ignore | Hero keeps playing while you are away; the diary is the return | Pet, arena duels, temple, ranking; community-written phrases | Hero auto-equips; "divine" items need the god | Spend godpower on a nudge | Rationed nudge budget; a diary that is funny enough to read. Web page says the god may "do nothing" or "direct him, though unsuccessfully" (V godvillegame.com). Costs 25 percent encourage/punish, 50 percent miracle, 5 percent voice (S, GodWiki search snippet) |
| Progress Quest | Progress bars fill; fight, sell, buy, quest | Roll a character, then watch | None needed; always on | Online ranking; no gameplay effect | Auto-sold when carrying cap is hit (Strength) | Almost none (V Wikipedia) | The satire: make the meaningless act legible; Fallout Shelter cites it (V) |
| Loop Hero | Hero circles a closed road and fights alone; you place cards drawn from drops | You build the world; hero is autonomous | XX (V Wikipedia) | Camp buildings are lasting boosts | Equipment drops auto-compare in a deck; you equip | Where to place a card; when to leave | Expedition with a retreat rule and a keep-fraction: our "go home now" order |
| Kolbot SoloPlay (bot) | Bot levels one class 1 to 99 | You pick a final build and start it; no config files (V bhdocs) | Runs unattended; goal "Hell in about 24 hours" (V) | Final build target | Auto-equip for char and merc, socketing, CTA and Spirit swaps (V) | Almost none after the profile | The real-world demand: players run a D2 bot to completion; build target plus auto-merc is the spec we already have |
| Dwarf Fortress | Colony sim | Designate jobs, never control a dwarf (V) | None | None, "Losing is Fun!" (V) | Stockpile rules | Orders and zones | Indirect control produces stories (Boatmurdered) |

### B. Idle RPGs and idle MMOs

| Game | Loop | Player vs game | Offline | Meta | Loot | Decisions | Borrow |
|---|---|---|---|---|---|---|---|
| Melvor Idle | RuneScape-like skills; one action at a time | Pick the skill to run | Simulated offline; cap reported 12 h early, 18 h third-party, 24 h Steam post (S, sources disagree) | Mastery, game modes (hardcore variants exist, U) | Items auto-bank | What to train, what to equip | Mastery per action ("do it more, get better at it") as the Codex of repeat |
| Legends of Idleon | MMO whose characters AFK-farm different things | Swap characters, spend; each character is a role | Offline gains are the core appeal; early gains feel slow (S) | Large account-level meta | Auto-loot sold as a 5 USD unit (S) | Which character does what | Account-level tally; also the burnout lesson: late pacing "excessively slow" and heavily monetised (S, gamebrain) |
| Idle Champions (D&D) | Auto-fight waves; formation of champions | The formation is the game; click damage optional | Offline play, adventure variants as content (S) | Patrons, variants, event chests | Gear is dropped onto a champion (V Wikipedia) | Who stands where, which specialization | Fixed-slot "formation" = our build + stance: choosing the party shape beats choosing a number. Complaint: variants and patron shops feel pay-to-win to some (S) |
| AFK Arena | Preset 5-hero order, auto battles | Chapter push, manual ultimates optional | Resources and XP accrue offline; Bounty Board runs offline (V Wikipedia) | Heroes, guild, arena | Gacha | When to push | Timed offline "bounties" that take the hero off camera |
| Idle Slayer | Auto-run and jump; kill and collect | Tap to jump, mostly idle | Idle gains (S) | Ascension with Slayer Points and a skill tree (S) | Auto | Ascension timing | Ascension tree shape; weak ARPG fit |
| Soda Dungeon, Rucoy, Hero Wars, Idle Heroes | Tavern/party idle RPGs | Recruit, equip, push | AFK rewards | Heroes | Gear auto | Party choice | Not read; U, skip. |
| Increlution / Idle Loops | Action queue, loop a day | Plan the queue, game executes (V fun-ideas) | Auto-pause when the queue empties | Generations / loops carry knowledge | Skills and items per loop | Queue order | Already in fun-ideas; only the "stuck means plan again" point is used here |

### C. ARPG-lite and auto-battler roguelites

| Game | Loop | Player vs game | Offline | Meta | Loot | Decisions | Borrow |
|---|---|---|---|---|---|---|---|
| Vampire Survivors | Auto-fire weapons, you only move; 15 to 30 min runs | Movement and level-up picks | None | Gold buys characters and permanent power; challenges unlock stages (V Wikipedia) | Chests give 1, 3 or 5 items; weapon evolves after a passive plus chest (V) | 3 or 4 choices per level | Cheap decision units; evolution = a condition that rewards discovery |
| Halls of Torment | Diablo-skinned bullet heaven; 30 min runs, bosses | Auto or manual aim | None | Well of items bought for permanent use, 11 characters (V Wikipedia) | Boss chest: choose armor or jewelry | 4 stat choices with a reroll | A boss "choose one of" chest and a reroll; Diablo look makes the genre read as ARPG |
| Brotato | Wave shooter, shop between waves | Build choices; auto-aim | None | Characters unlock | Weapons in a shop | Shop buy/lock | Shop beat between fights = our town trip as a decision beat (V Wikipedia) |
| Path of Achra | Turn-based auto-battler roguelike | Build crafting | None | Run-to-run unlocks | "Broken build sandbox" (S) | Build synergy | Players praise synergies; complaint is the steep first hour (S) |

### D. Loot-centred ARPGs and idle-adjacent features

| Game | Why it matters | Borrow |
|---|---|---|
| Path of Exile | Item filter language: Show/Hide blocks, conditions (`Rarity >= Rare`, `ItemLevel`), actions (colour, size, beam, minimap icon, 16 sounds) (V pathofexile.com/item-filter/about). FilterBlade adds strictness levels, themes, auto-adjust by character (V filterblade.xyz). SSF and hardcore are rule sets players pick (U) | The policy sheet is a filter by another name; ship strictness presets and effect lines |
| Last Epoch | In-game filter, up to 75 rules, each Show/Hide/Recolor, optional emphasis, level range and 5 condition types; evaluated top to bottom; "filtered items still drop" (V Maxroll). Footgun: a misplaced rule hides wanted items | Priority order is a footgun; a preview of "what this hides" is the fix |
| Diablo Immortal | No energy system; rifts; heavy criticism for pay-to-win, hidden caps on legendary drops and a record-low user score (V Wikipedia) | Hidden caps and surprise limits poison trust: every cap on our side stays visible |
| Diablo 4 Helltide, Torchlight Infinite, Grim Dawn, Warframe, Destiny | Event timers and treadmills. Torchlight Infinite reviews call map farming repetitive; no real idle mode in beta (S). Warframe: all gear earnable "which may involve grinding"; mission rewards lost on failure (V Wikipedia); developers saw players leave after a few hours if systems asked too much (V) | Failure that costs run rewards (Loop Hero, Warframe) is accepted when kept legible; grind alone is not content |

### E. Classic incrementals (prestige references)

| Game | Meta layer | Offline | Burnout |
|---|---|---|---|
| Cookie Clicker | Ascension gives heavenly chips; each prestige level is +1 percent, level cost grows as the cube (V Wikipedia); golden cookies as timed attention beats | Not described | IGN "greatest idle game" and Destructoid "illusion of progress" (V) |
| Clicker Heroes | Ascension after primal bosses; Hero Souls buy Ancients (V) | Idle DPS simulated (V fun-ideas) | "Too many brickwalls" (V Push Square, via Wikipedia) |
| Egg, Inc. | Soul eggs boost egg value by percent; epic research permanent; grain silos let the farm run longer (V Wikipedia); golden eggs are the premium currency and buy temporary boosts | Silos = offline length | "Ultimately a little pointless" (V Gamezebo) |
| Antimatter Dimensions | Layers: boost, galaxy, infinity, eternity, reality; each opens a new area (V Wikipedia) | -- | pacing reception not checked; U |
| Kittens Game, Realm Grinder, NGU Idle, Idle Wizard, Evolve, Trimps | Deep multi-currency incrementals | -- | Not read this pass; kept out of the map (U) |

## 2. Per-game notes where the pass found something new

### 2.1 Godville (closest kin; S, because the wiki is walled)

- **What the player does.** Encourage, punish, miracle, voice of god, send to the arena, prayer. A search snippet from the GodWiki gives costs of 25 percent godpower for encourage and punish, 50 percent for a miracle (needs a completed temple), 5 percent for a voice and 50 percent for the arena, with an accumulator that restores around half (S). The hero "may ignore your wishes"; the site says the god can "do nothing" or direct "unsuccessfully" (V godvillegame.com).
- **What the game does.** Fights, quests, sells loot, reports a diary line for every event, many of them community-written and funny (V Wikipedia). Duels and pets are the social layer; the winner of an arena duel takes coins and items, and divine meddling can help the opponent (V Wikipedia).
- **Reception.** Praised for writing and satire (Edge, NYT, Sterling; V Wikipedia); the critique is repetitive early adventures and, from one aggregator, "much, much more control" wished for (S). The control complaint matters: **a nudge the hero ignores feels like a bug unless the ignore is the joke.**
- **Our mapping.** The "hero ignores voice" idea does not fit a competence game (the user wants orders obeyed: ADR 0016), but a **reason line** does: when the hero declines an order, the journal says why in character. That is a free, honest feedback channel.

### 2.2 Loop Hero, camp and retreat (V Wikipedia)

Leaving outside combat is safe, 40 percent less if far from the campfire, defeat or fleeing mid-fight costs 70 percent. The player never pilots the hero but chooses terrain from drops. Critics (PC Gamer) felt the lack of new cards weakened the late game. Our version already has "go to town" as a safe retreat; the new idea is making **leaving early a priced option shown on the card** ("leave now: keep all; leave in the field: lose the unsold stash").

### 2.3 Loot filters as policy-sheet precedent (V)

PoE: blocks match when all conditions match; a `Continue` flag lets matching run on; the label can be coloured, sized, beamed, given a minimap icon and a sound. Last Epoch: 75 rules, top-to-bottom, a bottom catch-all Hide with Show rules above it, "filtered items still drop". FilterBlade layers strictness presets (Semi-Strict default) on top and an Auto-Adjust per character. Lessons for the sheet: (a) players want **strictness presets** more than rule editors; (b) rules are evaluated in order and mistakes are silent, so show the **effect line** and a dry-run count ("this would sell 41 of last night's 63 items"); (c) the visual vocabulary (colour, beam, sound) is a free "what the bot just found" channel for the watch UI.

### 2.4 Bots as spectacle (V/S)

Kolbot SoloPlay's own docs sell exactly our promise: a 1 to 99 run, hardcore or softcore, an auto-hired merc, "Hell in about 24 hours" (V bhdocs). The community's thread titles are about patching and breakage, not about enjoyment; no source found says why people watch bots (S, gap, see section 6). Twitch Plays Pokemon is the strongest evidence for **myth-making from chaos**: the Helix Fossil became a religion and Pokemon got nicknames ("Bird Jesus"); democracy mode was protested because it removed the randomness (V Wikipedia). Dwarf Fortress adds that **indirect control yields stories** (V Wikipedia).

## 3. Ranked top 12 mechanics to borrow

Effort: S under a day of Go plus docs, M a card of one to three days, L more. Impact is for the "find the fun" goal (ADR 0017), my judgement (U). "Maps" names the existing card or "new".

| # | Mechanic | From | Effort | Impact | Maps to |
|---|---|---|---|---|---|
| 1 | **Strictness presets plus dry-run preview** on the policy sheet (Lenient / Standard / Strict loot tiers; "would sell N of last night's M") | PoE FilterBlade, Last Epoch | M | High | Extends TASK-128 policysheet; new card (policy-dryrun) |
| 2 | **Reason line when the hero declines or defers an order** in character voice | Godville (S) | S | High | TASK-155 unlockvoice; new small card |
| 3 | **Retreat pricing on the expedition card** ("leave now: keep X, lose Y") and a "go home" order | Loop Hero (V) | M | High | TASK-127 returnscreen; TASK-161 order queue; new card |
| 4 | **Return screen as a diary**: 3 to 6 funny/true lines first, numbers second | Godville diary, Egg silos | M | High | TASK-127/109 returnmock; TASK-155 voice |
| 5 | **Nudge budget**: free orders limited by a slowly refilling "attention" meter; long-away refills it, over-managing drains it | Godville godpower (S) | M | Medium, risky | New card, needs a user decision (section 6) |
| 6 | **Offline bounties**: timed town-side tasks the hero runs when the player is away (clear X by tomorrow) | AFK Arena Bounty Board (V) | M | Medium | New card on the order queue |
| 7 | **Choose-one boss chest** (3 items, take one, reroll once) at named bosses | Halls of Torment, Vampire Survivors (V) | M | Medium | Codex firsts (TASK-130) as a question kind; no loot change if offered from the existing drop |
| 8 | **Evolution by condition**: a discovery that unlocks only when two prerequisites are met (a charm plus a first kill) | Vampire Survivors weapon evolution (V) | S | Medium | Unlock ladder TASK-153 trigger table (data only) |
| 9 | **Mastery per repeat zone** (repeat clears of a place raise a "familiarity" bonus to the hero's caution or tally) | Melvor mastery (U) | M | Medium | Codex pages; needs tally design |
| 10 | **Hall retire as a tree, not a number**: legacy is a small skill tree that the next hero reads | Idle Slayer ascension tree (S), Cookie Clicker chips (V) | L | Medium | TASK-135/163/171 Hall cards |
| 11 | **Hide-but-log**: filtered items still drop, count shown ("hidden: 212 whites") | Last Epoch (V) | S | Medium | Policy sheet effect lines |
| 12 | **Nickname and myth hooks**: the hero names its merc, boss that killed it, and a favourite item; the journal reuses them | Twitch Plays Pokemon, Dwarf Fortress (V) | S | Medium | TASK-155 unlockvoice |

## 4. Anti-patterns to avoid (with evidence)

1. **Hidden caps and surprise limits.** Diablo Immortal's undisclosed caps on legendary drops and rewards fed the record-low user score and a community site dropping coverage (V Wikipedia). Rule: every limit (daily tally, charter price, Pact heat cap) shows on the sheet.
2. **Walls instead of prices.** Clicker Heroes drew "too many brickwalls" (V Push Square via Wikipedia). Already ADR 0018 (price, never a wall); keep it, including after the 1.14d content is cleared.
3. **Grind as the only content.** Torchlight Infinite complaints about repeating maps (S), Warframe's own note that people leave after a few hours if systems ask too much (V), Idleon late game "excessively slow" (S). Rule: when a loop repeats three times, something new appears (Forager rule in fun-ideas).
4. **Monetised shortcuts on the core loop.** Idleon auto-loot as a purchase, Diablo Immortal crests, Idle Champions patron shop complaints (S/V). We have no store; do not sell or gate comfort that a tester would assume is baseline (auto-sell, auto-pickup).
5. **Silent filter mistakes.** Last Epoch's own guide lists "misplaced rules hide wanted items" and "deleting a filter is permanent" as gotchas (V Maxroll). Rule: the sheet needs undo, versioning and a dry-run.
6. **Illusion of progress.** Destructoid on Cookie Clicker, Gamezebo on Egg ("a little pointless"), Push Square (V). Rule: tally must stay tied to D2 events (firsts, kills, quests), not a free-running counter.
7. **Control the player cannot feel.** The Godville aggregator quote asking for "much more control" (S). Rule: an order has to be visible in the next five minutes of play, or the journal must say why not.
8. **Chaos with no handle.** Twitch Plays Pokemon democracy mode removed what drew viewers (V); in our case "safety" that removes every near-death moment would remove the first watch moment. Keep the stance row (TASK-191) honest: the cautious setting must still show fights.

## 5. Proposed cards (top 5; title, owns, size, ACs)

Card names follow `docs/backlog-howto.md`; owns are guesses to be checked against `docs/OWNS`/`docs/OWNERS` by the coordinator. These are proposals only; nothing was created.

**1. policydryrun: strictness presets and dry-run preview on the policy sheet.** Owns: `internal/policysheet/` (or the package TASK-128 uses), `docs/research/policy-dryrun.md`. Size M. ACs: (a) three preset levels per chore row, as data, no class literals; (b) preview computes counts from a recorded session without side effects, tested on two builds; (c) the effect line shows "hidden: N" with a count, never a silent hide; (d) undo for the last sheet change. Maps: TASK-128, TASK-146 (per-class chore defaults).

**2. declinereason: the hero says why it deferred an order.** Owns: `internal/journal/` reason templates (data), `docs/contracts/` line if needed. Size S. ACs: (a) every order the bot defers or refuses writes one in-character journal line citing the true cause (danger, kit, queue, pact); (b) lines are data keyed by cause, two phrasings each; (c) the return screen groups declines. Maps: TASK-155 unlockvoice, TASK-161.

**3. retirepricing: priced retreat on the expedition card.** Owns: `internal/expedition/`, the card template. Size M. ACs: (a) the card shows "go home now: keep / lose" with real numbers from the bot's stash and potion state, labelled estimate; (b) a "go home" order exists and routes through the order queue; (c) no change to the bot's own safety rules; (d) test on a Paladin and a Sorceress fixture. Maps: TASK-127, TASK-161.

**4. diaryreturn: the return screen opens on a diary.** Owns: the return-screen page and `internal/journal/diary.go` selector. Size M. ACs: (a) a deterministic selector picks 3 to 6 lines from events (first, near-death, boss, loot highlight, decline reason); (b) numbers come second, never first; (c) selector is data-weighted per class; (d) a fixture test with a 10 h away log. Maps: TASK-127, TASK-109, TASK-155.

**5. nudgebudget (proposal needing a user decision first): rationed free orders.** Owns: a design note only (`docs/research/nudge-budget.md`) until decided. Size S for the note. ACs: (a) a one-page model with the meter's refill and cost per order kind; (b) a paper test against the user's own two week play record (TASK-106 session record) showing how often the meter would have been empty; (c) a recommendation yes/no/shrink. Maps: new.

## 6. Open questions for the user

1. **Nudge budget or not.** Godville rations influence and that is what makes a nudge a choice (S). ADR 0016 says orders are free. Do you want a meter at all, or only for the "force" kind of order (override the cautious hero)?
2. **May the hero decline an order?** Only for safety (as now), or also in character ("I don't like this")? Godville's ignore is its joke; ours is a competence game.
3. **Diary tone.** Funny and community-written (Godville) or dry and factual (Journal today)? A joke bank costs writing time and must avoid private names.
4. **Offline bounties.** Is "do X while I am away" part of the order queue, or does it break "the hero proposes, the player answers"?
5. **Retreat pricing.** Do you want the keep/lose numbers shown to the player on the card, or stay as the hero's private reasoning?
6. **Presets.** Is a three-level strictness preset enough, or should the sheet also show a real rule list (power users)?
7. **Evidence gap.** I could not find a source on why people watch D2 bots. A short poll of your testers ("what did you watch for?") would be better than my guess.

## 7. What this pass confirms or challenges in the existing design

- **Confirms:** the policy sheet (loot filters are the genre's own proof, V); return screen first (Godville, AFK Arena, Egg: the away reward is the core); price not wall (Clicker Heroes); voluntary Hall retire (walls are resented); the stance row (Twitch Plays Pokemon, bullet heaven: players want to see risk); no store (Diablo Immortal).
- **Challenges:** "no failure state" does not hold for our genre: Loop Hero, Warframe and D2 all price failure; our return screen should show what a death or retreat cost. Also the 15 to 30 minute run length of bullet heaven suggests our expedition card should have a **visible short beat** even if the real trip is longer.
- **Gaps:** Fandom walled, so Idle Champions formation detail, Idleon's offline caps and Godville's rules are S only. Not read: Rucoy, Soda Dungeon, Hero Wars, Idle Wizard, NGU Idle, Realm Grinder, Evolve, Trimps, Kittens Game, Grim Dawn, Destiny, Fallen London, Raid-style summon games, Dungeon Fighter idle; all U and not used for any recommendation.

## Sources (V = read)

- V https://en.wikipedia.org/wiki/Godville ; V https://godvillegame.com/ ; S GodWiki via search (Godpower, Encouragement, Arena, Remote Control pages)
- V https://en.wikipedia.org/wiki/Progress_Quest
- V https://en.wikipedia.org/wiki/Loop_Hero
- V https://en.wikipedia.org/wiki/Idle_Champions_of_the_Forgotten_Realms
- V https://en.wikipedia.org/wiki/Egg,_Inc.
- V https://en.wikipedia.org/wiki/Vampire_Survivors ; V https://en.wikipedia.org/wiki/Halls_of_Torment ; V https://en.wikipedia.org/wiki/Brotato
- V https://en.wikipedia.org/wiki/Cookie_Clicker ; V https://en.wikipedia.org/wiki/Clicker_Heroes ; V https://en.wikipedia.org/wiki/Antimatter_Dimensions
- V https://en.wikipedia.org/wiki/Diablo_Immortal ; V https://en.wikipedia.org/wiki/AFK_Arena ; V https://en.wikipedia.org/wiki/Warframe
- V https://en.wikipedia.org/wiki/Twitch_Plays_Pok%C3%A9mon ; V https://en.wikipedia.org/wiki/Dwarf_Fortress
- V https://www.pathofexile.com/item-filter/about ; V https://www.filterblade.xyz/ ; V https://maxroll.gg/last-epoch/resources/loot-filter-guide
- V https://bhdocs.github.io/SoloPlay/intro (Kolbot SoloPlay)
- S search snippets: Melvor offline cap (itemlevel.net, wiki.melvoridle.com/w/V0.16.2), Idleon (mein-mmo, gamebrain, notebookcheck), Idle Champions (tvtropes, gamersdecide), Idle Slayer (levelwinner, App Store), Torchlight Infinite (gamerbraves, mobi.gg), Path of Achra (vaporlens, bearblog), Idle Loops forks (incrementaldb)
- Blocked and not read: wiki.godvillegame.com, godville.fandom.com, idlechampions.fandom.com, lastepoch.fandom.com, wiki.melvoridle.com (403), PC Gamer Idle Champions article (body not extracted)
