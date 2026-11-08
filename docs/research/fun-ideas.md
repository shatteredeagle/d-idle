# Fun ideas: the converged d.idle loop (design partner, rounds 1 and 2)

> status: DESIGN, 2026-11-07, branch agent/funideas (design-partner agent, Fable), docs only; nothing built, no Go or game touched. Input: the user's words of 2026-11-07 (dictated by voice; unclear phrases are flagged), his popup answers to Q-U1 to Q-U10, the d2-sme agent's answers to Q-S1 to Q-S10 (veteran memory, its UNSURE tags kept), the direction review (ADR 0017) and the repo docs named in the sources. This document is the round 2 result: two to three ideas concrete enough to prototype, the research they borrow from, and what was dropped.
> Tags: **V** read in this tree (file named) or read on the web (URL named); **S** secondary (a doc's or the SME's claim, not re-checked); **U** unverified or my judgment; **UNVERIFIED** a Diablo II mechanic to confirm before it enters a contract; **D** a user decision of 2026-11-07. Classic 1.14d throughout; no D2R fact is used.

## 0. Answer in ten lines

1. D2's kill, loot, level loop lacks the four incremental ingredients: a spendable accrual, an allocation decision, compounding and a return payoff. The thing that compounds in d.idle is the hero's competence (rules, zones, kit), not its XP (section 1).
2. **Idea A, the Expedition Ledger**, is the loop: every town trip is an expedition with a cost and a report card; tally accrues per card; the player spends tally on charters (the price for above-range trips), order slots, hero perks and merc upkeep; the policy sheet is where the player experiments (section 3).
3. **Idea B, the Act 1 Codex**, is the content: the hero's first meetings with Act 1 (Shaman, Corpsefire, Bishibosh, the Cold Plains waypoint, Blood Raven, the Cairn Stones, the Countess, Andariel) fire one discovery question each, fill a codex and unlock skills by quest or find, never by currency (section 4).
4. **Idea C, Legacy and the Hall**, is the day-scale loop: a retired or dead hero leaves a small visible legacy charm for the next one, scaled by account level, and a memorial line; hardcore rebirth later (section 5).
5. A and B are one product; B ships first because it is catalogue data. C waits for the overlay probe and the rebirth decision.
6. The pacing gate is a price with a warning, never a wall (D Q-U1); the hero proposes the next zone itself (D Q-U4); hard walls feel un-D2 (S SME).
7. Chores are not questions: one policy sheet, set at hero creation, changeable any time, allowed to be suboptimal, with consequences shown so the player learns (D Q-U3).
8. The first metacurrency test credits tally in the existing ledger from observed events, no item (D Q-U7); the Raven's Mark item is costed separately.
9. Blessings (a currency-bought skill tree), level walls, gambling, permanent stat points and the loot filter as a standalone game are dropped (section 6).
10. Research (section 2) sets the pacing targets: first decision within a minute (the sheet), first unlock within 15 minutes (the Den skill point), first spend within 30 minutes (a charter), a report card every 10 to 20 minutes, the first "reset" (retire a hero) at about 3 hours.

## 1. Diagnosis in short (round 1, kept)

An incremental has four ingredients (V section 2 sources): (1) a spendable accrual that ticks visibly; (2) an allocation decision ("what do I buy next"); (3) compounding (the purchase raises the accrual); (4) a return payoff (offline progress summarised) and later a prestige reset. D2 as-is has none: XP is an accrual but not spendable (the plan table spends points, V internal/plandata); gold is spendable but its sinks are weak (S veterans); loot is wealth but lumpy and expert-evaluated (the pickit does it silently, V internal/loot); power is not legible (the power-vs-danger rule of ADR 0011 is internal); content is a linear chain of zones gated by danger; and D2 has a failure state, death, that incrementals lack. What D2 has that no clicker has: the town loop as a natural cadence every 5 to 20 minutes, loot drama, named bosses, waypoints as checkpoints, a hero with a build. The only thing that compounds in d.idle today is the hero's competence: a learned rule is permanent automation (V ADR 0016). The design below makes that visible and buyable.

The user's level-gate idea (clvl 5 for Cold Plains, 10 for the Burial Grounds) is good as a legible milestone and bad as a wall: D2 gates by danger, a wall removes the one dramatic choice (go early and risk it), it contradicts the orders map, and at 1x the wait is one to two hours with nothing accruing. The user chose a price with a warning (D Q-U1); the SME's ranking is caution first, price second, a soft recommended-level label third, no hard walls (S).

Voice-transcription flags in the user's quote (2026-11-07): "not progress/tuning gates" is read as "not the project's soak and fidelity gates"; "the mini boss there whose name I forget" is Blood Raven (V docs/research/orders-ux-design.md); "stopping for whites" is read as "picking up white (normal-quality) items"; "what kind of potions" is read as potion tier and mix. If any reading is wrong, section 3 changes at the policy sheet.

## 2. Incremental games research

Read for this round (one line each; everything else about these games is from memory and tagged U):

