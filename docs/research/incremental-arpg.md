# Incremental and idle ARPGs: a deep pass on what to borrow

> status: research note of 2026-11-08, not a decision. `docs/decisions.md`, the ADRs and `GLOSSARY.md` are untouched. It builds on [fun-ideas.md](fun-ideas.md) section 2 (genre math, Pecorella, Guan, Machinations) and does not repeat it; read that first. Section 8 is a second pass of 2026-11-08 (walled games read, why people watch bots). Direction is fixed by ADR 0016 to 0019 (fun-first, Expedition Ledger, policy sheet, Codex, unlock ladder, Pacts, Hall). Target is classic 1.14d; nothing here changes vanilla XP or loot.

Tags: V = page read this session (URL given), S = search snippet or second-hand, U = from memory, unchecked. First pass: Fandom, Fextralife and the Godville wiki could not be read. Second pass (section 8, 2026-11-08): most of those claims are now V through Wayback copies (`web.archive.org/web/2024/<url>` fetched with curl), the Melvor and IdleOn wikis, Steam store pages and Steam community reviews, Hacker News and arXiv; claims marked V in the tables below were upgraded in place, and what stayed unreachable is listed in section 8.9. Section 8 also holds the bot-spectator evidence the first pass lacked.

## 0. Answer in twelve lines