| Source | Takeaway |
|---|---|
| Pecorella, The Math of Idle Games part I, https://www.gamedeveloper.com/design/the-math-of-idle-games-part-i (V) | Costs grow exponentially, production linearly; multipliers at owned-count thresholds are the catch-up; "the newest generator is nearly always dominant", so vary multipliers per generator or choices vanish; felt impact beats math. |
| Pecorella, part III (prestige), https://www.gamedeveloper.com/design/the-math-of-idle-games-part-iii (V) | A prestige reset gives a ladder-climbing feel and pulls numbers back to scale; the sample first run is about 120 minutes; prestige currency is a root of lifetime earnings so each run must push 3x to 8x further to double it; spreadsheets are rough guides, players who hit a milestone keep going. |
| Kongregate, Quest for Progress page, https://www.kongregate.com/pages/quest-for-progress-the-math-of-idle-games (V) | Public spreadsheets for generator choice and threshold bonuses; shifting the top generator keeps play varied. |
| Guan, Idle Game Design Principles, https://ericguan.substack.com/p/idle-game-design-principles (V) | Pacing follows decaying engagement (hourly check-ins become weekly); staggered production caps (20 min, 5 h, 2 days) are re-engagement clocks; extra currencies must differentiate playstyle, not denominate. |
| Machinations, idle games and how to design them, https://machinations.io/articles/idle-games-and-how-to-design-them (V) | Core loop (simple earn cycle with a visible counter) and meta loop (the layer that carries once the counter bores) are separate; a robust offline mode plus a return reward. |
| "I built 7 idle games in 30 days", https://dev.to/aguier/i-built-7-idle-games-in-30-days-what-i-learned-about-incremental-design-5d3f (V, one developer's numbers) | First upgrade within 60 s, first automation within 3 min, a meaningful decision within 5 min, first prestige within 10 to 15 min; upgrades must change how you play, not add 10 percent; show the next 2 or 3 upgrades, not all 20; offline pays about half rate. |
| Wikipedia, Incremental game, https://en.wikipedia.org/wiki/Incremental_game (V) | Genre history: Progress Quest (2002) as the automated-combat parody, Cookie Clicker, Clicker Heroes' ascend as the prestige pioneer, AdVenture Capitalist's offline earnings; the appeal is low pressure, no failure state, steady feedback; the critique is "glorified spreadsheets". |
| Clicker Heroes blog, offline progression, https://blog.clickerheroes.com/offline-progression-in-clicker-heroes/ (V) | Offline is a simulation of idle DPS only, with no time cap but a natural stop where idle DPS no longer advances; gold waits to be collected on return; offline progression is "a core design principle". |
| Forager's approach to idle grinding, https://www.gamedeveloper.com/design/when-grinding-is-fun-i-forager-i-s-compelling-approach-to-idle-game-design (V) | Many small achievable goals and slowly opened features drive retention; upgrades as tangible things ("rocks explode"), not percentages; when a task tires, something new appears; the reward for grind is automating it. |
| Increlution, Steam page, https://store.steampowered.com/app/1593350/Increlution/ (V) | An action queue the game executes; auto-pause when the queue is empty ("planning, not micro-management"); death starts a new generation; generation levels are per life, instinct levels carry over slowly. |
| Path of Exile forum, "Why I feel compelled to use a loot filter", https://www.pathofexile.com/forum/view-thread/3900183 (V) | Filters exist because loot is clutter; a filter colour-codes, hides and reduces; the game should cover the basics and leave only taste to the player. |
| Spiel et al., idle game designers interview study, https://par.nsf.gov/servlets/purl/10174274 (fetched as PDF, NOT read: no text extractor on this box; U) | Listed for the coordinator; not used. |

From memory, tagged U: Kittens Game chains resources through caps so a storage building is a decision; Melvor Idle simulates the one selected skill offline with a cap of many hours and shows a summary on return; Antimatter Dimensions layers resets (boost, galaxy, infinity) so the first prestige lands within the first hour; Idle Champions makes the formation the decision and uses adventure variants as content gates; NGU Idle and Realm Grinder are multi-currency and are the usual "too many currencies" example; Diablo Immortal and Diablo 4 season passes are the counter-example: a schedule of unlocks that is not earned by play.

Lessons, and what maps onto d.idle:

1. **One visible counter at the core, a richer meta on top** (Machinations, Guan). Maps: the expedition card is the counter beat; tally is the meta. Does not map: a per-second tick; the user chose bursty (D Q-U2) and D2's natural burst is the town trip.
2. **Costs exponential, production linear, with threshold multipliers** (Pecorella I). Maps: charter and perk prices rise geometrically (section 3.4); a learned rule is the multiplier (it raises yield per trip). Does not map: generators; d.idle has one hero, not a column of buildings. The "newest generator dominates" trap becomes "the newest zone dominates", which is fine: D2 wants the player to move on, and the Codex rewards it.
3. **First decision within minutes, first automation soon after** (dev.to, Forager). Maps: the policy sheet is filled before the hero takes a step (minute 0 decision); the Den of Evil skill point is the first unlock at about 15 to 25 minutes (S SME table); the first learned rule is the first automation. Does not map: a 60-second first upgrade; D2's first ten minutes are Fallen in the Blood Moor and the Codex has to carry them.
4. **Prestige early, visible, and a root of lifetime earnings** (Pecorella III, dev.to). Maps: the first "reset" is retiring a hero after Andariel at about 2.5 to 3.5 hours at 1x (S SME Q-S2), close to Pecorella's 120-minute first run; the legacy charm is the carry-over, scaled by a root of account tally so each hero must go further to grow it. Does not map: ascension that wipes everything; D2 veterans want the hero to live on (D ADR 0016 item 8), so retirement is voluntary and hardcore death is the later forced form.
5. **Offline progress is a simulation with a natural stop; the return collects** (Clicker Heroes, Machinations). Maps exactly: the bot plays for real while away, the return screen collects (section 3.6). Caps: the hero's own caution is the natural stop (it holds in town when the kit cannot be bought); no artificial cap is needed.
6. **Upgrades must change play, not add a percent** (dev.to, Forager). Maps: sinks that change what the player can order (charters, slots) or what the hero does (perks that open a behaviour, such as "clears dungeon levels" or "revives the merc"), not "+5 percent pickup radius".
7. **Show the next two or three purchases** (dev.to). Maps: the spend sheet lists the next three affordable or nearly affordable items only.
8. **Extra currencies must differentiate playstyle** (Guan). Maps: one tally; gold stays D2's gold; the Raven's Mark item is a *form* of tally, not a second unit (V ADR 0016 Amendment 2026-11-06). Does not map: NGU-style currency stacks.
9. **A queue and auto-pause make planning the game** (Increlution). Maps: standing orders as a queue the player fills; the hero does not pause when the queue is empty, it proposes (D Q-U4), which is the idle reading of Increlution's pause.
10. **Loot filters are clutter management; the game should cover the basics** (PoE forum). Maps: whites, junk magic, gold, potions are policy defaults, never questions (S SME Q-S5 agrees); only the taste decisions (imbue target, hold a base, a +skill item) are cards.
11. **Steady feedback, no failure state** (Wikipedia). Does not map and should not: death is D2's heartbeat and the SME's first watch moment is "a close call at 8 percent life". The return screen makes death a story, not a punishment.

## 3. Idea A: the Expedition Ledger

**Pitch.** The town trip is the unit of play. Every time the hero leaves town and comes back it has run an expedition: a target zone, a length, supplies spent, and a return of kills, XP, gold, drops, firsts and close calls, shown as one report card. Tally accrues per card. The player spends tally on charters, order slots, hero perks and merc upkeep, sets the policy sheet the hero plays by, and answers the hero's proposals and the few real questions. The hero warns before a trip it thinks is too hard and charges a price; it never refuses an in-range order.

**The loop.**
- Accrues: tally (ledger credit per observed event, section 3.4), expedition cards, zones cleared and waypoints lit on the map, codex entries (idea B), rules learned, gold and gear in the game.
- Spent: tally on (a) **charters**: a trip to a zone above the hero's recommended range, or beyond the lit frontier, costs tally; in-range trips are free; (b) **order slots**: queue depth 1 to 5 (V `MaxOpenOrders` is 5 today); (c) **hero perks**, each a behaviour the hero gains, data rows over knobs (V docs/knobs.md metadata, PLAN.md handicap-profile design): "sweeps the whole level before returning", "takes the waypoint instead of walking", "revives the merc on its own", "banks runes and gems every trip", "clears dungeon levels"; (d) **merc upkeep**: a subsidy row that pays part of a hire or revive from tally (gold stays the primary cost); (e) **rule skips**: one tally price to skip a repetition toward a rule (V ADR 0016 item 12); (f) **flavour**: a hero title, a journal voice. Gold stays in the game and buys what D2 sells (potions, scrolls, repair, merc), which is the "price" half of the gate: an above-range trip also requires the full kit (S SME: a price in potions and scrolls fits D2's economy).
- Comes back: higher yield per card (rules, perks, better zones), new zones (charters), fewer deaths, a longer queue so the hero runs longer alone.

**Decisions.** Minute scale (at a return): approve or change the hero's proposal for the next trip, answer the one or two real questions on the card, buy the next perk. Hour scale: push the frontier (a charter), reorder the queue, change a policy and watch its effect over three cards. Day scale: retire the hero after Andariel for a legacy (idea C), or keep going into Act 2 (content later).