1. Our product is closest to **Godville** (a hero who acts alone, the player nudges), **Progress Quest** (the satirical ancestor), **Loop Hero** (the player shapes the world, the hero fights alone, expedition and camp) and the **Kolbot SoloPlay** community (a bot that levels a class from 1 to 99). Nobody ships all of these with a real ARPG underneath; that is our gap and our risk.
2. The strongest transferable idea is a **small, regenerating nudge budget** (Godville's godpower): the player influences, never commands, and the influence is rationed so it stays a decision. Our question queue plus the policy sheet are already this; the budget idea would tell us what a "free" order costs. (V, section 8.1: GodWiki via Wayback)
3. The strongest proof that **the policy sheet is right** is the loot-filter ecosystem: Path of Exile and Last Epoch both ship a rule list (conditions, show/hide/recolor, top-to-bottom priority) and a community strictness ladder on top (V). Players use the filter as the game's real inventory UI.
4. The best **return-screen** precedents are Idle Champions style "what happened while you were gone", Egg, Inc. silos, AFK Arena idle rewards and Melvor's summary; none shows a story. Godville's diary is the one that does (V, section 8.1).
5. **Prestige** works when the reset is short and legible (Cookie Clicker's chips are a cube-root of lifetime; Clicker Heroes' ascension; Antimatter's layers) but players resent walls (Push Square on Clicker Heroes: "too many brickwalls", V). Our Hall retire is voluntary, which sidesteps that.
6. The loudest burnout evidence is **late-game pacing plus monetised shortcuts** (Idleon, Diablo Immortal, Idle Champions patron variants). We have no store, so the lesson is "never let the grind be the only content".
7. Bullet-heaven/ARPG-lite (Vampire Survivors, Halls of Torment, Brotato) proves **three-choice level-ups** and **weapon evolution by combination** are the cheapest high-feeling decision. Our analogue is the free-choice skill point and Codex unlocks, not a new system.
8. Spectator culture (Twitch Plays Pokemon, Dwarf Fortress, Kolbot) says watchers love **emergent stories and nicknames**, not optimal play. This backs "the first watch moment is a close call".

## 1. Comparison table (22 games, by archetype)

Columns: loop; player vs game; offline or return; prestige or meta; loot handling; decisions asked; one thing to borrow. Source tag at the end of each row.

### A. Zero-player and "hero acts alone" (our closest kin)

| Game | Loop | Player vs game | Offline / return | Meta | Loot | Decisions asked | Borrow |
|---|---|---|---|---|---|---|---|
| Godville | Hero wanders, fights, quests, sells, levels; every event is a diary line | God shapes with encourage, punish, miracle, voice, prayer; hero may ignore | Hero keeps playing while you are away; the diary is the return | Pet, arena duels, temple, ranking; community-written phrases | Hero auto-equips; "divine" items need the god | Spend godpower on a nudge | Rationed nudge budget; a diary that is funny enough to read. The god may "do nothing" or "direct him, though unsuccessfully" (V godvillegame.com). Costs: voice 5, encourage 25, punish 25, arena 50, miracle 50 percent of a 0 to 100 percent godpower meter (V GodWiki Remote Control, Wayback); the Remote Control can be switched off for a pure zero-player game (V, same page). Details in 8.1 |
| Progress Quest | Progress bars fill; fight, sell, buy, quest | Roll a character, then watch | None needed; always on | Online ranking; no gameplay effect | Auto-sold when carrying cap is hit (Strength) | Almost none (V Wikipedia) | The satire: make the meaningless act legible; Fallout Shelter cites it (V) |
| Loop Hero | Hero circles a closed road and fights alone; you place cards drawn from drops | You build the world; hero is autonomous | Leaving outside combat is safe; 40 percent less far from the campfire; defeat or fleeing mid-fight loses 70 percent (V Wikipedia) | Camp buildings are lasting boosts | Equipment drops auto-compare in a deck; you equip | Where to place a card; when to leave | Expedition with a retreat rule and a keep-fraction: our "go home now" order |
| Kolbot SoloPlay (bot) | Bot levels one class 1 to 99 | You pick a final build and start it; no config files (V bhdocs) | Runs unattended; goal "Hell in about 24 hours" (V) | Final build target | Auto-equip for char and merc, socketing, CTA and Spirit swaps (V) | Almost none after the profile | The real-world demand: players run a D2 bot to completion; build target plus auto-merc is the spec we already have |
| Dwarf Fortress | Colony sim | Designate jobs, never control a dwarf (V) | None | None, "Losing is Fun!" (V) | Stockpile rules | Orders and zones | Indirect control produces stories (Boatmurdered) |

### B. Idle RPGs and idle MMOs

| Game | Loop | Player vs game | Offline | Meta | Loot | Decisions | Borrow |
|---|---|---|---|---|---|---|---|
| Melvor Idle | RuneScape-like skills; one action at a time | Pick the skill to run | Simulated on return as if online, capped at 24 h today (V wiki.melvoridle.com Offline Progression, v1.3.1); an older wiki patch page said 12 h and a third-party guide said 18 h (S, outdated); combat and thieving offline are opt-in because they can kill (V) | Mastery, game modes (hardcore variants exist, U) | Items auto-bank | What to train, what to equip | Mastery per action ("do it more, get better at it") as the Codex of repeat |
| Legends of Idleon | MMO whose characters AFK-farm different things | Swap characters, spend; each character is a role | Store page: "Load up IdleOn the next day, spend your AFK gains earned while you were away, and buy MORE upgrades" (V Steam); an AFK Info screen shows survivability and the defence needed for zero damage (V idleon.wiki); the numeric cap was not found (U) | 11 characters, account-level meta | Items can show "Filtered Out" in the odds line (V Steam patch notes); auto-loot as a purchase (S) | Which character does what | Account-level tally; burnout lesson from long-time players: "charming idle MMO" turned "aggressively monetized", FOMO gacha, weekly paid packs (V Steam reviews, 8.2) |
| Idle Champions (D&D) | Auto-fight waves; formation of champions | The formation is the game; click damage optional; familiars automate levelling and gold pickup "so YOU can focus on optimizing your formation" (V Steam) | Champions "automatically fight enemies until defeated, even if you close the game" (V Steam) | Patrons unlock later: extra rewards for free-play adventures and variants, but more restrictions and higher requirements (V archive of the Fandom Patrons stub) | Gear is dropped onto a champion (V Wikipedia) | Who stands where, which specialization (specialization page unreachable) | Fixed-slot "formation" = our build + stance: choosing the party shape beats choosing a number. Complaints from 2k to 42k hour players: chests, champions, time-gate pieces, potions and familiars all sold; popups on every party change; event characters paywalled then nerfed (V Steam reviews, 8.2) |
| AFK Arena | Preset 5-hero order, auto battles | Chapter push, manual ultimates optional | Resources and XP accrue offline; Bounty Board runs offline (V Wikipedia) | Heroes, guild, arena | Gacha | When to push | Timed offline "bounties" that take the hero off camera |
| Idle Slayer | Auto-run and jump; kill and collect | Tap to jump, mostly idle | Idle gains (S) | Ascension with Slayer Points and a skill tree (S) | Auto | Ascension timing | Ascension tree shape; weak ARPG fit |
| Soda Dungeon | Tavern/party idle RPG, free, "no frustrating timers" (V Steam) | Recruit, equip, push; auto raids | -- | Heroes | Gear auto | Party setup | Negative review: "with auto ... just do the setup and then don't care"; the boss and world beats are "so small and far apart" (V Steam). An autopilot with no decision beat disengages. See 8.6 |
| Rucoy, Hero Wars, Idle Heroes | Mobile MMO / hero gacha | -- | -- | -- | -- | -- | Rucoy: only the official site was readable ("free to play 2D MMORPG for mobile", V rucoyonline.com); no idle claims verified; Hero Wars and Idle Heroes not read (U, skip) |
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
| Diablo 4 Helltide, Torchlight Infinite, Grim Dawn, Warframe, Destiny | Event timers and treadmills. Torchlight Infinite Steam reviews are dominated by monetisation (pet gacha as mandatory power, notification spam, a coin for every daily action; V), with casual solo players content (V); Diablo Immortal added an auto-battle in 2026 that Blizzard calls "deliberately less efficient" than manual play (V massivelyop, 8.5); Grim Dawn and Destiny negatives are about difficulty and progression after expansions (V Steam, 8.6). Warframe: all gear earnable "which may involve grinding"; mission rewards lost on failure (V Wikipedia); developers saw players leave after a few hours if systems asked too much (V) | Failure that costs run rewards (Loop Hero, Warframe) is accepted when kept legible; grind alone is not content |

### E. Classic incrementals (prestige references)

| Game | Meta layer | Offline | Burnout |
|---|---|---|---|
| Cookie Clicker | Ascension gives heavenly chips; each prestige level is +1 percent, level cost grows as the cube (V Wikipedia); golden cookies as timed attention beats | Not described | IGN "greatest idle game" and Destructoid "illusion of progress" (V) |
| Clicker Heroes | Ascension after primal bosses; Hero Souls buy Ancients (V) | Idle DPS simulated (V fun-ideas) | "Too many brickwalls" (V Push Square, via Wikipedia) |
| Egg, Inc. | Soul eggs boost egg value by percent; epic research permanent; grain silos let the farm run longer (V Wikipedia); golden eggs are the premium currency and buy temporary boosts | Silos = offline length | "Ultimately a little pointless" (V Gamezebo) |
| Antimatter Dimensions | Layers: boost, galaxy, infinity, eternity, reality; each opens a new area (V Wikipedia) | -- | pacing reception not checked; U |
| Kittens Game, Realm Grinder, NGU Idle | Deep multi-currency incrementals | -- | Steam store and reviews read in the second pass (8.6): Kittens "obtuse", "grindy"; Realm Grinder faction choice praised, mid-game "long gaps"; NGU "numbers go up" and "will absorb your life" (V) |
| Idle Wizard, Evolve, Trimps | Incrementals | -- | Not read (U) |

## 2. Per-game notes where the pass found something new

### 2.1 Godville (closest kin; first pass S, upgraded to V in 8.1)

- **What the player does.** Encourage, punish, miracle, voice of god, send to the arena, prayer. The GodWiki (read through Wayback in the second pass) confirms 25 percent for encourage and punish, 50 percent for a miracle (needs a completed temple), 5 percent for a voice and 50 percent for the arena, with an accumulator charge that restores 50 percent (V, 8.1). The hero "may ignore your wishes"; the site says the god can "do nothing" or direct "unsuccessfully" (V godvillegame.com).
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
| 1 | **Strictness presets plus dry-run preview** on the policy sheet (Lenient / Standard / Strict loot tiers; "would sell N of last night's M"; plus a one-key show-everything override; 8.4) | PoE NeverSink and FilterBlade (V: 7 levels, start semi-strict, simulator preview), Last Epoch | M | High | Extends TASK-128 policysheet; new card (policy-dryrun) |
| 2 | **Reason line when the hero declines or defers an order** in character voice | Godville (V: reply marked with an arrow in the diary) | S | High | TASK-155 unlockvoice; new small card |
| 3 | **Retreat pricing on the expedition card** ("leave now: keep X, lose Y") and a "go home" order | Loop Hero (V) | M | High | TASK-127 returnscreen; TASK-161 order queue; new card |
| 4 | **Return screen as a diary**: 3 to 6 funny/true lines first, numbers second (Godville shows 10 lines, bold for rare events, 8.1) | Godville diary (V), Egg silos (V) | M | High | TASK-127/109 returnmock; TASK-155 voice |
| 5 | **Nudge budget**: free orders limited by a slowly refilling "attention" meter; long-away refills it, over-managing drains it (Godville refills from the hero's level-ups and prayers, has a cost ladder, and can be switched off; 8.1) | Godville godpower (V) | M | Medium, risky | New card, needs a user decision (section 6) |
| 6 | **Offline bounties**: timed town-side tasks the hero runs when the player is away (clear X by tomorrow) | AFK Arena Bounty Board (V) | M | Medium | New card on the order queue |
| 7 | **Choose-one boss chest** (3 items, take one, reroll once) at named bosses | Halls of Torment, Vampire Survivors (V) | M | Medium | Codex firsts (TASK-130) as a question kind; no loot change if offered from the existing drop |
| 8 | **Evolution by condition**: a discovery that unlocks only when two prerequisites are met (a charm plus a first kill) | Vampire Survivors weapon evolution (V) | S | Medium | Unlock ladder TASK-153 trigger table (data only) |
| 9 | **Mastery per repeat zone** (repeat clears of a place raise a "familiarity" bonus to the hero's caution or tally) | Melvor mastery (U) | M | Medium | Codex pages; needs tally design |
| 10 | **Hall retire as a tree, not a number**: legacy is a small skill tree that the next hero reads | Idle Slayer ascension tree (S), Cookie Clicker chips (V) | L | Medium | TASK-135/163/171 Hall cards |
| 11 | **Hide-but-log** (rank next to #1): filtered items still drop, count shown ("hidden: 212 whites"), and the item says "Filtered Out" in place of its odds line | Last Epoch (V), IdleOn "Filtered Out" (V) | S | Medium | Policy sheet effect lines |
| 12 | **Nickname and myth hooks**: the hero names its merc, boss that killed it, and a favourite item; the journal reuses them | Twitch Plays Pokemon, Dwarf Fortress (V) | S | Medium | TASK-155 unlockvoice |

## 4. Anti-patterns to avoid (with evidence)

1. **Hidden caps and surprise limits.** Diablo Immortal's undisclosed caps on legendary drops and rewards fed the record-low user score and a community site dropping coverage (V Wikipedia). Rule: every limit (daily tally, charter price, Pact heat cap) shows on the sheet.
2. **Walls instead of prices.** Clicker Heroes drew "too many brickwalls" (V Push Square via Wikipedia). Already ADR 0018 (price, never a wall); keep it, including after the 1.14d content is cleared.
3. **Grind as the only content.** Torchlight Infinite complaints about repeating maps (S), Warframe's own note that people leave after a few hours if systems ask too much (V), Idleon long-time players describing a late game that became "psychologically exhausting" (V Steam reviews, 8.2), Soda Dungeon's "just do the setup and then don't care" (V). Rule: when a loop repeats three times, something new appears (Forager rule in fun-ideas).
4. **Monetised shortcuts on the core loop.** Idleon auto-loot as a purchase (S), Diablo Immortal crests, Idle Champions chests, time-gate pieces and familiars sold (V Steam reviews), Torchlight Infinite mandatory gacha pets (V). We have no store; do not sell or gate comfort that a tester would assume is baseline (auto-sell, auto-pickup).
5. **Silent filter mistakes.** Last Epoch's own guide lists "misplaced rules hide wanted items" and "deleting a filter is permanent" as gotchas (V Maxroll). Rule: the sheet needs undo, versioning and a dry-run.
6. **Illusion of progress.** Destructoid on Cookie Clicker, Gamezebo on Egg ("a little pointless"), Push Square (V). Rule: tally must stay tied to D2 events (firsts, kills, quests), not a free-running counter.
7. **Control the player cannot feel.** The Godville aggregator quote asking for "much more control" (S). Rule: an order has to be visible in the next five minutes of play, or the journal must say why not.
8. **Chaos with no handle.** Twitch Plays Pokemon democracy mode removed what drew viewers (V); in our case "safety" that removes every near-death moment would remove the first watch moment. Keep the stance row (TASK-191) honest: the cautious setting must still show fights.
9. **Offline length as a paid axis.** Egg, Inc. sells the away window: a standard permit allows 2 silos at half the offline rate, a pro permit 10 silos at the full rate (V archive of the Egg Inc wiki); AFK Arena's 12 h cap is raised by VIP (V appgamer). Rule: the away window and its rate are baseline and shown on the return screen; a longer window may be earned, never sold.
10. **Notification and popup spam.** Idle Champions reviewers cite a popup on every party change; Torchlight Infinite reviewers cite shop notification spam and a currency for every daily action (V Steam reviews, 8.2/8.5). Rule: one return screen, no modal on routine events, no daily-login chains.

## 5. Proposed cards (top 5; title, owns, size, ACs; three more from the second pass are in 8.8)

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
7. **Evidence gap (reduced, not closed).** Section 8.7 collects what could be read about why people watch bots (Neuro-sama fan survey, Claude Plays Pokemon, Twitch Plays Pokemon, IdleRPG/Progress Quest culture, Kolbot builders). Nothing readable covers D2 bots on Twitch specifically. A short poll of your testers ("what did you watch for?") is still better than any of it.

## 7. What this pass confirms or challenges in the existing design

- **Confirms:** the policy sheet (loot filters are the genre's own proof, V); return screen first (Godville, AFK Arena, Egg: the away reward is the core); price not wall (Clicker Heroes); voluntary Hall retire (walls are resented); the stance row (Twitch Plays Pokemon, bullet heaven: players want to see risk); no store (Diablo Immortal).
- **Challenges:** "no failure state" does not hold for our genre: Loop Hero, Warframe and D2 all price failure; our return screen should show what a death or retreat cost. Also the 15 to 30 minute run length of bullet heaven suggests our expedition card should have a **visible short beat** even if the real trip is longer.
- **Gaps (first pass; most closed by section 8):** Godville rules, Melvor cap and the unread Steam games are now V. Still open: Idle Champions specialization detail and IdleOn's numeric offline cap (8.9). Never read: Hero Wars, Idle Heroes, Idle Wizard, Evolve, Trimps, Fallen London, Raid-style summon games, Dungeon Fighter idle; U and not used for any recommendation.

## 8. Second pass (2026-11-08): the walled games read, and why people watch bots

Method: Wayback copies of walled wiki pages fetched with curl (`web.archive.org/web/2024/<url>` works; the WebFetch tool refuses archive.org), the IdleOn and Melvor wikis (not walled), Steam store pages and Steam community reviews (`steamcommunity.com/app/<id>/negativereviews/?browsefilter=toprated`, top-rated, English, a sample of 3 to 7 per game, so anecdotal), Hacker News through the Algolia API, arXiv and Wikipedia. Reddit could not be read at all (8.9). Review quotes are shortened; hours are the reviewer's playtime.

### 8.1 Godville in depth (V unless marked; GodWiki through Wayback)

- **The budget.** Godpower is a meter from 0 to 100 percent (200 for gods who built a shop). Costs: voice of god 5, encourage 25, punish 25, send to arena 50, challenge a friend 50, miracle 50 (needs the Temple), dungeon drop 50, resurrect 0, blessing 60 charges. So a full meter is 4 encourages or 20 voices (arithmetic on V numbers). Pages: wiki.godvillegame.com Godpower, Remote_Control.
- **The refill.** An accumulator (from level 5) turns a charge into 50 percent godpower (70 on some days), never above the maximum. Six sources: one charge per hero level-up; converting a full meter into a charge when empty (first three only); an arena win; certain artifacts; a built Temple "several charges per day"; and paid charges. The hero also prays on its own at shrines, temples or in town and that returns "15%+"; a temple donation adds roughly (gold - 100) / 10 + 20 percent (an 80 percent accurate community fit, quoted as such). Voting on other players' ideas for at least half of the last 48 hours boosts prayer for a day. Net: a free player gets a handful of big nudges a day and many cheap voices; the pacing lever is the hero's own level-ups and prayers, so the game pays you nudges for the hero's progress. Charges are sold (PayPal price table) and can be gifted.
- **Nudges are never certain.** Voice: free text, 5 percent, orders "aren't always followed well"; the hero ignores obscenity and anything silly (stopping a fight to pray); since June 2021 the hero marks a reply to a voice with an arrow in the diary. Encourage: always diary-logged, shifts the hero's alignment toward good, heals 60 to 180, can backfire (hurt the hero, heal the monster) or do nothing. Miracle: 100 percent success and no alignment change. A coarse gradient: cheap and uncertain, mid-price and risky, expensive and certain.
- **Zero-player is a switch.** The wiki: "To make Godville a complete ZPG-game, the Remote Control can be disabled at the options page." Retention evidence: TouchArcade's writer played about four years, "you can literally not even touch the game for a year at a time and come back" (V toucharcade.com 2014); GamesRadar's Jim Sterling is quoted "Not playing videogames has never been this much fun" (V as printed on that page). HN: "play as actively as you want, but you can also close the tab and not return for years" (V, item 28193680).
- **The diary is a window, not an archive.** The god sees exactly 10 entries (20 after the Book of Creation), which is 10 to 15 minutes of life. Entries are timestamped; colour carries source (default ink: the hero acted alone; blue: your influence; red: a rival god's); rare events are bold (temple done, level-up, death). A "Third Eye" keeps 5 to 20 important entries a day (purchases, boss kills) and a public "Chronicles" scratchpad is the god's own text. The phrases themselves are community-written through an Ideabox with voting. Pets unlock at level 18; arena duels cost 50 percent and the winner gets coins plus an accumulator charge.
- **What the critics say (S/V mix).** The writing is the product ("if it's not funny to you, pick something else", S aggregator); the complaint is grind and early repetition (S). I found no first-hand burnout threads (Reddit unreachable).
- **Mapping.** (a) The Chronicle should be a short rolling window plus a highlight tier plus bold for firsts, with the source of each line coloured (bot alone, your order, your policy sheet). (b) A nudge hierarchy (cheap and uncertain, costly and certain) is how Godville keeps influence meaningful; the open question stays whether we ration at all (section 6, item 1). (c) The meter refilling from the hero's own progress is a ready-made rule if the user chooses a budget.

### 8.2 Idle Champions, IdleOn, Melvor: decisions, caps, complaints

- **Idle Champions (V Steam store).** "Optimize formations using over 140 Champions"; familiars automate levelling and gold pickup "so YOU can focus on optimizing your formation strategy"; champions fight on even if you close the game; "multiple content updates every month". Patrons unlock later, add rewards for variants and add "new restrictions and increase requirements" (V archive of the Fandom stub). Positive reviews: playable one-handed, "no massive pressure 'login' every 15 minutes", "sit down for a few minutes then walk away". Negative (12.6k, 5.5k, 2.7k, 42k hour players): "they will sell literally everything possible: chests, champions, time gate pieces, potions, scrolls, cards, familiars"; "event characters behind a paywall too"; a character you paid for "gets nerfed"; "annoying pop-ups ... every single time you change your party"; high CPU use; "feels thin ... like butter scraped over too much bread". Pacing complaints are about monetised shortcuts, not about the formation decision itself, which nobody attacks. Specialization and adventure unlock detail: wiki unreachable (401).
- **IdleOn (V).** Store text: "Load up IdleOn the next day, spend your AFK gains earned while you were away, and buy MORE upgrades"; the free-to-play note promises no gated content. The wiki's AFK Info screen shows survivability (defence plus food plus health; "the lower the survivability, the less gains") and "NO DMG REQ: n" for the current map, and hourly kill caps per monster; fighting AFK gains depend on equipped active talents, weapon speed, spawn rate and crit. A player: "Active afk": you set up a build and leave it overnight for 5 to 20 upgrades. A patch note adds "Items that are filtered out now say 'Filtered Out' where they usually show their 1 in X odds display" (V, Steam patch notes thread). Burnout, from 994 h, 6.3k h, 8.5k h and 12k h reviewers: the first months were "one of the best games", then "FOMO gacha", a new premium currency, weekly paid packs, "psychologically exhausting"; others recommend playing full free-to-play. The numeric offline cap was not found (a TouchArcade listing claims "unlimited", S).
- **Melvor (V wiki, v1.3.1).** The game records the skill and action when closed, simulates the gap "as if the game had been left open" up to 24 hours, then shows "a summary of what has occurred while offline" and resumes. Combat and thieving offline are off until "Toggle Offline Combat" is on, because both can kill; food must be equipped, potions need auto re-use. Lesson: risky away-time is opt-in, with the survival precondition spelled out.

### 8.3 Return screens: what comes first (V where read)

| Game | First thing shown | Cap and how it is sold | Read time |
|---|---|---|---|
| Melvor | A "Welcome Back" summary of skill XP, mastery XP, items; action resumes | 24 h, baseline (V) | not measured |
| AFK Arena | A chest on the main screen: tap for XP, essence, coins, items (V appgamer) | 12 h, raised by VIP; a "fast rewards" button gives 2 h at once (S) | one tap |
| Egg, Inc. | Earnings accrue per stored hour in silos; one silo = one hour (V archive) | Standard permit 2 silos at half rate; pro permit 10 at full rate (V) | not measured |
| IdleOn | The AFK claim then spend it; AFK Info with survivability before you leave (V) | cap not found | not measured |
| Godville | A 10-line diary, newest first, bold for rare events (V) | none: runs for years | about 10 lines by design |

Nobody publishes read times; "not measured" is honest. The pattern: one tap or one screen, loot as a summary, the cap stated up front, and the risk (Melvor, IdleOn) shown before leaving, not after.

### 8.4 Loot-filter UX facts (V)

- **Strictness presets with a preview.** NeverSink's README: "7 different strictnesses, each hiding more items. Start with SEMI-STRICT if you're new. Increase the strictness as you become faster, stronger and more experienced". "The hidden items are always the cheapest layer. You don't have to worry about expensive drops, these are always shown." FilterBlade offers a loot simulator to preview every strictness and a fine-tune editor; styles by taste or visual impairment; a filter-ladder subscription that auto-updates. Source: raw.githubusercontent.com/NeverSinkDev/NeverSink-Filter README.
- **Viewer-aware audio.** A CUSTOMSOUNDS style exists "for streamers or viewers so that they don't get confused when the other side is getting an expensive drop". A watch screen with sounds needs distinct cues per drop class.
- **Last Epoch order footgun.** A forum guide (V forum.lastepoch.com): rules run top to bottom, the first full match stops evaluation, swapping two rules can hide everything, "X" temporarily disables the filter; the safe method is "recolor what matters first, then hide only what you are 100 percent sure about". Combined with the first-pass Maxroll facts, the sheet needs: a shipped default ladder, a preview, an order warning, a one-key "show everything" override, and a "Filtered Out" reason on the item (IdleOn's wording).
- **Torchlight Infinite auto-loot (S).** Search snippets from two guide sites: auto-loot rides on a Pactspirit and a premium tier targets high-value drops; a Steam user asks whether a strict filter is needed when auto-loot exists. The Maxroll TI filter page returned 404 and the TapTap dev blog is only an intro, so this stays S.

### 8.5 Auto-combat QoL: Torchlight Infinite and Diablo Immortal

- **Diablo Immortal (V massivelyop 2026-05-01, quoting the official post).** An auto-battle that runs around and kills monsters; the post "points out that this feature is 'deliberately less efficient than a player actively farming themselves'", some builds more so. Search snippets add one-hour sessions, a weekly cap and a level gate (S, not on the page I read). Lesson: auto may be built as a deliberately weaker mode; our bot is the opposite (it is the main player), so the analogue is "efficiency of idle vs. attended" must be stated, not hidden.
- **Torchlight Infinite (V Steam).** What players hate is not the automation but the layer around it: pets from a gacha described as mandatory for damage, "shop notification spam", daily-login currency chains ("Log every day to get coins ... Pay monthly to get more coins"), build "stone-walls" by timers. What they like: solo casual play with "0 microtransactions", "couple hours daily decompression". No verified list of "auto features players ask for" exists in what I could read (Reddit unreachable).

### 8.6 The unread games, short (Steam store text plus top-rated reviews; V)

| Game | One line | Burnout / lesson |
|---|---|---|
| Rucoy Online | Mobile 2D MMORPG, free (V official site only) | Nothing readable beyond the tagline; no claim |
| Soda Dungeon | Free idle dungeon raids, no timers | "with auto ... do the setup and then don't care"; beats "small and far apart" (a pure autopilot loses interest) |
| NGU Idle | Kongregate hit, "numbers going up", 300 bosses | 3.4k h to 12k h reviewers: "will absorb your life", "half-heartedly recommend"; humour carried it, depth hid late walls |
| Realm Grinder | Faction choice changes how a run plays | Praised: runs differ by faction; negative: "long gaps between interesting tasks", factions unbalanced. A player notes the undead faction "excelled at being left offline" (V, a player in the IdleOn Steam thread, recalling Realm Grinder) |
| Kittens Game | Text village sim, long haul | "extremely grindy", "horrible about communicating information"; loyal fans call it the genre founder |
| Grim Dawn | Dual-class ARPG, loot-focused | Negatives are about a 2026 expansion's one-shots and boss health, not loot; one reviewer says hours "with the application simply open" were sunk cost |
| Destiny 2 | Free-to-play shooter | "draconian progression systems and temporary-by-design story content"; "chasing the dragon". Not an idle precedent |

### 8.7 Why people watch bots (the evidence, and 10 takeaways)

What was read: the first fan study of an autonomous AI streamer (V arXiv 2509.10427: 334 survey responses, 12 interviews, chat logs); the Claude Plays Pokemon story (V Engadget); Twitch Plays Pokemon and Dwarf Fortress (V Wikipedia, first pass); IdleRPG/Progress Quest talk (V Hacker News); Kolbot builders (V GitHub, Discourse); a 1990s newsgroup post on NetHack bots (V chiark.greenend.org.uk); Salty Bet (S, abstract in a search result). Not found: any readable source about Twitch audiences for D2 bots, Median XL bots, Path of Exile SSF AFK viewing, Cataclysm bots, or Dungeon Crawl "crawl bots" (Sequell is only a game-record IRC bot, S). So this is a transfer from adjacent cases, labelled as such.

Takeaways, each tied to a feature:

1. **Stable persona, unpredictable events.** Neuro-sama's fans: 90 percent rate the "unpredictable, surprising atmosphere" important, 83 percent see a coherent personality, and "consistency rather than humanness anchors authenticity" (V arXiv). Feature: hero voice. Keep one fixed voice; let events vary. Do not rotate tone per class. Cost: write the persona once as data.
2. **Visible reasoning is a draw.** Claude Plays Pokemon prints a paragraph of reasoning before each button press ("As per my strategy, I should flee from this battle to conserve resources"), and viewers watch the assumptions form and fail (V Engadget). Feature: the watch screen and the questions queue. Show a one-line "why" under the current action and the reason each question exists.
3. **Failure with a clever recovery is the best story.** Stuck in Viridian Forest for days; escaped Mt. Moon by deliberately fainting the party; then deleted its own notes and looped (V Engadget). Dwarf Fortress "Losing is Fun" (V). Feature: Chronicle and risk meter. Prefer near-death, stuck and recovery lines over kill counts; the risk meter makes the danger visible before the story happens. Never hide a death.
4. **Names and myths grow from chaos.** The Helix Fossil religion and nicknames in Twitch Plays Pokemon (V, first pass); Neuro's "Swarm" meme "derive[s] directly from her AI-generated narratives" and fans adopt it (V arXiv). Feature: Chronicle and Hall. Name the merc, the nemesis monster, the favourite weapon, reuse them in lines, carry them to the Hall.
5. **Watchers want to steer.** In Neuro's chat the largest category is questions and commands (26 percent), the inverse of human streamers, and 85 percent of paid messages are proactive, to redirect the stream (V arXiv). Feature: questions and stance. A watcher who can set a stance or answer a question is a co-performer; this is also the strongest argument for the hero-proposes, player-answers loop. The paper warns that rationing co-creation by money is a governance choice; ours is free, so a budget (nudge meter) would be a pure design lever.
6. **Stats, streaks and milestones are the bot-runner's own spectator layer.** A Kolbot builder wrote a run tracker with best streak, average run time, session stats and "announce every 50 runs" milestones (V Discourse, 2026); PQ players compared character stats at work (V HN). Feature: Hall and Chronicle milestones. Count runs, streaks, personal bests; announce round numbers.
7. **Ambient, low-attention viewing.** "People listen to podcasts or watch TV in the background while working, but I'd rather have something like this constantly playing" (V HN, 28193680); PQ as a screensaver and a call-centre time-killer (V); IdleRPG rewarded not interacting (V HN). Feature: the watch screen. Calm by default, no modals, readable at a glance from across a room, sounds rare and distinct per drop class (the CUSTOMSOUNDS lesson, 8.4).
8. **Bots as company.** The d2-qolbot README builds bots that join your game, "crack jokes", gather waypoints and "play with you" (V GitHub). Feature: hero voice and merc. A little gallows humour per boss or death is the bot-as-companion idea, and it stays inside one persona (takeaway 1).
9. **Strategy is where bots are interesting and where they fail.** A veteran newsgroup post: NetHack bots are hard because of "difficult strategic decisions" (which branch, what to cache, when to dive) while the Angband Borg gets by on "a very high boredom threshold" and always farms a safe depth (V chiark). Feature: questions and quest hints. Our questions are exactly those strategic forks; surface them as the interesting beat, and let quest hints name the next fork before the bot reaches it.
10. **Clips and moments are the discovery path.** 96 percent of Neuro fans first met it through algorithmic clips (V arXiv). We have no network, but a local export of one Chronicle moment (text or image) is the offline analogue and what a tester would show a friend. Feature: Chronicle share card.
- Counterweights: a TechRadar piece argues AI streamers are "experiments people peek at" more than entertainers (S); D2R forum threads show the bot community also harms multiplayer economy and ladders (V Blizzard forums), which is why offline single-player and honest naming matter; Salty Bet's appeal is bets on "foolish AI" underdog stories (S).

### 8.8 Effects on the ranking and the card list

Changes to section 3 and 4 (everything else stands):
- **#1 policy presets**: now backed by V UX facts (7 strictness levels, start semi-strict, cheapest layer hidden first, simulator preview, a one-key show-all). Effort unchanged; the card gains an AC for the show-all override.
- **#11 hide-but-log**: IdleOn ships a "Filtered Out" label in the odds line (V), so the effect-line idea has a precedent. Promote next to #1 (rank 2, not a new row).
- **#2 reason line**: Godville marks the hero's reply to a voice with an arrow (V). Same card.
- **#5 nudge budget**: stays Medium, risky. New V: Godville keeps a three-tier cost ladder, refills the meter from the hero's own level-ups and prayers, and lets you disable the meter entirely (ZPG); a fan study says rationing co-creation is a governance choice. The card's AC (c) should also compare "no budget, but a ladder of certain/uncertain orders".
- **#3 retire pricing**: Melvor makes risky away-time opt-in with preconditions (food, success rate). Merge that into the card.
- **Anti-patterns**: added 9 (offline length as a paid axis) and 10 (notification and popup spam); item 3 and 4 are now V from Steam reviews. Items 2 and 6 unchanged.
- **Challenges to the "no failure" point in section 7 stay**; takeaway 3 adds that failure with a recovery is the best spectator beat.

New cards (3):

**6. watchfeed: Chronicle as a rolling window with source colours and a share card.** Owns: the Chronicle and watch-screen modules (coordinator to name), `docs/research/watchfeed.md`. Size M. ACs: (a) the watch screen shows 10 lines newest-first with bold for firsts, near-deaths and boss kills, colour by source (bot alone / your order / your sheet); (b) a highlights tier keeps the day's best 5 to 20 lines like Godville's Third Eye; (c) nicknames for merc, nemesis and favourite item are stored and reused; (d) one-click local export of a moment as text; (e) a fixture test on a recorded session; no new network feature. Maps: TASK-155 voice, watch screen, Hall.

**7. whyline: a one-line reason under every action and question.** Owns: the order/decision journal template data. Size S. ACs: (a) each bot decision that costs more than a few seconds (retreat, potion buy, skip a chest, route change) writes a cause key and a one-line why; (b) the watch screen shows it under the current action; (c) questions show the same cause; (d) two phrasings per cause, data only, tested on two builds. Maps: TASK-155, question queue. Overlaps card 2 (declinereason); the coordinator may merge them.

**8. awayfilter: away-time stance with a survivability line.** Owns: the away/stance data and the return-screen header. Size M. ACs: (a) before leaving, the stance row shows a survivability estimate and the stated cap (IdleOn AFK Info, Melvor combat toggle); (b) risky activities while away are off unless switched on, with preconditions listed; (c) the return screen states the cap used and what the stance skipped; (d) no change to the bot's own safety rules. Maps: TASK-191 stance, TASK-127.

### 8.9 What stayed unreachable, and why

- **Reddit** (r/godville, r/Idleon, r/Melvoridle, r/incremental_games, r/diablo2): old.reddit.com and www.reddit.com answer 403 to curl, Wayback has no usable snapshot, the public mirrors (redlib, pullpush) were down or refused. So no first-hand burnout threads for Godville and none for D2 bot culture beyond forum posts; Steam reviews stand in for burnout evidence.
- **Idle Champions specialization/formation wiki pages**: wiki.gg answers 401, the archived Fandom copies of those pages do not exist (the Patrons stub does).
- **Maxroll Torchlight Infinite filter guide** (404 through the headless route used in the first pass and by curl); TI auto-loot stays S.
- **IdleOn's numeric offline cap**: not stated on the wiki pages I could open or on Steam; Melvor's is V (24 h).
- **AFK Arena and Egg Inc. return screens**: only secondary guides and an archived wiki (V as guides), no first-party screenshots; read times unmeasured everywhere.
- **PC Gamer and Salty Bet articles, the LSBU spectatorship paper**: body not extracted or a bot challenge; Salty Bet stays S.
- **Twitch/YouTube audiences for Kolbot, Median XL bots, Path of Exile SSF AFK, Cataclysm bots, Dungeon Crawl bots**: no source found with the search tool; YouTube transcripts were not tried (no transcript tool here). Not claimed.
- **Godville monetisation beyond charge sales and burnout threads**: wiki pricing table seen, no player discussion.

## Sources (V = read)

Second pass (all fetched 2026-11-08):
- V Godville: https://web.archive.org/web/2024/https://wiki.godvillegame.com/ with pages Godpower, Remote_Control, Prayer, Diary, Encouragement, Voice_of_God, Miracle, Accumulator; https://toucharcade.com/2014/04/08/godville-is-the-only-ios-game-ive-played-for-nearly-four-years-straight/
- V Melvor: https://wiki.melvoridle.com/w/Offline_Progression ; V IdleOn: https://idleon.wiki/wiki/Game_Mechanics ; https://steamcommunity.com/app/1476970/discussions/0/6252725698201432654 ; https://steamcommunity.com/app/1476970/discussions/0/758394165818064926
- V Steam store pages and top-rated community reviews (steamcommunity.com/app/ID/negativereviews and positivereviews): 627690 Idle Champions, 1476970 IdleOn, 1974050 Torchlight Infinite, 1147690 NGU, 610080 Realm Grinder, 1097410 Kittens Game, 564710 Soda Dungeon, 219990 Grim Dawn, 1085660 Destiny 2
- V https://web.archive.org/web/2024/https://idlechampions.fandom.com/wiki/Patrons ; https://web.archive.org/web/2024/https://egg-inc.fandom.com/wiki/Silos ; https://www.appgamer.com/afk-arena/strategy-guide/afk-rewards
- V https://raw.githubusercontent.com/NeverSinkDev/NeverSink-Filter/master/README.md ; https://forum.lastepoch.com/t/a-basic-guide-on-how-loot-filter-works-and-how-to-set-up-your-own-loot-filter/25389 ; https://massivelyop.com/2026/05/01/diablo-immortal-introduces-an-autoplay-feature-open-captain-selection-and-new-legendary-affixes/
- V bots and spectators: https://arxiv.org/html/2509.10427v1 ; https://www.engadget.com/ai/claude-isnt-a-great-pokemon-player-and-thats-okay-151522448.html ; https://news.ycombinator.com/item?id=40141350 and item=28193680 (read through hn.algolia.com) ; https://github.com/whipowill/d2-qolbot ; https://blizzhackers.discourse.group/t/release-d2channelbot-a-stealthbot-alternative-for-kolbot-chaos-baal-run-tracker/1762 ; https://us.forums.blizzard.com/en/d2r/t/a-thread-for-botters/61350 ; https://www.chiark.greenend.org.uk/~damerell/games/nhbot.txt ; https://en.wikipedia.org/wiki/Rog-O-Matic
- S: search snippets for Salty Bet spectatorship (LSBU paper), TechRadar on AI streamers, Diablo Immortal auto-battle limits, Torchlight Infinite auto-loot tiers, AFK Arena fast rewards.

First pass:

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