**The hero proposes (D Q-U4).** A data table of recommended clvl per zone (the SME's Q-S2 table, tagged UNSURE by him; the orders mock already carries a veteran range, V docs/research/mocks/orders.html) drives one point: when the hero reaches the range of the next zone and holds the kit, it proposes "Cold Plains next?" with go / stay / later. Default: go, when the kit is OK (the hero's own caution is the safe default); stay, when it is THIN or NONE. An order from the map for a zone above range opens the warning and the charter price in the order sheet (the mock's warning box, V), confirmed with tally; the hero's guard still has the last word on arrival (V orders design section 5, as built).

### 3.1 Questions in idea A (types, trigger, default, consequence)

| # | Question | Trigger | Default (if away) | Consequence shown |
|---|---|---|---|---|
| 1 | "Next: Cold Plains? (you are level 6, veterans go at 6 to 12)" go / stay / later | clvl enters the next zone's range with kit OK | go | the next card is titled by the zone; the map node turns "order" |
| 2 | "You ordered the Burial Grounds below range: charter for 30 tally and I take a full kit, or wait?" | an above-range order from the map | wait | charter spent shows on the card as a cost line; the hero's warning is journaled |
| 3 | "Blood Raven sighted. Attempt her this trip?" fight / avoid | first sight of a named boss in range (S SME Q-S1: the first real wall) | avoid below range, fight in range | the card shows "fought and kited" or "avoided", the Codex entry opens (idea B) |
| 4 | "Yield fell by 40 percent over three trips in Cold Plains. Switch to Stony Field or stay?" | t2.slowrun (V catalogue) on three cards | stay | the next three cards are compared on the return screen |
| 5 | "A rare bow dropped. Identify it with the scroll, keep it for Charsi's imbue, or sell?" | a rare or unique item picked up (S SME Q-S5: the imbue target is a real decision) | identify | the identify reveal is on the card (SME watch moment 3) |
| 6 | "Two deaths in the Den this hour. Skip the Den for now?" | t2.failstreak (V catalogue) | skip after 4 (today's edge) | the Den node reads "skipped by you"; the policy sheet's zone-skip list gains a row |
| 7 | "Merc for hire at 400 gold. Hire now or save for potions?" | t2.mercenary gold floor (V catalogue; Rogue is free after Blood Raven, V ADR 0006) | wait | the card shows merc alive share; a subsidy perk appears in the spend sheet |
| 8 | "Make this a rule?" (any of 1 to 7 answered the same N times) | the ladder's streak (V ADR 0016 item 12) | no (unlearned) | the Rules panel gains a line; later cards say "by your rule" |

Interesting: 1, 2, 3, 5, 6. Chore-adjacent, kept because they carry a trade-off: 4, 7. Everything else from the catalogue (repair, restock, potion tier, return, town trigger, tp lost, stash full, identify of magic items) moves to the policy sheet and stops being a card.

### 3.2 The policy sheet (D Q-U3: set once, change any time, suboptimal allowed, consequences visible)

One page, grouped, every row a fixed choice (never a typed number), each row with a one-line "what this does" and a "since you set this" effect line computed from the cards after the change. The sheet is a first-class experimentation surface: an odd choice is allowed, journaled as an experiment ("14:02 you set the belt to all mana potions"), and the return screen compares the cards before and after.

| Group | Rows (choices) | Effect line shows |
|---|---|---|
| Belt and potions | belt mix (balanced / mostly healing / all mana / all healing); potion tier (best affordable / cheapest sufficient, V t1.potions.buy); how many healing potions is enough (2 / 3 / 4; S SME: 3 is a fair belt) | potions drunk per trip, deaths with an empty belt, gold spent on potions |
| Town trips | go to town when the pack is (60 / 80 / 95 percent) full or the belt is low (V t1.town.trigger bands); rest in town between trips (no / 1 min) | trip length, pack full events, time in town |
| Loot | pick up whites (no / only socketed and ethereal / yes); pick up magic (sell value over X / all / none); keep runes and gems (all, the default; S SME); keep charms and jewels (all / ask) | gold from sales, pack full rate, items kept |
| Repairs and identify | repair under (20 / 30 / 50 percent, V t1.repair bands); identify (scroll / Cain / leave magic unidentified) | gold on repairs, items left unidentified |
| Risk appetite | retreat at (cautious / normal / bold: today's drinklife bands 55-64, 26-49, 18-25 as the drink and leave lines, V catalogue); fight champion packs (always / only above 60 percent life / never); two packs at once (break away / fight; S SME Q-S7) | deaths, close calls (minimum life per trip), kills per trip |
| Zones | skip list (Den, Caves, any dungeon; S SME Q-S8 "zones to skip"); dungeon depth (first level only / all levels) | time per zone, deaths per zone |
| Merc | hire when affordable / never / keep alive first (revive before potions) | merc alive share, gold on the merc |
| Gold | priority: potions first / save for the merc / save for gambling (gambling greyed: "not worth it before clvl 25", S SME Q-S4) | gold curve on the return screen |

The learning path: the policy rows are the same data the ladder's chore points key on (V internal/decide/catalogue.json fields: belt, tripgold, packused, durability, itemkind, drinklife), so a sheet row writes the same rule the streak would have learned, with `source: policy` in the Rules panel. The hero's "why did you do that" (V `/api/why`) answers "your policy: all mana potions".

### 3.3 The expedition card (the counter beat)

One card per town trip, newest first, on the return screen and the live view: zone and length; cost (potions and scrolls used, repair gold, charter tally if any); return (kills, XP and XP/h, gold delta, items kept with the best one named, firsts, waypoint lit); the closest call (minimum life percent, with whether a potion and a portal were in hand: SME watch moment 1); deaths with cause; tally credited and why; one line of the hero's voice ("Blood Raven's archers kept me honest"). Journal lines already partition by town trip (V watch design section 1: the Hub's order and uptime events; U that a clean trip boundary can be read from today's events; the state machine has town and field states, V internal/bot).

### 3.4 Tally: sources, sinks, prices (first slice, no item; D Q-U7)

Credit in the existing ledger (V docs/currency.md: append-only, hash-chained, `credit` and `grant` kinds, per-source daily caps, vanilla mode refuses): kills (1 per 25 kills, from the kill events the Hub already counts, V t0-slice "kills from kill events"), a first (5, idea B), a quest done (10, V `EvQuest` events), a waypoint lit (10, U whether the bot emits one; the waypoint table is readable, V d2-1.14d-memory 2.10 and `dump quests` waypoint masks on d2d, V internal/d2dsrc/quests.go), a named boss first killed (15), an expedition completed as ordered (2). Expected in hour one at 1x (U, from the SME's pacing table and about 400 kills per hour, which is the lockstep 1,500 per hour at about 4x, V docs/research/metacurrency-design.md section 2.3): about 16 from kills, 25 to 40 from firsts, 10 from the Den, 10 from the Cold Plains waypoint: 60 to 75 tally in the first hour, then 30 to 40 per hour. Prices (U, geometric ratio about 2.4 as the metacurrency design sketched): charter 20, 40, 80 by how far above range; order slot 2 at 30, slot 3 at 75, slots 4 and 5 at 180 and 400; first perk 50, then 120, 300; merc subsidy 60 per use; rule skip 15. So the first spend lands in minutes 20 to 40 (a charter or the second slot), the first perk at about 1.5 hours, which is the dev.to "first automation" beat stretched to D2's pace. All numbers are registry rows (V `internal/currency/registry.json` schema has `upgrades` with `costs`), tuned from session records.

Item version, costed separately: the Raven's Mark stackable (overlay Misc row, bank by cube recipe, credit by observed pack loss) is 4 h of rows plus a 6 to 10 h d2d probe plus about 10 h of bank step and pickit (V metacurrency-design section 5), d.idle mode only; on game.exe it needs a `-direct -txt` mod-data layer (S supported in 1.14d single player, V docs/sme/builds-open-questions.md line 55; strings and `.tbl` U), the user's go, and 2 to 3 days (U). Build it only if the first sessions say the visible drop matters (session question: "did you notice when tally came in?").

### 3.5 Where it lives, cost, backends

Lever 1 throughout (bot rules, catalogue data, ledger, UI); no engine change; identical on d2d and game.exe. Pieces: expedition card builder over the journal and Hub events (10 h, U), proposal point and the zone range table (6 h), charter and price in the order sheet plus the ledger spend (8 h), policy sheet page and its mapping to rules (12 h), spend sheet (6 h), perks as knob rows (4 h for the first three, each a behaviour the bot already has behind a knob; a perk that needs new bot behaviour is a bot card). About 46 agent-hours (U), no fork work.

### 3.6 The return screen (D Q-U5: return screen first)

Opens on every return (and is the `/watch` landing when the hero ran for more than 20 minutes without a view). Top to bottom:
1. Headline: "Away 4 h 12 min. Sorra is level 11 in the Dark Wood. 3 expeditions, 1 death."
2. The five moments (S SME, additional (d)), one tile each, only when they happened: the closest call (life percent, potion and portal state, zone, what the hero did); the named boss (fought, kited or avoided, with the Codex entry); the best drop with its identify reveal; the level-up and what it unlocked (a skill by quest or find, idea B); the stats line that moved (gold, XP/h, deaths, zones lit on the small map).
3. Expedition cards, newest first (section 3.3).
4. Proposals waiting: "Stony Field next?" with the hero's default already applied ("I went; say stop if you disagree").
5. Questions missed, grouped by kind, each with "answer once for all" and "make this a rule" (V direction-review H2).
6. Tally earned and the next three things it can buy (dev.to: show two or three, not twenty).
7. Policy effects: for each row changed in the last day, the before and after cards side by side.
8. One text box: "three sentences about this session" (the session record of direction-review section 2 item 3).

### 3.7 Cheapest prototype and test

Day 1: a post-processor that groups an existing soak's journal into expedition cards and renders the return screen as a static HTML mock with real data (like orders.html, V), with a fake tally ledger credited by the rules in 3.4. Day 2: the policy sheet as a static mock whose rows are the catalogue's chore points, plus the proposal point as a catalogue variant. Test with the user: the H2 session of the direction review (a 4-hour unattended realtime run, then read the return screen; count rules made in five minutes and whether he leaves the hero running tonight). Test with a friend through the tester container (D ADR 0017 D3): the return screen replaces `/watch` as the landing page; the report bundle (task testerreport) carries the session record and the three sentences. Kill criteria: the user cannot say what to do with tally after reading the return screen; fewer than one real question per expedition; the policy sheet is set once and never touched in two sessions (then it is config, not a game); expedition cards read the same after the third one.

## 4. Idea B: the Act 1 Codex

**Pitch.** The hero's first meetings with Act 1 are the content. Each first (a monster kind, a named boss, a shrine, an item kind, a waypoint, a quest NPC) fires one discovery question, teaches one rule and fills a codex entry. Skills unlock by quest or find, never by currency (D Q-U9; S SME: fine only if it reads as "the hero has not learned this yet", better gated by quest, tome, shrine or boss kill). The codex and the rules list are the number-go-up; new zones bring new entries, so the Codex pulls the player along the map.

**Act 1 content arc by zone** (S SME Q-S1, Q-S2 for the clvl and minutes, his UNSURE tag kept on the area levels; hour-1 beats marked):

| Zone | clvl (S) | Minutes at 1x (S) | Firsts that fire | Skill unlock by quest or find (U design) |
|---|---|---|---|---|
| Blood Moor | 1 to 3 | 10 | Fallen pack; the Shaman reviving the dead (hour-1 beat); Quill Rats chip damage; a shrine; first gold, first potion | starting skill only |
| Den of Evil | 3 to 6 | 15 | Corpsefire (hour-1 beat, kite); running out of mana in a cave (S Q-S7); a cave with no waypoint | the Den's skill point from Akara: the first unlock, about minute 15 to 25 (V ADR 0004) |
| Cold Plains | 5 to 7 | 10 | Bishibosh (hour-1 beat); the Cold Plains waypoint lit (hour-1 beat, S: feels like a milestone); first champion pack | a second skill opens when the waypoint is lit (U: a plan-table row "unlock at waypoint 1") |
| Burial Grounds, Crypt, Mausoleum | 6 to 8 | 12 | Blood Raven (hour-1 wall, S: fire arrows and raised skeletons); skeleton archers; the free Rogue (V ADR 0006) | merc hired: the merc row of the policy sheet opens |
| Stony Field | 7 to 9 | 10 | Rakanishu (lightning enchanted); the Cairn Stones; the Stony Field waypoint | Static Field or Frost Nova opens on Rakanishu's death (U design; S SME Q-S8: "when to take Static Field" is a thing the human tells the hero) |
| Cave | 8 to 10 | 15 | tight corridors, two packs at once (S Q-S7) | none |
| Dark Wood | 9 to 11 | 12 | the Tree of Inifuss and its scroll; the Dark Wood waypoint | none |
| Tristram (from the Cairn Stones) | 10 to 12 | U | Griswold; rescuing Cain (S: needs the Scroll of Inifuss decoded by Akara; UNVERIFIED exact sequence) | Cain identifies for free: the identify row of the sheet gains "Cain" |
| Black Marsh and the Forgotten Tower | 11 to 14 | 20 | the Black Marsh waypoint (S: a milestone, shorter trips); the Countess and the first runes El to Tir (S) | a skill opens on the first rune (U design) |
| Tamoe Highland, Pit | 13 to 14 | 8 | Goatmen (S UNSURE) | none |
| Monastery to Catacombs 4 | 14 to 17 | 35 | Outer Cloister waypoint (S: a relief); Jail, Inner Cloister, Catacombs waypoints | Teleport at 18 (S Q-S8) opens in the Catacombs (U design) |
| Andariel | 17 to 20 | 10 | Andariel (the only quest that gates Act 2, S Q-S3); poison | Act 2: content later (V ADR 0016 item 2) |

A competent bot reaches Andariel in about 2.5 to 3.5 hours at 1x (S SME). Hour one therefore holds four beats (Shaman, Corpsefire, Bishibosh, the Cold Plains waypoint) and ends at the wall (Blood Raven): that is the first-10-minutes and first-hour curve the research asks for, made of D2's own content.

**Discovery questions (types).** Each fires once per profile, is risky or normal by the catalogue, and its answer is the rule: "First Fallen Shaman: they revive the pack. Kill shamans first?" (default yes); "Corpsefire: kite or stand?" (default kite below clvl 5, U); "First shrine: take shrines when no pack is near?" (V t0.shrine, default avoid today; proposed default take when calm); "First champion pack: fight above 60 percent life?" (V t0.unique.fight); "Cold Plains waypoint lit: start trips here from now on?" (default yes); "Blood Raven: attempt her now (level 6, veterans say 8)?" (default avoid, a charter if ordered); "First rune: keep all runes?" (default keep, S SME); "First death, by Corpsefire: avoid him until level 8?" (V t0.retreat.lethal). The Codex entry shows the monster or place, what the hero learned, the rule in force, and the count ("12 of 31 Act 1 entries").

**Skill unlocks by quest or find.** The plan table (V internal/plandata) gains an `unlock` column per skill row: `quest:den`, `waypoint:cold_plains`, `boss:rakanishu`, `item:rune`, `clvl:18`. The leveler spends a point only on rows whose unlock is met, banks the rest (Lever 1; V internal/leveler spends from the table today). The hero says "I have not learned Frost Nova yet; Rakanishu might teach me" in the Thoughts feed, which is the SME's acceptable reading. Measured runs use a plan table with every unlock met (the same rule as ADR 0016 item 18 for the rule file). Risk: banked points weaken the hero; keep the Den point and the first attack skill always unlocked; the first unlock table must keep the v3 skill order (V decisions.md 2026-11-01) reachable by Andariel.

**Where it lives, cost.** Lever 1 only: a `first` context flag in the catalogue (the store already dedupes by key, V t0-slice), the firsts list as data (monster class ids and area ids the bot reads, V d2-1.14d-memory 2.6 and 2.1; d2d `dump` and `debug nearby`, V tester plan G5), codex page (6 h), unlock column and leveler rule (8 h), cardtext for about 30 entries (4 h). About 24 agent-hours (U). Identical on both backends. Prototype: count distinct firsts per 10 minutes from an existing soak log (hours), then a catalogue variant run in the H1 session of the direction review; kill criteria: fewer than 3 firsts in the first 10 minutes at 1x, or the user says the entries read like a bestiary he did not ask for.

## 5. Idea C: Legacy and the Hall

**Pitch.** When a hero retires (after Andariel, voluntary) or dies on hardcore (later, V ADR 0016 item 9), the account keeps its tally and rules, and the next hero starts with a small visible **legacy charm** scaled by account level (D Q-U10; S SME: a small charm with +1 percent gold or a starting potion is acceptable; permanent stat points and a Paragon grind are un-D2), plus a memorial line in a **Hall** (S SME additional (b): a hall of fame, titles, memorials of dead heroes feel like D2). Known cube recipes and the codex are account-wide (S SME: known recipes feel account-wide).

**The loop.** Accrues: account tally across heroes, the Hall's lines, codex completion. Spent: nothing new; the charm's size is a root of lifetime account tally (Pecorella III: a root makes each hero push further to grow it; U the exponent, start with the square root over 100). Comes back: a slightly shorter dangerous early game for the next hero, never a skipped one (the playtest must show the reborn early game is still dangerous, V ADR 0016 item 9).

**Decisions.** Day scale: retire now (after Andariel, with the Act 1 codex full) or push into Act 2 when it exists. Hour scale on a new hero: none new; the charm is set at creation.

**Where it lives, cost, backends.** Charm: Lever 2 on d2d (overlay rows `Legacy I` to `Legacy V`, one Misc or charm row each with +life, +resists, +gold find; the overlay merges rows inside the fork's table read, V docs/overlay.md, custom-items section 3), granted at creation through `d2cheat give` as a declared SETUP write before `runstart` (Lever 3; the crutch rule is unchanged, V AGENTS.md). d2d: rows 4 h plus the Misc-code probe 6 to 10 h (V metacurrency-design section 5) plus 4 h grant tooling. game.exe: the same rows need the `-direct -txt` mod-data layer and the user's go (2 to 3 days, U; a mode-stamped save that vanilla refuses, V custom-items section 4). The Hall and account codex: Lever 1, 6 h. Prototype: the Hall and the retire flow with no charm (a journal page per hero), tested at the user's first Andariel kill; the charm waits for the overlay probe. Kill criteria: the user does not want to retire a hero that just killed Andariel (then rebirth must wait for hardcore and the charm is a hardcore-only reward).

## 6. Comparison and what was dropped

| Candidate | Verdict | Why |
|---|---|---|
| A. Expedition Ledger (expeditions, tally, charters, policy sheet, proposals, return screen) | **Keep, first** | Supplies the accrual, the spend and the gate in one shape; matches D Q-U1, Q-U2, Q-U3, Q-U4, Q-U5, Q-U7; Lever 1 only; reuses journal, kit gate, orders map, ledger |
| B. Act 1 Codex (firsts, discovery questions, skill unlocks by quest or find) | **Keep, ships first** | Fixes question density in hour one (V t0drill Result 3) with D2's own content; matches D Q-U9 and the SME's beats; catalogue data, Lever 1 |
| C. Legacy and the Hall | **Keep, later** | The only day-scale loop; matches D Q-U10 and the SME's account-wide list; waits for the overlay probe and the rebirth decision |
| Commander's Map (orders as the game) | Folded into A | Slice 1 is built (V orders design section 11); it becomes A's spend surface (charters in the order sheet) and B's progress map |
| Loot filter as a standalone minigame | Folded into A's policy sheet and question 5 | S SME Q-S5: before clvl 15 most loot decisions are chores; the real ones (imbue target, bases, +skill items) are cards; a fuller filter is a later tier (Nightmare, runewords) |
| Raven's Mark item | Deferred, costed in 3.4 | D Q-U7: ledger credit first; the item only if the visible drop proves to matter |
| Blessings (skill tree bought with tally) | **Dropped** | D Q-U9 and S SME: skills unlock by quest or find; a currency-bought tree reads as un-D2 |
| Level walls per zone | Dropped | D Q-U1: a price with a warning; S SME: hard walls feel un-D2; replaced by the proposal point and charters |
| Gambling as a sink | Dropped for Act 1 | S SME Q-S4: not worth it before clvl 25 to 30 and 20k gold; greyed on the sheet with the reason |
| Permanent stat points, Paragon | Dropped | S SME: un-D2, power creep; the legacy charm is the accepted form |
| Hero personality as its own idea | Folded into A's card voice and C's Hall | Flavour, not a loop |
| Faster answer fade as a sink | Dropped | d.idle is idle (D ADR 0017 D2): questions queue, nothing fades for the player |
| Altar object (fork) | Stays parked | The cube bank or a ledger credit does the job without a fork object (V ADR 0016 item 17) |

## 7. Pacing targets (numbers to log, never pass or fail; V ADR 0016 item 16 spirit)

| Beat | Target at 1x | Source of the target |
|---|---|---|
| First decision | before the hero moves (the policy sheet) | dev.to "meaningful decision within 5 min" (V) |
| First discovery question | within 10 minutes (Fallen Shaman) | S SME Q-S1 |
| First unlock | 15 to 25 minutes (the Den skill point) | S SME Q-S2, V ADR 0004 |
| First expedition card | 10 to 20 minutes | the town-trip cadence (V gate records, U at 1x) |
| First spend | 20 to 40 minutes (a charter or the second order slot) | section 3.4 prices |
| First milestone on the map | about 35 to 45 minutes (Cold Plains waypoint) | S SME Q-S6 |
| First wall | about 50 to 60 minutes (Blood Raven) | S SME Q-S1 |
| First perk | about 1.5 hours | section 3.4 |
| First retire (soft prestige) | 2.5 to 3.5 hours (Andariel) | S SME Q-S2; Pecorella III's 120-minute first run (V) |
| Return screen reads in | under 3 minutes for a 4-hour absence; 3 or more rules or policies set from it | V direction-review H2 bar |

## 8. Open questions

For the user (max 5): U1. Should the hero's proposal ("Cold Plains next?") default to go when the kit is OK, or always wait for you when you are watching and only default when you are away? U2. Charter prices in tally only, or tally plus a full kit (potions and scrolls) as the D2 half of the price? U3. Retiring a hero after Andariel as the first soft prestige: wanted, or does a hero only end by hardcore death? U4. Policy effect lines: per row on the sheet (always visible) or only on the return screen? U5. The first perk list: which three behaviours would you buy first (sweep the level, take the waypoint, revive the merc, bank runes, clear dungeon levels)?

For the SME (max 3): S1. Skill unlocks by event: which Sorceress skills may sensibly wait for the Cold Plains waypoint, Rakanishu, the first rune and clvl 18, so the v3 order still reaches Andariel at clvl 17 to 20? S2. Legacy charm sizes that read as Annihilus-light for Normal Act 1 (life, resists, gold find) without making Blood Raven trivial for a clvl 6 Sorceress with about 80 life at clvl 8? S3. Confirm the Tristram sequence (Cairn Stones, Scroll of Inifuss decoded by Akara, Griswold, Cain) and the Act 1 waypoint order for the content table (both UNVERIFIED here).

## 9. Proposed decision text (not written to the guard files)

For the coordinator or the user's grill session; agents do not edit docs/decisions.md, the ADRs or GLOSSARY.md.

- "2026-11-xx (user): d.idle's loop is the Expedition Ledger (docs/research/fun-ideas.md section 3): every town trip is an expedition with a report card; tally is credited in the ledger from observed events (no item in the first slice); tally buys charters, order slots, hero perks and merc upkeep; the pacing gate is a price with the hero's warning, never a wall; the hero proposes the next zone and the player may order it from the map; chores are a policy sheet, set once and changeable, whose consequences are shown."
- "2026-11-xx (user): Act 1 content is the Codex (section 4): firsts fire one discovery question each; skills unlock by quest or find from a plan-table column, never by currency; measured runs use a table with every unlock met."
- "2026-11-xx (user): account-wide carry-over is a small visible legacy charm scaled by account tally plus a Hall of heroes (section 5); permanent stat points are not built."
- Glossary candidates (not written): Expedition, Report card, Charter, Policy sheet, Codex, Legacy charm, Hall.

## 10. Not verified and dead ends

- No session has been played with any of this; every pacing number in section 7 is a target from the SME's memory and the cited articles, not a measurement.
- Clean trip boundaries from today's events (section 3.3) are U; the state machine has the states but the Hub's event mapping was not traced.
- Whether the bot emits a waypoint-lit event is U (the waypoint table is readable on both backends).
- The skill-unlock column (section 4) is a design; its interaction with the v3 skill rows and the clvl 17 Andariel floor is unchecked.
- The Spiel et al. paper was fetched but not read (no PDF text tool on this box); the Kongregate 2015 and AdVenture Capitalist lesson pages, the Melvor and Kittens wikis, the Cookie Clicker and Antimatter Dimensions wikis and the Idle Miner Tycoon deconstruction returned 402, 403 or 404 and are cited from memory only where used (section 2, tagged U).
- The Tristram quest sequence and the Act 1 waypoint order are the SME's memory (UNVERIFIED).
- Dead end: Blessings as a currency-bought skill tree (round 1 section H) was the cheapest allocation beat and is dropped on the user's and the SME's thematic judgment; the quest-or-find column keeps the beat without the currency.

## Sources

User answers Q-U1 to Q-U10 and SME answers Q-S1 to Q-S10 (relayed by the coordinator, 2026-11-07); docs/research/direction-review.md; docs/adr/0016-automation-ladder.md, 0017-direction-fun-first-2026-11-07.md, 0004, 0006, 0011; docs/decisions.md; GLOSSARY.md; docs/t0-slice.md; docs/probes/t0drill.md; internal/decide/catalogue.json; docs/research/orders-ux-design.md and mocks/orders.html; docs/research/onboarding-ux-design.md; docs/research/metacurrency-design.md; docs/research/custom-items.md; docs/currency.md; docs/overlay.md (named, not re-read); docs/research/tester-build-plan.md; docs/research/automation-ladder.md; docs/research/watch-screen-design.md; docs/d2-1.14d-memory.md; docs/sme/builds-open-questions.md (line 55, via custom-items.md); PLAN.md; the web sources of section 2.
